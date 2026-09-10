// Package receipt is the user portal's receipt business logic: upload,
// async extraction, review/confirm, images, and the account cascade.
// Collection access is common/services/receipt.
package receipt

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sort"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"smartreceipt/backend/common/pkg/extraction"
	gohttp "smartreceipt/backend/common/pkg/http"
	"smartreceipt/backend/common/pkg/models"
	"smartreceipt/backend/common/pkg/storage"
	"smartreceipt/backend/common/services/category"
	receiptrepo "smartreceipt/backend/common/services/receipt"
	"smartreceipt/backend/common/services/user"
)

// Deps are the infrastructure providers wired at boot.
type Deps struct {
	Store       storage.Provider
	Extractor   extraction.Extractor
	Pool        *extraction.Pool
	MaxUploadMB int64
}

var deps Deps

// Configure installs the providers. Call once from app.Run before serving.
func Configure(d Deps) { deps = d }

// Shutdown stops the worker pool, waiting up to ctx for in-flight jobs.
func Shutdown(ctx context.Context) {
	if deps.Pool != nil {
		deps.Pool.Shutdown(ctx)
	}
}

// AllowedTypes are accepted upload content types.
var AllowedTypes = map[string]bool{
	"image/jpeg": true, "image/png": true, "image/webp": true, "image/heic": true, "application/pdf": true,
}

// UploadFile is one incoming file.
type UploadFile struct {
	Name        string
	ContentType string
	Data        []byte
}

// UploadResult is returned per created receipt.
type UploadResult struct {
	ID        string               `json:"id"`
	Status    models.ReceiptStatus `json:"status"`
	Duplicate bool                 `json:"duplicate"`
	FileName  string               `json:"fileName"`
}

// Service orchestrates storage, extraction, and persistence.
type Service struct {
	repo  *receiptrepo.Service
	cats  *category.Service
	users *user.Service
	// OnExtracted is called after an extraction attempt finishes (notifications hook).
	OnExtracted func(ctx context.Context, rc *models.Receipt)
}

// New returns the service bound to the configured providers.
func New() *Service {
	return &Service{repo: receiptrepo.New(), cats: category.New(), users: user.New()}
}

// MaxBytes is the per-file upload limit.
func (s *Service) MaxBytes() int64 {
	if deps.MaxUploadMB <= 0 {
		return 20 << 20
	}
	return deps.MaxUploadMB << 20
}

func shortID() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func hashBytes(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

// Decorate fills computed fields (image URLs) before returning to clients.
func (s *Service) Decorate(rc *models.Receipt) *models.Receipt {
	for i := range rc.Images {
		rc.Images[i].URL = deps.Store.URL(rc.Images[i].Key)
	}
	return rc
}

// DecorateList fills thumbnails.
func (s *Service) DecorateList(rows []models.ReceiptListItem) []models.ReceiptListItem {
	for i := range rows {
		if rows[i].FirstKey != "" {
			rows[i].Thumbnail = deps.Store.URL(rows[i].FirstKey)
		}
	}
	return rows
}

func (s *Service) validate(f UploadFile) error {
	if !AllowedTypes[f.ContentType] {
		return gohttp.ErrBadRequest(fmt.Sprintf("%s: unsupported file type %s (use JPEG, PNG, WebP, HEIC or PDF)", f.Name, f.ContentType))
	}
	if int64(len(f.Data)) > s.MaxBytes() {
		return gohttp.ErrBadRequest(fmt.Sprintf("%s: file exceeds %d MB", f.Name, s.MaxBytes()>>20))
	}
	if len(f.Data) == 0 {
		return gohttp.ErrBadRequest(f.Name + ": empty file")
	}
	return nil
}

// Upload stores files and creates receipts. With group=true all files become
// pages of a single receipt; otherwise each file is its own receipt.
func (s *Service) Upload(ctx context.Context, uid bson.ObjectID, files []UploadFile, group bool) ([]UploadResult, error) {
	if len(files) == 0 {
		return nil, gohttp.ErrBadRequest("no files uploaded")
	}
	for _, f := range files {
		if err := s.validate(f); err != nil {
			return nil, err
		}
	}
	groups := [][]UploadFile{}
	if group {
		groups = append(groups, files)
	} else {
		for _, f := range files {
			groups = append(groups, []UploadFile{f})
		}
	}
	var out []UploadResult
	for _, g := range groups {
		res, err := s.createFromFiles(ctx, uid, g)
		if err != nil {
			return out, err
		}
		out = append(out, *res)
	}
	return out, nil
}

func (s *Service) createFromFiles(ctx context.Context, uid bson.ObjectID, files []UploadFile) (*UploadResult, error) {
	// Duplicate check on the combined content hash.
	var hashes []string
	for _, f := range files {
		hashes = append(hashes, hashBytes(f.Data))
	}
	combined := hashBytes([]byte(strings.Join(hashes, "|")))
	if existing, err := s.repo.FindByHash(ctx, uid, combined); err == nil {
		return &UploadResult{ID: existing.ID.Hex(), Status: existing.Status, Duplicate: true, FileName: files[0].Name}, nil
	} else if !errors.Is(err, receiptrepo.ErrNotFound) {
		return nil, err
	}

	rc := &models.Receipt{UserID: uid, Status: models.ReceiptProcessing, Currency: "INR", FileHash: combined, Images: []models.ReceiptImage{}}
	if u, err := s.users.FindByID(ctx, uid); err == nil && u.Preferences.Currency != "" {
		rc.Currency = u.Preferences.Currency
	}
	var stored []string
	for i, f := range files {
		key := fmt.Sprintf("receipts/%s/%s.%s", uid.Hex(), shortID(), storage.ExtFor(f.ContentType))
		obj, err := deps.Store.Put(ctx, key, bytes.NewReader(f.Data), f.ContentType)
		if err != nil {
			for _, k := range stored {
				_ = deps.Store.Delete(ctx, k)
			}
			return nil, fmt.Errorf("store upload: %w", err)
		}
		stored = append(stored, key)
		rc.Images = append(rc.Images, models.ReceiptImage{ID: shortID(), Key: key, ContentType: f.ContentType, Size: obj.Size, Hash: hashes[i], Order: i, FileName: f.Name})
	}
	if err := s.repo.Insert(ctx, rc); err != nil {
		for _, k := range stored {
			_ = deps.Store.Delete(ctx, k)
		}
		return nil, err
	}
	s.enqueue(uid, rc.ID, files)
	return &UploadResult{ID: rc.ID.Hex(), Status: rc.Status, FileName: files[0].Name}, nil
}

// enqueue schedules extraction using already-in-memory bytes.
func (s *Service) enqueue(uid, id bson.ObjectID, files []UploadFile) {
	exFiles := make([]extraction.File, 0, len(files))
	for _, f := range files {
		exFiles = append(exFiles, extraction.File{Data: f.Data, ContentType: f.ContentType})
	}
	deps.Pool.Submit(func(ctx context.Context) { s.runExtraction(ctx, uid, id, exFiles) })
}

// runExtraction calls the model with retries and persists the outcome.
func (s *Service) runExtraction(ctx context.Context, uid, id bson.ObjectID, files []extraction.File) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()

	cats, _ := s.cats.ListForUser(ctx, uid)
	slugs := make([]string, 0, len(cats))
	for _, c := range cats {
		slugs = append(slugs, c.Slug)
	}
	in := extraction.Input{Files: files, Categories: slugs, Currency: "INR", Timezone: "Asia/Kolkata"}
	if u, err := s.users.FindByID(ctx, uid); err == nil {
		in.Currency, in.Timezone = u.Preferences.Currency, u.Preferences.Timezone
	}

	var res *extraction.Result
	var err error
	attempts := 0
	for attempts < 3 {
		attempts++
		res, err = deps.Extractor.Extract(ctx, in)
		if err == nil || ctx.Err() != nil {
			break
		}
		slog.Warn("extraction attempt failed", "receipt", id.Hex(), "attempt", attempts, "err", err)
		select {
		case <-time.After(time.Duration(attempts*attempts) * time.Second):
		case <-ctx.Done():
		}
	}
	now := time.Now().UTC()
	if err != nil {
		_ = s.repo.Update(ctx, uid, id, bson.M{
			"status":    models.ReceiptFailed,
			"extracted": models.Extracted{Model: deps.Extractor.Name(), ExtractedAt: now, Error: err.Error(), Attempts: attempts},
		})
		s.notify(uid, id)
		return
	}

	set := bson.M{
		"status": models.ReceiptReview, "merchant": res.Merchant, "merchantNormalized": models.NormalizeMerchant(res.Merchant),
		"date": res.Date, "time": res.Time, "currency": res.Currency, "total": res.Total,
		"paymentMethod": res.PaymentMethod, "gstin": res.GSTIN, "taxBreakdown": res.TaxBreakdown,
		"extracted": models.Extracted{
			Model: res.Model, Confidence: res.Confidence, SuggestedCategory: res.SuggestedCategory,
			WarrantyCandidates: res.WarrantyCandidates, ExtractedAt: now, Attempts: attempts,
		},
	}
	if res.Subtotal != nil {
		set["subtotal"] = *res.Subtotal
	}
	if res.Tax != nil {
		set["tax"] = *res.Tax
	}
	items := make([]models.ReceiptItem, 0, len(res.Items))
	for _, it := range res.Items {
		items = append(items, models.ReceiptItem{ID: shortID(), Name: it.Name, Quantity: it.Quantity, UnitPrice: it.UnitPrice, Amount: it.Amount})
	}
	set["items"] = items
	if res.Date == "" {
		set["date"] = now.Format("2006-01-02")
	}
	slug := res.SuggestedCategory
	if slug == "" {
		slug = models.OthersSlug
	}
	if c, err := s.cats.BySlug(ctx, uid, slug); err == nil {
		set["categoryId"] = c.ID
	} else if c, err := s.cats.BySlug(ctx, uid, models.OthersSlug); err == nil {
		set["categoryId"] = c.ID
	}
	if err := s.repo.Update(ctx, uid, id, set); err != nil {
		slog.Error("persist extraction", "receipt", id.Hex(), "err", err)
		return
	}
	s.notify(uid, id)
}

func (s *Service) notify(uid, id bson.ObjectID) {
	if s.OnExtracted == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if rc, err := s.repo.Get(ctx, uid, id); err == nil {
		s.OnExtracted(ctx, rc)
	}
}

// Reextract re-runs the model on the stored files.
func (s *Service) Reextract(ctx context.Context, uid, id bson.ObjectID) (*models.Receipt, error) {
	rc, err := s.repo.Get(ctx, uid, id)
	if err != nil {
		return nil, mapErr(err)
	}
	if len(rc.Images) == 0 {
		return nil, gohttp.ErrBadRequest("receipt has no images")
	}
	files, err := s.loadFiles(ctx, rc)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, uid, id, bson.M{"status": models.ReceiptProcessing}); err != nil {
		return nil, mapErr(err)
	}
	deps.Pool.Submit(func(ctx context.Context) { s.runExtraction(ctx, uid, id, files) })
	rc.Status = models.ReceiptProcessing
	return s.Decorate(rc), nil
}

func (s *Service) loadFiles(ctx context.Context, rc *models.Receipt) ([]extraction.File, error) {
	imgs := append([]models.ReceiptImage(nil), rc.Images...)
	sort.Slice(imgs, func(i, j int) bool { return imgs[i].Order < imgs[j].Order })
	var files []extraction.File
	for _, im := range imgs {
		rcl, ct, err := deps.Store.Get(ctx, im.Key)
		if err != nil {
			return nil, fmt.Errorf("load %s: %w", im.Key, err)
		}
		data, err := io.ReadAll(io.LimitReader(rcl, s.MaxBytes()+1))
		rcl.Close()
		if err != nil {
			return nil, err
		}
		if ct == "" || ct == "application/octet-stream" {
			ct = im.ContentType
		}
		files = append(files, extraction.File{Data: data, ContentType: ct})
	}
	return files, nil
}

// Patch holds editable fields. Nil pointers are left untouched.
type Patch struct {
	Merchant      *string               `json:"merchant" validate:"omitempty,max=120"`
	Date          *string               `json:"date" validate:"omitempty,datetime=2006-01-02"`
	Time          *string               `json:"time" validate:"omitempty,max=8"`
	CategoryID    *string               `json:"categoryId"`
	Currency      *string               `json:"currency" validate:"omitempty,len=3,alpha"`
	Subtotal      *int64                `json:"subtotal" validate:"omitempty,gte=0"`
	Tax           *int64                `json:"tax" validate:"omitempty,gte=0"`
	Total         *int64                `json:"total" validate:"omitempty,gte=0"`
	PaymentMethod *string               `json:"paymentMethod"`
	GSTIN         *string               `json:"gstin" validate:"omitempty,max=20"`
	Items         *[]models.ReceiptItem `json:"items" validate:"omitempty,dive"`
	Notes         *string               `json:"notes" validate:"omitempty,max=2000"`
	Tags          *[]string             `json:"tags" validate:"omitempty,max=20,dive,min=1,max=30"`
}

func (s *Service) applyPatch(ctx context.Context, uid bson.ObjectID, p Patch) (bson.M, error) {
	set := bson.M{}
	if p.Merchant != nil {
		set["merchant"] = strings.TrimSpace(*p.Merchant)
	}
	if p.Date != nil {
		set["date"] = *p.Date
	}
	if p.Time != nil {
		set["time"] = *p.Time
	}
	if p.CategoryID != nil {
		cid, err := bson.ObjectIDFromHex(*p.CategoryID)
		if err != nil {
			return nil, gohttp.ErrValidation(map[string]string{"categoryId": "is invalid"})
		}
		if _, err := s.cats.Get(ctx, uid, cid); err != nil {
			return nil, gohttp.ErrValidation(map[string]string{"categoryId": "not found"})
		}
		set["categoryId"] = cid
	}
	if p.Currency != nil {
		set["currency"] = strings.ToUpper(*p.Currency)
	}
	if p.Subtotal != nil {
		set["subtotal"] = *p.Subtotal
	}
	if p.Tax != nil {
		set["tax"] = *p.Tax
	}
	if p.Total != nil {
		set["total"] = *p.Total
	}
	if p.PaymentMethod != nil {
		if *p.PaymentMethod != "" && !models.PaymentMethods[*p.PaymentMethod] {
			return nil, gohttp.ErrValidation(map[string]string{"paymentMethod": "is invalid"})
		}
		set["paymentMethod"] = *p.PaymentMethod
	}
	if p.GSTIN != nil {
		set["gstin"] = strings.ToUpper(strings.TrimSpace(*p.GSTIN))
	}
	if p.Items != nil {
		items := make([]models.ReceiptItem, 0, len(*p.Items))
		for _, it := range *p.Items {
			if strings.TrimSpace(it.Name) == "" {
				continue
			}
			if it.ID == "" {
				it.ID = shortID()
			}
			it.Name = strings.TrimSpace(it.Name)
			items = append(items, it)
		}
		set["items"] = items
	}
	if p.Notes != nil {
		set["notes"] = strings.TrimSpace(*p.Notes)
	}
	if p.Tags != nil {
		seen := map[string]bool{}
		tags := []string{}
		for _, t := range *p.Tags {
			t = strings.TrimSpace(t)
			if t != "" && !seen[strings.ToLower(t)] {
				seen[strings.ToLower(t)] = true
				tags = append(tags, t)
			}
		}
		set["tags"] = tags
	}
	return set, nil
}

// Update edits fields without changing status.
func (s *Service) Update(ctx context.Context, uid, id bson.ObjectID, p Patch) (*models.Receipt, error) {
	set, err := s.applyPatch(ctx, uid, p)
	if err != nil {
		return nil, err
	}
	if len(set) == 0 {
		return nil, gohttp.ErrBadRequest("nothing to update")
	}
	if err := s.repo.Update(ctx, uid, id, set); err != nil {
		return nil, mapErr(err)
	}
	return s.Get(ctx, uid, id)
}

// Confirm applies an optional patch and marks the receipt confirmed.
func (s *Service) Confirm(ctx context.Context, uid, id bson.ObjectID, p Patch) (*models.Receipt, error) {
	rc, err := s.repo.Get(ctx, uid, id)
	if err != nil {
		return nil, mapErr(err)
	}
	if rc.Status == models.ReceiptProcessing {
		return nil, gohttp.ErrConflict("still_processing", "extraction is still running")
	}
	set, err := s.applyPatch(ctx, uid, p)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	set["status"] = models.ReceiptConfirmed
	set["confirmedAt"] = now
	if _, ok := set["date"]; !ok && rc.Date == "" {
		set["date"] = now.Format("2006-01-02")
	}
	if _, ok := set["categoryId"]; !ok && rc.CategoryID == nil {
		if c, err := s.cats.BySlug(ctx, uid, models.OthersSlug); err == nil {
			set["categoryId"] = c.ID
		}
	}
	if err := s.repo.Update(ctx, uid, id, set); err != nil {
		return nil, mapErr(err)
	}
	return s.Get(ctx, uid, id)
}

// Get loads and decorates.
func (s *Service) Get(ctx context.Context, uid, id bson.ObjectID) (*models.Receipt, error) {
	rc, err := s.repo.Get(ctx, uid, id)
	if err != nil {
		return nil, mapErr(err)
	}
	return s.Decorate(rc), nil
}

// List pages receipts.
func (s *Service) List(ctx context.Context, uid bson.ObjectID, f receiptrepo.ListFilter) ([]models.ReceiptListItem, int64, error) {
	rows, total, err := s.repo.List(ctx, uid, f)
	if err != nil {
		return nil, 0, err
	}
	return s.DecorateList(rows), total, nil
}

// Recent returns the newest uploads for the upload screen.
func (s *Service) Recent(ctx context.Context, uid bson.ObjectID, n int) ([]models.Receipt, error) {
	rows, err := s.repo.Recent(ctx, uid, n)
	if err != nil {
		return nil, err
	}
	for i := range rows {
		s.Decorate(&rows[i])
	}
	return rows, nil
}

// Delete removes the receipt and its files.
func (s *Service) Delete(ctx context.Context, uid, id bson.ObjectID) error {
	rc, err := s.repo.Get(ctx, uid, id)
	if err != nil {
		return mapErr(err)
	}
	if err := s.repo.Delete(ctx, uid, id); err != nil {
		return mapErr(err)
	}
	for _, im := range rc.Images {
		if err := deps.Store.Delete(ctx, im.Key); err != nil {
			slog.Warn("delete image", "key", im.Key, "err", err)
		}
	}
	return nil
}

// AddImages appends pages to an existing receipt.
func (s *Service) AddImages(ctx context.Context, uid, id bson.ObjectID, files []UploadFile) (*models.Receipt, error) {
	rc, err := s.repo.Get(ctx, uid, id)
	if err != nil {
		return nil, mapErr(err)
	}
	if len(rc.Images)+len(files) > 10 {
		return nil, gohttp.ErrBadRequest("a receipt can have at most 10 images")
	}
	for _, f := range files {
		if err := s.validate(f); err != nil {
			return nil, err
		}
	}
	imgs := rc.Images
	for _, f := range files {
		key := fmt.Sprintf("receipts/%s/%s.%s", uid.Hex(), shortID(), storage.ExtFor(f.ContentType))
		obj, err := deps.Store.Put(ctx, key, bytes.NewReader(f.Data), f.ContentType)
		if err != nil {
			return nil, fmt.Errorf("store upload: %w", err)
		}
		imgs = append(imgs, models.ReceiptImage{ID: shortID(), Key: key, ContentType: f.ContentType, Size: obj.Size, Hash: hashBytes(f.Data), Order: len(imgs), FileName: f.Name})
	}
	if err := s.repo.Update(ctx, uid, id, bson.M{"images": imgs}); err != nil {
		return nil, mapErr(err)
	}
	return s.Get(ctx, uid, id)
}

// RemoveImage deletes one page.
func (s *Service) RemoveImage(ctx context.Context, uid, id bson.ObjectID, imageID string) (*models.Receipt, error) {
	rc, err := s.repo.Get(ctx, uid, id)
	if err != nil {
		return nil, mapErr(err)
	}
	var keep []models.ReceiptImage
	var removed *models.ReceiptImage
	for _, im := range rc.Images {
		if im.ID == imageID {
			cp := im
			removed = &cp
			continue
		}
		keep = append(keep, im)
	}
	if removed == nil {
		return nil, gohttp.ErrNotFound
	}
	if keep == nil {
		keep = []models.ReceiptImage{}
	}
	for i := range keep {
		keep[i].Order = i
	}
	if err := s.repo.Update(ctx, uid, id, bson.M{"images": keep}); err != nil {
		return nil, mapErr(err)
	}
	_ = deps.Store.Delete(ctx, removed.Key)
	return s.Get(ctx, uid, id)
}

// DeleteAllForUser removes receipts and files (account deletion cascade).
func (s *Service) DeleteAllForUser(ctx context.Context, uid bson.ObjectID) error {
	keys, err := s.repo.AllImageKeys(ctx, uid)
	if err != nil {
		return err
	}
	if err := s.repo.DeleteAllForUser(ctx, uid); err != nil {
		return err
	}
	for _, k := range keys {
		_ = deps.Store.Delete(ctx, k)
	}
	return nil
}

// ReassignCategory moves receipts between categories (category deletion).
func (s *Service) ReassignCategory(ctx context.Context, uid, from, to bson.ObjectID) (int64, error) {
	return s.repo.ReassignCategory(ctx, uid, from, to)
}

func mapErr(err error) error {
	if errors.Is(err, receiptrepo.ErrNotFound) {
		return gohttp.ErrNotFound
	}
	return err
}

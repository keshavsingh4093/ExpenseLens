// Package warranty is the user portal's warranty logic: create/list/update
// with a derived status, receipt linking, and the 30/7-day reminders.
// Collection access is common/services/warranty.
package warranty

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	gohttp "smartreceipt/backend/common/pkg/http"
	"smartreceipt/backend/common/pkg/models"
	"smartreceipt/backend/common/pkg/storage"
	receiptrepo "smartreceipt/backend/common/services/receipt"
	"smartreceipt/backend/common/services/user"
	warrantyrepo "smartreceipt/backend/common/services/warranty"

	"smartreceipt/backend/user/internal/services/analytics"
	"smartreceipt/backend/user/internal/services/notification"
)

var store storage.Provider

// Configure installs the storage provider used for image URLs.
func Configure(s storage.Provider) { store = s }

// ExpiringDays is the "expiring soon" window.
const ExpiringDays = 30

// Input is the create/update payload. Nil pointers are left untouched on update.
type Input struct {
	ProductName    *string `json:"productName" validate:"omitempty,min=1,max=120"`
	PurchaseDate   *string `json:"purchaseDate" validate:"omitempty,datetime=2006-01-02"`
	ExpiryDate     *string `json:"expiryDate" validate:"omitempty,datetime=2006-01-02"`
	DurationMonths *int    `json:"durationMonths" validate:"omitempty,gte=0,lte=600"`
	ReceiptID      *string `json:"receiptId"`
	ItemID         *string `json:"itemId" validate:"omitempty,max=40"`
	Notes          *string `json:"notes" validate:"omitempty,max=2000"`
}

// Service is the warranty domain logic.
type Service struct {
	repo     *warrantyrepo.Service
	receipts *receiptrepo.Service
	users    *user.Service
	notify   *notification.Service
}

// New returns the service.
func New() *Service {
	return &Service{repo: warrantyrepo.New(), receipts: receiptrepo.New(), users: user.New(), notify: notification.New()}
}

func (s *Service) today(ctx context.Context, uid bson.ObjectID) string {
	tz := "Asia/Kolkata"
	if u, err := s.users.FindByID(ctx, uid); err == nil && u.Preferences.Timezone != "" {
		tz = u.Preferences.Timezone
	}
	return analytics.Now(tz).Format(analytics.Day)
}

// Derive fills Status and DaysLeft relative to today (YYYY-MM-DD).
func Derive(w *models.Warranty, today string) {
	t, _ := time.Parse(analytics.Day, today)
	e, err := time.Parse(analytics.Day, w.ExpiryDate)
	if err != nil {
		w.Status, w.DaysLeft = models.WarrantyActive, 0
		return
	}
	w.DaysLeft = int(e.Sub(t).Hours() / 24)
	switch {
	case w.DaysLeft < 0:
		w.Status = models.WarrantyExpired
	case w.DaysLeft <= ExpiringDays:
		w.Status = models.WarrantyExpiring
	default:
		w.Status = models.WarrantyActive
	}
}

func (s *Service) decorate(w *models.Warranty, today string) *models.Warranty {
	Derive(w, today)
	if w.ImageKey != "" && store != nil {
		w.ImageURL = store.URL(w.ImageKey)
	}
	return w
}

// resolveReceipt verifies ownership and returns the receipt.
func (s *Service) resolveReceipt(ctx context.Context, uid bson.ObjectID, hexID string) (*models.Receipt, error) {
	rid, err := bson.ObjectIDFromHex(hexID)
	if err != nil {
		return nil, gohttp.ErrValidation(map[string]string{"receiptId": "is invalid"})
	}
	rc, err := s.receipts.Get(ctx, uid, rid)
	if err != nil {
		if errors.Is(err, receiptrepo.ErrNotFound) {
			return nil, gohttp.ErrValidation(map[string]string{"receiptId": "not found"})
		}
		return nil, err
	}
	return rc, nil
}

func addMonths(date string, months int) string {
	t, err := time.Parse(analytics.Day, date)
	if err != nil {
		return ""
	}
	return t.AddDate(0, months, 0).Format(analytics.Day)
}

// Create stores a warranty. Expiry defaults to purchase + durationMonths.
func (s *Service) Create(ctx context.Context, uid bson.ObjectID, in Input) (*models.Warranty, error) {
	fields := map[string]string{}
	w := &models.Warranty{UserID: uid}
	if in.ProductName == nil || strings.TrimSpace(*in.ProductName) == "" {
		fields["productName"] = "is required"
	} else {
		w.ProductName = strings.TrimSpace(*in.ProductName)
	}
	if in.PurchaseDate == nil || *in.PurchaseDate == "" {
		fields["purchaseDate"] = "is required"
	} else {
		w.PurchaseDate = *in.PurchaseDate
	}
	if in.DurationMonths != nil {
		w.DurationMonths = *in.DurationMonths
	}
	switch {
	case in.ExpiryDate != nil && *in.ExpiryDate != "":
		w.ExpiryDate = *in.ExpiryDate
	case w.DurationMonths > 0 && w.PurchaseDate != "":
		w.ExpiryDate = addMonths(w.PurchaseDate, w.DurationMonths)
	default:
		fields["expiryDate"] = "expiry date or duration is required"
	}
	if w.ExpiryDate != "" && w.PurchaseDate != "" && w.ExpiryDate < w.PurchaseDate {
		fields["expiryDate"] = "must be after the purchase date"
	}
	if in.Notes != nil {
		w.Notes = strings.TrimSpace(*in.Notes)
	}
	if len(fields) > 0 {
		return nil, gohttp.ErrValidation(fields)
	}
	if in.ReceiptID != nil && *in.ReceiptID != "" {
		rc, err := s.resolveReceipt(ctx, uid, *in.ReceiptID)
		if err != nil {
			return nil, err
		}
		w.ReceiptID = &rc.ID
		if len(rc.Images) > 0 {
			w.ImageKey = rc.Images[0].Key
		}
		if in.ItemID != nil {
			w.ItemID = *in.ItemID
		}
	}
	if err := s.repo.Insert(ctx, w); err != nil {
		return nil, err
	}
	return s.decorate(w, s.today(ctx, uid)), nil
}

// List returns the user's warranties, optionally filtered by derived status.
func (s *Service) List(ctx context.Context, uid bson.ObjectID, status string) ([]models.Warranty, map[string]int, error) {
	if status != "" && status != models.WarrantyActive && status != models.WarrantyExpiring && status != models.WarrantyExpired {
		return nil, nil, gohttp.ErrBadRequest("unknown status " + status)
	}
	rows, err := s.repo.List(ctx, uid)
	if err != nil {
		return nil, nil, err
	}
	today := s.today(ctx, uid)
	counts := map[string]int{models.WarrantyActive: 0, models.WarrantyExpiring: 0, models.WarrantyExpired: 0}
	out := []models.Warranty{}
	for i := range rows {
		s.decorate(&rows[i], today)
		counts[rows[i].Status]++
		if status == "" || rows[i].Status == status {
			out = append(out, rows[i])
		}
	}
	return out, counts, nil
}

// Get loads one warranty.
func (s *Service) Get(ctx context.Context, uid, id bson.ObjectID) (*models.Warranty, error) {
	w, err := s.repo.Get(ctx, uid, id)
	if err != nil {
		return nil, mapErr(err)
	}
	return s.decorate(w, s.today(ctx, uid)), nil
}

// Update patches a warranty.
func (s *Service) Update(ctx context.Context, uid, id bson.ObjectID, in Input) (*models.Warranty, error) {
	cur, err := s.repo.Get(ctx, uid, id)
	if err != nil {
		return nil, mapErr(err)
	}
	set := bson.M{}
	var unset []string
	fields := map[string]string{}
	if in.ProductName != nil {
		if strings.TrimSpace(*in.ProductName) == "" {
			fields["productName"] = "is required"
		}
		set["productName"] = strings.TrimSpace(*in.ProductName)
		cur.ProductName = strings.TrimSpace(*in.ProductName)
	}
	if in.PurchaseDate != nil {
		set["purchaseDate"] = *in.PurchaseDate
		cur.PurchaseDate = *in.PurchaseDate
	}
	if in.DurationMonths != nil {
		set["durationMonths"] = *in.DurationMonths
		cur.DurationMonths = *in.DurationMonths
	}
	switch {
	case in.ExpiryDate != nil && *in.ExpiryDate != "":
		cur.ExpiryDate = *in.ExpiryDate
	case (in.DurationMonths != nil || in.PurchaseDate != nil) && cur.DurationMonths > 0:
		cur.ExpiryDate = addMonths(cur.PurchaseDate, cur.DurationMonths)
	}
	set["expiryDate"] = cur.ExpiryDate
	if cur.ExpiryDate < cur.PurchaseDate {
		fields["expiryDate"] = "must be after the purchase date"
	}
	if in.Notes != nil {
		set["notes"] = strings.TrimSpace(*in.Notes)
	}
	if len(fields) > 0 {
		return nil, gohttp.ErrValidation(fields)
	}
	if in.ReceiptID != nil {
		if *in.ReceiptID == "" {
			unset = append(unset, "receiptId", "itemId", "imageKey")
		} else {
			rc, err := s.resolveReceipt(ctx, uid, *in.ReceiptID)
			if err != nil {
				return nil, err
			}
			set["receiptId"] = rc.ID
			if len(rc.Images) > 0 {
				set["imageKey"] = rc.Images[0].Key
			}
			if in.ItemID != nil {
				set["itemId"] = *in.ItemID
			}
		}
	} else if in.ItemID != nil {
		set["itemId"] = *in.ItemID
	}
	// Expiry moved: allow reminders to fire again for the new date.
	if cur.ExpiryDate != "" {
		unset = append(unset, "remindersSent")
	}
	if err := s.repo.Update(ctx, uid, id, set, unset...); err != nil {
		return nil, mapErr(err)
	}
	return s.Get(ctx, uid, id)
}

// Delete removes a warranty.
func (s *Service) Delete(ctx context.Context, uid, id bson.ObjectID) error {
	return mapErr(s.repo.Delete(ctx, uid, id))
}

// ListAllForUser returns every warranty (export).
func (s *Service) ListAllForUser(ctx context.Context, uid bson.ObjectID) ([]models.Warranty, error) {
	rows, err := s.repo.List(ctx, uid)
	if err != nil {
		return nil, err
	}
	today := s.today(ctx, uid)
	for i := range rows {
		s.decorate(&rows[i], today)
	}
	return rows, nil
}

// DeleteAllForUser removes every warranty (account cascade).
func (s *Service) DeleteAllForUser(ctx context.Context, uid bson.ObjectID) error {
	return s.repo.DeleteAllForUser(ctx, uid)
}

// SendReminders emits warranty_expiring notifications for warranties
// expiring within 30 days (tag "30") and 7 days (tag "7"), once each.
// today is the UTC date; returns the number of notifications created.
func (s *Service) SendReminders(ctx context.Context, today string) int {
	t, _ := time.Parse(analytics.Day, today)
	rows, err := s.repo.ExpiringBetween(ctx, today, t.AddDate(0, 0, ExpiringDays).Format(analytics.Day))
	if err != nil {
		return 0
	}
	prefs := map[string]bool{}
	sent := 0
	for i := range rows {
		w := &rows[i]
		key := w.UserID.Hex()
		allowed, seen := prefs[key]
		if !seen {
			u, err := s.users.FindByID(ctx, w.UserID)
			allowed = err == nil && u.Preferences.Notifications.WarrantyReminders
			prefs[key] = allowed
		}
		if !allowed {
			continue
		}
		Derive(w, today)
		tag := "30"
		if w.DaysLeft <= 7 {
			tag = "7"
		}
		already := false
		for _, r := range w.RemindersSent {
			if r == tag {
				already = true
			}
		}
		if already {
			continue
		}
		var body string
		switch {
		case w.DaysLeft <= 0:
			body = fmt.Sprintf("The warranty for %s expires today (%s).", w.ProductName, w.ExpiryDate)
		case w.DaysLeft == 1:
			body = fmt.Sprintf("The warranty for %s expires tomorrow (%s).", w.ProductName, w.ExpiryDate)
		default:
			body = fmt.Sprintf("The warranty for %s expires in %d days (%s).", w.ProductName, w.DaysLeft, w.ExpiryDate)
		}
		data := map[string]any{"warrantyId": w.ID.Hex(), "expiryDate": w.ExpiryDate, "daysLeft": w.DaysLeft}
		if w.ReceiptID != nil {
			data["receiptId"] = w.ReceiptID.Hex()
		}
		if s.notify.Notify(ctx, w.UserID, models.NotifyWarrantyExpiring, "Warranty expiring: "+w.ProductName, body, data, "warranty:"+w.ID.Hex()+":"+tag) {
			sent++
		}
		_ = s.repo.MarkReminderSent(ctx, w.ID, tag)
	}
	return sent
}

func mapErr(err error) error {
	if errors.Is(err, warrantyrepo.ErrNotFound) {
		return gohttp.ErrNotFound
	}
	return err
}

package category

import (
	"context"
	"errors"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"smartreceipt/backend/common/pkg/database"
	"smartreceipt/backend/common/pkg/models"
)

var (
	ErrNotFound = errors.New("category not found")
	ErrExists   = errors.New("category already exists")
	ErrSystem   = errors.New("system categories cannot be modified")
)

// Service reads and writes the categories collection.
type Service struct{}

// New returns the service.
func New() *Service { return &Service{} }

// EnsureIndexes builds the categories indexes (tooling only).
func EnsureIndexes(ctx context.Context) error { return database.EnsureIndexes(ctx, Indexes()) }

// Seed upserts the system defaults (tooling only).
func Seed(ctx context.Context) error {
	now := time.Now().UTC()
	for _, c := range models.DefaultCategories {
		_, err := col().UpdateOne(ctx,
			bson.M{"userId": nil, "slug": c.Slug},
			bson.M{
				"$set":         bson.M{"name": c.Name, "icon": c.Icon, "color": c.Color, "updatedAt": now},
				"$setOnInsert": bson.M{"createdAt": now, "userId": nil},
			},
			options.UpdateOne().SetUpsert(true))
		if err != nil {
			return err
		}
	}
	return nil
}

// ListForUser returns system categories followed by the user's own.
func (s *Service) ListForUser(ctx context.Context, uid bson.ObjectID) ([]models.Category, error) {
	cur, err := col().Find(ctx,
		bson.M{"$or": bson.A{bson.M{"userId": nil}, bson.M{"userId": uid}}},
		options.Find().SetSort(bson.D{{Key: "userId", Value: 1}, {Key: "name", Value: 1}}))
	if err != nil {
		return nil, err
	}
	var out []models.Category
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	for i := range out {
		out[i].IsSystem = out[i].UserID == nil
	}
	if out == nil {
		out = []models.Category{}
	}
	return out, nil
}

// ListSystem returns the system categories only.
func (s *Service) ListSystem(ctx context.Context) ([]models.Category, error) {
	cur, err := col().Find(ctx, bson.M{"userId": nil}, options.Find().SetSort(bson.D{{Key: "name", Value: 1}}))
	if err != nil {
		return nil, err
	}
	var out []models.Category
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	for i := range out {
		out[i].IsSystem = true
	}
	if out == nil {
		out = []models.Category{}
	}
	return out, nil
}

// Get returns a category visible to the user (system or own).
func (s *Service) Get(ctx context.Context, uid, id bson.ObjectID) (*models.Category, error) {
	var c models.Category
	err := col().FindOne(ctx, bson.M{"_id": id, "$or": bson.A{bson.M{"userId": nil}, bson.M{"userId": uid}}}).Decode(&c)
	if err != nil {
		if database.IsNoDocuments(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	c.IsSystem = c.UserID == nil
	return &c, nil
}

// BySlug resolves a slug for the user (own category wins over system).
func (s *Service) BySlug(ctx context.Context, uid bson.ObjectID, slug string) (*models.Category, error) {
	var c models.Category
	err := col().FindOne(ctx,
		bson.M{"slug": slug, "$or": bson.A{bson.M{"userId": uid}, bson.M{"userId": nil}}},
		options.FindOne().SetSort(bson.D{{Key: "userId", Value: -1}})).Decode(&c)
	if err != nil {
		if database.IsNoDocuments(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	c.IsSystem = c.UserID == nil
	return &c, nil
}

// Create adds a user category.
func (s *Service) Create(ctx context.Context, uid bson.ObjectID, name, icon, color string) (*models.Category, error) {
	now := time.Now().UTC()
	c := models.Category{UserID: &uid, Name: strings.TrimSpace(name), Slug: models.Slugify(name), Icon: icon, Color: color, CreatedAt: now, UpdatedAt: now}
	if c.Slug == "" {
		c.Slug = "category"
	}
	// avoid clashing with a system slug so extraction stays unambiguous
	if n, _ := col().CountDocuments(ctx, bson.M{"userId": nil, "slug": c.Slug}); n > 0 {
		c.Slug += "-custom"
	}
	res, err := col().InsertOne(ctx, c)
	if err != nil {
		if database.IsDuplicateKey(err) {
			return nil, ErrExists
		}
		return nil, err
	}
	c.ID = res.InsertedID.(bson.ObjectID)
	return &c, nil
}

// Update patches a user's own category.
func (s *Service) Update(ctx context.Context, uid, id bson.ObjectID, set bson.M) error {
	set["updatedAt"] = time.Now().UTC()
	res, err := col().UpdateOne(ctx, bson.M{"_id": id, "userId": uid}, bson.M{"$set": set})
	if err != nil {
		if database.IsDuplicateKey(err) {
			return ErrExists
		}
		return err
	}
	if res.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}

// Delete removes a user's own category.
func (s *Service) Delete(ctx context.Context, uid, id bson.ObjectID) error {
	res, err := col().DeleteOne(ctx, bson.M{"_id": id, "userId": uid})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteAllForUser removes every category owned by uid (account deletion).
func (s *Service) DeleteAllForUser(ctx context.Context, uid bson.ObjectID) error {
	_, err := col().DeleteMany(ctx, bson.M{"userId": uid})
	return err
}

// ---- System categories (admin portal) ----

// ErrOthersProtected is returned when deleting the fallback category.
var ErrOthersProtected = errors.New("the others category cannot be deleted")

// GetSystem loads one system category.
func (s *Service) GetSystem(ctx context.Context, id bson.ObjectID) (*models.Category, error) {
	var c models.Category
	if err := col().FindOne(ctx, bson.M{"_id": id, "userId": nil}).Decode(&c); err != nil {
		if database.IsNoDocuments(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	c.IsSystem = true
	return &c, nil
}

// SystemBySlug loads a system category by slug.
func (s *Service) SystemBySlug(ctx context.Context, slug string) (*models.Category, error) {
	var c models.Category
	if err := col().FindOne(ctx, bson.M{"slug": slug, "userId": nil}).Decode(&c); err != nil {
		if database.IsNoDocuments(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	c.IsSystem = true
	return &c, nil
}

// CreateSystem adds a system category visible to every user. slug is
// derived from name when empty; it must be unique among system categories
// (the unique {userId, slug} index enforces it).
func (s *Service) CreateSystem(ctx context.Context, name, slug, icon, color string) (*models.Category, error) {
	now := time.Now().UTC()
	if slug == "" {
		slug = models.Slugify(name)
	} else {
		slug = models.Slugify(slug)
	}
	if slug == "" {
		slug = "category"
	}
	c := models.Category{UserID: nil, Name: strings.TrimSpace(name), Slug: slug, Icon: icon, Color: color, CreatedAt: now, UpdatedAt: now}
	res, err := col().InsertOne(ctx, c)
	if err != nil {
		if database.IsDuplicateKey(err) {
			return nil, ErrExists
		}
		return nil, err
	}
	c.ID = res.InsertedID.(bson.ObjectID)
	c.IsSystem = true
	return &c, nil
}

// UpdateSystem patches a system category. The others slug is immutable so
// extraction and deletion fallbacks keep resolving.
func (s *Service) UpdateSystem(ctx context.Context, id bson.ObjectID, set bson.M) error {
	if slug, ok := set["slug"].(string); ok {
		cur, err := s.GetSystem(ctx, id)
		if err != nil {
			return err
		}
		if cur.Slug == models.OthersSlug && slug != models.OthersSlug {
			return ErrOthersProtected
		}
	}
	set["updatedAt"] = time.Now().UTC()
	res, err := col().UpdateOne(ctx, bson.M{"_id": id, "userId": nil}, bson.M{"$set": set})
	if err != nil {
		if database.IsDuplicateKey(err) {
			return ErrExists
		}
		return err
	}
	if res.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteSystem removes a system category. Callers reassign receipts first
// (receipt.ReassignCategoryAll). The others category is protected.
func (s *Service) DeleteSystem(ctx context.Context, id bson.ObjectID) error {
	c, err := s.GetSystem(ctx, id)
	if err != nil {
		return err
	}
	if c.Slug == models.OthersSlug {
		return ErrOthersProtected
	}
	res, err := col().DeleteOne(ctx, bson.M{"_id": id, "userId": nil})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return ErrNotFound
	}
	return nil
}

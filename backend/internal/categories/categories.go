// Package categories manages system default and user-defined expense categories.
package categories

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"expenselens/backend/internal/db"
)

// Category is a spending bucket. System categories have a nil UserID.
type Category struct {
	ID        bson.ObjectID  `bson:"_id,omitempty" json:"id"`
	UserID    *bson.ObjectID `bson:"userId,omitempty" json:"-"`
	Name      string         `bson:"name" json:"name"`
	Slug      string         `bson:"slug" json:"slug"`
	Icon      string         `bson:"icon" json:"icon"`   // lucide icon name
	Color     string         `bson:"color" json:"color"` // hex
	IsSystem  bool           `bson:"-" json:"isSystem"`
	CreatedAt time.Time      `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time      `bson:"updatedAt" json:"updatedAt"`
}

// Defaults are seeded at boot. Slugs are stable identifiers used by extraction.
var Defaults = []Category{
	{Name: "Food", Slug: "food", Icon: "utensils", Color: "#22c55e"},
	{Name: "Groceries", Slug: "groceries", Icon: "shopping-basket", Color: "#84cc16"},
	{Name: "Fuel", Slug: "fuel", Icon: "fuel", Color: "#f59e0b"},
	{Name: "Shopping", Slug: "shopping", Icon: "shopping-bag", Color: "#3b82f6"},
	{Name: "Medical", Slug: "medical", Icon: "heart-pulse", Color: "#8b5cf6"},
	{Name: "Entertainment", Slug: "entertainment", Icon: "clapperboard", Color: "#ec4899"},
	{Name: "Travel", Slug: "travel", Icon: "plane", Color: "#06b6d4"},
	{Name: "Utilities", Slug: "utilities", Icon: "plug-zap", Color: "#64748b"},
	{Name: "Others", Slug: "others", Icon: "circle-dashed", Color: "#f97316"},
}

// OthersSlug is the fallback category.
const OthersSlug = "others"

var (
	ErrNotFound = errors.New("category not found")
	ErrExists   = errors.New("category already exists")
	ErrSystem   = errors.New("system categories cannot be modified")
)

// Repo is the categories collection.
type Repo struct{ col *mongo.Collection }

// NewRepo binds to "categories".
func NewRepo(d *db.DB) *Repo { return &Repo{col: d.Collection("categories")} }

// Indexes: unique slug per owner (nil owner = system).
func Indexes() db.IndexSet {
	return db.IndexSet{Collection: "categories", Models: []mongo.IndexModel{
		{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "slug", Value: 1}}, Options: options.Index().SetUnique(true)},
	}}
}

// Seed upserts the system defaults.
func (r *Repo) Seed(ctx context.Context) error {
	now := time.Now().UTC()
	for _, c := range Defaults {
		_, err := r.col.UpdateOne(ctx,
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
func (r *Repo) ListForUser(ctx context.Context, uid bson.ObjectID) ([]Category, error) {
	cur, err := r.col.Find(ctx,
		bson.M{"$or": bson.A{bson.M{"userId": nil}, bson.M{"userId": uid}}},
		options.Find().SetSort(bson.D{{Key: "userId", Value: 1}, {Key: "name", Value: 1}}))
	if err != nil {
		return nil, err
	}
	var out []Category
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	for i := range out {
		out[i].IsSystem = out[i].UserID == nil
	}
	if out == nil {
		out = []Category{}
	}
	return out, nil
}

// Get returns a category visible to the user (system or own).
func (r *Repo) Get(ctx context.Context, uid, id bson.ObjectID) (*Category, error) {
	var c Category
	err := r.col.FindOne(ctx, bson.M{"_id": id, "$or": bson.A{bson.M{"userId": nil}, bson.M{"userId": uid}}}).Decode(&c)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	c.IsSystem = c.UserID == nil
	return &c, nil
}

// BySlug resolves a slug for the user (own category wins over system).
func (r *Repo) BySlug(ctx context.Context, uid bson.ObjectID, slug string) (*Category, error) {
	var c Category
	err := r.col.FindOne(ctx,
		bson.M{"slug": slug, "$or": bson.A{bson.M{"userId": uid}, bson.M{"userId": nil}}},
		options.FindOne().SetSort(bson.D{{Key: "userId", Value: -1}})).Decode(&c)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	c.IsSystem = c.UserID == nil
	return &c, nil
}

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

// Slugify makes a URL-safe slug.
func Slugify(name string) string {
	s := slugRe.ReplaceAllString(strings.ToLower(strings.TrimSpace(name)), "-")
	return strings.Trim(s, "-")
}

// Create adds a user category.
func (r *Repo) Create(ctx context.Context, uid bson.ObjectID, name, icon, color string) (*Category, error) {
	now := time.Now().UTC()
	c := Category{UserID: &uid, Name: strings.TrimSpace(name), Slug: Slugify(name), Icon: icon, Color: color, CreatedAt: now, UpdatedAt: now}
	if c.Slug == "" {
		c.Slug = "category"
	}
	// avoid clashing with a system slug so extraction stays unambiguous
	for _, d := range Defaults {
		if d.Slug == c.Slug {
			c.Slug += "-custom"
		}
	}
	res, err := r.col.InsertOne(ctx, c)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return nil, ErrExists
		}
		return nil, err
	}
	c.ID = res.InsertedID.(bson.ObjectID)
	return &c, nil
}

// Update patches a user's own category.
func (r *Repo) Update(ctx context.Context, uid, id bson.ObjectID, set bson.M) error {
	set["updatedAt"] = time.Now().UTC()
	res, err := r.col.UpdateOne(ctx, bson.M{"_id": id, "userId": uid}, bson.M{"$set": set})
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
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
func (r *Repo) Delete(ctx context.Context, uid, id bson.ObjectID) error {
	res, err := r.col.DeleteOne(ctx, bson.M{"_id": id, "userId": uid})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteAllForUser removes every category owned by uid (account deletion).
func (r *Repo) DeleteAllForUser(ctx context.Context, uid bson.ObjectID) error {
	_, err := r.col.DeleteMany(ctx, bson.M{"userId": uid})
	return err
}

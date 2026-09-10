package models

import (
	"regexp"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// CategoriesCollection is the categories collection name.
const CategoriesCollection = "categories"

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

// DefaultCategories are seeded by ensure-indexes. Slugs are stable
// identifiers used by extraction.
var DefaultCategories = []Category{
	{Name: "Shopping", Slug: "shopping", Icon: "shopping-bag", Color: "#2a78d6"},
	{Name: "Fuel", Slug: "fuel", Icon: "fuel", Color: "#eb6834"},
	{Name: "Food", Slug: "food", Icon: "utensils", Color: "#1baf7a"},
	{Name: "Groceries", Slug: "groceries", Icon: "shopping-basket", Color: "#eda100"},
	{Name: "Entertainment", Slug: "entertainment", Icon: "clapperboard", Color: "#e87ba4"},
	{Name: "Travel", Slug: "travel", Icon: "plane", Color: "#008300"},
	{Name: "Medical", Slug: "medical", Icon: "heart-pulse", Color: "#4a3aa7"},
	{Name: "Utilities", Slug: "utilities", Icon: "plug-zap", Color: "#e34948"},
	{Name: "Others", Slug: "others", Icon: "circle-dashed", Color: "#8a8f99"},
}

// OthersSlug is the fallback category.
const OthersSlug = "others"

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

// Slugify makes a URL-safe slug.
func Slugify(name string) string {
	s := slugRe.ReplaceAllString(strings.ToLower(strings.TrimSpace(name)), "-")
	return strings.Trim(s, "-")
}

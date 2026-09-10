package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// BudgetsCollection is the budgets collection name.
const BudgetsCollection = "budgets"

// Budget is a per-category monthly limit. Limit is in minor units.
type Budget struct {
	ID         bson.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID     bson.ObjectID `bson:"userId" json:"-"`
	CategoryID bson.ObjectID `bson:"categoryId" json:"categoryId"`
	Month      string        `bson:"month" json:"month"` // YYYY-MM
	Limit      int64         `bson:"limit" json:"limit"`
	Currency   string        `bson:"currency" json:"currency"`
	CreatedAt  time.Time     `bson:"createdAt" json:"createdAt"`
	UpdatedAt  time.Time     `bson:"updatedAt" json:"updatedAt"`
}

package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// WarrantiesCollection is the warranties collection name.
const WarrantiesCollection = "warranties"

// Warranty statuses derived from ExpiryDate.
const (
	WarrantyActive   = "active"
	WarrantyExpiring = "expiring" // within 30 days
	WarrantyExpired  = "expired"
)

// Warranty tracks a product warranty, optionally linked to a receipt item.
// Dates are YYYY-MM-DD strings in the user's timezone.
type Warranty struct {
	ID             bson.ObjectID  `bson:"_id,omitempty" json:"id"`
	UserID         bson.ObjectID  `bson:"userId" json:"-"`
	ProductName    string         `bson:"productName" json:"productName"`
	PurchaseDate   string         `bson:"purchaseDate" json:"purchaseDate"`
	ExpiryDate     string         `bson:"expiryDate" json:"expiryDate"`
	DurationMonths int            `bson:"durationMonths,omitempty" json:"durationMonths,omitempty"`
	ReceiptID      *bson.ObjectID `bson:"receiptId,omitempty" json:"receiptId,omitempty"`
	ItemID         string         `bson:"itemId,omitempty" json:"itemId,omitempty"`
	ImageKey       string         `bson:"imageKey,omitempty" json:"-"`
	ImageURL       string         `bson:"-" json:"imageUrl,omitempty"`
	Notes          string         `bson:"notes,omitempty" json:"notes"`
	RemindersSent  []string       `bson:"remindersSent,omitempty" json:"-"` // "30", "7"
	CreatedAt      time.Time      `bson:"createdAt" json:"createdAt"`
	UpdatedAt      time.Time      `bson:"updatedAt" json:"updatedAt"`
	// Derived at read time.
	Status   string `bson:"-" json:"status"`
	DaysLeft int    `bson:"-" json:"daysLeft"`
}

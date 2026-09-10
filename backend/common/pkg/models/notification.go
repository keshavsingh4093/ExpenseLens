package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// NotificationsCollection is the notifications collection name.
const NotificationsCollection = "notifications"

// Notification types.
const (
	NotifyBudgetAlert      = "budget_alert"
	NotifyWarrantyExpiring = "warranty_expiring"
	NotifyExtractionDone   = "extraction_done"
	NotifyExtractionFailed = "extraction_failed"
	NotifyWeeklySummary    = "weekly_summary"
)

// Notification is an in-app message. DedupeKey (unique per user when set)
// stops the scheduler from repeating the same alert.
type Notification struct {
	ID        bson.ObjectID  `bson:"_id,omitempty" json:"id"`
	UserID    bson.ObjectID  `bson:"userId" json:"-"`
	Type      string         `bson:"type" json:"type"`
	Title     string         `bson:"title" json:"title"`
	Body      string         `bson:"body" json:"body"`
	Data      map[string]any `bson:"data,omitempty" json:"data,omitempty"`
	DedupeKey string         `bson:"dedupeKey,omitempty" json:"-"`
	ReadAt    *time.Time     `bson:"readAt,omitempty" json:"readAt,omitempty"`
	CreatedAt time.Time      `bson:"createdAt" json:"createdAt"`
}

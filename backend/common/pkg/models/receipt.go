package models

import (
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"smartreceipt/backend/common/pkg/extraction"
)

// ReceiptsCollection is the receipts collection name.
const ReceiptsCollection = "receipts"

// ReceiptStatus is the receipt lifecycle.
type ReceiptStatus string

const (
	ReceiptProcessing ReceiptStatus = "processing"
	ReceiptReview     ReceiptStatus = "review"
	ReceiptConfirmed  ReceiptStatus = "confirmed"
	ReceiptFailed     ReceiptStatus = "failed"
)

// ReceiptItem is a line item. Money in minor units.
type ReceiptItem struct {
	ID        string   `bson:"id" json:"id"`
	Name      string   `bson:"name" json:"name"`
	Quantity  *float64 `bson:"quantity,omitempty" json:"quantity,omitempty"`
	UnitPrice *int64   `bson:"unitPrice,omitempty" json:"unitPrice,omitempty"`
	Amount    int64    `bson:"amount" json:"amount"`
}

// ReceiptImage is one stored page/photo.
type ReceiptImage struct {
	ID          string `bson:"id" json:"id"`
	Key         string `bson:"key" json:"-"`
	URL         string `bson:"-" json:"url"`
	ContentType string `bson:"contentType" json:"contentType"`
	Size        int64  `bson:"size" json:"size"`
	Hash        string `bson:"hash" json:"-"`
	Order       int    `bson:"order" json:"order"`
	FileName    string `bson:"fileName,omitempty" json:"fileName,omitempty"`
}

// Extracted records what the model returned, for audit and re-review.
type Extracted struct {
	Model              string                         `bson:"model" json:"model"`
	Confidence         float64                        `bson:"confidence" json:"confidence"`
	SuggestedCategory  string                         `bson:"suggestedCategory,omitempty" json:"suggestedCategory,omitempty"`
	WarrantyCandidates []extraction.WarrantyCandidate `bson:"warrantyCandidates,omitempty" json:"warrantyCandidates,omitempty"`
	ExtractedAt        time.Time                      `bson:"extractedAt" json:"extractedAt"`
	Error              string                         `bson:"error,omitempty" json:"-"`
	ErrorCode          string                         `bson:"errorCode,omitempty" json:"errorCode,omitempty"`
	Attempts           int                            `bson:"attempts" json:"attempts"`
}

// Receipt is the persisted document.
type Receipt struct {
	ID                 bson.ObjectID        `bson:"_id,omitempty" json:"id"`
	UserID             bson.ObjectID        `bson:"userId" json:"-"`
	Status             ReceiptStatus        `bson:"status" json:"status"`
	Merchant           string               `bson:"merchant" json:"merchant"`
	MerchantNormalized string               `bson:"merchantNormalized" json:"-"`
	Date               string               `bson:"date" json:"date"` // YYYY-MM-DD in user's tz
	Time               string               `bson:"time,omitempty" json:"time,omitempty"`
	CategoryID         *bson.ObjectID       `bson:"categoryId,omitempty" json:"categoryId,omitempty"`
	Currency           string               `bson:"currency" json:"currency"`
	Subtotal           *int64               `bson:"subtotal,omitempty" json:"subtotal,omitempty"`
	Tax                *int64               `bson:"tax,omitempty" json:"tax,omitempty"`
	Total              int64                `bson:"total" json:"total"`
	PaymentMethod      string               `bson:"paymentMethod,omitempty" json:"paymentMethod,omitempty"`
	GSTIN              string               `bson:"gstin,omitempty" json:"gstin,omitempty"`
	TaxBreakdown       []extraction.TaxLine `bson:"taxBreakdown,omitempty" json:"taxBreakdown,omitempty"`
	Items              []ReceiptItem        `bson:"items" json:"items"`
	Images             []ReceiptImage       `bson:"images" json:"images"`
	Notes              string               `bson:"notes,omitempty" json:"notes"`
	Tags               []string             `bson:"tags" json:"tags"`
	Extracted          *Extracted           `bson:"extracted,omitempty" json:"extracted,omitempty"`
	FileHash           string               `bson:"fileHash" json:"-"`
	CreatedAt          time.Time            `bson:"createdAt" json:"createdAt"`
	UpdatedAt          time.Time            `bson:"updatedAt" json:"updatedAt"`
	ConfirmedAt        *time.Time           `bson:"confirmedAt,omitempty" json:"confirmedAt,omitempty"`
}

// ReceiptListItem is the compact row for list views. FirstKey is the storage
// key of the first image; the service turns it into Thumbnail.
type ReceiptListItem struct {
	ID         bson.ObjectID  `bson:"_id" json:"id"`
	Status     ReceiptStatus  `bson:"status" json:"status"`
	Merchant   string         `bson:"merchant" json:"merchant"`
	Date       string         `bson:"date" json:"date"`
	CategoryID *bson.ObjectID `bson:"categoryId,omitempty" json:"categoryId,omitempty"`
	Currency   string         `bson:"currency" json:"currency"`
	Total      int64          `bson:"total" json:"total"`
	ItemCount  int            `bson:"itemCount" json:"itemCount"`
	Thumbnail  string         `bson:"-" json:"thumbnail,omitempty"`
	FirstKey   string         `bson:"firstKey" json:"-"`
	CreatedAt  time.Time      `bson:"createdAt" json:"createdAt"`
}

// PaymentMethods accepted by the API.
var PaymentMethods = map[string]bool{"cash": true, "upi": true, "card": true, "netbanking": true, "wallet": true, "other": true}

// NormalizeMerchant lowercases and collapses whitespace for grouping/search.
func NormalizeMerchant(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(s)), " ")
}

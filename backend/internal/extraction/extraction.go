// Package extraction turns receipt images/PDFs into structured data. The
// Extractor interface hides the model provider; Gemini is the production
// implementation and Fake serves tests and offline development.
package extraction

import (
	"context"
	"encoding/json"
	"fmt"

	"expenselens/backend/internal/config"
)

// File is one page/photo of a receipt.
type File struct {
	Data        []byte
	ContentType string // image/jpeg, image/png, image/webp, application/pdf
}

// Input is everything the model needs.
type Input struct {
	Files      []File
	Categories []string // slugs the model must choose from
	Currency   string   // user's home currency, a hint only
	Timezone   string
}

// Item is a line on the receipt. Money is in minor units (paise).
type Item struct {
	Name      string   `json:"name"`
	Quantity  *float64 `json:"quantity,omitempty"`
	UnitPrice *int64   `json:"unitPrice,omitempty"`
	Amount    int64    `json:"amount"`
}

// TaxLine is one tax component (e.g. CGST 9%).
type TaxLine struct {
	Label  string `json:"label"`
	Amount int64  `json:"amount"`
}

// WarrantyCandidate flags an item that probably carries a warranty.
type WarrantyCandidate struct {
	ItemName string `json:"itemName"`
	Reason   string `json:"reason"`
}

// Result is the normalized extraction output.
type Result struct {
	Merchant           string              `json:"merchant"`
	Date               string              `json:"date"` // YYYY-MM-DD
	Time               string              `json:"time,omitempty"`
	Currency           string              `json:"currency"`
	Subtotal           *int64              `json:"subtotal,omitempty"`
	Tax                *int64              `json:"tax,omitempty"`
	Total              int64               `json:"total"`
	PaymentMethod      string              `json:"paymentMethod,omitempty"`
	GSTIN              string              `json:"gstin,omitempty"`
	TaxBreakdown       []TaxLine           `json:"taxBreakdown,omitempty"`
	Items              []Item              `json:"items"`
	SuggestedCategory  string              `json:"suggestedCategory"`
	WarrantyCandidates []WarrantyCandidate `json:"warrantyCandidates,omitempty"`
	Confidence         float64             `json:"confidence"`
	Model              string              `json:"model"`
	Raw                json.RawMessage     `json:"raw,omitempty"`
}

// Extractor is implemented by model providers.
type Extractor interface {
	Name() string
	Extract(ctx context.Context, in Input) (*Result, error)
}

// New picks an implementation from config.
func New(cfg config.GeminiConfig) (Extractor, error) {
	switch cfg.Extractor {
	case "fake":
		return &Fake{}, nil
	case "gemini", "":
		if cfg.APIKey == "" {
			return nil, fmt.Errorf("extraction: GEMINI_API_KEY is required (or set EXTRACTOR=fake)")
		}
		return NewGemini(cfg.APIKey, cfg.Model), nil
	default:
		return nil, fmt.Errorf("extraction: unknown extractor %q", cfg.Extractor)
	}
}

// toMinor converts a decimal amount (e.g. 1230.50) to minor units (123050).
func toMinor(v float64) int64 {
	if v >= 0 {
		return int64(v*100 + 0.5)
	}
	return int64(v*100 - 0.5)
}

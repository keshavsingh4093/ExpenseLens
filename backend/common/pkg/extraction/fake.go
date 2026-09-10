package extraction

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"
)

// Fake returns deterministic data derived from the file bytes. Used when
// EXTRACTOR=fake and in tests.
type Fake struct {
	// Fail, when set, makes every call return an error (tests).
	Fail bool
}

func (f *Fake) Name() string { return "fake" }

func (f *Fake) Extract(_ context.Context, in Input) (*Result, error) {
	if f.Fail {
		return nil, fmt.Errorf("extraction: fake failure")
	}
	var seed byte
	if len(in.Files) > 0 && len(in.Files[0].Data) > 0 {
		seed = sha256.Sum256(in.Files[0].Data)[0]
	}
	cat := "others"
	if len(in.Categories) > 0 {
		cat = in.Categories[int(seed)%len(in.Categories)]
	}
	tax := int64(5000)
	q := 2.0
	unit := int64(32500)
	return &Result{
		Merchant: "Sample Mart", Date: time.Now().Format("2006-01-02"), Time: "18:42",
		Currency: "INR", Total: 100000 + int64(seed)*100, Tax: &tax, PaymentMethod: "upi",
		Items: []Item{
			{Name: "Rice 5kg", Quantity: &q, UnitPrice: &unit, Amount: 65000},
			{Name: "Sunflower Oil 1L", Amount: 21000},
			{Name: "Toor Dal 1kg", Amount: 14000 + int64(seed)*100},
		},
		SuggestedCategory: cat, Confidence: 0.9, Model: "fake",
		WarrantyCandidates: nil,
	}, nil
}

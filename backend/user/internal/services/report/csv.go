// Package report builds downloadable exports: CSV of receipts, a monthly
// PDF summary, and the all-data JSON export.
package report

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"

	"smartreceipt/backend/common/pkg/models"
	"smartreceipt/backend/common/services/category"
	receiptrepo "smartreceipt/backend/common/services/receipt"

	"smartreceipt/backend/user/internal/services/analytics"
)

// Service builds reports.
type Service struct {
	receipts *receiptrepo.Service
	cats     *category.Service
	stats    *analytics.Service
}

// New returns the service.
func New() *Service {
	return &Service{receipts: receiptrepo.New(), cats: category.New(), stats: analytics.New()}
}

// decimal renders minor units as "1234.50" (no grouping, for spreadsheets).
func decimal(minor int64) string {
	neg := minor < 0
	if neg {
		minor = -minor
	}
	s := fmt.Sprintf("%d.%02d", minor/100, minor%100)
	if neg {
		s = "-" + s
	}
	return s
}

func optDecimal(v *int64) string {
	if v == nil {
		return ""
	}
	return decimal(*v)
}

func (s *Service) categoryNames(ctx context.Context, uid bson.ObjectID) map[string]string {
	names := map[string]string{}
	if cats, err := s.cats.ListForUser(ctx, uid); err == nil {
		for _, c := range cats {
			names[c.ID.Hex()] = c.Name
		}
	}
	return names
}

func categoryName(names map[string]string, id *bson.ObjectID) string {
	if id == nil {
		return "Uncategorized"
	}
	if n, ok := names[id.Hex()]; ok {
		return n
	}
	return "Uncategorized"
}

func itemsCell(items []models.ReceiptItem) string {
	parts := make([]string, 0, len(items))
	for _, it := range items {
		p := it.Name
		if it.Quantity != nil && *it.Quantity > 0 {
			p += fmt.Sprintf(" x %g", *it.Quantity)
		}
		p += " = " + decimal(it.Amount)
		parts = append(parts, p)
	}
	return strings.Join(parts, "; ")
}

// CSV renders every counted receipt in the range as UTF-8 CSV with a BOM
// (so Excel opens it correctly).
func (s *Service) CSV(ctx context.Context, uid bson.ObjectID, rg analytics.Range) ([]byte, error) {
	rows, err := s.receipts.FindAll(ctx, uid, rg.From, rg.To)
	if err != nil {
		return nil, err
	}
	names := s.categoryNames(ctx, uid)
	var buf bytes.Buffer
	buf.WriteString("\xEF\xBB\xBF")
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"date", "time", "merchant", "category", "status", "currency", "subtotal", "tax", "total", "paymentMethod", "gstin", "items", "tags", "notes", "id"})
	for _, rc := range rows {
		_ = w.Write([]string{
			rc.Date, rc.Time, rc.Merchant, categoryName(names, rc.CategoryID), string(rc.Status), rc.Currency,
			optDecimal(rc.Subtotal), optDecimal(rc.Tax), decimal(rc.Total), rc.PaymentMethod, rc.GSTIN,
			itemsCell(rc.Items), strings.Join(rc.Tags, ", "), rc.Notes, rc.ID.Hex(),
		})
	}
	w.Flush()
	return buf.Bytes(), w.Error()
}

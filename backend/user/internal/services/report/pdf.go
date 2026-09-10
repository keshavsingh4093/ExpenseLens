package report

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"
	"go.mongodb.org/mongo-driver/v2/bson"

	"smartreceipt/backend/user/internal/services/analytics"
	"smartreceipt/backend/user/internal/services/budget"
)

// PDF renders a spending summary for the range: totals, by category, top
// merchants and a daily bar chart. Core fonts only, so amounts use the
// currency code rather than a symbol.
func (s *Service) PDF(ctx context.Context, uid bson.ObjectID, rg analytics.Range, currency, userName string) ([]byte, error) {
	if currency == "" {
		currency = "INR"
	}
	points, err := s.stats.ByDay(ctx, uid, rg)
	if err != nil {
		return nil, err
	}
	byCat, err := s.stats.ByCategory(ctx, uid, rg)
	if err != nil {
		return nil, err
	}
	merchants, err := s.stats.TopMerchants(ctx, uid, rg, 8)
	if err != nil {
		return nil, err
	}
	stats := analytics.ComputeStats(points, rg)
	names := s.categoryNames(ctx, uid)

	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetTitle("ExpenseLens summary "+rg.From+" to "+rg.To, true)
	pdf.SetAutoPageBreak(true, 18)
	pdf.SetMargins(16, 16, 16)
	pdf.AddPage()
	green := func() { pdf.SetTextColor(22, 163, 74) }
	ink := func() { pdf.SetTextColor(30, 30, 30) }
	muted := func() { pdf.SetTextColor(110, 110, 110) }

	green()
	pdf.SetFont("Helvetica", "B", 20)
	pdf.CellFormat(0, 10, "ExpenseLens", "", 1, "L", false, 0, "")
	ink()
	pdf.SetFont("Helvetica", "B", 14)
	pdf.CellFormat(0, 8, "Spending summary", "", 1, "L", false, 0, "")
	muted()
	pdf.SetFont("Helvetica", "", 10)
	period := fmt.Sprintf("%s to %s", human(rg.From), human(rg.To))
	if userName != "" {
		period = userName + "  |  " + period
	}
	pdf.CellFormat(0, 6, period, "", 1, "L", false, 0, "")
	pdf.CellFormat(0, 6, "Generated "+time.Now().UTC().Format("02 Jan 2006 15:04 UTC"), "", 1, "L", false, 0, "")
	pdf.Ln(4)

	// KPI row.
	tiles := [][2]string{
		{"Total spent", budget.Money(stats.Total, currency)},
		{"Receipts", fmt.Sprintf("%d", stats.Transactions)},
		{"Average per day", budget.Money(stats.AvgDaily, currency)},
		{"Average per receipt", budget.Money(stats.AvgPerTx, currency)},
	}
	w := (210 - 32 - 9) / 4.0
	x0, y0 := pdf.GetX(), pdf.GetY()
	for i, t := range tiles {
		x := x0 + float64(i)*(w+3)
		pdf.SetFillColor(240, 253, 244)
		pdf.Rect(x, y0, w, 18, "F")
		pdf.SetXY(x+3, y0+3)
		muted()
		pdf.SetFont("Helvetica", "", 8)
		pdf.CellFormat(w-6, 4, t[0], "", 2, "L", false, 0, "")
		ink()
		pdf.SetFont("Helvetica", "B", 11)
		pdf.CellFormat(w-6, 7, t[1], "", 0, "L", false, 0, "")
	}
	pdf.SetXY(x0, y0+24)
	if stats.HighestDay != nil {
		muted()
		pdf.SetFont("Helvetica", "", 9)
		pdf.CellFormat(0, 5, fmt.Sprintf("Highest day: %s (%s, %d receipts)", human(stats.HighestDay.Date), budget.Money(stats.HighestDay.Total, currency), stats.HighestDay.Count), "", 1, "L", false, 0, "")
	}
	pdf.Ln(3)

	// Daily bar chart.
	section(pdf, "Daily spend")
	drawBars(pdf, analytics.Rollup(points, rg, analytics.DefaultGranularity(rg)), 178, 40)
	pdf.Ln(6)

	// By category.
	section(pdf, "By category")
	table(pdf, []string{"Category", "Receipts", "Share", "Amount"}, []float64{80, 30, 28, 40}, func() [][]string {
		rows := [][]string{}
		for _, c := range byCat {
			rows = append(rows, []string{categoryName(names, c.CategoryID), fmt.Sprintf("%d", c.Count), fmt.Sprintf("%.0f%%", c.Percent), budget.Money(c.Total, currency)})
		}
		return rows
	}())
	pdf.Ln(4)

	// Top merchants.
	section(pdf, "Top merchants")
	table(pdf, []string{"Merchant", "Receipts", "Amount"}, []float64{108, 30, 40}, func() [][]string {
		rows := [][]string{}
		for _, m := range merchants {
			rows = append(rows, []string{m.Merchant, fmt.Sprintf("%d", m.Count), budget.Money(m.Total, currency)})
		}
		return rows
	}())

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func human(day string) string {
	t, err := time.Parse(analytics.Day, day)
	if err != nil {
		return day
	}
	return t.Format("02 Jan 2006")
}

func section(pdf *fpdf.Fpdf, title string) {
	pdf.SetTextColor(30, 30, 30)
	pdf.SetFont("Helvetica", "B", 11)
	pdf.CellFormat(0, 7, title, "", 1, "L", false, 0, "")
	pdf.SetDrawColor(225, 225, 225)
	x, y := pdf.GetX(), pdf.GetY()
	pdf.Line(x, y, x+178, y)
	pdf.Ln(2)
}

func table(pdf *fpdf.Fpdf, headers []string, widths []float64, rows [][]string) {
	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetFillColor(245, 245, 245)
	pdf.SetTextColor(90, 90, 90)
	for i, h := range headers {
		align := "L"
		if i > 0 {
			align = "R"
		}
		pdf.CellFormat(widths[i], 7, h, "", 0, align, true, 0, "")
	}
	pdf.Ln(-1)
	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(30, 30, 30)
	if len(rows) == 0 {
		pdf.SetTextColor(110, 110, 110)
		pdf.CellFormat(0, 7, "No data in this period", "", 1, "L", false, 0, "")
		return
	}
	for _, r := range rows {
		for i, c := range r {
			align := "L"
			if i > 0 {
				align = "R"
			}
			pdf.CellFormat(widths[i], 6.5, clip(pdf, c, widths[i]-2), "B", 0, align, false, 0, "")
		}
		pdf.Ln(-1)
	}
}

func clip(pdf *fpdf.Fpdf, s string, w float64) string {
	s = strings.Map(func(r rune) rune {
		if r > 255 {
			return '?'
		}
		return r
	}, s)
	for pdf.GetStringWidth(s) > w && len(s) > 3 {
		s = s[:len(s)-2]
	}
	return s
}

func drawBars(pdf *fpdf.Fpdf, buckets []analytics.Bucket, width, height float64) {
	x0, y0 := pdf.GetX(), pdf.GetY()
	var max int64
	for _, b := range buckets {
		if b.Total > max {
			max = b.Total
		}
	}
	pdf.SetDrawColor(225, 225, 225)
	pdf.Line(x0, y0+height, x0+width, y0+height)
	if max == 0 || len(buckets) == 0 {
		pdf.SetTextColor(110, 110, 110)
		pdf.SetFont("Helvetica", "", 9)
		pdf.SetXY(x0, y0+height/2)
		pdf.CellFormat(width, 6, "No spending in this period", "", 1, "C", false, 0, "")
		pdf.SetXY(x0, y0+height+8)
		return
	}
	n := float64(len(buckets))
	slot := width / n
	bw := slot * 0.7
	if bw > 12 {
		bw = 12
	}
	pdf.SetFillColor(22, 163, 74)
	for i, b := range buckets {
		h := float64(b.Total) / float64(max) * (height - 4)
		x := x0 + float64(i)*slot + (slot-bw)/2
		if h > 0 {
			pdf.Rect(x, y0+height-h, bw, h, "F")
		}
	}
	// Labels: at most ~8.
	step := int(n/8) + 1
	pdf.SetFont("Helvetica", "", 7)
	pdf.SetTextColor(110, 110, 110)
	for i := 0; i < len(buckets); i += step {
		x := x0 + float64(i)*slot
		pdf.SetXY(x-4, y0+height+1)
		pdf.CellFormat(slot+8, 4, buckets[i].Label, "", 0, "C", false, 0, "")
	}
	pdf.SetXY(x0, y0+height+8)
}

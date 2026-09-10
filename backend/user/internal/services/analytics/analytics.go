// Package analytics computes dashboard and analytics aggregates over receipts.
// Receipt dates are YYYY-MM-DD strings in the user's timezone, so all range
// math happens on date strings; the caller passes the timezone.
package analytics

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"smartreceipt/backend/common/pkg/database"
	"smartreceipt/backend/common/pkg/models"
)

// countedStatuses are receipts that contribute to totals.
var countedStatuses = bson.A{"review", "confirmed"}

func col() *mongo.Collection { return database.GetDBConnection().Collection(models.ReceiptsCollection) }

// Service runs aggregations on the receipts collection.
type Service struct{}

// New returns the service.
func New() *Service { return &Service{} }

// Day is the date layout used throughout.
const Day = "2006-01-02"

// Range is an inclusive date window.
type Range struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// Now returns the current time in tz (falls back to Asia/Kolkata).
func Now(tz string) time.Time {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc, _ = time.LoadLocation("Asia/Kolkata")
	}
	return time.Now().In(loc)
}

// ParseRange resolves a named range ("7d","30d","3m","6m","1y","month","year","all")
// or explicit from/to into a Range ending today.
func ParseRange(name, from, to string, now time.Time) (Range, error) {
	today := now.Format(Day)
	if from != "" || to != "" {
		if from == "" {
			from = "1970-01-01"
		}
		if to == "" {
			to = today
		}
		for _, d := range []string{from, to} {
			if _, err := time.Parse(Day, d); err != nil {
				return Range{}, fmt.Errorf("invalid date %q", d)
			}
		}
		return Range{From: from, To: to}, nil
	}
	switch name {
	case "7d":
		return Range{From: now.AddDate(0, 0, -6).Format(Day), To: today}, nil
	case "", "30d":
		return Range{From: now.AddDate(0, 0, -29).Format(Day), To: today}, nil
	case "3m":
		return Range{From: now.AddDate(0, -3, 0).Format(Day), To: today}, nil
	case "6m":
		return Range{From: now.AddDate(0, -6, 0).Format(Day), To: today}, nil
	case "1y":
		return Range{From: now.AddDate(-1, 0, 0).Format(Day), To: today}, nil
	case "month":
		return Range{From: now.Format("2006-01") + "-01", To: today}, nil
	case "year":
		return Range{From: now.Format("2006") + "-01-01", To: today}, nil
	case "all":
		return Range{From: "1970-01-01", To: today}, nil
	}
	return Range{}, fmt.Errorf("unknown range %q", name)
}

// Days returns the number of days in the range (at least 1).
func (r Range) Days() int {
	f, _ := time.Parse(Day, r.From)
	t, _ := time.Parse(Day, r.To)
	d := int(t.Sub(f).Hours()/24) + 1
	if d < 1 {
		return 1
	}
	return d
}

// Previous returns the same-length window immediately before r.
func (r Range) Previous() Range {
	f, _ := time.Parse(Day, r.From)
	n := r.Days()
	return Range{From: f.AddDate(0, 0, -n).Format(Day), To: f.AddDate(0, 0, -1).Format(Day)}
}

func match(uid bson.ObjectID, rg Range) bson.M {
	return bson.M{"userId": uid, "status": bson.M{"$in": countedStatuses}, "date": bson.M{"$gte": rg.From, "$lte": rg.To}}
}

// Totals is a sum + count.
type Totals struct {
	Total int64 `bson:"total" json:"total"`
	Count int64 `bson:"count" json:"count"`
}

// Sum totals receipts in the range.
func (s *Service) Sum(ctx context.Context, uid bson.ObjectID, rg Range) (Totals, error) {
	cur, err := col().Aggregate(ctx, mongo.Pipeline{
		{{Key: "$match", Value: match(uid, rg)}},
		{{Key: "$group", Value: bson.M{"_id": nil, "total": bson.M{"$sum": "$total"}, "count": bson.M{"$sum": 1}}}},
	})
	if err != nil {
		return Totals{}, err
	}
	var out []Totals
	if err := cur.All(ctx, &out); err != nil {
		return Totals{}, err
	}
	if len(out) == 0 {
		return Totals{}, nil
	}
	return out[0], nil
}

// DayPoint is a per-day total.
type DayPoint struct {
	Date  string `bson:"_id" json:"date"`
	Total int64  `bson:"total" json:"total"`
	Count int64  `bson:"count" json:"count"`
}

// ByDay returns per-day totals sorted ascending (days with no spend omitted).
func (s *Service) ByDay(ctx context.Context, uid bson.ObjectID, rg Range) ([]DayPoint, error) {
	cur, err := col().Aggregate(ctx, mongo.Pipeline{
		{{Key: "$match", Value: match(uid, rg)}},
		{{Key: "$group", Value: bson.M{"_id": "$date", "total": bson.M{"$sum": "$total"}, "count": bson.M{"$sum": 1}}}},
		{{Key: "$sort", Value: bson.D{{Key: "_id", Value: 1}}}},
	})
	if err != nil {
		return nil, err
	}
	var out []DayPoint
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = []DayPoint{}
	}
	return out, nil
}

// Bucket is a trend point after roll-up.
type Bucket struct {
	Period string `json:"period"` // YYYY-MM-DD, YYYY-Www, or YYYY-MM
	Label  string `json:"label"`
	Total  int64  `json:"total"`
	Count  int64  `json:"count"`
}

// Rollup fills every period in the range at the given granularity.
func Rollup(points []DayPoint, rg Range, granularity string) []Bucket {
	from, _ := time.Parse(Day, rg.From)
	to, _ := time.Parse(Day, rg.To)
	key := func(t time.Time) (string, string) {
		switch granularity {
		case "month":
			return t.Format("2006-01"), t.Format("Jan 06")
		case "week":
			y, w := t.ISOWeek()
			monday := t.AddDate(0, 0, -int((t.Weekday()+6)%7))
			return fmt.Sprintf("%04d-W%02d", y, w), monday.Format("02 Jan")
		}
		return t.Format(Day), t.Format("02 Jan")
	}
	order := []string{}
	buckets := map[string]*Bucket{}
	for t := from; !t.After(to); t = t.AddDate(0, 0, 1) {
		k, label := key(t)
		if _, ok := buckets[k]; !ok {
			buckets[k] = &Bucket{Period: k, Label: label}
			order = append(order, k)
		}
	}
	for _, p := range points {
		t, err := time.Parse(Day, p.Date)
		if err != nil {
			continue
		}
		k, _ := key(t)
		if b, ok := buckets[k]; ok {
			b.Total += p.Total
			b.Count += p.Count
		}
	}
	out := make([]Bucket, 0, len(order))
	for _, k := range order {
		out = append(out, *buckets[k])
	}
	return out
}

// CategoryTotal is spend per category.
type CategoryTotal struct {
	CategoryID *bson.ObjectID `bson:"_id" json:"categoryId"`
	Total      int64          `bson:"total" json:"total"`
	Count      int64          `bson:"count" json:"count"`
	Percent    float64        `bson:"-" json:"percent"`
}

// ByCategory returns spend per category, largest first, with percentages.
func (s *Service) ByCategory(ctx context.Context, uid bson.ObjectID, rg Range) ([]CategoryTotal, error) {
	cur, err := col().Aggregate(ctx, mongo.Pipeline{
		{{Key: "$match", Value: match(uid, rg)}},
		{{Key: "$group", Value: bson.M{"_id": "$categoryId", "total": bson.M{"$sum": "$total"}, "count": bson.M{"$sum": 1}}}},
		{{Key: "$sort", Value: bson.D{{Key: "total", Value: -1}}}},
	})
	if err != nil {
		return nil, err
	}
	var out []CategoryTotal
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	var sum int64
	for _, c := range out {
		sum += c.Total
	}
	for i := range out {
		if sum > 0 {
			out[i].Percent = float64(out[i].Total) * 100 / float64(sum)
		}
	}
	if out == nil {
		out = []CategoryTotal{}
	}
	return out, nil
}

// MerchantTotal is spend per merchant.
type MerchantTotal struct {
	Merchant string `bson:"merchant" json:"merchant"`
	Total    int64  `bson:"total" json:"total"`
	Count    int64  `bson:"count" json:"count"`
}

// TopMerchants returns the biggest merchants by spend.
func (s *Service) TopMerchants(ctx context.Context, uid bson.ObjectID, rg Range, limit int) ([]MerchantTotal, error) {
	if limit <= 0 || limit > 50 {
		limit = 5
	}
	cur, err := col().Aggregate(ctx, mongo.Pipeline{
		{{Key: "$match", Value: match(uid, rg)}},
		{{Key: "$group", Value: bson.M{"_id": "$merchantNormalized", "merchant": bson.M{"$first": "$merchant"}, "total": bson.M{"$sum": "$total"}, "count": bson.M{"$sum": 1}}}},
		{{Key: "$sort", Value: bson.D{{Key: "total", Value: -1}}}},
		{{Key: "$limit", Value: int64(limit)}},
	})
	if err != nil {
		return nil, err
	}
	var out []MerchantTotal
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	for i := range out {
		if strings.TrimSpace(out[i].Merchant) == "" {
			out[i].Merchant = "Unknown"
		}
	}
	if out == nil {
		out = []MerchantTotal{}
	}
	return out, nil
}

// Stats summarizes a range.
type Stats struct {
	Total        int64     `json:"total"`
	Transactions int64     `json:"transactions"`
	AvgDaily     int64     `json:"avgDaily"`
	AvgPerTx     int64     `json:"avgPerTransaction"`
	HighestDay   *DayPoint `json:"highestDay,omitempty"`
	Days         int       `json:"days"`
}

// ComputeStats derives Stats from per-day points.
func ComputeStats(points []DayPoint, rg Range) Stats {
	s := Stats{Days: rg.Days()}
	for _, p := range points {
		s.Total += p.Total
		s.Transactions += p.Count
		if s.HighestDay == nil || p.Total > s.HighestDay.Total {
			cp := p
			s.HighestDay = &cp
		}
	}
	s.AvgDaily = s.Total / int64(s.Days)
	if s.Transactions > 0 {
		s.AvgPerTx = s.Total / s.Transactions
	}
	return s
}

// KPI is one dashboard tile: value plus change vs the previous period.
type KPI struct {
	Total       int64    `json:"total"`
	Count       int64    `json:"count"`
	Previous    int64    `json:"previous"`
	ChangePct   *float64 `json:"changePct"` // nil when previous is 0
	PeriodLabel string   `json:"periodLabel"`
}

func kpi(cur, prev Totals, label string) KPI {
	k := KPI{Total: cur.Total, Count: cur.Count, Previous: prev.Total, PeriodLabel: label}
	if prev.Total > 0 {
		p := (float64(cur.Total) - float64(prev.Total)) * 100 / float64(prev.Total)
		k.ChangePct = &p
	}
	return k
}

// Summary is the four dashboard tiles.
type Summary struct {
	TotalExpenses KPI `json:"totalExpenses"`
	ThisMonth     KPI `json:"thisMonth"`
	Today         KPI `json:"today"`
	AverageDaily  KPI `json:"averageDaily"`
}

// Summary computes the dashboard KPI tiles for now.
func (s *Service) Summary(ctx context.Context, uid bson.ObjectID, now time.Time) (Summary, error) {
	today := now.Format(Day)
	allTime := Range{From: "1970-01-01", To: today}
	thisMonth := Range{From: now.Format("2006-01") + "-01", To: today}
	lastMonthStart := now.AddDate(0, -1, -now.Day()+1)
	lastMonth := Range{From: lastMonthStart.Format(Day), To: now.AddDate(0, 0, -now.Day()).Format(Day)}
	todayR := Range{From: today, To: today}
	yesterday := Range{From: now.AddDate(0, 0, -1).Format(Day), To: now.AddDate(0, 0, -1).Format(Day)}
	last30 := Range{From: now.AddDate(0, 0, -29).Format(Day), To: today}
	prev30 := last30.Previous()

	var firstErr error
	get := func(rg Range) Totals {
		t, err := s.Sum(ctx, uid, rg)
		if err != nil && firstErr == nil {
			firstErr = err
		}
		return t
	}
	all := get(allTime)
	tm, lm := get(thisMonth), get(lastMonth)
	td, yd := get(todayR), get(yesterday)
	l30, p30 := get(last30), get(prev30)
	if firstErr != nil {
		return Summary{}, firstErr
	}
	avg := Totals{Total: l30.Total / 30, Count: l30.Count}
	prevAvg := Totals{Total: p30.Total / 30, Count: p30.Count}
	return Summary{
		TotalExpenses: kpi(all, lm, "vs last month"),
		ThisMonth:     kpi(tm, lm, "vs last month"),
		Today:         kpi(td, yd, "vs yesterday"),
		AverageDaily:  kpi(avg, prevAvg, "vs last 30 days"),
	}, nil
}

// DefaultGranularity picks day/week/month for a range length.
func DefaultGranularity(rg Range) string {
	switch {
	case rg.Days() <= 31:
		return "day"
	case rg.Days() <= 120:
		return "week"
	}
	return "month"
}

// sortPoints keeps ByDay output deterministic in tests.
func sortPoints(p []DayPoint) { sort.Slice(p, func(i, j int) bool { return p[i].Date < p[j].Date }) }

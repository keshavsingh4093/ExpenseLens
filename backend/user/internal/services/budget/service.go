// Package budget is the user portal's budget logic: monthly limits per
// category with spend progress, and the 80%/100% alerts. Collection access is
// common/services/budget.
package budget

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	gohttp "smartreceipt/backend/common/pkg/http"
	"smartreceipt/backend/common/pkg/models"
	budgetrepo "smartreceipt/backend/common/services/budget"
	"smartreceipt/backend/common/services/category"
	"smartreceipt/backend/common/services/user"

	"smartreceipt/backend/user/internal/services/analytics"
	"smartreceipt/backend/user/internal/services/notification"
)

var monthRe = regexp.MustCompile(`^\d{4}-(0[1-9]|1[0-2])$`)

// ValidMonth reports whether m is YYYY-MM.
func ValidMonth(m string) bool { return monthRe.MatchString(m) }

// MonthRange returns the inclusive date window of a YYYY-MM month.
func MonthRange(month string) analytics.Range {
	t, _ := time.Parse("2006-01", month)
	last := t.AddDate(0, 1, -1)
	return analytics.Range{From: month + "-01", To: last.Format(analytics.Day)}
}

// Row is one budget with its spend for the month.
type Row struct {
	models.Budget
	Spent     int64   `json:"spent"`
	Remaining int64   `json:"remaining"`
	Percent   float64 `json:"percent"`
	Count     int64   `json:"count"`
}

// Summary totals the month.
type Summary struct {
	TotalLimit int64 `json:"totalLimit"`
	TotalSpent int64 `json:"totalSpent"`
}

// Service is the budget domain logic.
type Service struct {
	repo   *budgetrepo.Service
	cats   *category.Service
	users  *user.Service
	stats  *analytics.Service
	notify *notification.Service
}

// New returns the service.
func New() *Service {
	return &Service{repo: budgetrepo.New(), cats: category.New(), users: user.New(), stats: analytics.New(), notify: notification.New()}
}

func (s *Service) rows(ctx context.Context, uid bson.ObjectID, month string) ([]Row, Summary, error) {
	budgets, err := s.repo.ListByMonth(ctx, uid, month)
	if err != nil {
		return nil, Summary{}, err
	}
	spent := map[string]analytics.CategoryTotal{}
	if len(budgets) > 0 {
		totals, err := s.stats.ByCategory(ctx, uid, MonthRange(month))
		if err != nil {
			return nil, Summary{}, err
		}
		for _, t := range totals {
			if t.CategoryID != nil {
				spent[t.CategoryID.Hex()] = t
			}
		}
	}
	out := make([]Row, 0, len(budgets))
	var sum Summary
	for _, b := range budgets {
		t := spent[b.CategoryID.Hex()]
		r := Row{Budget: b, Spent: t.Total, Count: t.Count, Remaining: b.Limit - t.Total}
		if b.Limit > 0 {
			r.Percent = float64(t.Total) * 100 / float64(b.Limit)
		}
		sum.TotalLimit += b.Limit
		sum.TotalSpent += t.Total
		out = append(out, r)
	}
	return out, sum, nil
}

// List returns the month's budgets with progress.
func (s *Service) List(ctx context.Context, uid bson.ObjectID, month string) ([]Row, Summary, error) {
	if !ValidMonth(month) {
		return nil, Summary{}, gohttp.ErrValidation(map[string]string{"month": "must be YYYY-MM"})
	}
	return s.rows(ctx, uid, month)
}

// Upsert creates or updates the limit for (category, month).
func (s *Service) Upsert(ctx context.Context, uid bson.ObjectID, categoryID, month string, limit int64) (*Row, error) {
	fields := map[string]string{}
	if !ValidMonth(month) {
		fields["month"] = "must be YYYY-MM"
	}
	if limit <= 0 {
		fields["limit"] = "must be greater than zero"
	}
	cid, err := bson.ObjectIDFromHex(categoryID)
	if err != nil {
		fields["categoryId"] = "is invalid"
	}
	if len(fields) > 0 {
		return nil, gohttp.ErrValidation(fields)
	}
	if _, err := s.cats.Get(ctx, uid, cid); err != nil {
		return nil, gohttp.ErrValidation(map[string]string{"categoryId": "not found"})
	}
	currency := "INR"
	if u, err := s.users.FindByID(ctx, uid); err == nil && u.Preferences.Currency != "" {
		currency = u.Preferences.Currency
	}
	b, err := s.repo.Upsert(ctx, uid, cid, month, limit, currency)
	if err != nil {
		return nil, err
	}
	rows, _, err := s.rows(ctx, uid, month)
	if err != nil {
		return nil, err
	}
	for i := range rows {
		if rows[i].ID == b.ID {
			return &rows[i], nil
		}
	}
	return &Row{Budget: *b, Remaining: b.Limit}, nil
}

// Delete removes a budget.
func (s *Service) Delete(ctx context.Context, uid, id bson.ObjectID) error {
	if err := s.repo.Delete(ctx, uid, id); err != nil {
		if errors.Is(err, budgetrepo.ErrNotFound) {
			return gohttp.ErrNotFound
		}
		return err
	}
	return nil
}

// ListAllForUser returns every budget (export).
func (s *Service) ListAllForUser(ctx context.Context, uid bson.ObjectID) ([]models.Budget, error) {
	return s.repo.ListAllForUser(ctx, uid)
}

// DeleteAllForUser removes every budget (account cascade).
func (s *Service) DeleteAllForUser(ctx context.Context, uid bson.ObjectID) error {
	return s.repo.DeleteAllForUser(ctx, uid)
}

// UsersWithBudgets lists users that have a budget for the month (scheduler).
func (s *Service) UsersWithBudgets(ctx context.Context, month string) ([]bson.ObjectID, error) {
	return s.repo.UsersWithBudgets(ctx, month)
}

// CheckAlerts emits budget_alert notifications for budgets at ≥80% and
// ≥100% of their limit. Each threshold fires once per budget (dedupe key).
// Returns how many notifications were created.
func (s *Service) CheckAlerts(ctx context.Context, uid bson.ObjectID, month string) int {
	u, err := s.users.FindByID(ctx, uid)
	if err != nil || !u.Preferences.Notifications.BudgetAlerts {
		return 0
	}
	rows, _, err := s.rows(ctx, uid, month)
	if err != nil {
		return 0
	}
	names := map[string]string{}
	if cats, err := s.cats.ListForUser(ctx, uid); err == nil {
		for _, c := range cats {
			names[c.ID.Hex()] = c.Name
		}
	}
	sent := 0
	for _, r := range rows {
		name := names[r.CategoryID.Hex()]
		if name == "" {
			name = "a category"
		}
		label := formatMonth(month)
		data := map[string]any{"budgetId": r.ID.Hex(), "categoryId": r.CategoryID.Hex(), "month": month, "spent": r.Spent, "limit": r.Limit, "percent": int(r.Percent)}
		if r.Percent >= 100 {
			data["threshold"] = 100
			if s.notify.Notify(ctx, uid, models.NotifyBudgetAlert, name+" budget exceeded",
				fmt.Sprintf("You have spent %s of your %s %s budget for %s.", money(r.Spent, r.Currency), money(r.Limit, r.Currency), name, label),
				data, "budget:"+r.ID.Hex()+":100") {
				sent++
			}
		} else if r.Percent >= 80 {
			data["threshold"] = 80
			if s.notify.Notify(ctx, uid, models.NotifyBudgetAlert, name+" budget at "+fmt.Sprintf("%d%%", int(r.Percent)),
				fmt.Sprintf("You have used %s of your %s %s budget for %s.", money(r.Spent, r.Currency), money(r.Limit, r.Currency), name, label),
				data, "budget:"+r.ID.Hex()+":80") {
				sent++
			}
		}
	}
	return sent
}

func formatMonth(month string) string {
	t, err := time.Parse("2006-01", month)
	if err != nil {
		return month
	}
	return t.Format("January 2006")
}

// money renders minor units as "INR 1,234.50" using plain ASCII.
func money(minor int64, currency string) string {
	neg := minor < 0
	if neg {
		minor = -minor
	}
	whole, frac := minor/100, minor%100
	digits := fmt.Sprintf("%d", whole)
	// Indian grouping: last 3, then 2s.
	var parts []string
	if len(digits) > 3 {
		head, tail := digits[:len(digits)-3], digits[len(digits)-3:]
		for len(head) > 2 {
			parts = append([]string{head[len(head)-2:]}, parts...)
			head = head[:len(head)-2]
		}
		parts = append([]string{head}, parts...)
		parts = append(parts, tail)
		digits = strings.Join(parts, ",")
	}
	out := digits
	if frac != 0 {
		out = fmt.Sprintf("%s.%02d", digits, frac)
	}
	if neg {
		out = "-" + out
	}
	if currency == "" {
		currency = "INR"
	}
	return currency + " " + out
}

// Money is exported for reports/notifications that need the same rendering.
func Money(minor int64, currency string) string { return money(minor, currency) }

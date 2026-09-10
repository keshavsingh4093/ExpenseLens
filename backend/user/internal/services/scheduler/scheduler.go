// Package scheduler runs the periodic jobs of the user backend in one
// goroutine: budget alerts, warranty reminders and the Monday weekly
// summary. It ticks hourly and runs the daily job once per UTC day; every
// notification carries a dedupe key, so an extra run is harmless.
package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"smartreceipt/backend/common/pkg/models"
	"smartreceipt/backend/common/services/user"

	"smartreceipt/backend/user/internal/services/analytics"
	"smartreceipt/backend/user/internal/services/budget"
	"smartreceipt/backend/user/internal/services/notification"
	"smartreceipt/backend/user/internal/services/warranty"
)

var (
	mu      sync.Mutex
	cancel  context.CancelFunc
	done    chan struct{}
	lastDay string
)

// InitialDelay and Interval are exported so tests can shorten them.
var (
	InitialDelay = 10 * time.Second
	Interval     = time.Hour
)

// Start launches the scheduler goroutine. Safe to call once.
func Start() {
	mu.Lock()
	defer mu.Unlock()
	if cancel != nil {
		return
	}
	var ctx context.Context
	ctx, cancel = context.WithCancel(context.Background())
	done = make(chan struct{})
	go loop(ctx, done)
}

// Stop cancels the goroutine and waits for it (server OnShutdown hook).
func Stop(ctx context.Context) {
	mu.Lock()
	c, d := cancel, done
	cancel, done = nil, nil
	mu.Unlock()
	if c == nil {
		return
	}
	c()
	select {
	case <-d:
	case <-ctx.Done():
	}
}

func loop(ctx context.Context, done chan struct{}) {
	defer close(done)
	t := time.NewTimer(InitialDelay)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		RunDaily(ctx, time.Now().UTC(), false)
		t.Reset(Interval)
	}
}

// RunDaily executes the daily job for now. Unless force is set it runs at
// most once per UTC day per process. Returns what it did (for logs/tests).
func RunDaily(ctx context.Context, now time.Time, force bool) map[string]int {
	day := now.Format(analytics.Day)
	mu.Lock()
	if !force && lastDay == day {
		mu.Unlock()
		return nil
	}
	lastDay = day
	mu.Unlock()

	ctx, cancelRun := context.WithTimeout(ctx, 10*time.Minute)
	defer cancelRun()
	out := map[string]int{}
	start := time.Now()

	out["budgetAlerts"] = budgetAlerts(ctx, now.Format("2006-01"))
	out["warrantyReminders"] = warranty.New().SendReminders(ctx, day)
	if now.Weekday() == time.Monday {
		out["weeklySummaries"] = weeklySummaries(ctx, now)
	}
	slog.Info("scheduler daily job", "day", day, "result", out, "took", time.Since(start).Round(time.Millisecond))
	return out
}

func budgetAlerts(ctx context.Context, month string) int {
	svc := budget.New()
	ids, err := svc.UsersWithBudgets(ctx, month)
	if err != nil {
		slog.Warn("scheduler: users with budgets", "err", err)
		return 0
	}
	n := 0
	for _, uid := range ids {
		n += svc.CheckAlerts(ctx, uid, month)
	}
	return n
}

func weeklySummaries(ctx context.Context, now time.Time) int {
	ids, err := user.New().IDsWithWeeklySummary(ctx)
	if err != nil {
		slog.Warn("scheduler: weekly summary users", "err", err)
		return 0
	}
	stats := analytics.New()
	notify := notification.New()
	users := user.New()
	y, w := now.ISOWeek()
	key := fmt.Sprintf("weekly:%04d-W%02d", y, w)
	n := 0
	for _, uid := range ids {
		tz := "Asia/Kolkata"
		currency := "INR"
		if u, err := users.FindByID(ctx, uid); err == nil {
			if u.Preferences.Timezone != "" {
				tz = u.Preferences.Timezone
			}
			if u.Preferences.Currency != "" {
				currency = u.Preferences.Currency
			}
		}
		local := now.In(analytics.Now(tz).Location())
		rg := analytics.Range{From: local.AddDate(0, 0, -7).Format(analytics.Day), To: local.AddDate(0, 0, -1).Format(analytics.Day)}
		tot, err := stats.Sum(ctx, uid, rg)
		if err != nil {
			continue
		}
		prev, _ := stats.Sum(ctx, uid, rg.Previous())
		body := fmt.Sprintf("You spent %s across %d receipts last week", budget.Money(tot.Total, currency), tot.Count)
		if prev.Total > 0 {
			pct := (float64(tot.Total) - float64(prev.Total)) * 100 / float64(prev.Total)
			if pct >= 0 {
				body += fmt.Sprintf(", up %.0f%% from the week before", pct)
			} else {
				body += fmt.Sprintf(", down %.0f%% from the week before", -pct)
			}
		}
		body += "."
		if notify.Notify(ctx, uid, models.NotifyWeeklySummary, "Your weekly spending summary", body,
			map[string]any{"from": rg.From, "to": rg.To, "total": tot.Total, "count": tot.Count}, key) {
			n++
		}
	}
	return n
}

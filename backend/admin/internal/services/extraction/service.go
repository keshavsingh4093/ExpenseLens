// Package extraction computes the admin's extraction monitoring numbers and
// forwards retries to the user backend (which owns the worker pool).
package extraction

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	gohttp "smartreceipt/backend/common/pkg/http"
	"smartreceipt/backend/common/services/receipt"
	usersvc "smartreceipt/backend/common/services/user"

	"smartreceipt/backend/admin/internal/config"
)

// Stats is the monitoring payload.
type Stats struct {
	Range       string                    `json:"range"`
	From        string                    `json:"from"`
	Totals      receipt.ExtractionTotals  `json:"totals"`
	FailurePct  float64                   `json:"failurePct"`
	ByDay       []receipt.ExtractionDay   `json:"byDay"`
	ByModel     []receipt.ExtractionModel `json:"byModel"`
	Extractor   string                    `json:"extractor"`
	Model       string                    `json:"model"`
	GeneratedAt time.Time                 `json:"generatedAt"`
}

// Service computes stats and retries.
type Service struct {
	receipts *receipt.Service
	users    *usersvc.Service
	client   *http.Client
}

// New returns the service.
func New() *Service {
	return &Service{receipts: receipt.New(), users: usersvc.New(), client: &http.Client{Timeout: 15 * time.Second}}
}

// rangeDays maps the query range to a window; the default is 30 days.
func rangeDays(r string) (string, int) {
	switch r {
	case "7d":
		return "7d", 7
	case "90d":
		return "90d", 90
	default:
		return "30d", 30
	}
}

// Stats aggregates receipts created within the range, filling missing days
// with zeros so charts have a continuous axis.
func (s *Service) Stats(ctx context.Context, rng string) (*Stats, error) {
	name, days := rangeDays(rng)
	now := time.Now().UTC()
	since := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -(days - 1))
	totals, err := s.receipts.ExtractionTotalsSince(ctx, since)
	if err != nil {
		return nil, err
	}
	days_, err := s.receipts.ExtractionByDay(ctx, since)
	if err != nil {
		return nil, err
	}
	models, err := s.receipts.ExtractionByModel(ctx, since)
	if err != nil {
		return nil, err
	}
	byDay := make(map[string]receipt.ExtractionDay, len(days_))
	for _, d := range days_ {
		byDay[d.Date] = d
	}
	filled := make([]receipt.ExtractionDay, 0, days)
	for i := 0; i < days; i++ {
		key := since.AddDate(0, 0, i).Format("2006-01-02")
		d, ok := byDay[key]
		if !ok {
			d = receipt.ExtractionDay{Date: key}
		}
		filled = append(filled, d)
	}
	out := &Stats{Range: name, From: since.Format("2006-01-02"), Totals: *totals, ByDay: filled, ByModel: models, GeneratedAt: now}
	if totals.Total > 0 {
		out.FailurePct = float64(totals.Failed) * 100 / float64(totals.Total)
	}
	out.Extractor = config.GetConfigVal("EXTRACTOR", "gemini")
	out.Model = config.GetConfigVal("GEMINI_MODEL", "gemini-3.1-pro-preview")
	return out, nil
}

// Failed pages failed receipts, joined with the owner's email.
func (s *Service) Failed(ctx context.Context, page, pageSize int) ([]receipt.FailedRow, int64, error) {
	rows, total, err := s.receipts.ListFailed(ctx, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	ids := make([]bson.ObjectID, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.UserID)
	}
	emails, err := s.users.EmailsByIDs(ctx, ids)
	if err != nil {
		return nil, 0, err
	}
	for i := range rows {
		rows[i].UserEmail = emails[rows[i].UserID]
	}
	return rows, total, nil
}

// ErrRetryUnavailable is returned when the user backend link is not configured.
var ErrRetryUnavailable = gohttp.ErrWithCode(http.StatusServiceUnavailable, "retry_unavailable",
	"retry needs CONFIG_RELOAD_SECRET and USER_API_URL configured on both services")

// RetryResult is what a retry returns.
type RetryResult struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

// Retry asks the user backend to re-run extraction for the receipt.
func (s *Service) Retry(ctx context.Context, id bson.ObjectID) (*RetryResult, error) {
	rc, err := s.receipts.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, receipt.ErrNotFound) {
			return nil, gohttp.ErrNotFound
		}
		return nil, err
	}
	if len(rc.Images) == 0 {
		return nil, gohttp.ErrBadRequest("receipt has no images to extract from")
	}
	c := config.AppConfig
	if c.ConfigReloadSecret == "" || c.UserAPIURL == "" {
		return nil, ErrRetryUnavailable
	}
	url := strings.TrimRight(c.UserAPIURL, "/") + "/openapi/receipts/" + id.Hex() + "/reextract"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Config-Reload-Secret", c.ConfigReloadSecret)
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, gohttp.ErrWithCode(http.StatusBadGateway, "user_backend_unreachable", "user backend did not respond: "+err.Error())
	}
	defer resp.Body.Close()
	var env struct {
		Success bool        `json:"success"`
		Data    RetryResult `json:"data"`
		Error   string      `json:"error"`
		Code    string      `json:"code"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil || !env.Success {
		msg := env.Error
		if msg == "" {
			msg = fmt.Sprintf("user backend returned HTTP %d", resp.StatusCode)
		}
		return nil, gohttp.ErrWithCode(http.StatusBadGateway, "retry_failed", msg)
	}
	return &env.Data, nil
}

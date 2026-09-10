package analytics

import (
	"context"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/bson"

	"expenselens/backend/internal/server"
)

// Handler exposes /analytics.
type Handler struct {
	repo   *Repo
	userID func(context.Context) bson.ObjectID
}

// NewHandler constructs the handler.
func NewHandler(repo *Repo, userID func(context.Context) bson.ObjectID) *Handler {
	return &Handler{repo: repo, userID: userID}
}

// Mount registers routes (caller applies auth middleware).
func (h *Handler) Mount(r chi.Router) {
	r.Route("/analytics", func(r chi.Router) {
		r.Get("/summary", h.summary)
		r.Get("/trend", h.trend)
		r.Get("/categories", h.categories)
		r.Get("/merchants", h.merchants)
		r.Get("/stats", h.stats)
	})
}

func tz(r *http.Request) string {
	if t := r.URL.Query().Get("tz"); t != "" {
		return t
	}
	return "Asia/Kolkata"
}

func rangeFrom(r *http.Request) (Range, error) {
	q := r.URL.Query()
	rg, err := ParseRange(q.Get("range"), q.Get("from"), q.Get("to"), Now(tz(r)))
	if err != nil {
		return rg, server.BadRequest(err.Error())
	}
	return rg, nil
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

// summary returns the four dashboard tiles.
func (h *Handler) summary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	uid := h.userID(ctx)
	now := Now(tz(r))
	today := now.Format(day)

	allTime := Range{From: "1970-01-01", To: today}
	thisMonth := Range{From: now.Format("2006-01") + "-01", To: today}
	lastMonthStart := now.AddDate(0, -1, -now.Day()+1)
	lastMonth := Range{From: lastMonthStart.Format(day), To: now.AddDate(0, 0, -now.Day()).Format(day)}
	todayR := Range{From: today, To: today}
	yesterday := Range{From: now.AddDate(0, 0, -1).Format(day), To: now.AddDate(0, 0, -1).Format(day)}
	last30 := Range{From: now.AddDate(0, 0, -29).Format(day), To: today}
	prev30 := last30.Previous()

	get := func(rg Range) Totals { t, _ := h.repo.Sum(ctx, uid, rg); return t }
	all := get(allTime)
	tm, lm := get(thisMonth), get(lastMonth)
	td, yd := get(todayR), get(yesterday)
	l30, p30 := get(last30), get(prev30)
	avg := Totals{Total: l30.Total / 30, Count: l30.Count}
	prevAvg := Totals{Total: p30.Total / 30, Count: p30.Count}

	server.JSON(w, http.StatusOK, map[string]any{
		"totalExpenses": kpi(all, lm, "vs last month"),
		"thisMonth":     kpi(tm, lm, "vs last month"),
		"today":         kpi(td, yd, "vs yesterday"),
		"averageDaily":  kpi(avg, prevAvg, "vs last 30 days"),
	})
}

func (h *Handler) trend(w http.ResponseWriter, r *http.Request) {
	rg, err := rangeFrom(r)
	if err != nil {
		server.Error(w, r, err)
		return
	}
	g := r.URL.Query().Get("granularity")
	if g == "" {
		switch {
		case rg.Days() <= 31:
			g = "day"
		case rg.Days() <= 120:
			g = "week"
		default:
			g = "month"
		}
	}
	points, err := h.repo.ByDay(r.Context(), h.userID(r.Context()), rg)
	if err != nil {
		server.Error(w, r, err)
		return
	}
	server.JSON(w, http.StatusOK, map[string]any{"range": rg, "granularity": g, "points": Rollup(points, rg, g)})
}

func (h *Handler) categories(w http.ResponseWriter, r *http.Request) {
	rg, err := rangeFrom(r)
	if err != nil {
		server.Error(w, r, err)
		return
	}
	out, err := h.repo.ByCategory(r.Context(), h.userID(r.Context()), rg)
	if err != nil {
		server.Error(w, r, err)
		return
	}
	server.JSON(w, http.StatusOK, map[string]any{"range": rg, "categories": out})
}

func (h *Handler) merchants(w http.ResponseWriter, r *http.Request) {
	rg, err := rangeFrom(r)
	if err != nil {
		server.Error(w, r, err)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	out, err := h.repo.TopMerchants(r.Context(), h.userID(r.Context()), rg, limit)
	if err != nil {
		server.Error(w, r, err)
		return
	}
	server.JSON(w, http.StatusOK, map[string]any{"range": rg, "merchants": out})
}

func (h *Handler) stats(w http.ResponseWriter, r *http.Request) {
	rg, err := rangeFrom(r)
	if err != nil {
		server.Error(w, r, err)
		return
	}
	points, err := h.repo.ByDay(r.Context(), h.userID(r.Context()), rg)
	if err != nil {
		server.Error(w, r, err)
		return
	}
	s := ComputeStats(points, rg)
	server.JSON(w, http.StatusOK, map[string]any{"range": rg, "stats": s})
}

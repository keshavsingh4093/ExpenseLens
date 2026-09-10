package analytics

import (
	"testing"
	"time"
)

func TestParseRangeAndPrevious(t *testing.T) {
	now := time.Date(2026, 6, 12, 10, 0, 0, 0, time.UTC)
	rg, err := ParseRange("7d", "", "", now)
	if err != nil || rg.From != "2026-06-06" || rg.To != "2026-06-12" || rg.Days() != 7 {
		t.Fatalf("7d: %+v %v", rg, err)
	}
	prev := rg.Previous()
	if prev.From != "2026-05-30" || prev.To != "2026-06-05" {
		t.Fatalf("previous: %+v", prev)
	}
	if rg, _ := ParseRange("month", "", "", now); rg.From != "2026-06-01" {
		t.Fatalf("month: %+v", rg)
	}
	if _, err := ParseRange("", "2026-13-01", "", now); err == nil {
		t.Fatal("expected invalid date")
	}
}

func TestRollupAndStats(t *testing.T) {
	rg := Range{From: "2026-06-01", To: "2026-06-03"}
	pts := []DayPoint{{Date: "2026-06-01", Total: 100, Count: 1}, {Date: "2026-06-03", Total: 300, Count: 2}}
	sortPoints(pts)
	b := Rollup(pts, rg, "day")
	if len(b) != 3 || b[0].Total != 100 || b[1].Total != 0 || b[2].Total != 300 || b[2].Label != "03 Jun" {
		t.Fatalf("day rollup: %+v", b)
	}
	m := Rollup(pts, Range{From: "2026-05-30", To: "2026-06-02"}, "month")
	if len(m) != 2 || m[0].Period != "2026-05" || m[1].Total != 400 {
		t.Fatalf("month rollup: %+v", m)
	}
	s := ComputeStats(pts, rg)
	if s.Total != 400 || s.Transactions != 3 || s.AvgDaily != 133 || s.AvgPerTx != 133 || s.HighestDay.Date != "2026-06-03" {
		t.Fatalf("stats: %+v", s)
	}
}

func TestDefaultGranularity(t *testing.T) {
	if g := DefaultGranularity(Range{From: "2026-06-01", To: "2026-06-30"}); g != "day" {
		t.Fatalf("30d: %s", g)
	}
	if g := DefaultGranularity(Range{From: "2026-04-01", To: "2026-06-30"}); g != "week" {
		t.Fatalf("3m: %s", g)
	}
	if g := DefaultGranularity(Range{From: "2025-06-01", To: "2026-06-30"}); g != "month" {
		t.Fatalf("1y: %s", g)
	}
}

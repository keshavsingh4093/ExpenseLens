package extraction

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

func TestParseWire(t *testing.T) {
	data := []byte(`{"merchant":" Reliance Smart ","date":"2026-06-11","time":"18:42","currency":"inr",
	"subtotal":1120.5,"tax":110,"total":1230.5,"paymentMethod":"upi","gstin":"27abcde1234f1z5",
	"taxBreakdown":[{"label":"CGST 9%","amount":55},{"label":"SGST 9%","amount":55}],
	"items":[{"name":"Rice 5kg","quantity":2,"unitPrice":325,"amount":650},{"name":"Oil","quantity":null,"unitPrice":null,"amount":210.25}],
	"suggestedCategory":"groceries","warrantyCandidates":[],"confidence":0.93}`)
	r, err := parseWire(data, "gemini:test")
	if err != nil {
		t.Fatal(err)
	}
	if r.Merchant != "Reliance Smart" || r.Currency != "INR" || r.Total != 123050 || *r.Tax != 11000 || *r.Subtotal != 112050 {
		t.Fatalf("bad header: %+v", r)
	}
	if r.GSTIN != "27ABCDE1234F1Z5" || len(r.TaxBreakdown) != 2 || r.TaxBreakdown[0].Amount != 5500 {
		t.Fatalf("bad tax: %+v", r)
	}
	if len(r.Items) != 2 || r.Items[0].Amount != 65000 || *r.Items[0].UnitPrice != 32500 || r.Items[1].Amount != 21025 || r.Items[1].UnitPrice != nil {
		t.Fatalf("bad items: %+v", r.Items)
	}
	if r.SuggestedCategory != "groceries" || r.Confidence != 0.93 || r.Model != "gemini:test" {
		t.Fatalf("bad meta: %+v", r)
	}
}

func TestToMinor(t *testing.T) {
	for _, c := range []struct {
		in   float64
		want int64
	}{{0, 0}, {1, 100}, {0.1, 10}, {1230.5, 123050}, {19.99, 1999}, {0.29, 29}, {-5.5, -550}} {
		if got := toMinor(c.in); got != c.want {
			t.Errorf("toMinor(%v) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestCodeOf(t *testing.T) {
	for _, c := range []struct {
		err  error
		want string
	}{
		{nil, ""},
		{newError(codeForStatus(429), "quota"), CodeRateLimited},
		{newError(codeForStatus(503), "down"), CodeUnavailable},
		{newError(codeForStatus(400), "bad key"), CodeFailed},
		{newError(CodeBlocked, "blocked"), CodeBlocked},
		{fmt.Errorf("wrapped: %w", newError(CodeBadResponse, "json")), CodeBadResponse},
		{context.DeadlineExceeded, CodeUnavailable},
		{errors.New("plain"), CodeFailed},
	} {
		if got := CodeOf(c.err); got != c.want {
			t.Errorf("CodeOf(%v) = %q, want %q", c.err, got, c.want)
		}
	}
}

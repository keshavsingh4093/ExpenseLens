package storage

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestLocalRoundTrip(t *testing.T) {
	l, err := NewLocal(t.TempDir(), "http://x/files", "secret")
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	obj, err := l.Put(ctx, "receipts/u1/a.png", strings.NewReader("png-bytes"), "image/png")
	if err != nil {
		t.Fatal(err)
	}
	if obj.Size != 9 || obj.ContentType != "image/png" {
		t.Fatalf("bad object %+v", obj)
	}
	rc, ct, err := l.Get(ctx, "receipts/u1/a.png")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(rc)
	rc.Close()
	if string(b) != "png-bytes" || ct != "image/png" {
		t.Fatalf("got %q %q", b, ct)
	}
	if _, err := l.Put(ctx, "../escape", strings.NewReader("x"), "text/plain"); err == nil {
		t.Fatal("expected invalid key error")
	}

	r := chi.NewRouter()
	r.Get("/files/*", l.Handler().ServeHTTP)
	srv := httptest.NewServer(r)
	defer srv.Close()

	u := strings.Replace(l.URL("receipts/u1/a.png"), "http://x", srv.URL, 1)
	resp, err := http.Get(u)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("signed url status %d", resp.StatusCode)
	}
	resp.Body.Close()
	resp, _ = http.Get(strings.Replace(u, "sig=", "sig=00", 1))
	if resp.StatusCode != 403 {
		t.Fatalf("tampered sig status %d", resp.StatusCode)
	}
	resp.Body.Close()

	if err := l.Delete(ctx, "receipts/u1/a.png"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := l.Get(ctx, "receipts/u1/a.png"); err != ErrNotFound {
		t.Fatalf("expected not found, got %v", err)
	}
}

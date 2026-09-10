package storage

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

// Local stores files on disk and serves them through a signed URL handler.
type Local struct {
	root    string
	baseURL string
	secret  []byte
	ttl     time.Duration
}

// NewLocal creates the root directory if needed.
func NewLocal(root, baseURL, secret string) (*Local, error) {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, err
	}
	return &Local{root: root, baseURL: baseURL, secret: []byte(secret), ttl: 2 * time.Hour}, nil
}

func (l *Local) Name() string { return "local" }

func (l *Local) Put(_ context.Context, key string, r io.Reader, contentType string) (*Object, error) {
	if err := SafeKey(key); err != nil {
		return nil, err
	}
	p := filepath.Join(l.root, filepath.FromSlash(key))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return nil, err
	}
	f, err := os.Create(p)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	n, err := io.Copy(f, r)
	if err != nil {
		return nil, err
	}
	// sidecar with the content type so Get can return it
	_ = os.WriteFile(p+".type", []byte(contentType), 0o644)
	return &Object{Key: key, URL: l.URL(key), ContentType: contentType, Size: n}, nil
}

func (l *Local) Get(_ context.Context, key string) (io.ReadCloser, string, error) {
	if err := SafeKey(key); err != nil {
		return nil, "", err
	}
	p := filepath.Join(l.root, filepath.FromSlash(key))
	f, err := os.Open(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, "", ErrNotFound
		}
		return nil, "", err
	}
	ct, _ := os.ReadFile(p + ".type")
	if len(ct) == 0 {
		ct = []byte("application/octet-stream")
	}
	return f, string(ct), nil
}

func (l *Local) Delete(_ context.Context, key string) error {
	if err := SafeKey(key); err != nil {
		return err
	}
	p := filepath.Join(l.root, filepath.FromSlash(key))
	_ = os.Remove(p + ".type")
	if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// URL returns baseURL/<key>?exp=<unix>&sig=<hmac>.
func (l *Local) URL(key string) string {
	exp := strconv.FormatInt(time.Now().Add(l.ttl).Unix(), 10)
	return fmt.Sprintf("%s/%s?exp=%s&sig=%s", l.baseURL, key, exp, l.sign(key, exp))
}

func (l *Local) sign(key, exp string) string {
	m := hmac.New(sha256.New, l.secret)
	m.Write([]byte(key + "|" + exp))
	return hex.EncodeToString(m.Sum(nil))
}

// Handler serves GET /files/* with signature and expiry checks. Mount it on
// the API router without auth middleware; the signature is the credential.
func (l *Local) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := chi.URLParam(r, "*")
		if key == "" {
			key, _ = url.PathUnescape(r.URL.Path)
		}
		q := r.URL.Query()
		exp, sig := q.Get("exp"), q.Get("sig")
		expAt, err := strconv.ParseInt(exp, 10, 64)
		if err != nil || time.Now().Unix() > expAt || !hmac.Equal([]byte(sig), []byte(l.sign(key, exp))) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		rc, ct, err := l.Get(r.Context(), key)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		defer rc.Close()
		w.Header().Set("Content-Type", ct)
		w.Header().Set("Cache-Control", "private, max-age=3600")
		_, _ = io.Copy(w, rc)
	})
}

// Package storage abstracts file storage so the platform (Cloudinary, local
// disk, S3 later) can be swapped without touching receipts code.
package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
)

// Object describes a stored file.
type Object struct {
	Key         string `json:"key"`
	URL         string `json:"url"`
	ContentType string `json:"contentType"`
	Size        int64  `json:"size"`
}

// Provider is implemented by each storage backend.
type Provider interface {
	// Name identifies the backend ("local", "cloudinary").
	Name() string
	// Put stores r under key and returns the object. Keys look like
	// "receipts/<userId>/<random>.<ext>".
	Put(ctx context.Context, key string, r io.Reader, contentType string) (*Object, error)
	// Get streams the stored bytes back (used for re-extraction).
	Get(ctx context.Context, key string) (io.ReadCloser, string, error)
	// Delete removes the object; missing objects are not an error.
	Delete(ctx context.Context, key string) error
	// URL returns a browser-loadable URL for key, valid for at least an hour.
	URL(key string) string
}

// ErrNotFound is returned by Get for missing keys.
var ErrNotFound = errors.New("storage: object not found")

// Config selects and configures the provider.
type Config struct {
	Provider      string // local | cloudinary
	LocalDir      string // local: root directory
	LocalBaseURL  string // local: public base URL of the signed-file handler
	LocalSecret   string // local: HMAC secret for signed URLs
	CloudinaryURL string // cloudinary://key:secret@cloud
}

// New picks the provider from config.
func New(cfg Config) (Provider, error) {
	switch cfg.Provider {
	case "local", "":
		return NewLocal(cfg.LocalDir, cfg.LocalBaseURL, cfg.LocalSecret)
	case "cloudinary":
		return NewCloudinary(cfg.CloudinaryURL)
	default:
		return nil, fmt.Errorf("storage: unknown provider %q", cfg.Provider)
	}
}

// ExtFor maps a content type to a file extension.
func ExtFor(contentType string) string {
	switch contentType {
	case "image/jpeg":
		return "jpg"
	case "image/png":
		return "png"
	case "image/webp":
		return "webp"
	case "image/heic":
		return "heic"
	case "application/pdf":
		return "pdf"
	}
	return "bin"
}

// SafeKey rejects keys that could escape the storage root.
func SafeKey(key string) error {
	if key == "" || strings.Contains(key, "..") || strings.HasPrefix(key, "/") || path.Clean(key) != key {
		return fmt.Errorf("storage: invalid key %q", key)
	}
	return nil
}

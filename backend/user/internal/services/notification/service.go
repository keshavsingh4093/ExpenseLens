// Package notification is the user portal's in-app notification logic. It
// wraps the common collection service and is the single place other domains
// (budgets, warranties, extraction) call to emit a message.
package notification

import (
	"context"
	"errors"
	"log/slog"

	"go.mongodb.org/mongo-driver/v2/bson"

	gohttp "smartreceipt/backend/common/pkg/http"
	"smartreceipt/backend/common/pkg/models"
	notifrepo "smartreceipt/backend/common/services/notification"
)

// Service emits and reads notifications.
type Service struct {
	repo *notifrepo.Service
}

// New returns the service.
func New() *Service { return &Service{repo: notifrepo.New()} }

// Notify stores a notification. When dedupeKey is set and a notification with
// that key already exists for the user, nothing is stored and false is
// returned. Errors are logged, never returned: notifications are best effort.
func (s *Service) Notify(ctx context.Context, uid bson.ObjectID, typ, title, body string, data map[string]any, dedupeKey string) bool {
	n := &models.Notification{UserID: uid, Type: typ, Title: title, Body: body, Data: data, DedupeKey: dedupeKey}
	err := s.repo.Insert(ctx, n)
	switch {
	case err == nil:
		return true
	case errors.Is(err, notifrepo.ErrDuplicate):
		return false
	default:
		slog.Warn("notify failed", "user", uid.Hex(), "type", typ, "err", err)
		return false
	}
}

// Page is a list response.
type Page struct {
	Notifications []models.Notification `json:"notifications"`
	Total         int64                 `json:"total"`
	Unread        int64                 `json:"unread"`
	Page          int                   `json:"page"`
	PageSize      int                   `json:"pageSize"`
}

// List returns one page plus the unread count.
func (s *Service) List(ctx context.Context, uid bson.ObjectID, page, pageSize int) (*Page, error) {
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	if page < 1 {
		page = 1
	}
	rows, total, err := s.repo.List(ctx, uid, page, pageSize)
	if err != nil {
		return nil, err
	}
	unread, err := s.repo.UnreadCount(ctx, uid)
	if err != nil {
		return nil, err
	}
	return &Page{Notifications: rows, Total: total, Unread: unread, Page: page, PageSize: pageSize}, nil
}

// UnreadCount returns how many notifications are unread.
func (s *Service) UnreadCount(ctx context.Context, uid bson.ObjectID) (int64, error) {
	return s.repo.UnreadCount(ctx, uid)
}

// Read marks one notification read.
func (s *Service) Read(ctx context.Context, uid, id bson.ObjectID) (*models.Notification, error) {
	n, err := s.repo.MarkRead(ctx, uid, id)
	if errors.Is(err, notifrepo.ErrNotFound) {
		return nil, gohttp.ErrNotFound
	}
	return n, err
}

// ReadAll marks everything read and returns the count.
func (s *Service) ReadAll(ctx context.Context, uid bson.ObjectID) (int64, error) {
	return s.repo.MarkAllRead(ctx, uid)
}

// ListAllForUser returns every notification (export), newest first, capped.
func (s *Service) ListAllForUser(ctx context.Context, uid bson.ObjectID) ([]models.Notification, error) {
	rows, _, err := s.repo.List(ctx, uid, 1, 100)
	return rows, err
}

// DeleteAllForUser removes every notification (account cascade).
func (s *Service) DeleteAllForUser(ctx context.Context, uid bson.ObjectID) error {
	return s.repo.DeleteAllForUser(ctx, uid)
}

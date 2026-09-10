// Package stats computes the admin dashboard numbers.
package stats

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"smartreceipt/backend/common/services/receipt"
	"smartreceipt/backend/common/services/user"
)

// Overview is the admin dashboard payload.
type Overview struct {
	Users struct {
		Total       int64 `json:"total"`
		Last7Days   int64 `json:"last7Days"`
		Last30Days  int64 `json:"last30Days"`
		WithReceipt int64 `json:"withReceipts"`
	} `json:"users"`
	Receipts struct {
		Total      int64            `json:"total"`
		Last7Days  int64            `json:"last7Days"`
		Last30Days int64            `json:"last30Days"`
		ByStatus   map[string]int64 `json:"byStatus"`
		Failed     int64            `json:"failed"`
		FailurePct float64          `json:"failurePct"`
	} `json:"receipts"`
	GeneratedAt time.Time `json:"generatedAt"`
}

// Service computes stats.
type Service struct {
	users    *user.Service
	receipts *receipt.Service
}

// New returns the service.
func New() *Service { return &Service{users: user.New(), receipts: receipt.New()} }

// Overview gathers counts. Any query error aborts.
func (s *Service) Overview(ctx context.Context) (*Overview, error) {
	now := time.Now().UTC()
	var o Overview
	var err error
	if o.Users.Total, err = s.users.Count(ctx); err != nil {
		return nil, err
	}
	if o.Users.Last7Days, err = s.users.CountSince(ctx, now.AddDate(0, 0, -7)); err != nil {
		return nil, err
	}
	if o.Users.Last30Days, err = s.users.CountSince(ctx, now.AddDate(0, 0, -30)); err != nil {
		return nil, err
	}
	if o.Users.WithReceipt, err = s.receipts.CountUsersWithReceipts(ctx); err != nil {
		return nil, err
	}
	if o.Receipts.Total, err = s.receipts.Count(ctx, nil); err != nil {
		return nil, err
	}
	if o.Receipts.Last7Days, err = s.receipts.Count(ctx, bson.M{"createdAt": bson.M{"$gte": now.AddDate(0, 0, -7)}}); err != nil {
		return nil, err
	}
	if o.Receipts.Last30Days, err = s.receipts.Count(ctx, bson.M{"createdAt": bson.M{"$gte": now.AddDate(0, 0, -30)}}); err != nil {
		return nil, err
	}
	if o.Receipts.ByStatus, err = s.receipts.CountByStatus(ctx); err != nil {
		return nil, err
	}
	for _, st := range []string{"processing", "review", "confirmed", "failed"} {
		if _, ok := o.Receipts.ByStatus[st]; !ok {
			o.Receipts.ByStatus[st] = 0
		}
	}
	o.Receipts.Failed = o.Receipts.ByStatus["failed"]
	if o.Receipts.Total > 0 {
		o.Receipts.FailurePct = float64(o.Receipts.Failed) * 100 / float64(o.Receipts.Total)
	}
	o.GeneratedAt = now
	return &o, nil
}

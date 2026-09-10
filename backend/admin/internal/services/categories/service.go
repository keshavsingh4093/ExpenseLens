// Package categories manages the system (default) categories every user sees.
package categories

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/bson"

	gohttp "smartreceipt/backend/common/pkg/http"
	"smartreceipt/backend/common/pkg/models"
	"smartreceipt/backend/common/services/category"
	"smartreceipt/backend/common/services/receipt"
)

// Row is a system category with its usage count.
type Row struct {
	models.Category
	ReceiptCount int64 `json:"receiptCount"`
}

// Service manages system categories.
type Service struct {
	cats     *category.Service
	receipts *receipt.Service
}

// New returns the service.
func New() *Service { return &Service{cats: category.New(), receipts: receipt.New()} }

// List returns every system category with receipt counts across all users.
func (s *Service) List(ctx context.Context) ([]Row, error) {
	cats, err := s.cats.ListSystem(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Row, 0, len(cats))
	for _, c := range cats {
		n, _ := s.receipts.CountByCategory(ctx, c.ID)
		out = append(out, Row{Category: c, ReceiptCount: n})
	}
	return out, nil
}

// Create adds a system category.
func (s *Service) Create(ctx context.Context, name, slug, icon, color string) (*models.Category, error) {
	c, err := s.cats.CreateSystem(ctx, name, slug, icon, color)
	if err != nil {
		return nil, mapErr(err)
	}
	return c, nil
}

// Update patches a system category and returns it.
func (s *Service) Update(ctx context.Context, id bson.ObjectID, set bson.M) (*models.Category, error) {
	if err := s.cats.UpdateSystem(ctx, id, set); err != nil {
		return nil, mapErr(err)
	}
	c, err := s.cats.GetSystem(ctx, id)
	if err != nil {
		return nil, mapErr(err)
	}
	return c, nil
}

// Delete removes a system category after moving every user's receipts in
// it to "others". Returns the deleted category and how many receipts moved.
func (s *Service) Delete(ctx context.Context, id bson.ObjectID) (*models.Category, int64, error) {
	c, err := s.cats.GetSystem(ctx, id)
	if err != nil {
		return nil, 0, mapErr(err)
	}
	if c.Slug == models.OthersSlug {
		return nil, 0, mapErr(category.ErrOthersProtected)
	}
	others, err := s.cats.SystemBySlug(ctx, models.OthersSlug)
	if err != nil {
		return nil, 0, mapErr(err)
	}
	moved, err := s.receipts.ReassignCategoryAll(ctx, id, others.ID)
	if err != nil {
		return nil, 0, err
	}
	if err := s.cats.DeleteSystem(ctx, id); err != nil {
		return nil, 0, mapErr(err)
	}
	return c, moved, nil
}

func mapErr(err error) error {
	switch {
	case errors.Is(err, category.ErrNotFound):
		return gohttp.ErrNotFound
	case errors.Is(err, category.ErrExists):
		return gohttp.ErrConflict("category_exists", "a system category with that slug already exists")
	case errors.Is(err, category.ErrOthersProtected):
		return gohttp.ErrConflict("others_protected", "the others category cannot be deleted or renamed")
	}
	return err
}

package user

import (
	"context"
	"errors"
	"regexp"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"smartreceipt/backend/common/pkg/database"
	"smartreceipt/backend/common/pkg/models"
)

var (
	// ErrNotFound is returned when no user matches.
	ErrNotFound = errors.New("user not found")
	// ErrEmailTaken is returned on duplicate email.
	ErrEmailTaken = errors.New("email already registered")
)

// Service reads and writes the users collection.
type Service struct{}

// New returns the service.
func New() *Service { return &Service{} }

// EnsureIndexes builds the users indexes (tooling only).
func EnsureIndexes(ctx context.Context) error { return database.EnsureIndexes(ctx, Indexes()) }

// Create inserts a new user, filling timestamps and defaults.
func (s *Service) Create(ctx context.Context, u *models.User) error {
	now := time.Now().UTC()
	u.Email = models.NormalizeEmail(u.Email)
	u.CreatedAt, u.UpdatedAt = now, now
	if u.Providers == nil {
		u.Providers = []models.ProviderLink{}
	}
	if u.Preferences.Currency == "" {
		u.Preferences = models.DefaultPreferences()
	}
	res, err := col().InsertOne(ctx, u)
	if err != nil {
		if database.IsDuplicateKey(err) {
			return ErrEmailTaken
		}
		return err
	}
	u.ID = res.InsertedID.(bson.ObjectID)
	return nil
}

// FindByID loads a user by id.
func (s *Service) FindByID(ctx context.Context, id bson.ObjectID) (*models.User, error) {
	return s.findOne(ctx, bson.M{"_id": id})
}

// FindByEmail loads a user by normalized email.
func (s *Service) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	return s.findOne(ctx, bson.M{"email": models.NormalizeEmail(email)})
}

// FindByProvider loads a user by a linked social identity.
func (s *Service) FindByProvider(ctx context.Context, provider, providerUserID string) (*models.User, error) {
	return s.findOne(ctx, bson.M{"providers": bson.M{"$elemMatch": bson.M{
		"provider": provider, "providerUserId": providerUserID,
	}}})
}

func (s *Service) findOne(ctx context.Context, filter bson.M) (*models.User, error) {
	var u models.User
	if err := col().FindOne(ctx, filter).Decode(&u); err != nil {
		if database.IsNoDocuments(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

// IsDisabled reports the account flag with a projected lookup (auth guard).
func (s *Service) IsDisabled(ctx context.Context, id bson.ObjectID) (bool, error) {
	var row struct {
		Disabled bool `bson:"disabled"`
	}
	err := col().FindOne(ctx, bson.M{"_id": id}, options.FindOne().SetProjection(bson.M{"disabled": 1})).Decode(&row)
	if err != nil {
		if database.IsNoDocuments(err) {
			return false, ErrNotFound
		}
		return false, err
	}
	return row.Disabled, nil
}

// Update applies a $set patch and bumps updatedAt.
func (s *Service) Update(ctx context.Context, id bson.ObjectID, set bson.M) error {
	set["updatedAt"] = time.Now().UTC()
	res, err := col().UpdateByID(ctx, id, bson.M{"$set": set})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}

// LinkProvider appends a social identity to the user.
func (s *Service) LinkProvider(ctx context.Context, id bson.ObjectID, link models.ProviderLink) error {
	link.LinkedAt = time.Now().UTC()
	_, err := col().UpdateByID(ctx, id, bson.M{
		"$push": bson.M{"providers": link},
		"$set":  bson.M{"updatedAt": link.LinkedAt},
	})
	return err
}

// Delete removes the user document. Callers are responsible for cascading.
func (s *Service) Delete(ctx context.Context, id bson.ObjectID) error {
	_, err := col().DeleteOne(ctx, bson.M{"_id": id})
	return err
}

// ListFilter drives List (admin portal).
type ListFilter struct {
	Query    string // matches email or name, case-insensitive
	Status   string // "" | active | disabled
	Page     int
	PageSize int
}

// List returns a page of users, newest first, plus the total count.
func (s *Service) List(ctx context.Context, f ListFilter) ([]models.User, int64, error) {
	filter := bson.M{}
	if q := f.Query; q != "" {
		re := bson.Regex{Pattern: regexp.QuoteMeta(q), Options: "i"}
		filter["$or"] = bson.A{bson.M{"email": re}, bson.M{"name": re}}
	}
	switch f.Status {
	case "disabled":
		filter["disabled"] = true
	case "active":
		filter["disabled"] = bson.M{"$ne": true}
	}
	if f.PageSize <= 0 || f.PageSize > 100 {
		f.PageSize = 20
	}
	if f.Page < 1 {
		f.Page = 1
	}
	total, err := col().CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	cur, err := col().Find(ctx, filter, options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: -1}}).
		SetSkip(int64((f.Page-1)*f.PageSize)).SetLimit(int64(f.PageSize)))
	if err != nil {
		return nil, 0, err
	}
	var out []models.User
	if err := cur.All(ctx, &out); err != nil {
		return nil, 0, err
	}
	if out == nil {
		out = []models.User{}
	}
	return out, total, nil
}

// Count returns the number of users.
func (s *Service) Count(ctx context.Context) (int64, error) {
	return col().CountDocuments(ctx, bson.M{})
}

// CountSince returns users created at or after t.
func (s *Service) CountSince(ctx context.Context, t time.Time) (int64, error) {
	return col().CountDocuments(ctx, bson.M{"createdAt": bson.M{"$gte": t}})
}

var _ = mongo.ErrNoDocuments

// EmailsByIDs returns id → email for the given users (admin joins).
func (s *Service) EmailsByIDs(ctx context.Context, ids []bson.ObjectID) (map[bson.ObjectID]string, error) {
	out := map[bson.ObjectID]string{}
	if len(ids) == 0 {
		return out, nil
	}
	cur, err := col().Find(ctx, bson.M{"_id": bson.M{"$in": ids}}, options.Find().SetProjection(bson.M{"email": 1}))
	if err != nil {
		return nil, err
	}
	var rows []struct {
		ID    bson.ObjectID `bson:"_id"`
		Email string        `bson:"email"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[r.ID] = r.Email
	}
	return out, nil
}

// IDsWithWeeklySummary returns the ids of users who opted into the weekly
// summary notification (scheduler).
func (s *Service) IDsWithWeeklySummary(ctx context.Context) ([]bson.ObjectID, error) {
	cur, err := col().Find(ctx, bson.M{"preferences.notifications.weeklySummary": true, "disabled": bson.M{"$ne": true}},
		options.Find().SetProjection(bson.M{"_id": 1}))
	if err != nil {
		return nil, err
	}
	var rows []struct {
		ID bson.ObjectID `bson:"_id"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return nil, err
	}
	out := make([]bson.ObjectID, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.ID)
	}
	return out, nil
}

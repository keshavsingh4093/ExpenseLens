package users

import (
	"context"
	"errors"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"expenselens/backend/internal/db"
)

// ErrNotFound is returned when no user matches.
var ErrNotFound = errors.New("user not found")

// ErrEmailTaken is returned on duplicate email.
var ErrEmailTaken = errors.New("email already registered")

// Repo is the users collection.
type Repo struct{ col *mongo.Collection }

// NewRepo binds to the "users" collection.
func NewRepo(d *db.DB) *Repo { return &Repo{col: d.Collection("users")} }

// Indexes declares the users indexes.
func Indexes() db.IndexSet {
	return db.IndexSet{Collection: "users", Models: []mongo.IndexModel{
		{Keys: bson.D{{Key: "email", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "providers.provider", Value: 1}, {Key: "providers.providerUserId", Value: 1}}},
	}}
}

// NormalizeEmail lowercases and trims.
func NormalizeEmail(e string) string { return strings.ToLower(strings.TrimSpace(e)) }

// Create inserts a new user, filling timestamps and defaults.
func (r *Repo) Create(ctx context.Context, u *User) error {
	now := time.Now().UTC()
	u.Email = NormalizeEmail(u.Email)
	u.CreatedAt, u.UpdatedAt = now, now
	if u.Providers == nil {
		u.Providers = []ProviderLink{}
	}
	if u.Preferences.Currency == "" {
		u.Preferences = DefaultPreferences()
	}
	res, err := r.col.InsertOne(ctx, u)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return ErrEmailTaken
		}
		return err
	}
	u.ID = res.InsertedID.(bson.ObjectID)
	return nil
}

// FindByID loads a user by id.
func (r *Repo) FindByID(ctx context.Context, id bson.ObjectID) (*User, error) {
	return r.findOne(ctx, bson.M{"_id": id})
}

// FindByEmail loads a user by normalized email.
func (r *Repo) FindByEmail(ctx context.Context, email string) (*User, error) {
	return r.findOne(ctx, bson.M{"email": NormalizeEmail(email)})
}

// FindByProvider loads a user by a linked social identity.
func (r *Repo) FindByProvider(ctx context.Context, provider, providerUserID string) (*User, error) {
	return r.findOne(ctx, bson.M{"providers": bson.M{"$elemMatch": bson.M{
		"provider": provider, "providerUserId": providerUserID,
	}}})
}

func (r *Repo) findOne(ctx context.Context, filter bson.M) (*User, error) {
	var u User
	if err := r.col.FindOne(ctx, filter).Decode(&u); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

// Update applies a $set patch and bumps updatedAt.
func (r *Repo) Update(ctx context.Context, id bson.ObjectID, set bson.M) error {
	set["updatedAt"] = time.Now().UTC()
	res, err := r.col.UpdateByID(ctx, id, bson.M{"$set": set})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}

// LinkProvider appends a social identity to the user.
func (r *Repo) LinkProvider(ctx context.Context, id bson.ObjectID, link ProviderLink) error {
	link.LinkedAt = time.Now().UTC()
	_, err := r.col.UpdateByID(ctx, id, bson.M{
		"$push": bson.M{"providers": link},
		"$set":  bson.M{"updatedAt": link.LinkedAt},
	})
	return err
}

// Delete removes the user document. Callers are responsible for cascading.
func (r *Repo) Delete(ctx context.Context, id bson.ObjectID) error {
	_, err := r.col.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

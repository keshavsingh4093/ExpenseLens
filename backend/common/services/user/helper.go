// Package user is the users collection access shared by the portals.
package user

import (
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"smartreceipt/backend/common/pkg/database"
	"smartreceipt/backend/common/pkg/models"
)

func col() *mongo.Collection { return database.GetDBConnection().Collection(models.UsersCollection) }

// Indexes declares the users indexes.
func Indexes() database.IndexSet {
	return database.IndexSet{Collection: models.UsersCollection, Models: []mongo.IndexModel{
		{Keys: bson.D{{Key: "email", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "providers.provider", Value: 1}, {Key: "providers.providerUserId", Value: 1}}},
		{Keys: bson.D{{Key: "createdAt", Value: -1}}},
	}}
}

// Package receipt is the receipts collection access shared by the user API
// and admin statistics. Business logic (upload, extraction, confirm) lives in
// the user backend.
package receipt

import (
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"smartreceipt/backend/common/pkg/database"
	"smartreceipt/backend/common/pkg/models"
)

func col() *mongo.Collection { return database.GetDBConnection().Collection(models.ReceiptsCollection) }

// Indexes declares receipt indexes.
func Indexes() database.IndexSet {
	return database.IndexSet{Collection: models.ReceiptsCollection, Models: []mongo.IndexModel{
		{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "date", Value: -1}, {Key: "createdAt", Value: -1}}},
		{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "categoryId", Value: 1}}},
		{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "fileHash", Value: 1}}},
		{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "status", Value: 1}}},
		{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "merchantNormalized", Value: 1}}},
		{Keys: bson.D{{Key: "status", Value: 1}, {Key: "createdAt", Value: -1}}},
	}}
}

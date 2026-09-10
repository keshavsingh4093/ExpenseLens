// Package category manages system default and user-defined expense categories.
package category

import (
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"smartreceipt/backend/common/pkg/database"
	"smartreceipt/backend/common/pkg/models"
)

func col() *mongo.Collection {
	return database.GetDBConnection().Collection(models.CategoriesCollection)
}

// Indexes: unique slug per owner (nil owner = system).
func Indexes() database.IndexSet {
	return database.IndexSet{Collection: models.CategoriesCollection, Models: []mongo.IndexModel{
		{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "slug", Value: 1}}, Options: options.Index().SetUnique(true)},
	}}
}

package dbconnections

import "go.mongodb.org/mongo-driver/mongo"

type CacheDBConnection interface {
	Collection(collectionName string) *mongo.Collection
}

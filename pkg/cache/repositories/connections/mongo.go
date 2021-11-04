package dbconnections

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type CacheDBConfig struct {
	ConnectionString string
}

type CacheDBProductionConnection struct {
	config CacheDBConfig
	client *mongo.Client
}

var _ CacheDBConnection = (*CacheDBProductionConnection)(nil)

func NewCacheDBProductionConnection(ctx context.Context, config CacheDBConfig) (CacheDBConnection, error) {
	client, err := mongo.NewClient(options.Client().ApplyURI(config.ConnectionString))
	if err != nil {
		return nil, err
	}

	err = client.Connect(ctx)
	if err != nil {
		return nil, err
	}

	return &CacheDBProductionConnection{
		config: config,
		client: client,
	}, nil
}

func (c *CacheDBProductionConnection) Collection(collectionName string) *mongo.Collection {
	return c.client.Database("imcaxy").Collection(collectionName)
}

package connections

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoEnv struct {
	MongoConnectionString string
}

type IMongoConnectionFactory interface {
	CreateMongoConnection() (*mongo.Client, *context.Context, error)
}

type MongoConnectionFactory struct {
	connectionString string
}

func (factory MongoConnectionFactory) CreateMongoConnection() (*mongo.Client, *context.Context, error) {
	client, err := mongo.NewClient(options.Client().ApplyURI(factory.connectionString))
	if err != nil {
		return nil, nil, err
	}

	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	err = client.Connect(ctx)
	if err != nil {
		return nil, nil, err
	}

	return client, &ctx, nil
}

func CreateMongoConnectionFactory(env MongoEnv) IMongoConnectionFactory {
	return MongoConnectionFactory{
		connectionString: env.MongoConnectionString,
	}
}

package dbconnections

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type CacheDBTestingConnection struct {
	testDBName string
	client     *mongo.Client
}

var _ CacheDBConnection = (*CacheDBTestingConnection)(nil)

func NewCacheDBTestingConnection(t *testing.T) *CacheDBTestingConnection {
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI("mongodb://test:test@IntegrationTests.Imcaxy.Mongo:27017"))
	if err != nil {
		panic("Cannot connect to mongodb: " + err.Error())
	}

	testDBName := generateTestDBName(client)
	conn := &CacheDBTestingConnection{testDBName, client}

	t.Cleanup(conn.Cleanup)
	return conn
}

func (c *CacheDBTestingConnection) Collection(name string) *mongo.Collection {
	return c.client.Database(c.testDBName).Collection(name)
}

func (c *CacheDBTestingConnection) Cleanup() {
	ctx := context.Background()
	err := c.client.Database(c.testDBName).Drop(ctx)
	if err != nil {
		panic("Cannot cleanup testing database '" + c.testDBName + "': " + err.Error())
	}
}

func generateTestDBName(client *mongo.Client) string {
	for i := 0; i < 10; i++ {
		id := uuid.New().String()
		if checkDatabaseExists(client, id) {
			continue
		}

		client.Database(id)
		return id
	}

	panic("Cannot generate unique test DB name")
}

func checkDatabaseExists(client *mongo.Client, databaseName string) bool {
	databases, err := client.ListDatabaseNames(context.Background(), bson.M{})
	if err != nil {
		panic("Cannot fetch database names list: " + err.Error())
	}

	for _, name := range databases {
		if name == databaseName {
			return true
		}
	}

	return false
}

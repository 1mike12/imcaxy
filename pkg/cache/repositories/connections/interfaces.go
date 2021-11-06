package dbconnections

import (
	"context"
	"io"

	"github.com/minio/minio-go/v7"
	"go.mongodb.org/mongo-driver/mongo"
)

type CacheDBConnection interface {
	Collection(collectionName string) *mongo.Collection
}

type MinioBlockStorageConnection interface {
	GetObject(ctx context.Context, objectName string) (*minio.Object, error)
	PutObject(ctx context.Context, objectName string, objectSize int64, mimeType string, reader io.Reader) error
	DeleteObject(ctx context.Context, objectName string) error
	ObjectExists(ctx context.Context, objectName string) (exists bool, err error)
}

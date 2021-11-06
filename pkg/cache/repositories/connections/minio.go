package dbconnections

import (
	"context"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinioBlockStorageProductionConnectionConfig struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	Location  string
	UseSSL    bool
}

type MinioBlockStorageProductionConnection struct {
	config MinioBlockStorageProductionConnectionConfig
	client *minio.Client
}

var _ MinioBlockStorageConnection = (*MinioBlockStorageProductionConnection)(nil)

func NewMinioBlockStorageProductionConnection(ctx context.Context, config MinioBlockStorageProductionConnectionConfig) (conn MinioBlockStorageProductionConnection, err error) {
	client, err := minio.New(config.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(config.AccessKey, config.SecretKey, ""),
		Secure: config.UseSSL,
	})

	if err != nil {
		return
	}

	makeBucketOptions := minio.MakeBucketOptions{Region: config.Location}
	if err = client.MakeBucket(ctx, config.Bucket, makeBucketOptions); err != nil {
		return
	}

	conn = MinioBlockStorageProductionConnection{
		config: config,
		client: client,
	}

	return
}

func (c *MinioBlockStorageProductionConnection) GetObject(ctx context.Context, objectName string) (*minio.Object, error) {
	return c.client.GetObject(ctx, c.config.Bucket, objectName, minio.GetObjectOptions{})
}

func (c *MinioBlockStorageProductionConnection) PutObject(
	ctx context.Context,
	objectName string,
	objectSize int64,
	mimeType string,
	reader io.Reader,
) error {
	_, err := c.client.PutObject(
		ctx,
		c.config.Bucket,
		objectName,
		reader,
		objectSize,
		minio.PutObjectOptions{ContentType: mimeType},
	)
	return err
}

func (c *MinioBlockStorageProductionConnection) DeleteObject(ctx context.Context, objectName string) error {
	return c.client.RemoveObject(ctx, c.config.Bucket, objectName, minio.RemoveObjectOptions{})
}

func (c *MinioBlockStorageProductionConnection) ObjectExists(ctx context.Context, objectName string) (exists bool, err error) {
	_, err = c.client.StatObject(ctx, c.config.Bucket, objectName, minio.StatObjectOptions{})
	if err != nil {
		if minio.ToErrorResponse(err).Code == "NoSuchKey" {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

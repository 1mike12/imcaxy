package dbconnections

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinioBlockStorageTestingConnection struct {
	MinioBlockStorageProductionConnection
}

func NewMinioBlockStorageTestingConnection(t *testing.T) *MinioBlockStorageTestingConnection {
	conn, err := NewMinioBlockStorageProductionConnection(context.Background(), MinioBlockStorageProductionConnectionConfig{
		Endpoint:  testingServerEndpoint,
		AccessKey: testingServerAccessKey,
		SecretKey: testingServerSecretKey,
		Bucket:    getRandomTestingBucketName(),
		Location:  "us-east-1",
		UseSSL:    false,
	})
	if err != nil {
		panic("Error when connecting to minio block storage: " + err.Error())
	}

	testingConn := &MinioBlockStorageTestingConnection{conn}
	t.Cleanup(testingConn.dropTestBucket)

	return testingConn
}

func (c *MinioBlockStorageTestingConnection) dropTestBucket() {
	// if err := c.client.RemoveBucketWithOptions(context.Background(), c.config.Bucket, minio.BucketOptions{
	// 	ForceDelete: true,
	// }); err != nil {
	// 	panic("Error when dropping test bucket: " + err.Error())
	// }
}

func getRandomTestingBucketName() string {
	minioClient, err := minio.New(testingServerEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(testingServerAccessKey, testingServerSecretKey, ""),
		Secure: false,
	})
	if err != nil {
		panic("Error when generating random name of test bucket: " + err.Error())
	}

	for i := 0; i < 10; i++ {
		id := uuid.New().String()
		bucketName := id + "-testing-bucket"

		exists, err := minioClient.BucketExists(context.Background(), bucketName)
		if err != nil {
			panic("Error when checking if bucket name exists: " + err.Error())
		}
		if !exists {
			return bucketName
		}
	}

	panic("Could not generate random bucket name")
}

const testingServerEndpoint = "IntegrationTests.Imcaxy.Minio:9000"
const testingServerAccessKey = "minio"
const testingServerSecretKey = "minio123"

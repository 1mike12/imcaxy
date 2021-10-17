package main

import (
	"context"
	"log"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func connectToMinio() (*minio.Client, error) {
	endpoint := "dev-imcaxy-minio:9000"
	accessKeyID := "minio"
	secretAccessKey := "minio123"

	// Initialize minio client object.
	return minio.New(endpoint, &minio.Options{
		Creds: credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
	})
}

func makeBucketIfNotExists(bucketName string) error {
	ctx := context.Background()
	client, err := connectToMinio()
	if err != nil {
		return err
	}

	exists, errBucketExists := client.BucketExists(ctx, bucketName)
	if errBucketExists != nil {
		return errBucketExists
	}

	if !exists {
		err = client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{Region: "us-east-1"})

		if err != nil {
			return err
		}
	}

	return nil
}

func uploadExampleFileToMinio() {
	bucketName := "test-bucket"
	filePath := "/data/image.jpg"
	uploadName := "data/image.jpg"

	makeBucketIfNotExists(bucketName)

	ctx := context.Background()
	client, err := connectToMinio()
	if err != nil {
		log.Fatalln(err)
	}

	_, err = client.FPutObject(ctx, bucketName, uploadName, filePath, minio.PutObjectOptions{ContentType: "image/jpg"})
	if err != nil {
		log.Fatalln(err)
	}

	log.Println("Successfully uploaded", filePath, "as", uploadName, "into bucket", bucketName)
}

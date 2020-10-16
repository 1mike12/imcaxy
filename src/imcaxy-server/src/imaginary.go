package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/minio/minio-go/v7"
)

func cropExampleImageAndUploadToMinio() {
	bucketName := "test-bucket"
	inputFile := "/data/image.jpg"
	outputFile := "data/output.jpg"

	file, err := os.Open(inputFile)
	if err != nil {
		log.Panicln(err)
	}
	defer file.Close()

	response, err := http.Post("http://dev-imcaxy-imaginary:8080/crop?width=1000", "image/jpg, multipart/form-data", file)
	if err != nil {
		log.Panicln(err)
	}

	makeBucketIfNotExists(bucketName)

	ctx := context.Background()
	client, err := connectToMinio()
	if err != nil {
		log.Fatalln(err)
	}

	_, err = client.PutObject(ctx, bucketName, outputFile, response.Body, response.ContentLength, minio.PutObjectOptions{ContentType: "image/jpg"})
	if err != nil {
		log.Fatalln(err)
	}
}

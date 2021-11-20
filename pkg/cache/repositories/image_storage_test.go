package cacherepositories

import (
	"bytes"
	"context"
	"io/ioutil"
	"os"
	"testing"

	dbconnections "github.com/thebartekbanach/imcaxy/pkg/cache/repositories/connections"
	mock_hub "github.com/thebartekbanach/imcaxy/pkg/hub/mocks"
)

func loadTestFile(t *testing.T) []byte {
	file, err := os.Open("./../../../test/data/image.jpg")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	data, err := ioutil.ReadAll(file)
	if err != nil {
		t.Fatal(err)
	}

	return data
}

func TestCachedImagesStorageIntegration_ShouldCorrectlyUploadImage(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping cachedImagesRepository integration tests")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	testData := loadTestFile(t)
	mockDataStreamOutput := mock_hub.NewMockTestingDataStreamOutputUsingSingleChunkOfData(t, testData, nil, nil)
	mockDataStreamInput := mock_hub.NewMockTestingDataStreamInput(t, nil, nil, nil)

	conn := dbconnections.NewMinioBlockStorageTestingConnection(t)
	storage := NewCachedImagesStorage(conn)

	err := storage.Save(ctx, "test-signature", "imaginary", "image/jpeg", int64(len(testData)), mockDataStreamOutput)
	if err != nil {
		t.Fatalf("Error ocurred while saving image to block storage: %s", err)
	}

	// make sure stream output was closed
	mockDataStreamOutput.Wait()

	err = storage.Get(ctx, "test-signature", "imaginary", &mockDataStreamInput)
	if err != nil {
		t.Fatalf("Error ocurred while getting image from block storage: %s", err)
	}

	mockDataStreamInput.Wait()
	if mockDataStreamInput.ForwardedError != nil {
		t.Fatalf("Error was forwarded when fetching file from storage: %s", mockDataStreamInput.ForwardedError)
	}

	response := mockDataStreamInput.GetWholeResponse()
	if !bytes.Equal(response, testData) {
		t.Fatalf("Readed data is not equal to original data")
	}
}

func TestCachedImagesStorageIntegration_ShouldReturnSaveErrorWhenTryingToUploadImageThatAlreadyExists(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping cachedImagesRepository integration tests")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	testData := loadTestFile(t)
	mockDataStreamFirstOutput := mock_hub.NewMockTestingDataStreamOutputUsingSingleChunkOfData(t, testData, nil, nil)
	mockDataStreamSecondOutput := mock_hub.NewMockTestingDataStreamOutputUsingSingleChunkOfData(t, testData, nil, nil)

	conn := dbconnections.NewMinioBlockStorageTestingConnection(t)
	storage := NewCachedImagesStorage(conn)

	err := storage.Save(ctx, "test-signature", "imaginary", "image/jpeg", int64(len(testData)), mockDataStreamFirstOutput)
	if err != nil {
		t.Fatalf("Error ocurred while saving image to block storage: %s", err)
	}

	err = storage.Save(ctx, "test-signature", "imaginary", "image/jpeg", int64(len(testData)), mockDataStreamSecondOutput)
	if err != ErrImageAlreadyExists {
		t.Fatalf("Error was not returned when trying to save image that already exists, got: %v", err)
	}
}

func TestCachedImagesStorageIntegration_ShouldReturnGetErrorIfImageDoesNotExist(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping cachedImagesRepository integration tests")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mockDataStreamInput := mock_hub.NewMockTestingDataStreamInput(t, nil, nil, nil)

	conn := dbconnections.NewMinioBlockStorageTestingConnection(t)
	storage := NewCachedImagesStorage(conn)

	err := storage.Get(ctx, "unknown-signature", "imaginary", &mockDataStreamInput)
	if err != ErrImageNotFound {
		t.Fatalf("Error was not returned when trying to get image that does not exist, got %v", err)
	}
}

func TestCachedImagesStorageIntegration_ShouldCorrectlyDeleteImage(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping cachedImagesRepository integration tests")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	testData := loadTestFile(t)
	mockDataStreamOutput := mock_hub.NewMockTestingDataStreamOutputUsingSingleChunkOfData(t, testData, nil, nil)
	mockDataStreamInput := mock_hub.NewMockTestingDataStreamInput(t, nil, nil, nil)
	mockDataStreamResultInput := mock_hub.NewMockTestingDataStreamInput(t, nil, nil, nil)

	conn := dbconnections.NewMinioBlockStorageTestingConnection(t)
	storage := NewCachedImagesStorage(conn)

	err := storage.Save(ctx, "test-signature", "imaginary", "image/jpeg", int64(len(testData)), mockDataStreamOutput)
	if err != nil {
		t.Fatalf("Error ocurred while saving image to block storage: %s", err)
	}

	err = storage.Get(ctx, "test-signature", "imaginary", &mockDataStreamInput)
	if err != nil {
		t.Fatalf("Error ocurred while getting image from block storage: %s", err)
	}

	mockDataStreamInput.Wait()
	if mockDataStreamInput.ForwardedError != nil {
		t.Fatalf("Error was forwarded when fetching file from storage: %s", mockDataStreamInput.ForwardedError)
	}
	if !bytes.Equal(mockDataStreamInput.GetWholeResponse(), testData) {
		t.Fatalf("Readed data is not equal to original data")
	}

	err = storage.Delete(ctx, "test-signature", "imaginary")
	if err != nil {
		t.Fatalf("Error ocurred while deleting image from block storage: %s", err)
	}

	err = storage.Get(ctx, "test-signature", "imaginary", &mockDataStreamResultInput)
	if err == nil {
		t.Fatalf("Image was not deleted")
	}
}

func TestCachedImagesStorageIntegration_ShouldReturnDeleteErrorIfImageDoesNotExist(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping cachedImagesRepository integration tests")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conn := dbconnections.NewMinioBlockStorageTestingConnection(t)
	storage := NewCachedImagesStorage(conn)

	err := storage.Delete(ctx, "unknown-signature", "imaginary")
	if err != ErrImageNotFound {
		t.Fatalf("Error was not returned when trying to delete image that does not exist, got %v", err)
	}
}

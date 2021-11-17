package cache_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/thebartekbanach/imcaxy/pkg/cache"
	cacherepositories "github.com/thebartekbanach/imcaxy/pkg/cache/repositories"
	dbconnections "github.com/thebartekbanach/imcaxy/pkg/cache/repositories/connections"
	mock_cacherepositories "github.com/thebartekbanach/imcaxy/pkg/cache/repositories/mocks"
	"github.com/thebartekbanach/imcaxy/pkg/hub"
	mock_hub "github.com/thebartekbanach/imcaxy/pkg/hub/mocks"
)

func TestCacheService_GetCorrectlyGetsInformationFromImageStorage(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockImagesRepo := mock_cacherepositories.NewMockCachedImagesRepository(mockCtrl)
	mockImagesStorage := mock_cacherepositories.NewMockCachedImagesStorage()
	mockStreamInput := mock_hub.NewMockTestingDataStreamInput(t, nil, nil, nil)
	testData := []byte("test data")

	mockImagesStorage.InstantSave("test-signature", "imaginary", testData)

	cacheService := cache.NewCacheService(mockImagesRepo, mockImagesStorage)
	cacheService.Get(context.Background(), "test-signature", "imaginary", &mockStreamInput)

	mockStreamInput.Wait()

	if !bytes.Equal(mockStreamInput.GetWholeResponse(), testData) {
		t.Errorf("Expected %v, got %v", testData, mockStreamInput.GetWholeResponse())
	}
}

func TestCacheService_GetShouldReturnErrorIfEntryNotFound(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockImagesRepo := mock_cacherepositories.NewMockCachedImagesRepository(mockCtrl)
	mockImagesStorage := mock_cacherepositories.NewMockCachedImagesStorage()
	mockStreamInput := mock_hub.NewMockTestingDataStreamInput(t, nil, nil, nil)

	cacheService := cache.NewCacheService(mockImagesRepo, mockImagesStorage)
	err := cacheService.Get(context.Background(), "test-signature", "imaginary", &mockStreamInput)

	if err != cache.ErrEntryNotFound {
		t.Errorf("Expected ErrEntryNotFound error, got: %v", err)
	}
}

func TestCacheService_GetDoesNotCloseStreamInputOnErrImageNotFoundError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockImagesRepo := mock_cacherepositories.NewMockCachedImagesRepository(mockCtrl)
	mockImagesStorage := mock_cacherepositories.NewMockCachedImagesStorage()
	mockStreamInput := mock_hub.NewMockDataStreamInput(mockCtrl)

	mockStreamInput.EXPECT().Close(gomock.Any()).Times(0)

	cacheService := cache.NewCacheService(mockImagesRepo, mockImagesStorage)
	// image with signature "unknown-signature" processed by "imaginary" processor
	// is not defined in cache (so cache mock returns ErrImageNotFound)
	cacheService.Get(context.Background(), "unknown-signature", "imaginary", mockStreamInput)
}

func TestCacheService_GetClosesStreamDoesNotCloseInputOnAnyImagesStorageError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockImagesRepo := mock_cacherepositories.NewMockCachedImagesRepository(mockCtrl)
	mockImagesStorage := mock_cacherepositories.NewMockCachedImagesStorage()
	mockStreamInput := mock_hub.NewMockDataStreamInput(mockCtrl)

	testError := errors.New("some error")
	mockImagesStorage.ReturnError(testError)
	mockStreamInput.EXPECT().Close(gomock.Any()).Times(0)

	cacheService := cache.NewCacheService(mockImagesRepo, mockImagesStorage)
	cacheService.Get(context.Background(), "unknown-signature", "imaginary", mockStreamInput)
}

func TestCacheService_SaveShouldCorrectlySaveImage(t *testing.T) {
	testData := [][]byte{{0x1, 0x2, 0x3}}
	mockCtrl := gomock.NewController(t)
	mockImagesRepo := mock_cacherepositories.NewMockCachedImagesRepository(mockCtrl)
	mockImagesStorage := mock_cacherepositories.NewMockCachedImagesStorage()
	mockStreamOutput := mock_hub.NewMockTestingDataStreamOutput(t, testData, nil, nil)
	mockStreamInput := mock_hub.NewMockTestingDataStreamInput(t, testData, nil, nil)

	cachedImageInfo := cacherepositories.CachedImageModel{
		RawRequest:        "/crop?width=500&height=500&url=http://google.com/image.jpg",
		RequestSignature:  "|/crop|height=500|url=http://google.com/image.jpg|width=500|",
		ProcessorType:     "imaginary",
		MimeType:          "image/jpeg",
		ProcessorEndpoint: "/crop",
		SourceImageURL:    "http://google.com/image.jpg",
		ProcessingParams: map[string][]string{
			"width":  {"500"},
			"height": {"500"},
			"url":    {"http://google.com/image.jpg"},
		},
	}
	mockImagesRepo.EXPECT().CreateCachedImageInfo(gomock.Any(), cachedImageInfo).Return(nil)

	cacheService := cache.NewCacheService(mockImagesRepo, mockImagesStorage)
	err := cacheService.Save(context.Background(), cachedImageInfo, mockStreamOutput)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	cacheService.Get(context.Background(), cachedImageInfo.RequestSignature, "imaginary", &mockStreamInput)
	mockStreamInput.Wait()
}

func TestCacheService_SaveShouldReturnErrorIfEntryAlreadyExists(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockImagesRepo := mock_cacherepositories.NewMockCachedImagesRepository(mockCtrl)
	mockImagesStorage := mock_cacherepositories.NewMockCachedImagesStorage()
	mockStreamOutput := mock_hub.NewMockTestingDataStreamOutput(t, nil, nil, nil)

	cachedImageInfo := cacherepositories.CachedImageModel{
		RawRequest:        "/crop?width=500&height=500&url=http://google.com/image.jpg",
		RequestSignature:  "|/crop|height=500|url=http://google.com/image.jpg|width=500|",
		ProcessorType:     "imaginary",
		MimeType:          "image/jpeg",
		ProcessorEndpoint: "/crop",
		SourceImageURL:    "http://google.com/image.jpg",
		ProcessingParams: map[string][]string{
			"width":  {"500"},
			"height": {"500"},
			"url":    {"http://google.com/image.jpg"},
		},
	}
	mockImagesRepo.EXPECT().CreateCachedImageInfo(gomock.Any(), cachedImageInfo).Return(cacherepositories.ErrCachedImageAlreadyExists)

	cacheService := cache.NewCacheService(mockImagesRepo, mockImagesStorage)
	err := cacheService.Save(context.Background(), cachedImageInfo, mockStreamOutput)

	if err != cache.ErrEntryAlreadyExists {
		t.Errorf("Expected ErrEntryAlreadyExists error, got: %v", err)
	}
}

func TestCacheService_SaveShouldReturnErrorReturnedByImagesRepository(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockImagesRepo := mock_cacherepositories.NewMockCachedImagesRepository(mockCtrl)
	mockImagesStorage := mock_cacherepositories.NewMockCachedImagesStorage()
	mockStreamOutput := mock_hub.NewMockTestingDataStreamOutput(t, nil, nil, nil)

	cachedImageInfo := cacherepositories.CachedImageModel{
		RawRequest:        "/crop?width=500&height=500&url=http://google.com/image.jpg",
		RequestSignature:  "|/crop|height=500|url=http://google.com/image.jpg|width=500|",
		ProcessorType:     "imaginary",
		MimeType:          "image/jpeg",
		ProcessorEndpoint: "/crop",
		SourceImageURL:    "http://google.com/image.jpg",
		ProcessingParams: map[string][]string{
			"width":  {"500"},
			"height": {"500"},
			"url":    {"http://google.com/image.jpg"},
		},
	}
	createError := errors.New("network error")
	mockImagesRepo.EXPECT().CreateCachedImageInfo(gomock.Any(), cachedImageInfo).Return(createError)

	cacheService := cache.NewCacheService(mockImagesRepo, mockImagesStorage)
	err := cacheService.Save(context.Background(), cachedImageInfo, mockStreamOutput)

	if err != createError {
		t.Errorf("Expected error returned by images repository, got: %v", err)
	}
}

func TestCacheService_SaveShouldReturnErrorReturnedByStreamOutput(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockImagesRepo := mock_cacherepositories.NewMockCachedImagesRepository(mockCtrl)
	mockImagesStorage := mock_cacherepositories.NewMockCachedImagesStorage()
	streamReadError := errors.New("network error")
	mockStreamOutput := mock_hub.NewMockTestingDataStreamOutput(t, [][]byte{{0x1, 0x2, 0x3}}, streamReadError, nil)

	cachedImageInfo := cacherepositories.CachedImageModel{
		RawRequest:        "/crop?width=500&height=500&url=http://google.com/image.jpg",
		RequestSignature:  "|/crop|height=500|url=http://google.com/image.jpg|width=500|",
		ProcessorType:     "imaginary",
		MimeType:          "image/jpeg",
		ProcessorEndpoint: "/crop",
		SourceImageURL:    "http://google.com/image.jpg",
		ProcessingParams: map[string][]string{
			"width":  {"500"},
			"height": {"500"},
			"url":    {"http://google.com/image.jpg"},
		},
	}
	mockImagesRepo.EXPECT().CreateCachedImageInfo(gomock.Any(), cachedImageInfo).Return(nil)
	mockImagesRepo.EXPECT().DeleteCachedImageInfo(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

	cacheService := cache.NewCacheService(mockImagesRepo, mockImagesStorage)
	err := cacheService.Save(context.Background(), cachedImageInfo, mockStreamOutput)

	if err != streamReadError {
		t.Errorf("Expected error returned by stream output, got: %v", err)
	}
}

func TestCacheService_SaveShouldRollbackChangesFromImagesRepositoryOnStorageSaveError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockImagesRepo := mock_cacherepositories.NewMockCachedImagesRepository(mockCtrl)
	mockImagesStorage := mock_cacherepositories.NewMockCachedImagesStorage()
	mockStreamOutput := mock_hub.NewMockTestingDataStreamOutput(t, nil, errors.New("network error"), nil)

	cachedImageInfo := cacherepositories.CachedImageModel{
		RawRequest:        "/crop?width=500&height=500&url=http://google.com/image.jpg",
		RequestSignature:  "|/crop|height=500|url=http://google.com/image.jpg|width=500|",
		ProcessorType:     "imaginary",
		MimeType:          "image/jpeg",
		ProcessorEndpoint: "/crop",
		SourceImageURL:    "http://google.com/image.jpg",
		ProcessingParams: map[string][]string{
			"width":  {"500"},
			"height": {"500"},
			"url":    {"http://google.com/image.jpg"},
		},
	}
	mockImagesRepo.EXPECT().CreateCachedImageInfo(gomock.Any(), cachedImageInfo).Return(nil)
	mockImagesRepo.EXPECT().DeleteCachedImageInfo(gomock.Any(), cachedImageInfo.RequestSignature, cachedImageInfo.ProcessorType).Return(nil)

	cacheService := cache.NewCacheService(mockImagesRepo, mockImagesStorage)
	cacheService.Save(context.Background(), cachedImageInfo, mockStreamOutput)
}

func TestCacheService_SaveShouldNotSaveImageInStorageOnImagesRepositoryInfoSaveError(t *testing.T) {
	testData := [][]byte{{0x1, 0x2, 0x3}}
	mockCtrl := gomock.NewController(t)
	mockImagesRepo := mock_cacherepositories.NewMockCachedImagesRepository(mockCtrl)
	mockImagesStorage := mock_cacherepositories.NewMockCachedImagesStorage()
	mockStreamOutput := mock_hub.NewMockTestingDataStreamOutput(t, testData, nil, nil)

	cachedImageInfo := cacherepositories.CachedImageModel{
		RawRequest:        "/crop?width=500&height=500&url=http://google.com/image.jpg",
		RequestSignature:  "|/crop|height=500|url=http://google.com/image.jpg|width=500|",
		ProcessorType:     "imaginary",
		MimeType:          "image/jpeg",
		ProcessorEndpoint: "/crop",
		SourceImageURL:    "http://google.com/image.jpg",
		ProcessingParams: map[string][]string{
			"width":  {"500"},
			"height": {"500"},
			"url":    {"http://google.com/image.jpg"},
		},
	}
	mockImagesRepo.EXPECT().CreateCachedImageInfo(gomock.Any(), cachedImageInfo).Return(errors.New("some error"))

	cacheService := cache.NewCacheService(mockImagesRepo, mockImagesStorage)
	cacheService.Save(context.Background(), cachedImageInfo, mockStreamOutput)

	if mockImagesStorage.Exists(cachedImageInfo.RequestSignature, cachedImageInfo.ProcessorType) {
		t.Errorf("Expected image to not be saved in storage, but it was")
	}
}

func TestCacheService_SaveClosesStreamOutputOnErrCachedImageAlreadyExistsImagesRepoError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockImagesRepo := mock_cacherepositories.NewMockCachedImagesRepository(mockCtrl)
	mockImagesStorage := mock_cacherepositories.NewMockCachedImagesStorage()
	mockStreamOutput := mock_hub.NewMockDataStreamOutput(mockCtrl)

	imageInfo := cacherepositories.CachedImageModel{
		RawRequest:        "/crop?width=500&height=500&url=http://google.com/image.jpg",
		RequestSignature:  "|/crop|height=500|url=http://google.com/image.jpg|width=500|",
		ProcessorType:     "imaginary",
		MimeType:          "image/jpeg",
		ProcessorEndpoint: "/crop",
		SourceImageURL:    "http://google.com/image.jpg",
		ProcessingParams: map[string][]string{
			"width":  {"500"},
			"height": {"500"},
			"url":    {"http://google.com/image.jpg"},
		},
	}
	mockImagesRepo.EXPECT().CreateCachedImageInfo(gomock.Any(), imageInfo).Return(cacherepositories.ErrCachedImageAlreadyExists)
	mockStreamOutput.EXPECT().Close().Return(nil)

	cacheService := cache.NewCacheService(mockImagesRepo, mockImagesStorage)
	cacheService.Save(context.Background(), imageInfo, mockStreamOutput)
}

func TestCacheService_SaveClosesStreamOutputOnAnyImagesRepoError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockImagesRepo := mock_cacherepositories.NewMockCachedImagesRepository(mockCtrl)
	mockImagesStorage := mock_cacherepositories.NewMockCachedImagesStorage()
	mockStreamOutput := mock_hub.NewMockDataStreamOutput(mockCtrl)

	imageInfo := cacherepositories.CachedImageModel{
		RawRequest:        "/crop?width=500&height=500&url=http://google.com/image.jpg",
		RequestSignature:  "|/crop|height=500|url=http://google.com/image.jpg|width=500|",
		ProcessorType:     "imaginary",
		MimeType:          "image/jpeg",
		ProcessorEndpoint: "/crop",
		SourceImageURL:    "http://google.com/image.jpg",
		ProcessingParams: map[string][]string{
			"width":  {"500"},
			"height": {"500"},
			"url":    {"http://google.com/image.jpg"},
		},
	}
	testError := errors.New("some error")
	mockImagesRepo.EXPECT().CreateCachedImageInfo(gomock.Any(), imageInfo).Return(testError)
	mockStreamOutput.EXPECT().Close().Return(nil)

	cacheService := cache.NewCacheService(mockImagesRepo, mockImagesStorage)
	cacheService.Save(context.Background(), imageInfo, mockStreamOutput)
}

func TestCacheService_SaveClosesStreamOutputOnAnyImagesStorageError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockImagesRepo := mock_cacherepositories.NewMockCachedImagesRepository(mockCtrl)
	mockImagesStorage := mock_cacherepositories.NewMockCachedImagesStorage()
	mockStreamOutput := mock_hub.NewMockDataStreamOutput(mockCtrl)

	imageInfo := cacherepositories.CachedImageModel{
		RawRequest:        "/crop?width=500&height=500&url=http://google.com/image.jpg",
		RequestSignature:  "|/crop|height=500|url=http://google.com/image.jpg|width=500|",
		ProcessorType:     "imaginary",
		MimeType:          "image/jpeg",
		ProcessorEndpoint: "/crop",
		SourceImageURL:    "http://google.com/image.jpg",
		ProcessingParams: map[string][]string{
			"width":  {"500"},
			"height": {"500"},
			"url":    {"http://google.com/image.jpg"},
		},
	}
	mockImagesRepo.EXPECT().CreateCachedImageInfo(gomock.Any(), imageInfo).Return(nil)
	mockImagesRepo.EXPECT().DeleteCachedImageInfo(gomock.Any(), imageInfo.RequestSignature, imageInfo.ProcessorType).Return(nil)
	mockStreamOutput.EXPECT().Close().Return(nil)
	testError := errors.New("some error")
	mockImagesStorage.ReturnError(testError)

	cacheService := cache.NewCacheService(mockImagesRepo, mockImagesStorage)
	cacheService.Save(context.Background(), imageInfo, mockStreamOutput)
}

func TestCacheService_InvalidateAllEntriesForURLShouldDeleteAllCachedImagesOfGivenSource(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockImagesRepo := mock_cacherepositories.NewMockCachedImagesRepository(mockCtrl)
	mockImagesStorage := mock_cacherepositories.NewMockCachedImagesStorage()

	cachedImages := []cacherepositories.CachedImageModel{
		{
			RawRequest:        "/crop?width=500&height=500&url=http://google.com/image.jpg",
			RequestSignature:  "|/crop|height=500|url=http://google.com/image.jpg|width=500|",
			ProcessorType:     "imaginary",
			MimeType:          "image/jpeg",
			ProcessorEndpoint: "/crop",
			SourceImageURL:    "http://google.com/image.jpg",
			ProcessingParams: map[string][]string{
				"width":  {"500"},
				"height": {"500"},
				"url":    {"http://google.com/image.jpg"},
			},
		},
		{
			RawRequest:        "/crop?width=400&height=400&url=http://google.com/image.jpg",
			RequestSignature:  "|/crop|height=400|url=http://google.com/image.jpg|width=400|",
			ProcessorType:     "imaginary",
			MimeType:          "image/jpeg",
			ProcessorEndpoint: "/crop",
			SourceImageURL:    "http://google.com/image.jpg",
			ProcessingParams: map[string][]string{
				"width":  {"400"},
				"height": {"400"},
				"url":    {"http://google.com/image.jpg"},
			},
		},
	}
	mockImagesRepo.EXPECT().GetCachedImageInfosOfSource(gomock.Any(), "http://google.com/image.jpg").Return(cachedImages, nil)
	for _, image := range cachedImages {
		mockImagesStorage.InstantSave(image.RequestSignature, image.ProcessorType, []byte{})
		mockImagesRepo.EXPECT().DeleteCachedImageInfo(gomock.Any(), image.RequestSignature, image.ProcessorType).Return(nil)
	}

	cacheService := cache.NewCacheService(mockImagesRepo, mockImagesStorage)
	removedEntries, _ := cacheService.InvalidateAllEntriesForURL(context.Background(), "http://google.com/image.jpg")

	if len(removedEntries) != 2 {
		t.Errorf("Expected 2 entries to be removed, but got %d", len(removedEntries))
	}

	for index, image := range cachedImages {
		if mockImagesStorage.Exists(image.RequestSignature, image.ProcessorType) {
			t.Errorf("Expected image to be deleted from storage, but it was not")
		}

		if removedEntries[index].RequestSignature != image.RequestSignature {
			t.Errorf("Expected to return invalidated cached image in correct order, but it was not")
		}
	}
}

func TestCacheService_InvalidateAllEntriesForURLShouldStopDeletingAndReturnErrorIfRepositoryImageDeleteErrorOccurs(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockImagesRepo := mock_cacherepositories.NewMockCachedImagesRepository(mockCtrl)
	mockImagesStorage := mock_cacherepositories.NewMockCachedImagesStorage()

	cachedImages := []cacherepositories.CachedImageModel{
		{
			RawRequest:        "/crop?width=500&height=500&url=http://google.com/image.jpg",
			RequestSignature:  "|/crop|height=500|url=http://google.com/image.jpg|width=500|",
			ProcessorType:     "imaginary",
			MimeType:          "image/jpeg",
			ProcessorEndpoint: "/crop",
			SourceImageURL:    "http://google.com/image.jpg",
			ProcessingParams: map[string][]string{
				"width":  {"500"},
				"height": {"500"},
				"url":    {"http://google.com/image.jpg"},
			},
		},
		{
			RawRequest:        "/crop?width=400&height=400&url=http://google.com/image.jpg",
			RequestSignature:  "|/crop|height=400|url=http://google.com/image.jpg|width=400|",
			ProcessorType:     "imaginary",
			MimeType:          "image/jpeg",
			ProcessorEndpoint: "/crop",
			SourceImageURL:    "http://google.com/image.jpg",
			ProcessingParams: map[string][]string{
				"width":  {"400"},
				"height": {"400"},
				"url":    {"http://google.com/image.jpg"},
			},
		},
	}

	mockImagesRepo.EXPECT().GetCachedImageInfosOfSource(gomock.Any(), "http://google.com/image.jpg").Return(cachedImages, nil)
	mockImagesStorage.InstantSave(cachedImages[0].RequestSignature, cachedImages[0].ProcessorType, []byte{0x0})
	mockImagesRepo.EXPECT().DeleteCachedImageInfo(gomock.Any(), cachedImages[0].RequestSignature, cachedImages[0].ProcessorType).Return(errors.New("some error"))

	cacheService := cache.NewCacheService(mockImagesRepo, mockImagesStorage)
	cacheService.InvalidateAllEntriesForURL(context.Background(), "http://google.com/image.jpg")

	if !mockImagesStorage.Exists(cachedImages[0].RequestSignature, cachedImages[0].ProcessorType) {
		t.Errorf("Expected image to be not deleted from storage, but it was")
	}
}

func TestCacheService_InvalidateAllEntriesForURLShouldStopDeletingAndReturnErrorIfImageStorageDeleteErrorOccurs(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockImagesRepo := mock_cacherepositories.NewMockCachedImagesRepository(mockCtrl)
	mockImagesStorage := mock_cacherepositories.NewMockCachedImagesStorage()

	cachedImages := []cacherepositories.CachedImageModel{
		{
			RawRequest:        "/crop?width=500&height=500&url=http://google.com/image.jpg",
			RequestSignature:  "|/crop|height=500|url=http://google.com/image.jpg|width=500|",
			ProcessorType:     "imaginary",
			MimeType:          "image/jpeg",
			ProcessorEndpoint: "/crop",
			SourceImageURL:    "http://google.com/image.jpg",
			ProcessingParams: map[string][]string{
				"width":  {"500"},
				"height": {"500"},
				"url":    {"http://google.com/image.jpg"},
			},
		},
		{
			RawRequest:        "/crop?width=400&height=400&url=http://google.com/image.jpg",
			RequestSignature:  "|/crop|height=400|url=http://google.com/image.jpg|width=400|",
			ProcessorType:     "imaginary",
			MimeType:          "image/jpeg",
			ProcessorEndpoint: "/crop",
			SourceImageURL:    "http://google.com/image.jpg",
			ProcessingParams: map[string][]string{
				"width":  {"400"},
				"height": {"400"},
				"url":    {"http://google.com/image.jpg"},
			},
		},
	}

	mockImagesRepo.EXPECT().GetCachedImageInfosOfSource(gomock.Any(), "http://google.com/image.jpg").Return(cachedImages, nil)
	mockImagesStorage.InstantSave(cachedImages[0].RequestSignature, cachedImages[0].ProcessorType, []byte{0x0})
	for _, image := range cachedImages {
		mockImagesRepo.EXPECT().DeleteCachedImageInfo(gomock.Any(), image.RequestSignature, image.ProcessorType).Return(nil)
	}

	// the cachedImages[1] is unknown to storage, so it will return not found error and because of that
	// it should not call mockImagesRepo.DeleteCachedImageInfo

	cacheService := cache.NewCacheService(mockImagesRepo, mockImagesStorage)
	cacheService.InvalidateAllEntriesForURL(context.Background(), "http://google.com/image.jpg")
}

func TestCacheService_InvalidateAllEntriesForURLShouldReturnAllRemovedImagesInfos(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockImagesRepo := mock_cacherepositories.NewMockCachedImagesRepository(mockCtrl)
	mockImagesStorage := mock_cacherepositories.NewMockCachedImagesStorage()

	cachedImages := []cacherepositories.CachedImageModel{
		{
			RawRequest:        "/crop?width=500&height=500&url=http://google.com/image.jpg",
			RequestSignature:  "|/crop|height=500|url=http://google.com/image.jpg|width=500|",
			ProcessorType:     "imaginary",
			MimeType:          "image/jpeg",
			ProcessorEndpoint: "/crop",
			SourceImageURL:    "http://google.com/image.jpg",
			ProcessingParams: map[string][]string{
				"width":  {"500"},
				"height": {"500"},
				"url":    {"http://google.com/image.jpg"},
			},
		},
		{
			RawRequest:        "/crop?width=400&height=400&url=http://google.com/image.jpg",
			RequestSignature:  "|/crop|height=400|url=http://google.com/image.jpg|width=400|",
			ProcessorType:     "imaginary",
			MimeType:          "image/jpeg",
			ProcessorEndpoint: "/crop",
			SourceImageURL:    "http://google.com/image.jpg",
			ProcessingParams: map[string][]string{
				"width":  {"400"},
				"height": {"400"},
				"url":    {"http://google.com/image.jpg"},
			},
		},
	}

	mockImagesRepo.EXPECT().GetCachedImageInfosOfSource(gomock.Any(), "http://google.com/image.jpg").Return(cachedImages, nil)
	for _, image := range cachedImages {
		mockImagesRepo.EXPECT().DeleteCachedImageInfo(gomock.Any(), image.RequestSignature, image.ProcessorType).Return(nil)
		mockImagesStorage.InstantSave(image.RequestSignature, image.ProcessorType, []byte{0x0})
	}

	cacheService := cache.NewCacheService(mockImagesRepo, mockImagesStorage)
	removedImages, _ := cacheService.InvalidateAllEntriesForURL(context.Background(), "http://google.com/image.jpg")

	if len(removedImages) != len(cachedImages) {
		t.Errorf("Expected to return all removed images, but it was not")
	}
}

func TestCacheService_InvalidateAllEntriesForURLShouldReturnOnlyRemovedImagesIfErrorOcurred(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockImagesRepo := mock_cacherepositories.NewMockCachedImagesRepository(mockCtrl)
	mockImagesStorage := mock_cacherepositories.NewMockCachedImagesStorage()

	cachedImages := []cacherepositories.CachedImageModel{
		{
			RawRequest:        "/crop?width=500&height=500&url=http://google.com/image.jpg",
			RequestSignature:  "|/crop|height=500|url=http://google.com/image.jpg|width=500|",
			ProcessorType:     "imaginary",
			MimeType:          "image/jpeg",
			ProcessorEndpoint: "/crop",
			SourceImageURL:    "http://google.com/image.jpg",
			ProcessingParams: map[string][]string{
				"width":  {"500"},
				"height": {"500"},
				"url":    {"http://google.com/image.jpg"},
			},
		},
		{
			RawRequest:        "/crop?width=400&height=400&url=http://google.com/image.jpg",
			RequestSignature:  "|/crop|height=400|url=http://google.com/image.jpg|width=400|",
			ProcessorType:     "imaginary",
			MimeType:          "image/jpeg",
			ProcessorEndpoint: "/crop",
			SourceImageURL:    "http://google.com/image.jpg",
			ProcessingParams: map[string][]string{
				"width":  {"400"},
				"height": {"400"},
				"url":    {"http://google.com/image.jpg"},
			},
		},
	}

	mockImagesRepo.EXPECT().GetCachedImageInfosOfSource(gomock.Any(), "http://google.com/image.jpg").Return(cachedImages, nil)
	mockImagesRepo.EXPECT().DeleteCachedImageInfo(gomock.Any(), cachedImages[0].RequestSignature, cachedImages[0].ProcessorType).Return(nil)
	mockImagesRepo.EXPECT().DeleteCachedImageInfo(gomock.Any(), cachedImages[1].RequestSignature, cachedImages[1].ProcessorType).Return(errors.New("some error"))
	for _, image := range cachedImages {
		mockImagesStorage.InstantSave(image.RequestSignature, image.ProcessorType, []byte{0x0})
	}

	cacheService := cache.NewCacheService(mockImagesRepo, mockImagesStorage)
	removedImages, _ := cacheService.InvalidateAllEntriesForURL(context.Background(), "http://google.com/image.jpg")

	if len(removedImages) != 1 {
		t.Errorf("Expected to return only removed images, but it returned %v images instead of 1", len(removedImages))
	}
}

func loadTestFile(t *testing.T) []byte {
	file, err := os.Open("./../../test/data/image.jpg")
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

func getTestDataReadStream(t *testing.T) (hub.DataStreamOutput, cacherepositories.CachedImageModel, []byte) {
	data := loadTestFile(t)
	output := mock_hub.NewMockTestingDataStreamOutputUsingSingleChunkOfData(t, data, nil, nil)
	imageInfo := cacherepositories.CachedImageModel{
		RawRequest:        "/crop?width=500&height=500&url=http://google.com/image.jpg",
		RequestSignature:  "|/crop|height=500|url=http://google.com/image.jpg|width=500|",
		ProcessorType:     "imaginary",
		MimeType:          "image/jpeg",
		ProcessorEndpoint: "/crop",
		SourceImageURL:    "http://google.com/image.jpg",
		ProcessingParams: map[string][]string{
			"width":  {"500"},
			"height": {"500"},
			"url":    {"http://google.com/image.jpg"},
		},
		ImageSize: int64(len(data)),
	}
	return output, imageInfo, data
}

func TestCacheServiceIntegration_SavesAndGetsImageCorrectly(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping cacheService integration tests")
	}

	mongoTestingConnection := dbconnections.NewCacheDBTestingConnection(t)
	minioTestingConnection := dbconnections.NewMinioBlockStorageTestingConnection(t)
	imagesCache := cacherepositories.NewCachedImagesRepository(mongoTestingConnection)
	imagesStorage := cacherepositories.NewCachedImagesStorage(minioTestingConnection)
	dataStreamOutput, imageInfo, testData := getTestDataReadStream(t)
	mockDataStreamInput := mock_hub.NewMockTestingDataStreamInput(t, nil, nil, nil)

	cacheService := cache.NewCacheService(imagesCache, imagesStorage)

	if err := cacheService.Save(context.Background(), imageInfo, dataStreamOutput); err != nil {
		t.Fatal(err)
	}

	if err := cacheService.Get(context.Background(), imageInfo.RequestSignature, imageInfo.ProcessorType, &mockDataStreamInput); err != nil {
		t.Fatal(err)
	}

	mockDataStreamInput.Wait()

	if !bytes.Equal(testData, mockDataStreamInput.GetWholeResponse()) {
		t.Errorf("Expected to get correct data from cache, but data loaded from cache is not equal to original data")

		if len(mockDataStreamInput.GetWholeResponse()) != len(testData) {
			t.Errorf("Expected to get %v bytes from cache, but got %v bytes instead", len(testData), len(mockDataStreamInput.GetWholeResponse()))
		}
	}
}

func TestCacheServiceIntegration_GetReturnsErrorWhenImageIsNotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping cacheService integration tests")
	}

	mongoTestingConnection := dbconnections.NewCacheDBTestingConnection(t)
	minioTestingConnection := dbconnections.NewMinioBlockStorageTestingConnection(t)
	imagesCache := cacherepositories.NewCachedImagesRepository(mongoTestingConnection)
	imagesStorage := cacherepositories.NewCachedImagesStorage(minioTestingConnection)
	mockDataStreamInput := mock_hub.NewMockTestingDataStreamInput(t, nil, nil, nil)

	cacheService := cache.NewCacheService(imagesCache, imagesStorage)

	if err := cacheService.Get(context.Background(), "unknown-signature", "imaginary", &mockDataStreamInput); err != cache.ErrEntryNotFound {
		t.Errorf("Expected to get ErrEntryNotFound error, but got: %v", err)
	}
}

func TestCacheServiceIntegration_SaveReturnsErrorWhenImageAlreadyExists(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping cacheService integration tests")
	}

	mongoTestingConnection := dbconnections.NewCacheDBTestingConnection(t)
	minioTestingConnection := dbconnections.NewMinioBlockStorageTestingConnection(t)
	imagesCache := cacherepositories.NewCachedImagesRepository(mongoTestingConnection)
	imagesStorage := cacherepositories.NewCachedImagesStorage(minioTestingConnection)
	dataStreamOutput, imageInfo, _ := getTestDataReadStream(t)

	cacheService := cache.NewCacheService(imagesCache, imagesStorage)
	cacheService.Save(context.Background(), imageInfo, dataStreamOutput)

	if err := cacheService.Save(context.Background(), imageInfo, dataStreamOutput); err != cache.ErrEntryAlreadyExists {
		t.Errorf("Expected to get ErrEntryAlreadyExists error, but got: %v", err)
	}
}

func TestCacheServiceIntegration_InvaildateAllEntriesForURLRemovesAllEntriesOfGivenURL(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping cacheService integration tests")
	}

	mongoTestingConnection := dbconnections.NewCacheDBTestingConnection(t)
	minioTestingConnection := dbconnections.NewMinioBlockStorageTestingConnection(t)
	imagesCache := cacherepositories.NewCachedImagesRepository(mongoTestingConnection)
	imagesStorage := cacherepositories.NewCachedImagesStorage(minioTestingConnection)

	cacheService := cache.NewCacheService(imagesCache, imagesStorage)

	signaturesExpectedToBeDeleted := make([]string, 3)
	for i := 0; i < 3; i++ {
		dataStreamOutput, imageInfo, _ := getTestDataReadStream(t)

		// We changed the RequestSignature, what means that every signature is unique and
		// differs from signature returned by getTestDataReadStream function.
		imageInfo.RequestSignature = fmt.Sprintf("%v-%d", imageInfo.RequestSignature, i+1)
		signaturesExpectedToBeDeleted[i] = imageInfo.RequestSignature

		// At this point, we have 3 entries in the cache for the same URL
		// that we will want to be deleted.
		cacheService.Save(context.Background(), imageInfo, dataStreamOutput)
	}

	dataStreamOutput, infoOfImageThatWillStay, _ := getTestDataReadStream(t)
	infoOfImageThatWillStay.SourceImageURL = "http://google.com/image-that-will-stay.jpg"
	cacheService.Save(context.Background(), infoOfImageThatWillStay, dataStreamOutput)

	invalidatedEntries, err := cacheService.InvalidateAllEntriesForURL(context.Background(), "http://google.com/image.jpg")
	if err != nil {
		t.Fatal(err)
	}

	if len(invalidatedEntries) != 3 {
		t.Errorf("Expected to get 3 invalidated entries, but got: %v", invalidatedEntries)
	}

	for _, invalidatedEntry := range invalidatedEntries {
		if !containsInvalidatedEntry(signaturesExpectedToBeDeleted, invalidatedEntry) {
			t.Errorf("Expected cache entry \"%s\" to be deleted", invalidatedEntry.RequestSignature)
		}
	}
}

func containsInvalidatedEntry(signaturesExpectedToBeDeleted []string, invalidatedEntry cacherepositories.CachedImageModel) bool {
	for _, signature := range signaturesExpectedToBeDeleted {
		if signature == invalidatedEntry.RequestSignature {
			return true
		}
	}
	return false
}

package cacherepositories

import (
	"context"
	"reflect"
	"testing"

	dbconnections "github.com/thebartekbanach/imcaxy/pkg/cache/repositories/connections"
)

func TestCachedImagesRepositoryIntegration_CreatesCachedImage(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping cachedImagesRepository integration tests")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	info := CachedImageModel{
		RawRequest:       "/crop?width=500&height=500&url=http://google.com/image.jpg",
		RequestSignature: "|/crop|http://google.com/image.jpg|height=500&width=500|",

		ProcessorType:     "imaginary",
		ProcessorEndpoint: "/crop",

		MimeType:       "image/jpeg",
		SourceImageURL: "http://google.com/image.jpg",
		ProcessingParams: map[string][]string{
			"width":  {"500"},
			"height": {"500"},
		},
	}

	conn := dbconnections.NewCacheDBTestingConnection(t)
	repo := NewCachedImagesRepository(conn)

	if err := repo.CreateCachedImageInfo(ctx, info); err != nil {
		t.Errorf("Error creating cached image info: %s", err)
	}

	infoFromDB, err := repo.GetCachedImageInfo(ctx, info.RequestSignature, info.ProcessorType)
	if err != nil {
		t.Errorf("Error getting cached image info: %s", err)
	}

	if !reflect.DeepEqual(infoFromDB, info) {
		t.Errorf("Cached image info from DB does not match the created one")
	}
}

func TestCachedImagesRepositoryIntegration_ReturnsErrorWhenCachedImageAlreadyExists(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping cachedImagesRepository integration tests")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	info := CachedImageModel{
		RawRequest:       "/crop?width=500&height=500&url=http://google.com/image.jpg",
		RequestSignature: "|/crop|http://google.com/image.jpg|height=500&width=500|",

		ProcessorType:     "imaginary",
		ProcessorEndpoint: "/crop",

		MimeType:       "image/jpeg",
		SourceImageURL: "http://google.com/image.jpg",
		ProcessingParams: map[string][]string{
			"width":  {"500"},
			"height": {"500"},
		},
	}

	conn := dbconnections.NewCacheDBTestingConnection(t)
	repo := NewCachedImagesRepository(conn)

	if err := repo.CreateCachedImageInfo(ctx, info); err != nil {
		t.Errorf("Error creating cached image info: %s", err)
	}

	if err := repo.CreateCachedImageInfo(ctx, info); err != ErrCachedImageAlreadyExists {
		t.Errorf("Expected error ErrCachedImageAlreadyExists when creating cached image info that already exists, got: %s", err)
	}
}

func TestCachedImagesRepositoryIntegration_ReturnsErrorWhenCachedImageDoesNotExist(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping cachedImagesRepository integration tests")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conn := dbconnections.NewCacheDBTestingConnection(t)
	repo := NewCachedImagesRepository(conn)

	if _, err := repo.GetCachedImageInfo(ctx, "some-random-signature", "imaginary"); err != ErrCachedImageNotFound {
		t.Errorf("Expected error ErrCachedImageNotFound when getting cached image info that does not exist, got: %s", err)
	}
}

func TestCachedImagesRepositoryIntegration_DeletesCachedImage(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping cachedImagesRepository integration tests")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	info := CachedImageModel{
		RawRequest:       "/crop?width=500&height=500&url=http://google.com/image.jpg",
		RequestSignature: "|/crop|http://google.com/image.jpg|height=500&width=500|",

		ProcessorType:     "imaginary",
		ProcessorEndpoint: "/crop",

		MimeType:       "image/jpeg",
		SourceImageURL: "http://google.com/image.jpg",
		ProcessingParams: map[string][]string{
			"width":  {"500"},
			"height": {"500"},
		},
	}

	conn := dbconnections.NewCacheDBTestingConnection(t)
	repo := NewCachedImagesRepository(conn)

	if err := repo.CreateCachedImageInfo(ctx, info); err != nil {
		t.Errorf("Error creating cached image info: %s", err)
	}

	if _, err := repo.GetCachedImageInfo(ctx, info.RequestSignature, info.ProcessorType); err != nil {
		t.Errorf("Cached image was not created correctly: %s", err)
	}

	if err := repo.DeleteCachedImageInfo(ctx, info.RequestSignature, info.ProcessorType); err != nil {
		t.Errorf("Error deleting cached image info: %s", err)
	}

	if _, err := repo.GetCachedImageInfo(ctx, info.RequestSignature, info.ProcessorType); err != ErrCachedImageNotFound {
		t.Errorf("Cached image was not deleted correctly: %s", err)
	}
}

func TestCachedImagesRepositoryIntegration_ReturnsAllCachedImageInfosOfGivenURL(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping cachedImagesRepository integration tests")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// those two image infos differ in width and height of requested crop
	info1 := CachedImageModel{
		RawRequest:       "/crop?width=400&height=400&url=http://google.com/image.jpg",
		RequestSignature: "|/crop|http://google.com/image.jpg|height=400&width=400|",

		ProcessorType:     "imaginary",
		ProcessorEndpoint: "/crop",

		MimeType:       "image/jpeg",
		SourceImageURL: "http://google.com/image.jpg",
		ProcessingParams: map[string][]string{
			"width":  {"400"},
			"height": {"400"},
		},
	}

	info2 := CachedImageModel{
		RawRequest:       "/crop?width=500&height=500&url=http://google.com/image.jpg",
		RequestSignature: "|/crop|http://google.com/image.jpg|height=500&width=500|",

		ProcessorType:     "imaginary",
		ProcessorEndpoint: "/crop",

		MimeType:       "image/jpeg",
		SourceImageURL: "http://google.com/image.jpg",
		ProcessingParams: map[string][]string{
			"width":  {"500"},
			"height": {"500"},
		},
	}

	conn := dbconnections.NewCacheDBTestingConnection(t)
	repo := NewCachedImagesRepository(conn)
	repo.CreateCachedImageInfo(ctx, info1)
	repo.CreateCachedImageInfo(ctx, info2)
	infos, err := repo.GetCachedImageInfosOfSource(ctx, "http://google.com/image.jpg")

	if err != nil {
		t.Errorf("Error getting cached image infos of source image: %s", err)
	}

	if len(infos) != 2 {
		t.Errorf("Expected 2 cached image infos of source image, got: %d", len(infos))
	}

	for _, info := range infos {
		if reflect.DeepEqual(info, info1) || reflect.DeepEqual(info, info2) {
			continue
		}

		t.Errorf("Expected cached image info to be one of the two, got: %v", info)
	}
}

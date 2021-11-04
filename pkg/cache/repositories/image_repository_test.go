package cacherepositories

import (
	"context"
	"reflect"
	"testing"

	dbconnections "github.com/thebartekbanach/imcaxy/pkg/cache/repositories/connections"
)

func TestCachedImagesRepository_CreatesCachedImage(t *testing.T) {
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

func TestCachedImagesRepository_ReturnsErrorWhenCachedImageAlreadyExists(t *testing.T) {
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

func TestCachedImagesRepository_ReturnsErrorWhenCachedImageDoesNotExist(t *testing.T) {
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

func TestCachedImagesRepository_DeletesCachedImage(t *testing.T) {
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

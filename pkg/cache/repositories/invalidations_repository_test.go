package cacherepositories

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"testing"
	"time"

	dbconnections "github.com/thebartekbanach/imcaxy/pkg/cache/repositories/connections"
)

func createInvalidationModel(projectName, commitHash string, creationTime time.Time, requestedInvalidations, invalidatedImages []string) InvalidationModel {
	invalidatedImagesResult := make([]CachedImageModel, len(invalidatedImages))
	for i, image := range invalidatedImages {
		size := (i + 1) * 100
		invalidatedImagesResult[i] = CachedImageModel{
			RawRequest:       fmt.Sprintf("/crop?url=%s&width=%v&height=%v", image, size, size),
			RequestSignature: fmt.Sprintf("|/crop|%s|width=%v&height=%v|", image, size, size),

			ProcessorType:     "imaginary",
			ProcessorEndpoint: "/crop",

			MimeType:       "image/jpeg",
			ImageSize:      int64(size * size),
			SourceImageURL: image,
			ProcessingParams: map[string][]string{
				"url":    {image},
				"width":  {strconv.Itoa(size)},
				"height": {strconv.Itoa(size)},
			},
		}
	}

	return InvalidationModel{
		ProjectName: projectName,
		CommitHash:  commitHash,

		InvalidationDate:       creationTime,
		RequestedInvalidations: requestedInvalidations,
		InvalidatedImages:      invalidatedImagesResult,
	}
}

func createSuccessfullInvalidationModel(projectName, commitHash string, creationTime time.Time, invalidatedImages []string) InvalidationModel {
	return createInvalidationModel(projectName, commitHash, creationTime, invalidatedImages, invalidatedImages)
}

func TestInvalidationsRepositoryIntegration_CreatesInvalidationCorrectly(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping invalidationsRepository integration tests")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	info := createSuccessfullInvalidationModel("project", "abcdef", time.Now(), []string{
		"http://google.com/image1.jpg",
		"http://google.com/image2.jpg",
	})

	conn := dbconnections.NewCacheDBTestingConnection(t)
	repo := NewInvalidationsRepository(conn)

	err := repo.CreateInvalidation(ctx, info)
	if err != nil {
		t.Errorf("Unexpected error when creating invalidation entry: %v", err)
	}

	invalidation, err := repo.GetLatestInvalidation(ctx, "project")
	if err != nil {
		t.Errorf("Unexpected error when getting latest invalidation: %v", err)
	}

	// we cant DeepEqual the whole objects, because InvalidationDate field differs a little bit
	if !reflect.DeepEqual(invalidation.RequestedInvalidations, info.RequestedInvalidations) {
		t.Errorf("Invalidation is not the same as the one created")
	}
}

func TestInvalidationsRepositoryIntegration_ReturnsLatestInvalidationInfo(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping invalidationsRepository integration tests")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	info1 := createSuccessfullInvalidationModel("project", "abcdef", time.Now(), []string{
		"http://google.com/image1.jpg",
		"http://google.com/image2.jpg",
	})

	info2 := createSuccessfullInvalidationModel("project", "ghijkl", time.Now().Add(time.Minute), []string{
		"http://google.com/image3.jpg",
		"http://google.com/image4.jpg",
	})

	conn := dbconnections.NewCacheDBTestingConnection(t)
	repo := NewInvalidationsRepository(conn)

	err := repo.CreateInvalidation(ctx, info1)
	if err != nil {
		t.Errorf("Unexpected error when creating first invalidation entry: %v", err)
	}

	err = repo.CreateInvalidation(ctx, info2)
	if err != nil {
		t.Errorf("Unexpected error when creating second invalidation entry: %v", err)
	}

	invalidation, err := repo.GetLatestInvalidation(ctx, "project")
	if err != nil {
		t.Errorf("Unexpected error when getting latest invalidation: %v", err)
	}

	// we cant DeepEqual the whole objects, because InvalidationDate field differs a little bit
	if !reflect.DeepEqual(invalidation.RequestedInvalidations, info2.RequestedInvalidations) {
		t.Errorf("Invalidation is not the same as the last one created: \n%v \n!= \n%v", info2.RequestedInvalidations, invalidation.RequestedInvalidations)
	}
}

func TestInvalidationsRepositoryIntegration_ReturnsErrCommitHashNotAllowedIfCommitHashIsEmpty(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping invalidationsRepository integration tests")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	info := createSuccessfullInvalidationModel("project", "", time.Now(), []string{
		"http://google.com/image1.jpg",
		"http://google.com/image2.jpg",
	})

	conn := dbconnections.NewCacheDBTestingConnection(t)
	repo := NewInvalidationsRepository(conn)

	err := repo.CreateInvalidation(ctx, info)
	if err != ErrCommitHashNotAllowed {
		t.Errorf("Expected to return ErrCommitHashNotAllowed, got: %v", err)
	}
}

func TestInvalidationsRepositoryIntegration_ReturnsErrProjectNameNotAllowedIfProjectNameIsEmpty(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping invalidationsRepository integration tests")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	info := createSuccessfullInvalidationModel("", "abcdef", time.Now(), []string{
		"http://google.com/image1.jpg",
		"http://google.com/image2.jpg",
	})

	conn := dbconnections.NewCacheDBTestingConnection(t)
	repo := NewInvalidationsRepository(conn)

	err := repo.CreateInvalidation(ctx, info)
	if err != ErrProjectNameNotAllowed {
		t.Errorf("Expected to return ErrProjectNameNotAllowed, got: %v", err)
	}
}

func TestInvalidationsRepositoryIntegration_ReturnsErrProjectNameNotAllowedIfTryingToGetLatestInvalidationWithEmptyProjectName(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping invalidationsRepository integration tests")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conn := dbconnections.NewCacheDBTestingConnection(t)
	repo := NewInvalidationsRepository(conn)

	_, err := repo.GetLatestInvalidation(ctx, "")
	if err != ErrProjectNameNotAllowed {
		t.Errorf("Expected to return ErrProjectNameNotAllowed, got: %v", err)
	}
}

func TestInvalidationsRepositoryIntegration_ReturnsErrProjectNotFoundIfThereIsNoInvalidationsAssociatedToProjectYet(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping invalidationsRepository integration tests")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	conn := dbconnections.NewCacheDBTestingConnection(t)
	repo := NewInvalidationsRepository(conn)

	// the repository is empty at this point because we did not add anything
	_, err := repo.GetLatestInvalidation(ctx, "project")
	if err != ErrProjectNotFound {
		t.Errorf("Expected ErrProjectNotFound to be returned, got: %v", err)
	}
}

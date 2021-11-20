package cache

import (
	"context"
	"time"

	cacherepositories "github.com/thebartekbanach/imcaxy/pkg/cache/repositories"
)

type InvalidationServiceImplementation struct {
	invalidationsRepository cacherepositories.InvalidationsRepository
	cacheService            CacheService
}

var _ InvalidationService = (*InvalidationServiceImplementation)(nil)

func NewInvalidationService(invalidationsRepository cacherepositories.InvalidationsRepository, cacheService CacheService) InvalidationService {
	return &InvalidationServiceImplementation{invalidationsRepository, cacheService}
}

func (s *InvalidationServiceImplementation) GetLastKnownInvalidation(ctx context.Context, projectName string) (cacherepositories.InvalidationModel, error) {
	if projectName == "" {
		return cacherepositories.InvalidationModel{}, cacherepositories.ErrProjectNameNotAllowed
	}

	return s.invalidationsRepository.GetLatestInvalidation(ctx, projectName)
}

func (s *InvalidationServiceImplementation) Invalidate(ctx context.Context, projectName, latestCommitHash string, urls []string) (cacherepositories.InvalidationModel, error) {
	if projectName == "" {
		return cacherepositories.InvalidationModel{}, cacherepositories.ErrProjectNameNotAllowed
	}

	if latestCommitHash == "" {
		return cacherepositories.InvalidationModel{}, cacherepositories.ErrCommitHashNotAllowed
	}

	invalidationInfo := cacherepositories.InvalidationModel{
		ProjectName:            projectName,
		CommitHash:             latestCommitHash,
		RequestedInvalidations: urls,
		DoneInvalidations:      []string{},
		InvalidatedImages:      []cacherepositories.CachedImageModel{},
	}

	var invalidationError error

	for _, url := range urls {
		invalidatedEntries, err := s.cacheService.InvalidateAllEntriesForURL(ctx, url)
		invalidationInfo.InvalidatedImages = append(invalidationInfo.InvalidatedImages, invalidatedEntries...)

		if err != nil {
			invalidationError = err
			errText := err.Error()
			invalidationInfo.InvalidationError = &errText
			break
		}

		invalidationInfo.DoneInvalidations = append(invalidationInfo.DoneInvalidations, url)
	}

	invalidationInfo.InvalidationDate = time.Now()
	if err := s.invalidationsRepository.CreateInvalidation(ctx, invalidationInfo); err != nil {
		return invalidationInfo, err
	}

	return invalidationInfo, invalidationError
}

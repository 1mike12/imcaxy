package cache

import (
	"context"

	cacherepositories "github.com/thebartekbanach/imcaxy/pkg/cache/repositories"
	"github.com/thebartekbanach/imcaxy/pkg/hub"
)

type CacheService interface {
	Get(ctx context.Context, requestSignature, processorType string, w hub.DataStreamInput) error
	Save(ctx context.Context, imageInfo cacherepositories.CachedImageModel, r hub.DataStreamOutput) error
	InvalidateAllEntriesForURL(ctx context.Context, sourceImageURL string) ([]cacherepositories.CachedImageModel, error)
}

type InvalidationService interface {
	GetLastKnownInvalidation(ctx context.Context, projectName string) (cacherepositories.InvalidationModel, error)
	Invalidate(ctx context.Context, projectName string, latestCommitHash string, urls []string) (cacherepositories.InvalidationModel, error)
}

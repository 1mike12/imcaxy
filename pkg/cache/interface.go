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

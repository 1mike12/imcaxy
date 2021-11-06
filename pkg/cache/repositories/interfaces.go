package cacherepositories

import (
	"context"

	"github.com/thebartekbanach/imcaxy/pkg/hub"
)

type CachedImagesRepository interface {
	CreateCachedImageInfo(ctx context.Context, info CachedImageModel) error
	DeleteCachedImageInfo(ctx context.Context, requestSignature, processorType string) error
	GetCachedImageInfo(ctx context.Context, requestSignature, processorType string) (CachedImageModel, error)
}

type CachedImagesStorage interface {
	Save(ctx context.Context, requestSignature, processorType, mimeType string, size int64, reader hub.DataStreamOutput) error
	Get(ctx context.Context, requestSignature, processorType string, writer hub.DataStreamInput) error
	Delete(ctx context.Context, requestSignature, processorType string) error
}

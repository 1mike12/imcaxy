package cacherepositories

import (
	"context"
	"time"

	"github.com/thebartekbanach/imcaxy/pkg/hub"
)

type CachedImageModel struct {
	RawRequest       string `json:"rawRequest" bson:"rawRequest"`
	RequestSignature string `json:"requestSignature" bson:"requestSignature"`

	ProcessorType     string `json:"processorType" bson:"processorType"`
	ProcessorEndpoint string `json:"processorEndpoint" bson:"processorEndpoint"`

	MimeType         string              `json:"mimeType" bson:"mimeType"`
	ImageSize        int64               `json:"imageSize" bson:"imageSize"`
	SourceImageURL   string              `json:"sourceImageURL" bson:"sourceImageURL"`
	ProcessingParams map[string][]string `json:"processingParams" bson:"processingParams"`
}

type CachedImagesRepository interface {
	CreateCachedImageInfo(ctx context.Context, info CachedImageModel) error
	DeleteCachedImageInfo(ctx context.Context, requestSignature, processorType string) error
	GetCachedImageInfo(ctx context.Context, requestSignature, processorType string) (CachedImageModel, error)
	GetCachedImageInfosOfSource(ctx context.Context, sourceImageURL string) ([]CachedImageModel, error)
}

type CachedImagesStorage interface {
	Save(ctx context.Context, requestSignature, processorType, mimeType string, size int64, reader hub.DataStreamOutput) error
	Get(ctx context.Context, requestSignature, processorType string, writer hub.DataStreamInput) error
	Delete(ctx context.Context, requestSignature, processorType string) error
}

type InvalidationModel struct {
	ProjectName string `json:"projectName" bson:"projectName"`
	CommitHash  string `json:"commitHash" bson:"commitHash"`

	InvalidationDate       time.Time          `json:"invalidationDate" bson:"invalidationDate"`
	RequestedInvalidations []string           `json:"requestedInvalidations" bson:"requestedInvalidations"`
	DoneInvalidations      []string           `json:"doneInvalidations" bson:"doneInvalidations"`
	InvalidatedImages      []CachedImageModel `json:"invalidatedImages" bson:"invalidatedImages"`
	InvalidationError      *string            `json:"invalidationError" bson:"invalidationError"`
}

type InvalidationsRepository interface {
	CreateInvalidation(ctx context.Context, invalidation InvalidationModel) error
	GetLatestInvalidation(ctx context.Context, projectName string) (InvalidationModel, error)
}

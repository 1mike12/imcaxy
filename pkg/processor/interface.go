package processor

import (
	"context"

	"github.com/thebartekbanach/imcaxy/pkg/hub"
)

type ParsedRequest struct {
	Signature         string
	SourceImageURL    string
	ProcessorEndpoint string
	ProcessingParams  map[string][]string
}

type ProcessingService interface {
	ParseRequest(requestPath string) (ParsedRequest, error)

	ProcessImage(
		ctx context.Context,
		request ParsedRequest,
		streamInput hub.DataStreamInput,
	) (
		responseContentType string,
		responseSize int64,
		err error,
	)
}

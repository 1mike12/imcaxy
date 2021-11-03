package processor

import "github.com/thebartekbanach/imcaxy/pkg/hub"

type ParsedRequest struct {
	Signature         string
	SourceImageURL    string
	ProcessorEndpoint string
	ProcessingParams  map[string][]string
}

type ProcessingService interface {
	ParseRequest(requestPath string) (ParsedRequest, error)

	ProcessImage(
		request ParsedRequest,
		streamInput hub.DataStreamInput,
	) (
		responseContentType string,
		err error,
	)
}

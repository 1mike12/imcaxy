package imaginaryprocessor

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/thebartekbanach/imcaxy/pkg/hub"
	"github.com/thebartekbanach/imcaxy/pkg/processor"
)

type httpRequestFunc func(req *http.Request) (*http.Response, error)

type Processor struct {
	config      Config
	makeRequest httpRequestFunc
}

var _ processor.ProcessingService = (*Processor)(nil)

func NewProcessor(config Config) Processor {
	return Processor{config, http.DefaultClient.Do}
}

func (proc *Processor) ParseRequest(requestPath string) (processor.ParsedRequest, error) {
	info, err := url.Parse(requestPath)
	if err != nil {
		return processor.ParsedRequest{}, err
	}

	if !info.Query().Has("url") {
		return processor.ParsedRequest{}, ErrURLParamNotIncluded
	}

	if !proc.isOperationSupported(info.Path) {
		return processor.ParsedRequest{}, ErrOperationNotSupported
	}

	source := info.Query().Get("url")
	signature := proc.generateSignature(info.Path, source, info.Query())

	request := processor.ParsedRequest{
		ProcessorEndpoint: info.Path,
		SourceImageURL:    source,
		ProcessingParams:  info.Query(),
		Signature:         signature,
	}

	return request, nil
}

func (proc *Processor) ProcessImage(
	ctx context.Context,
	request processor.ParsedRequest,
	streamInput hub.DataStreamInput,
) (responseContentType string, responseSize int64, err error) {
	req := proc.buildRequest(request)

	response, err := proc.makeRequest(req.WithContext(ctx))
	if err != nil {
		return
	}

	if response.StatusCode != 200 {
		response.Body.Close()
		err = ErrResponseStatusNotOK
		return
	}

	contentType, exists := response.Header["Content-Type"]
	if !exists || len(contentType) == 0 {
		response.Body.Close()
		err = ErrUnknownContentType
		return
	}

	responseSizeHeader, exists := response.Header["Content-Length"]
	if !exists || len(responseSizeHeader) == 0 {
		response.Body.Close()
		err = ErrUnknownContentLength
		return
	}

	responseSizeHeaderValue, err := strconv.Atoi(responseSizeHeader[0])
	if err != nil || responseSizeHeaderValue <= 0 {
		response.Body.Close()
		err = ErrUnknownContentLength
		return
	}

	go func() {
		_, err := streamInput.ReadFrom(response.Body)
		streamInput.Close(err)
		response.Body.Close()
	}()

	responseContentType = contentType[0]
	responseSize = int64(responseSizeHeaderValue)
	return
}

func (proc *Processor) buildRequest(request processor.ParsedRequest) *http.Request {
	req := http.Request{
		Method: http.MethodGet,
		URL: &url.URL{
			Host: proc.config.ImaginaryServiceURL,
			Path: request.ProcessorEndpoint,
		},
	}

	query := req.URL.Query()
	for key, values := range request.ProcessingParams {
		for _, value := range values {
			query.Add(key, value)
		}
	}

	req.URL.RawQuery = query.Encode()
	return &req
}

func (proc *Processor) generateSignature(path, source string, params map[string][]string) string {
	signature := "|" + path + "|" + source + "|"
	for _, key := range proc.getSortedMapKeys(params) {
		currentValue := ""
		for _, value := range params[key] {
			currentValue += value + ","
		}

		currentValue = strings.TrimRight(currentValue, ",")
		signature += key + "=" + currentValue + "|"
	}

	return signature
}

func (proc *Processor) getSortedMapKeys(mapToSort map[string][]string) []string {
	keys := make([]string, len(mapToSort))

	i := 0
	for key := range mapToSort {
		keys[i] = key
		i++
	}

	sort.Strings(keys)
	return keys
}

func (proc *Processor) isOperationSupported(endpoint string) bool {
	for _, supportedEndpoint := range supportedImaginaryEndpoints {
		if supportedEndpoint == endpoint {
			return true
		}
	}

	return false
}

var (
	ErrResponseStatusNotOK   = errors.New("response status not OK")
	ErrUnknownContentType    = errors.New("unknown response content type")
	ErrUnknownContentLength  = errors.New("unknown response content length")
	ErrURLParamNotIncluded   = errors.New("url param not included")
	ErrOperationNotSupported = errors.New("operation not supported")
)

var supportedImaginaryEndpoints = []string{
	"/info",
	"/crop",
	"/smartcrop",
	"/resize",
	"/enlarge",
	"/extract",
	"/zoom",
	"/thumbnail",
	"/fit",
	"/rotate",
	"/autorotate",
	"/flip",
	"/flop",
	"/convert",
	"/pipeline",
	"/watermark",
	"/watermarkimage",
	"/blur",
}

package proxy

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"strings"

	"github.com/ryanuber/go-glob"
	"github.com/thebartekbanach/imcaxy/pkg/cache"
	cacherepositories "github.com/thebartekbanach/imcaxy/pkg/cache/repositories"
	"github.com/thebartekbanach/imcaxy/pkg/filefetcher"
	"github.com/thebartekbanach/imcaxy/pkg/hub"
	"github.com/thebartekbanach/imcaxy/pkg/processor"
)

type ProxyServiceConfig struct {
	Processors     map[string]processor.ProcessingService
	AllowedDomains []string
	AllowedOrigins []string
}

type ProxyServiceImplementation struct {
	config  ProxyServiceConfig
	cache   cache.CacheService
	datahub hub.DataHub
	fetcher filefetcher.Fetcher
}

var _ ProxyService = (*ProxyServiceImplementation)(nil)

func NewProxyService(config ProxyServiceConfig, cache cache.CacheService, datahub hub.DataHub, fetcher filefetcher.Fetcher) ProxyService {
	return &ProxyServiceImplementation{
		config:  config,
		cache:   cache,
		datahub: datahub,
		fetcher: fetcher,
	}
}

func (p *ProxyServiceImplementation) Handle(ctx context.Context, rawRequestPath, callerOrigin string, rw ProxyResponseWriter) {
	parsedRequest, processorType, processor, err := p.parseRequest(rawRequestPath, callerOrigin, rw)
	if err != nil {
		return
	}

	imageOutput, imageInput, err := p.datahub.GetOrCreateStream(parsedRequest.Signature)
	if err != nil {
		log.Printf("failed to get or create stream: %s", err)
		rw.WriteError(500, "data stream creation error")
	}
	defer imageOutput.Close()

	if imageInput == nil {
		rw.WriteOK(imageOutput)
		return
	}

	if success := p.tryToGetImageFromCache(ctx, parsedRequest, processorType, imageInput, imageOutput, rw); success {
		return
	}

	p.tryToProcessAndServeImage(ctx, parsedRequest, rawRequestPath, processorType, processor, imageInput, imageOutput, rw)
}

func (p *ProxyServiceImplementation) parseRequest(rawRequestPath string, callerOrigin string, rw ProxyResponseWriter) (
	parsedRequest processor.ParsedRequest,
	processorType string,
	processor processor.ProcessingService,
	err error,
) {
	if !p.isAllowedOrigin(callerOrigin) {
		rw.WriteError(403, "request origin not allowed")
		err = errors.New("request origin not allowed")
		return
	}

	processorType, requestPath, err := p.parseRawRequestPath(rawRequestPath)
	if err != nil {
		rw.WriteError(400, "bad request")
		err = errors.New("bad request")
		return
	}

	processor, found := p.config.Processors[processorType]
	if !found {
		rw.WriteError(400, "unknown processor")
		err = errors.New("unknown processor")
		return
	}

	parsedRequest, err = processor.ParseRequest(requestPath)
	if err != nil {
		rw.WriteError(400, "request parsing error")
		err = errors.New("request parsing error")
		return
	}

	if !p.isAllowedImageSourceDomain(parsedRequest.SourceImageURL) {
		rw.WriteError(403, "source image domain not allowed")
		err = errors.New("source image domain not allowed")
		return
	}

	return
}

// returns: get success
func (p *ProxyServiceImplementation) tryToGetImageFromCache(
	ctx context.Context,
	parsedRequest processor.ParsedRequest,
	processorType string,
	input hub.DataStreamInput,
	output hub.DataStreamOutput,
	rw ProxyResponseWriter,
) bool {
	// get does not close input stream on get error,
	// so we can reuse the same stream later
	err := p.cache.Get(ctx, parsedRequest.Signature, processorType, input)
	if err != cache.ErrEntryNotFound && err != nil {
		log.Printf("cache error ocurred: %s", err)

		input.Close(err)
		rw.WriteError(500, "cache error")
		return false
	}

	if err == nil {
		rw.WriteOK(output)
		return true
	}

	return false
}

func (p *ProxyServiceImplementation) tryToProcessAndServeImage(
	ctx context.Context,
	parsedRequest processor.ParsedRequest,
	rawRequestPath, processorType string,
	processor processor.ProcessingService,
	input hub.DataStreamInput,
	output hub.DataStreamOutput,
	rw ProxyResponseWriter,
) error {
	// process image does not close the input stream on initial fetch error,
	// only when error occurs when fetches the image from processing service
	contentType, size, err := processor.ProcessImage(ctx, parsedRequest, input)
	if err != nil {
		log.Printf("writing fallback image because: %s", err)
		return p.writeFallbackImage(
			ctx,
			500,
			"processing error ocurred",
			parsedRequest,
			input,
			output,
			rw,
		)
	}

	imageInfo := cacherepositories.CachedImageModel{
		RawRequest:       rawRequestPath,
		RequestSignature: parsedRequest.Signature,

		ProcessorType:     processorType,
		ProcessorEndpoint: parsedRequest.ProcessorEndpoint,

		MimeType:         contentType,
		ImageSize:        size,
		SourceImageURL:   parsedRequest.SourceImageURL,
		ProcessingParams: parsedRequest.ProcessingParams,
	}

	p.saveImageInCache(ctx, imageInfo)

	rw.WriteOK(output)
	return nil
}

func (p *ProxyServiceImplementation) writeFallbackImage(
	ctx context.Context,
	originalCode int,
	originalMessage string,
	parsedRequest processor.ParsedRequest,
	input hub.DataStreamInput,
	output hub.DataStreamOutput,
	rw ProxyResponseWriter,
) error {
	err := p.fetcher.Fetch(ctx, parsedRequest.SourceImageURL, input)
	if err != nil {
		rw.WriteError(404, "image not found")
		return err
	}

	rw.WriteErrorWithFallback(originalCode, originalMessage, output)
	output.Close()
	return nil
}

func (p *ProxyServiceImplementation) saveImageInCache(ctx context.Context, imageInfo cacherepositories.CachedImageModel) {
	processedImageOutput, err := p.datahub.GetStreamOutput(imageInfo.RequestSignature)
	if err != nil {
		log.Printf("failed to get stream output to save image in cache: %s", err)
		return
	}

	go func() {
		defer processedImageOutput.Close()

		if err := p.cache.Save(ctx, imageInfo, processedImageOutput); err != nil {
			log.Printf("failed to save entry to cache: %s", err)
		}
	}()
}

func (p *ProxyServiceImplementation) parseRawRequestPath(rawRequestPath string) (processorType string, requestPath string, err error) {
	url, err := url.Parse(rawRequestPath)
	if err != nil {
		return
	}

	pathSegments := strings.SplitN(url.Path, "/", 3)
	if len(pathSegments) != 3 || pathSegments[0] != "" {
		err = errors.New("parsed path consists of more or less that 2 fragments")
		return
	}

	processorType = pathSegments[1]
	requestPath = fmt.Sprintf("/%s?%s", pathSegments[2], url.RawQuery)
	return
}

func (p *ProxyServiceImplementation) isAllowedOrigin(origin string) bool {
	if len(p.config.AllowedOrigins) == 0 {
		return true
	}

	for _, allowedOrigin := range p.config.AllowedOrigins {
		if glob.Glob(allowedOrigin, origin) {
			return true
		}
	}

	return false
}

func (p *ProxyServiceImplementation) isAllowedImageSourceDomain(sourceImageURL string) bool {
	if len(p.config.AllowedDomains) == 0 {
		return true
	}

	url, err := url.Parse(sourceImageURL)
	if err != nil {
		return false
	}

	sourceImageDomain := url.Hostname()
	for _, allowedDomain := range p.config.AllowedDomains {
		if glob.Glob(allowedDomain, sourceImageDomain) {
			return true
		}
	}

	return false
}

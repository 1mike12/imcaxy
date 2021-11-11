package proxy

import (
	"context"
	"errors"
	"fmt"
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

type proxyService struct {
	config  ProxyServiceConfig
	cache   cache.CacheService
	datahub hub.DataHub
	fetcher filefetcher.Fetcher
}

var _ ProxyService = (*proxyService)(nil)

func NewProxyService(config ProxyServiceConfig, cache cache.CacheService, datahub hub.DataHub, fetcher filefetcher.Fetcher) ProxyService {
	return &proxyService{
		config:  config,
		cache:   cache,
		datahub: datahub,
		fetcher: fetcher,
	}
}

func (p *proxyService) Handle(ctx context.Context, rawRequestPath, callerOrigin string, responseWriter ProxyResponseWriter) {
	if !p.isAllowedOrigin(callerOrigin) {
		responseWriter.WriteError(403, "request origin not allowed")
		return
	}

	processorType, requestPath, err := p.parseRawRequestPath(rawRequestPath)
	if err != nil {
		responseWriter.WriteError(400, "bad request")
		return
	}

	processor, found := p.config.Processors[processorType]
	if !found {
		responseWriter.WriteError(400, "unknown processor")
		return
	}

	parsedRequest, err := processor.ParseRequest(requestPath)
	if err != nil {
		responseWriter.WriteError(400, "request parsing error")
		return
	}

	if !p.isAllowedImageSourceDomain(parsedRequest.SourceImageURL) {
		responseWriter.WriteError(403, "source image domain not allowed")
		return
	}

	output, input, err := p.datahub.GetOrCreateStream(parsedRequest.Signature)
	if err != nil {
		responseWriter.WriteError(500, "data stream creation error")
		return
	}
	defer output.Close()

	if input == nil {
		responseWriter.WriteOK(output)
		return
	}

	err = p.cache.Get(ctx, parsedRequest.Signature, processorType, input)
	if err != cache.ErrEntryNotFound && err != nil {
		input.Close(err)
		responseWriter.WriteError(500, "cache error")
		return
	}

	if err == nil {
		responseWriter.WriteOK(output)
		return
	}

	contentType, size, err := processor.ProcessImage(ctx, parsedRequest, input)
	if err != nil {
		output, err := p.fetcher.Fetch(parsedRequest.SourceImageURL)
		if err != nil {
			responseWriter.WriteError(404, "image not found")
			return
		}

		responseWriter.WriteErrorWithFallback(500, "processing service error", output)
		return
	}

	clientResponseOutputStream, err := p.datahub.GetStreamOutput(parsedRequest.Signature)
	if err != nil {
		input.Close(err)
		responseWriter.WriteError(500, "datahub error when getting output stream for client response")
		return
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

	go func() {
		p.cache.Save(ctx, imageInfo, output)
		output.Close()
	}()

	responseWriter.WriteOK(clientResponseOutputStream)
}

func (p *proxyService) parseRawRequestPath(rawRequestPath string) (processorType string, requestPath string, err error) {
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

func (p *proxyService) isAllowedOrigin(origin string) bool {
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

func (p *proxyService) isAllowedImageSourceDomain(sourceImageURL string) bool {
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

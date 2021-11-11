package proxy_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/thebartekbanach/imcaxy/pkg/cache"
	mock_cache "github.com/thebartekbanach/imcaxy/pkg/cache/mocks"
	cacherepositories "github.com/thebartekbanach/imcaxy/pkg/cache/repositories"
	"github.com/thebartekbanach/imcaxy/pkg/filefetcher"
	mock_filefetcher "github.com/thebartekbanach/imcaxy/pkg/filefetcher/mocks"
	"github.com/thebartekbanach/imcaxy/pkg/hub"
	mock_hub "github.com/thebartekbanach/imcaxy/pkg/hub/mocks"
	datahubstorage "github.com/thebartekbanach/imcaxy/pkg/hub/storage"
	"github.com/thebartekbanach/imcaxy/pkg/processor"
	mock_processor "github.com/thebartekbanach/imcaxy/pkg/processor/mocks"
	"github.com/thebartekbanach/imcaxy/pkg/proxy"
	mock_proxy "github.com/thebartekbanach/imcaxy/pkg/proxy/mocks"
)

type proxyServiceTestingConfig struct {
	processors     map[string]*mock_processor.MockProcessingService
	allowedDomains []string
	allowedOrigins []string
}

type testingProxyServiceDeps struct {
	cache          *mock_cache.MockCacheService
	datahub        hub.DataHub
	fetcher        *mock_filefetcher.MockFetcher
	responseWriter *mock_proxy.MockProxyResponseWriter
	config         *proxyServiceTestingConfig
}

type testingProxyServiceCreationConfig struct {
	processorMocks []string
	allowedDomains []string
	allowedOrigins []string
}

func createTestingProxyService(t *testing.T, cfg testingProxyServiceCreationConfig) (proxy.ProxyService, *testingProxyServiceDeps, *gomock.Controller) {
	mockCtrl := gomock.NewController(t)
	cacheService := mock_cache.NewMockCacheService(mockCtrl)
	fetcher := mock_filefetcher.NewMockFetcher(mockCtrl)
	responseWriter := mock_proxy.NewMockProxyResponseWriter(mockCtrl)
	datahubStorage := datahubstorage.NewStorage()
	datahub := hub.NewDataHub(&datahubStorage)

	ctx, stopDatahubMonitors := context.WithCancel(context.Background())
	t.Cleanup(stopDatahubMonitors)

	go datahub.StartMonitors(ctx)

	if len(cfg.processorMocks) < 1 {
		cfg.processorMocks = []string{"imaginary"}
	}

	processors := map[string]processor.ProcessingService{}
	processorMocks := map[string]*mock_processor.MockProcessingService{}
	for _, name := range cfg.processorMocks {
		mockProcessingService := mock_processor.NewMockProcessingService(mockCtrl)
		processors[name] = mockProcessingService
		processorMocks[name] = mockProcessingService
	}

	if cfg.allowedDomains == nil || len(cfg.allowedDomains) == 0 {
		cfg.allowedDomains = []string{"*"}
	}

	if cfg.allowedOrigins == nil || len(cfg.allowedOrigins) == 0 {
		cfg.allowedOrigins = []string{"*"}
	}

	config := proxy.ProxyServiceConfig{
		Processors:     processors,
		AllowedDomains: cfg.allowedDomains,
		AllowedOrigins: cfg.allowedOrigins,
	}

	mockConfig := proxyServiceTestingConfig{
		processors:     processorMocks,
		allowedDomains: cfg.allowedDomains,
		allowedOrigins: cfg.allowedOrigins,
	}

	proxyService := proxy.NewProxyService(config, cacheService, datahub, fetcher)
	deps := &testingProxyServiceDeps{
		cacheService,
		datahub,
		fetcher,
		responseWriter,
		&mockConfig,
	}

	return proxyService, deps, mockCtrl
}

type goroutineSync struct {
	waitGroup sync.WaitGroup
}

func newGoroutineSync() *goroutineSync {
	return &goroutineSync{
		waitGroup: sync.WaitGroup{},
	}
}

func (g *goroutineSync) Wait(t *testing.T) {
	response := make(chan struct{})

	go func() {
		g.waitGroup.Wait()
		response <- struct{}{}
	}()

	select {
	case <-response:
		return
	case <-time.After(time.Second * 1):
		t.Fatal("goroutine sync timed out after 1 second")
	}
}

func (g *goroutineSync) WaitForCacheSave() func(ctx context.Context, imageInfo cacherepositories.CachedImageModel, r hub.DataStreamOutput) error {
	g.waitGroup.Add(1)
	return func(ctx context.Context, imageInfo cacherepositories.CachedImageModel, r hub.DataStreamOutput) error {
		g.waitGroup.Done()
		return nil
	}
}

func TestProxyService_FirstHandleShouldProcessImageAndSaveInCacheAndReturn(t *testing.T) {
	proxy, deps, _ := createTestingProxyService(t, testingProxyServiceCreationConfig{})

	requestURLWithoutProcessor := "/test?url=http://google.com/image.jpg"
	requestURL := "/imaginary" + requestURLWithoutProcessor
	parsedRequest := processor.ParsedRequest{
		Signature:         "test-signature",
		SourceImageURL:    "http://google.com/image.jpg",
		ProcessorEndpoint: "/test",
		ProcessingParams:  map[string][]string{"url": {"http://google.com/image.jpg"}},
	}

	deps.config.processors["imaginary"].EXPECT().ParseRequest(requestURLWithoutProcessor).Return(parsedRequest, nil)
	deps.cache.EXPECT().Get(gomock.Any(), parsedRequest.Signature, "imaginary", gomock.Any()).Return(cache.ErrEntryNotFound)
	deps.config.processors["imaginary"].EXPECT().ProcessImage(gomock.Any(), parsedRequest, gomock.Any()).Return("image/jpeg", int64(1), nil)

	imageInfo := cacherepositories.CachedImageModel{
		RawRequest:       requestURL,
		RequestSignature: parsedRequest.Signature,

		ProcessorType:     "imaginary",
		ProcessorEndpoint: parsedRequest.ProcessorEndpoint,

		MimeType:         "image/jpeg",
		ImageSize:        1,
		ProcessingParams: parsedRequest.ProcessingParams,
		SourceImageURL:   parsedRequest.SourceImageURL,
	}

	sync := newGoroutineSync()
	defer sync.Wait(t)

	deps.cache.EXPECT().Save(gomock.Any(), imageInfo, gomock.Any()).Do(sync.WaitForCacheSave()).Return(nil)
	deps.responseWriter.EXPECT().WriteOK(gomock.Any())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	proxy.Handle(ctx, requestURL, "github.com", deps.responseWriter)
}

func TestProxyService_SecondHandleShouldReturnImageFromCache(t *testing.T) {
	proxy, deps, _ := createTestingProxyService(t, testingProxyServiceCreationConfig{})

	requestURLWithoutProcessor := "/test?url=http://google.com/image.jpg"
	requestURL := "/imaginary" + requestURLWithoutProcessor
	parsedRequest := processor.ParsedRequest{
		Signature:         "test-signature",
		SourceImageURL:    "http://google.com/image.jpg",
		ProcessorEndpoint: "/test",
		ProcessingParams:  map[string][]string{"url": {"http://google.com/image.jpg"}},
	}

	deps.config.processors["imaginary"].EXPECT().ParseRequest(requestURLWithoutProcessor).Return(parsedRequest, nil)
	deps.cache.EXPECT().Get(gomock.Any(), parsedRequest.Signature, "imaginary", gomock.Any()).Return(nil)
	deps.responseWriter.EXPECT().WriteOK(gomock.Any())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	proxy.Handle(ctx, requestURL, "github.com", deps.responseWriter)
}

func TestProxyService_Returns400BadRequestWhenRequestURLIsBroken(t *testing.T) {
	proxy, deps, _ := createTestingProxyService(t, testingProxyServiceCreationConfig{})

	requestURL := "/something-weird-happened-there"

	deps.responseWriter.EXPECT().WriteError(400, "bad request")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	proxy.Handle(ctx, requestURL, "github.com", deps.responseWriter)
}

func TestProxyService_Returns400BadRequestOnRequestParsingError(t *testing.T) {
	proxy, deps, _ := createTestingProxyService(t, testingProxyServiceCreationConfig{})

	requestURLWithoutProcessor := "/test?url=http://google.com/image.jpg"
	requestURL := "/imaginary" + requestURLWithoutProcessor
	parsedRequest := processor.ParsedRequest{
		Signature:         "test-signature",
		SourceImageURL:    "http://google.com/image.jpg",
		ProcessorEndpoint: "/test",
		ProcessingParams:  map[string][]string{"url": {"http://google.com/image.jpg"}},
	}

	deps.config.processors["imaginary"].EXPECT().ParseRequest(requestURLWithoutProcessor).Return(parsedRequest, errors.New("some error"))
	deps.responseWriter.EXPECT().WriteError(400, "request parsing error")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	proxy.Handle(ctx, requestURL, "github.com", deps.responseWriter)
}

func TestProxyService_RejectsRequestIfSourceImageDomainIsNotAllowed(t *testing.T) {
	proxy, deps, _ := createTestingProxyService(t, testingProxyServiceCreationConfig{
		allowedDomains: []string{"github.com"},
	})

	requestURLWithoutProcessor := "/test?url=http://google.com/image.jpg"
	requestURL := "/imaginary" + requestURLWithoutProcessor
	parsedRequest := processor.ParsedRequest{
		Signature:         "test-signature",
		SourceImageURL:    "http://google.com/image.jpg",
		ProcessorEndpoint: "/test",
		ProcessingParams:  map[string][]string{"url": {"http://google.com/image.jpg"}},
	}

	deps.config.processors["imaginary"].EXPECT().ParseRequest(requestURLWithoutProcessor).Return(parsedRequest, nil)
	deps.responseWriter.EXPECT().WriteError(403, "source image domain not allowed")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	proxy.Handle(ctx, requestURL, "github.com", deps.responseWriter)
}

func TestProxyService_AllowsRequestIfSourceImageDomainIsAllowed(t *testing.T) {
	proxy, deps, _ := createTestingProxyService(t, testingProxyServiceCreationConfig{
		allowedDomains: []string{"github.com", "google.com"},
	})

	requestURLWithoutProcessor := "/test?url=http://google.com/image.jpg"
	requestURL := "/imaginary" + requestURLWithoutProcessor
	parsedRequest := processor.ParsedRequest{
		Signature:         "test-signature",
		SourceImageURL:    "http://google.com/image.jpg",
		ProcessorEndpoint: "/test",
		ProcessingParams:  map[string][]string{"url": {"http://google.com/image.jpg"}},
	}

	deps.config.processors["imaginary"].EXPECT().ParseRequest(requestURLWithoutProcessor).Return(parsedRequest, nil)
	deps.cache.EXPECT().Get(gomock.Any(), parsedRequest.Signature, "imaginary", gomock.Any()).Return(cache.ErrEntryNotFound)
	deps.config.processors["imaginary"].EXPECT().ProcessImage(gomock.Any(), parsedRequest, gomock.Any()).Return("image/jpeg", int64(1), nil)

	imageInfo := cacherepositories.CachedImageModel{
		RawRequest:       requestURL,
		RequestSignature: parsedRequest.Signature,

		ProcessorType:     "imaginary",
		ProcessorEndpoint: parsedRequest.ProcessorEndpoint,

		MimeType:         "image/jpeg",
		ImageSize:        1,
		ProcessingParams: parsedRequest.ProcessingParams,
		SourceImageURL:   parsedRequest.SourceImageURL,
	}

	sync := newGoroutineSync()
	defer sync.Wait(t)

	deps.cache.EXPECT().Save(gomock.Any(), imageInfo, gomock.Any()).Do(sync.WaitForCacheSave()).Return(nil)
	deps.responseWriter.EXPECT().WriteOK(gomock.Any())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	proxy.Handle(ctx, requestURL, "github.com", deps.responseWriter)
}

func TestProxyService_AllowsRequestIfSourceImageDomainIsAllowedUsingGlobPattern(t *testing.T) {
	proxy, deps, _ := createTestingProxyService(t, testingProxyServiceCreationConfig{
		allowedDomains: []string{"github.com", "google.*"},
	})

	requestURLWithoutProcessor := "/test?url=http://google.com/image.jpg"
	requestURL := "/imaginary" + requestURLWithoutProcessor
	parsedRequest := processor.ParsedRequest{
		Signature:         "test-signature",
		SourceImageURL:    "http://google.com/image.jpg",
		ProcessorEndpoint: "/test",
		ProcessingParams:  map[string][]string{"url": {"http://google.com/image.jpg"}},
	}

	deps.config.processors["imaginary"].EXPECT().ParseRequest(requestURLWithoutProcessor).Return(parsedRequest, nil)
	deps.cache.EXPECT().Get(gomock.Any(), parsedRequest.Signature, "imaginary", gomock.Any()).Return(cache.ErrEntryNotFound)
	deps.config.processors["imaginary"].EXPECT().ProcessImage(gomock.Any(), parsedRequest, gomock.Any()).Return("image/jpeg", int64(1), nil)

	imageInfo := cacherepositories.CachedImageModel{
		RawRequest:       requestURL,
		RequestSignature: parsedRequest.Signature,

		ProcessorType:     "imaginary",
		ProcessorEndpoint: parsedRequest.ProcessorEndpoint,

		MimeType:         "image/jpeg",
		ImageSize:        1,
		ProcessingParams: parsedRequest.ProcessingParams,
		SourceImageURL:   parsedRequest.SourceImageURL,
	}

	sync := newGoroutineSync()
	defer sync.Wait(t)

	deps.cache.EXPECT().Save(gomock.Any(), imageInfo, gomock.Any()).Do(sync.WaitForCacheSave()).Return(nil)
	deps.responseWriter.EXPECT().WriteOK(gomock.Any())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	proxy.Handle(ctx, requestURL, "github.com", deps.responseWriter)
}

func TestProxyService_RejectsRequestIfRequesterOriginIsNotAllowed(t *testing.T) {
	proxy, deps, _ := createTestingProxyService(t, testingProxyServiceCreationConfig{
		allowedOrigins: []string{"google.com"},
	})

	requestURL := "/imaginary/test?url=http://google.com/image.jpg"

	deps.responseWriter.EXPECT().WriteError(403, "request origin not allowed")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	proxy.Handle(ctx, requestURL, "github.com", deps.responseWriter)
}

func TestProxyService_AllowsRequestIfRequesterOriginIsAllowed(t *testing.T) {
	proxy, deps, _ := createTestingProxyService(t, testingProxyServiceCreationConfig{
		allowedOrigins: []string{"google.com", "github.com"},
	})

	requestURLWithoutProcessor := "/test?url=http://google.com/image.jpg"
	requestURL := "/imaginary" + requestURLWithoutProcessor
	parsedRequest := processor.ParsedRequest{
		Signature:         "test-signature",
		SourceImageURL:    "http://google.com/image.jpg",
		ProcessorEndpoint: "/test",
		ProcessingParams:  map[string][]string{"url": {"http://google.com/image.jpg"}},
	}

	deps.config.processors["imaginary"].EXPECT().ParseRequest(requestURLWithoutProcessor).Return(parsedRequest, nil)
	deps.cache.EXPECT().Get(gomock.Any(), parsedRequest.Signature, "imaginary", gomock.Any()).Return(cache.ErrEntryNotFound)
	deps.config.processors["imaginary"].EXPECT().ProcessImage(gomock.Any(), parsedRequest, gomock.Any()).Return("image/jpeg", int64(1), nil)

	imageInfo := cacherepositories.CachedImageModel{
		RawRequest:       requestURL,
		RequestSignature: parsedRequest.Signature,

		ProcessorType:     "imaginary",
		ProcessorEndpoint: parsedRequest.ProcessorEndpoint,

		MimeType:         "image/jpeg",
		ImageSize:        1,
		ProcessingParams: parsedRequest.ProcessingParams,
		SourceImageURL:   parsedRequest.SourceImageURL,
	}

	sync := newGoroutineSync()
	defer sync.Wait(t)

	deps.cache.EXPECT().Save(gomock.Any(), imageInfo, gomock.Any()).Do(sync.WaitForCacheSave()).Return(nil)
	deps.responseWriter.EXPECT().WriteOK(gomock.Any())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	proxy.Handle(ctx, requestURL, "google.com", deps.responseWriter)
}

func TestProxyService_AllowsRequestIfRequesterOriginIsAllowedUsingGlobPattern(t *testing.T) {
	proxy, deps, _ := createTestingProxyService(t, testingProxyServiceCreationConfig{
		allowedOrigins: []string{"google.*", "github.com"},
	})

	requestURLWithoutProcessor := "/test?url=http://google.com/image.jpg"
	requestURL := "/imaginary" + requestURLWithoutProcessor
	parsedRequest := processor.ParsedRequest{
		Signature:         "test-signature",
		SourceImageURL:    "http://google.com/image.jpg",
		ProcessorEndpoint: "/test",
		ProcessingParams:  map[string][]string{"url": {"http://google.com/image.jpg"}},
	}

	deps.config.processors["imaginary"].EXPECT().ParseRequest(requestURLWithoutProcessor).Return(parsedRequest, nil)
	deps.cache.EXPECT().Get(gomock.Any(), parsedRequest.Signature, "imaginary", gomock.Any()).Return(cache.ErrEntryNotFound)
	deps.config.processors["imaginary"].EXPECT().ProcessImage(gomock.Any(), parsedRequest, gomock.Any()).Return("image/jpeg", int64(1), nil)

	imageInfo := cacherepositories.CachedImageModel{
		RawRequest:       requestURL,
		RequestSignature: parsedRequest.Signature,

		ProcessorType:     "imaginary",
		ProcessorEndpoint: parsedRequest.ProcessorEndpoint,

		MimeType:         "image/jpeg",
		ImageSize:        1,
		ProcessingParams: parsedRequest.ProcessingParams,
		SourceImageURL:   parsedRequest.SourceImageURL,
	}

	sync := newGoroutineSync()
	defer sync.Wait(t)

	deps.cache.EXPECT().Save(gomock.Any(), imageInfo, gomock.Any()).Do(sync.WaitForCacheSave()).Return(nil)
	deps.responseWriter.EXPECT().WriteOK(gomock.Any())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	proxy.Handle(ctx, requestURL, "google.com", deps.responseWriter)
}

func TestProxyService_HandlesProcessorErrorByReturningOriginalImageAsFallback(t *testing.T) {
	proxy, deps, mockCtrl := createTestingProxyService(t, testingProxyServiceCreationConfig{})

	requestURLWithoutProcessor := "/test?url=http://google.com/image.jpg"
	requestURL := "/imaginary" + requestURLWithoutProcessor
	parsedRequest := processor.ParsedRequest{
		Signature:         "test-signature",
		SourceImageURL:    "http://google.com/image.jpg",
		ProcessorEndpoint: "/test",
		ProcessingParams:  map[string][]string{"url": {"http://google.com/image.jpg"}},
	}

	deps.config.processors["imaginary"].EXPECT().ParseRequest(requestURLWithoutProcessor).Return(parsedRequest, nil)
	deps.cache.EXPECT().Get(gomock.Any(), parsedRequest.Signature, "imaginary", gomock.Any()).Return(cache.ErrEntryNotFound)
	deps.config.processors["imaginary"].EXPECT().ProcessImage(gomock.Any(), parsedRequest, gomock.Any()).Return("", int64(0), errors.New("some error"))

	fetchOutputStream := mock_hub.NewMockDataStreamOutput(mockCtrl)
	deps.fetcher.EXPECT().Fetch(parsedRequest.SourceImageURL).Return(fetchOutputStream, nil)

	deps.responseWriter.EXPECT().WriteErrorWithFallback(500, "processing service error", fetchOutputStream)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	proxy.Handle(ctx, requestURL, "google.com", deps.responseWriter)
}

func TestProxyService_HandlesProcessorErrorByReturning404IfImageDoesNotExist(t *testing.T) {
	proxy, deps, _ := createTestingProxyService(t, testingProxyServiceCreationConfig{})

	requestURLWithoutProcessor := "/test?url=http://google.com/image.jpg"
	requestURL := "/imaginary" + requestURLWithoutProcessor
	parsedRequest := processor.ParsedRequest{
		Signature:         "test-signature",
		SourceImageURL:    "http://google.com/image.jpg",
		ProcessorEndpoint: "/test",
		ProcessingParams:  map[string][]string{"url": {"http://google.com/image.jpg"}},
	}

	deps.config.processors["imaginary"].EXPECT().ParseRequest(requestURLWithoutProcessor).Return(parsedRequest, nil)
	deps.cache.EXPECT().Get(gomock.Any(), parsedRequest.Signature, "imaginary", gomock.Any()).Return(cache.ErrEntryNotFound)
	deps.config.processors["imaginary"].EXPECT().ProcessImage(gomock.Any(), parsedRequest, gomock.Any()).Return("", int64(0), errors.New("some error"))

	deps.fetcher.EXPECT().Fetch(parsedRequest.SourceImageURL).Return(nil, filefetcher.ErrResponseStatusNotOK)
	deps.responseWriter.EXPECT().WriteError(404, "image not found")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	proxy.Handle(ctx, requestURL, "google.com", deps.responseWriter)
}

func TestProxyService_HandlesProcessorErrorByReturning400IfRequestIsNotCorrect(t *testing.T) {
	proxy, deps, _ := createTestingProxyService(t, testingProxyServiceCreationConfig{})

	requestURLWithoutProcessor := "/test?url=http://google.com/image.jpg"
	requestURL := "/imaginary" + requestURLWithoutProcessor
	deps.config.processors["imaginary"].EXPECT().ParseRequest(requestURLWithoutProcessor).Return(processor.ParsedRequest{}, errors.New("some error"))
	deps.responseWriter.EXPECT().WriteError(400, "request parsing error")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	proxy.Handle(ctx, requestURL, "google.com", deps.responseWriter)
}

func TestProxyService_Returns400IfTriedToUseUnknownProcessor(t *testing.T) {
	proxy, deps, _ := createTestingProxyService(t, testingProxyServiceCreationConfig{})

	requestURL := "/unknown/test?url=http://google.com/image.jpg"
	deps.responseWriter.EXPECT().WriteError(400, "unknown processor")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	proxy.Handle(ctx, requestURL, "google.com", deps.responseWriter)
}

func TestProxyService_HandlesImageUsingCorrectProcessor(t *testing.T) {
	proxy, deps, _ := createTestingProxyService(t, testingProxyServiceCreationConfig{
		processorMocks: []string{"imaginary-1", "imaginary-2"},
	})

	requestURLWithoutProcessor := "/test?url=http://google.com/image.jpg"
	requestURL := "/imaginary-2" + requestURLWithoutProcessor
	parsedRequest := processor.ParsedRequest{
		Signature:         "test-signature",
		SourceImageURL:    "http://google.com/image.jpg",
		ProcessorEndpoint: "/test",
		ProcessingParams:  map[string][]string{"url": {"http://google.com/image.jpg"}},
	}

	deps.config.processors["imaginary-2"].EXPECT().ParseRequest(requestURLWithoutProcessor).Return(parsedRequest, nil)
	deps.cache.EXPECT().Get(gomock.Any(), parsedRequest.Signature, "imaginary-2", gomock.Any()).Return(cache.ErrEntryNotFound)
	deps.config.processors["imaginary-2"].EXPECT().ProcessImage(gomock.Any(), parsedRequest, gomock.Any()).Return("image/jpeg", int64(1), nil)

	imageInfo := cacherepositories.CachedImageModel{
		RawRequest:       requestURL,
		RequestSignature: parsedRequest.Signature,

		ProcessorType:     "imaginary-2",
		ProcessorEndpoint: parsedRequest.ProcessorEndpoint,

		MimeType:         "image/jpeg",
		ImageSize:        1,
		ProcessingParams: parsedRequest.ProcessingParams,
		SourceImageURL:   parsedRequest.SourceImageURL,
	}

	sync := newGoroutineSync()
	defer sync.Wait(t)

	deps.cache.EXPECT().Save(gomock.Any(), imageInfo, gomock.Any()).Do(sync.WaitForCacheSave()).Return(nil)
	deps.responseWriter.EXPECT().WriteOK(gomock.Any())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	proxy.Handle(ctx, requestURL, "github.com", deps.responseWriter)
}

func TestProxyService_ReturnsProcessedImageEvenIfCacheCannotSaveIt(t *testing.T) {
	proxy, deps, _ := createTestingProxyService(t, testingProxyServiceCreationConfig{})

	requestURLWithoutProcessor := "/test?url=http://google.com/image.jpg"
	requestURL := "/imaginary" + requestURLWithoutProcessor
	parsedRequest := processor.ParsedRequest{
		Signature:         "test-signature",
		SourceImageURL:    "http://google.com/image.jpg",
		ProcessorEndpoint: "/test",
		ProcessingParams:  map[string][]string{"url": {"http://google.com/image.jpg"}},
	}

	deps.config.processors["imaginary"].EXPECT().ParseRequest(requestURLWithoutProcessor).Return(parsedRequest, nil)
	deps.cache.EXPECT().Get(gomock.Any(), parsedRequest.Signature, "imaginary", gomock.Any()).Return(cache.ErrEntryNotFound)
	deps.config.processors["imaginary"].EXPECT().ProcessImage(gomock.Any(), parsedRequest, gomock.Any()).Return("image/jpeg", int64(1), nil)

	imageInfo := cacherepositories.CachedImageModel{
		RawRequest:       requestURL,
		RequestSignature: parsedRequest.Signature,

		ProcessorType:     "imaginary",
		ProcessorEndpoint: parsedRequest.ProcessorEndpoint,

		MimeType:         "image/jpeg",
		ImageSize:        1,
		ProcessingParams: parsedRequest.ProcessingParams,
		SourceImageURL:   parsedRequest.SourceImageURL,
	}

	sync := newGoroutineSync()
	defer sync.Wait(t)

	deps.cache.EXPECT().Save(gomock.Any(), imageInfo, gomock.Any()).Do(sync.WaitForCacheSave()).Return(errors.New("some error"))
	deps.responseWriter.EXPECT().WriteOK(gomock.Any())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	proxy.Handle(ctx, requestURL, "github.com", deps.responseWriter)
}

func TestProxyService_ReturnsErrorWhenCacheServiceGetReturnsNonErrEntryNotFoundErrorWithProcessedImageAsFallbackAndTriesToSaveItInCache(t *testing.T) {
	proxy, deps, _ := createTestingProxyService(t, testingProxyServiceCreationConfig{})

	requestURLWithoutProcessor := "/test?url=http://google.com/image.jpg"
	requestURL := "/imaginary" + requestURLWithoutProcessor
	parsedRequest := processor.ParsedRequest{
		Signature:         "test-signature",
		SourceImageURL:    "http://google.com/image.jpg",
		ProcessorEndpoint: "/test",
		ProcessingParams:  map[string][]string{"url": {"http://google.com/image.jpg"}},
	}

	deps.config.processors["imaginary"].EXPECT().ParseRequest(requestURLWithoutProcessor).Return(parsedRequest, nil)
	deps.cache.EXPECT().Get(gomock.Any(), parsedRequest.Signature, "imaginary", gomock.Any()).Return(cache.ErrEntryNotFound)
	deps.config.processors["imaginary"].EXPECT().ProcessImage(gomock.Any(), parsedRequest, gomock.Any()).Return("image/jpeg", int64(1), nil)

	imageInfo := cacherepositories.CachedImageModel{
		RawRequest:       requestURL,
		RequestSignature: parsedRequest.Signature,

		ProcessorType:     "imaginary",
		ProcessorEndpoint: parsedRequest.ProcessorEndpoint,

		MimeType:         "image/jpeg",
		ImageSize:        1,
		ProcessingParams: parsedRequest.ProcessingParams,
		SourceImageURL:   parsedRequest.SourceImageURL,
	}

	sync := newGoroutineSync()
	defer sync.Wait(t)

	deps.cache.EXPECT().Save(gomock.Any(), imageInfo, gomock.Any()).Do(sync.WaitForCacheSave()).Return(nil)
	deps.responseWriter.EXPECT().WriteOK(gomock.Any())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	proxy.Handle(ctx, requestURL, "github.com", deps.responseWriter)
}

func TestProxyService_ReturnsStreamFromImageIfItIsAlreadyProcessing(t *testing.T) {
	proxy, deps, _ := createTestingProxyService(t, testingProxyServiceCreationConfig{})

	requestURLWithoutProcessor := "/test?url=http://google.com/image.jpg"
	requestURL := "/imaginary" + requestURLWithoutProcessor
	parsedRequest := processor.ParsedRequest{
		Signature:         "test-signature",
		SourceImageURL:    "http://google.com/image.jpg",
		ProcessorEndpoint: "/test",
		ProcessingParams:  map[string][]string{"url": {"http://google.com/image.jpg"}},
	}

	deps.config.processors["imaginary"].EXPECT().ParseRequest(requestURLWithoutProcessor).Return(parsedRequest, nil)
	deps.datahub.CreateStream("test-signature")

	deps.responseWriter.EXPECT().WriteOK(gomock.Any())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	proxy.Handle(ctx, requestURL, "github.com", deps.responseWriter)
}

// Integration:

// multiple requests at once should be handled correctly by calling processor only once and returning image twice

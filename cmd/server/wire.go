//go:build wireinject
// +build wireinject

package main

import (
	"context"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/google/wire"
	"github.com/thebartekbanach/imcaxy/pkg/cache"
	cacherepositories "github.com/thebartekbanach/imcaxy/pkg/cache/repositories"
	dbconnections "github.com/thebartekbanach/imcaxy/pkg/cache/repositories/connections"
	"github.com/thebartekbanach/imcaxy/pkg/filefetcher"
	"github.com/thebartekbanach/imcaxy/pkg/hub"
	datahubstorage "github.com/thebartekbanach/imcaxy/pkg/hub/storage"
	"github.com/thebartekbanach/imcaxy/pkg/processor"
	imaginaryprocessor "github.com/thebartekbanach/imcaxy/pkg/processor/imaginary"
	"github.com/thebartekbanach/imcaxy/pkg/proxy"
)

func InitializeMongoConnectionConfig() dbconnections.CacheDBConfig {
	config := dbconnections.CacheDBConfig{
		ConnectionString: os.Getenv("IMCAXY_MONGO_CONNECTION_STRING"),
	}

	if config.ConnectionString == "" {
		log.Panic("IMCAXY_MONGO_CONNECTION_STRING is required environment variable")
	}

	parsedConnectionString, err := url.Parse(config.ConnectionString)
	if err != nil {
		log.Panicf("Error ocurred when parsing IMCAXY_MONGO_CONNECTION_STRING: %s", err)
	}

	if parsedConnectionString.User == nil {
		log.Panicf("IMCAXY_MONGO_CONNECTION_STRING must contain credentials")
	}

	return config
}

func InitializeMongoConnection(ctx context.Context, mongoConfig dbconnections.CacheDBConfig) dbconnections.CacheDBConnection {
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	cacheDbConnection, err := dbconnections.NewCacheDBProductionConnection(ctx, mongoConfig)
	if err != nil {
		log.Panicf("Error ocurred when initializing MongoDB connection: %s", err)
	}

	return cacheDbConnection
}

func InitializeMinioConnectionConfig() dbconnections.MinioBlockStorageProductionConnectionConfig {
	config := dbconnections.MinioBlockStorageProductionConnectionConfig{
		Endpoint:  os.Getenv("IMCAXY_MINIO_ENDPOINT"),
		AccessKey: os.Getenv("IMCAXY_MINIO_ACCESS_KEY"),
		SecretKey: os.Getenv("IMCAXY_MINIO_SECRET_KEY"),
		Location:  os.Getenv("IMCAXY_MINIO_LOCATION"),
		Bucket:    os.Getenv("IMCAXY_MINIO_BUCKET"),
		UseSSL:    os.Getenv("IMCAXY_MINIO_SSL") == "true",
	}

	if config.Endpoint == "" {
		log.Panic("IMCAXY_MINIO_ENDPOINT is required environment variable")
	}

	if _, err := url.Parse(config.Endpoint); err != nil {
		log.Panicf("Error ocurred when parsing IMCAXY_MINIO_ENDPOINT: %s", err)
	}

	if config.AccessKey == "" {
		log.Panic("IMCAXY_MINIO_ACCESS_KEY is required environment variable")
	}

	if config.SecretKey == "" {
		log.Panic("IMCAXY_MINIO_SECRET_KEY is required environment variable")
	}

	if config.Location == "" {
		config.Location = "us-east-1"
	}

	if config.Bucket == "" {
		log.Panic("IMCAXY_MINIO_BUCKET is required environment variable")
	}

	return config
}

func InitializeMinioConnection(ctx context.Context, minioConfig dbconnections.MinioBlockStorageProductionConnectionConfig) dbconnections.MinioBlockStorageConnection {
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	minioBlockStorageConnection, err := dbconnections.NewMinioBlockStorageProductionConnection(ctx, minioConfig)
	if err != nil {
		log.Panicf("Error ocurred when initializing Minio connection: %s", err)
	}

	return &minioBlockStorageConnection
}

func InitializeImaginaryProcessingService() imaginaryprocessor.Processor {
	config := imaginaryprocessor.Config{
		ImaginaryServiceURL: os.Getenv("IMCAXY_IMAGINARY_SERVICE_URL"),
	}

	if config.ImaginaryServiceURL == "" {
		log.Panic("IMCAXY_IMAGINARY_SERVICE_URL is required environment variable")
	}

	if _, err := url.Parse(config.ImaginaryServiceURL); err != nil {
		log.Panicf("Error ocurred when parsing IMCAXY_IMAGINARY_SERVICE_URL: %s", err)
	}

	return imaginaryprocessor.NewProcessor(config)
}

func InitializeDataHub(ctx context.Context, storage datahubstorage.StorageAdapter) hub.DataHub {
	dataHub := hub.NewDataHub(storage)
	dataHub.StartMonitors(ctx)
	return dataHub
}

func InitializeProxyConfig(imaginaryProcessingService imaginaryprocessor.Processor) proxy.ProxyServiceConfig {
	config := proxy.ProxyServiceConfig{
		Processors: map[string]processor.ProcessingService{
			"imaginary": &imaginaryProcessingService,
		},
		AllowedDomains: strings.Split(os.Getenv("IMCAXY_ALLOWED_DOMAINS"), ","),
		AllowedOrigins: strings.Split(os.Getenv("IMCAXY_ALLOWED_ORIGINS"), ","),
	}

	if len(config.AllowedDomains) == 0 || config.AllowedDomains[0] == "" && len(config.AllowedDomains) == 1 {
		config.AllowedDomains = []string{"*"}
	}

	if len(config.AllowedOrigins) == 0 || config.AllowedOrigins[0] == "" && len(config.AllowedOrigins) == 1 {
		config.AllowedOrigins = []string{"*"}
	}

	return config
}

func InitializeCache(ctx context.Context) cache.CacheService {
	wire.Build(
		InitializeMinioConnectionConfig,
		InitializeMinioConnection,
		cacherepositories.NewCachedImagesStorage,

		InitializeMongoConnectionConfig,
		InitializeMongoConnection,
		cacherepositories.NewCachedImagesRepository,

		cache.NewCacheService,
	)

	return &cache.CacheServiceImplementation{}
}

func InitializeInvalidator(ctx context.Context, cacheService cache.CacheService) cache.InvalidationService {
	wire.Build(
		InitializeMongoConnectionConfig,
		InitializeMongoConnection,
		cacherepositories.NewInvalidationsRepository,
		cache.NewInvalidationService,
	)

	return &cache.InvalidationServiceImplementation{}
}

func InitializeProxy(ctx context.Context, cache cache.CacheService) proxy.ProxyService {
	wire.Build(
		datahubstorage.NewStorage,
		InitializeDataHub,

		filefetcher.NewDataHubFetcher,
		InitializeImaginaryProcessingService,

		InitializeProxyConfig,
		proxy.NewProxyService,
	)

	return &proxy.ProxyServiceImplementation{}
}

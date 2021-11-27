# Imcaxy - the Imaginary cache and proxy service

The high performance cache and proxy service for Imaginary.

This project aims to serve frontend website assets (mainly images) at the lowest possible level of consumption of cellular data and lowest possible latency without sacrificing the quality of our assets providing by that best personalized quality of assets for each user on every device separately.

# Features

This project supports automatic file caching and cache invalidation through HTTP API. It just forwards all requests that call `/imaginary` endpoint to the Imaginary service if given request is not cached yet.

It also supports cache invalidation for files that were changed during development. It is done by saving latest commit hash in database and then fetching the this hash during `CI` build, finding last known commit in `Git` history, listing all changed files that fit given regular expressions under given directory and sending all changed file names for invalidation.

API, invalidator and react libraries can be found in [this repository](https://github.com/thebartekbanach/imcaxy-client).

**This project supports only URL source of image, no local file source support**. Every image that you want to process have to be accessible from public network.

Also, you may allow only known origins for incoming requests and allow processing for images only from known domains.

# API

This service share following HTTP endpoints:

- `GET /imaginary/...` - call it like normal Imaginary service, but it will cache the response if it is not cached yet. All available endpoints and parameters are available [here](https://github.com/h2non/imaginary#get-).
- `GET /latestInvalidation` - returns latest invalidation info. It is used by `CI` build to invalidate get info about latest invalidation to get know from which commit to look for file changes. This endpoint is secured by access token set by `IMCAXY_INVALIDATE_SECURITY_TOKEN` environment variable sent to server using `Authorization` HTTP header. You need to include `projectName` query parameter with project name that the invalidation is done for. It returns following json:

  ```typescript
  interface InvalidationModel {
    projectName: string;
    commitHash: string;

    invalidationDate: Date;
    requestedInvalidations: string[];
    doneInvalidations: string[];

    invalidatedImages: {
      rawRequest: string;
      requestSignature: string;

      processorType: string;
      processorEndpoint: string;

      mimeType: string;
      imageSize: number;
      sourceImageURL: string;
      processingParams: Record<string, string[]>;
    }[];

    invalidationError: string | null;
  }
  ```

- `DELETE /invalidate` - invalidates given cached images. This endpoint is secured by access token set by `IMCAXY_INVALIDATE_SECURITY_TOKEN` environment variable sent to server using `Authorization` HTTP header. You need to include following query params:

  - `projectName` - project name that the invalidation is done for
  - `latestCommitHash` - the latest commit hash that was available, used later for finding which images were changed during development
  - `urls` - the urls of images that should be invalidated, for example: `http://your-domain.com/image.png`

  It returns same json as `GET /latestInvalidation` endpoint:

  ```typescript
  interface InvalidationModel {
    projectName: string;
    commitHash: string;

    invalidationDate: Date;
    requestedInvalidations: string[];
    doneInvalidations: string[];

    invalidatedImages: {
      rawRequest: string;
      requestSignature: string;

      processorType: string;
      processorEndpoint: string;

      mimeType: string;
      imageSize: number;
      sourceImageURL: string;
      processingParams: Record<string, string[]>;
    }[];

    invalidationError: string | null;
  }
  ```

# Setup

To setup the project you should follow these steps:

1. Make sure that Imaginary service is running and available from Imcaxy service.
2. Make sure that MongoDB service is running and available from Imcaxy service.
3. Make sure that Minio service is running and available from Imcaxy service.
4. Set environment variables. You can find examples in `./config/env/examples` directory. All environment variables are described below.
5. Run `make build` script to build the executable.
6. Now your executable is available in `./bin` directory, just run it.

Environment variables:

- `IMCAXY_MONGO_CONNECTION_STRING` - MongoDB connection string
- `IMCAXY_MINIO_ENDPOINT` - Minio service endpoint, in pattern: `DOMAIN:PORT` - without `http(s)://` prefix
- `IMCAXY_MINIO_ACCESS_KEY` - Minio service access key
- `IMCAXY_MINIO_SECRET_KEY` - Minio service secret key
- `IMCAXY_MINIO_BUCKET` - Bucket name that will be used as Imcaxy service data bucket
- `IMCAXY_MINIO_LOCATION` - _optional_, location of the bucket
- `IMCAXY_MINIO_SSL` - _optional_, set it to `true` if you want to use SSL
- `IMCAXY_IMAGINARY_SERVICE_URL` - Imaginary service endpoint, in pattern: `DOMAIN:PORT` - without `http(s)://` prefix
- `IMCAXY_INVALIDATE_SECURITY_TOKEN` - security token that is used to access invalidation endpoint, use long random string for that
- `IMCAXY_ALLOWED_DOMAINS` - _optional_, list of allowed domains, separated with comma, for example: `example.com,example.net`, if not set, all domains are allowed
- `IMCAXY_ALLOWED_ORIGINS` - _optional_, list of allowed origins, separated with comma, for example: `example.com,example.net`, if not set, all origins are allowed

# Development

## Requirements

You wll need `make`, `docker`, `docker compose` and `go` with version at least `1.17`.

## Makefile scripts

There are available few useful `make` scripts:

- `make build` - builds the executable and puts it in `./bin` directory
- `make dev` - starts development environment for manual testing of the service, it starts also all necessary services: `MongoDB`, `Minio` and `Imaginary`, attaches to server logs
- `make stop-dev` - stops development environment, but preserves volumes
- `make cleanup-dev` - stops development environment and removes all containers and volumes
- `make build-dev` - builds all containers of development environment
- `make test` - runs unit tests
- `make integration-tests` - runs unit and integration tests
- `make build-integration-tests` - builds all containers of integration tests environment
- `make cleanup-integration-tests` - stops integration testing environment and removes all containers and volumes

## Project structure

Project structure is defined as follows:

- `./cmd/server/` - contains imcaxy main executable
- `./config/env/` - contains environment variables configuration and examples
- `./pkg` - root packages directory
- `test` - contains test data, global mocks and test utils

## Main packages

### Hub

`Hub` package contains `DataHub` structure. It is responsible for multiplexing incoming data from single source into multiple destination.

But the special feature of this package and what makes it different that simple multiplexer is that it caches all incoming data in memory, which allows to attach new data stream listeners even when data fetching is almost done.

It unloads the network and because of that, there is no way to double the processing work on same requests when proxy is already processing request with same signature.

### Cache

`Cache` package is responsible for data persistence layer. It allows to read cached images from `Minio` block storage and write new ones to `Minio` and `MongoDB` services.

This package shares also a `InvalidationService` which takes care about images invalidation by implementing simple invalidation API that is used directly by HTTP handlers.

### Processor

`Processor` package contains image processing service abstraction. Under this package placed are all available processing service packages.

Currently only `imaginary` processing service is available. But you can create your own processor if you want to implement support for another processing service.

### Proxy

`Proxy` package contains service that proxies all requests to our known processors and caches them. If given request is already cached, it returns it from cache.

This service also takes care about requests validation and checks if request origin is allowed as well as the domain of source image to process.

## Server cmd

The cmd in `./cmd/server/` directory contains main package of the service. It uses `Wire` package to initialize all services and its dependencies in correct order.

HTTP handlers are defined in in `./cmd/server/handlers.go` file.

## Testing

This project uses unit and integration testing techniques.

To run unit tests, run `make test` command.

If you want to run integration and unit tests, run `make integration-tests` command. Integration tests are skipped when running `make test` command, as it runs only unit tests.

You can remove containers and volumes of integration testing environment by running `make cleanup-integration-tests` command.

# License

This project is licensed under MIT license. Feel free to do with it whatever you want.

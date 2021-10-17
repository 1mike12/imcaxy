# Imcaxy - the imaginary cache and proxy service

This project is under development.

Expected functionality:

- [ ] simple proxy for imaginary service
- [ ] cache imaginary results in minio block storage, save cache info inside mongo database
- [ ] scan for changes in files, if some file is changed, discard all cached responses of this file
- [ ] if file is unavailable (deleted), discard all cached responses of this file
- [ ] collects statistics about resources usage, including:
  - [ ] how frequently is selected resource used
  - [ ] last time when was resource was used
- [ ] from time to time scans for unknown cached data and unknown database entries
- [ ] uses only URLs, does not access local files
- [ ] currently processed files registry with process parameters, if file we want to optimize is already processing (and process parameters are the same) it should wait for end of optimization process and get the request from minio server instead of sending two or more same requests to imaginary worker
- [ ] allows to send `revalidate` files api request, with optional file URL parameter that was changed

Project parts:
- `proxy` - proxies all traffic from client to imaginary service and uses `cache` to get or add new resources to cache
- `cache` - caches all resources using `mongo` database and `minio` block storage
  - `monitoring` - monitors usage of resources, imaginary response times
  - `integration` - cares about `mongo` and `minio` integration, and also about handling `revalidate` api requests, automatically scans for remote file changes and more

Proposed way to split things:
- `proxy`
  - `handlers.go` - will contain all proxy http handlers
  - `validate.go` - will validate incoming requests
  - `forward.go` - will forward all non-cached requests to imaginary service
- `cache/repositories` - will contain repositories that will proxy between rest of the code and mongo / minio services
  - `models.go` - will contain all necessary models definitions
  - `mongoRepositories.go` - will contain all mongodb repositories definitions
  - `minioRepositories.go` - will contain all minio repositories definitions
  - `...` or should I use `repositoryName.go` pattern which would contain all models for selected repository and all necessary definitions?
- `cache/monitoring`
  - `usage.go` - will handle usage statistics
  - `response.go` - will handle response times
- `cache/integration`
  
  
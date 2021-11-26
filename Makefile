.PHONY: test

build:
	@echo Building executable file
	@cd ./cmd/server && go build -o ../../bin/server . && cd ../..

dev:
	@echo Starting development environment
	@docker compose -f dev.docker-compose.yml up dev-imcaxy-imaginary dev-imcaxy-mongo dev-imcaxy-minio --detach --remove-orphans
	
	@echo Starting Imcaxy server
	@docker compose -f dev.docker-compose.yml up dev-imcaxy-server --no-log-prefix

stop-dev:
	@echo Stopping development environment
	@docker-compose -f dev.docker-compose.yml down

cleanup-dev:
	@echo Cleaning up development environment
	@docker-compose -f dev.docker-compose.yml down --volumes --remove-orphans

build-dev:
	@echo Building development environment
	@docker-compose -f dev.docker-compose.yml build

test:
	@echo Running unit tests on local machine
	@go test -short -timeout 5s ./...

integration-tests:
	@echo Starting integration testing environment
	@docker compose -f integration-tests.docker-compose.yml up integration-tests-imcaxy-imaginary integration-tests-imcaxy-mongo integration-tests-imcaxy-minio --detach --remove-orphans

	@echo Running integration tests
	@docker compose -f integration-tests.docker-compose.yml up integration-tests-imcaxy-server --no-log-prefix

build-integration-tests:
	@echo Building integration test runner images
	@docker compose -f integration-tests.docker-compose.yml build

cleanup-integration-tests:
	@echo Cleaning up integration test environment
	@docker-compose -f integration-tests.docker-compose.yml down --volumes --remove-orphans
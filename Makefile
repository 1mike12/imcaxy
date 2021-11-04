.PHONY: test

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
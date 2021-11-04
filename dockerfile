FROM golang:1.17.2 AS base

WORKDIR /go/src/app
COPY ./go.* ./
RUN go mod download

FROM base AS dev

RUN go get github.com/githubnemo/CompileDaemon
CMD ["CompileDaemon", "-build", "go build -o ./bin/server ./cmd/server", "-command", "./bin/server"]

FROM base AS integration-tests
CMD ["go", "test", "-timeout=5s", "./..."]
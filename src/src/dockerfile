FROM golang:1.17.2 AS base

WORKDIR /go/src/app

FROM base AS dev

RUN go get github.com/githubnemo/CompileDaemon
CMD ["CompileDaemon", "-build", "go build -o ./bin/server ./cmd/server", "-command", "./bin/server"]

FROM base AS prod

COPY . .

RUN go get -d -v ./...
RUN go install -v ./...

# TODO: add cmd
FROM golang:1.15.2

RUN go get github.com/githubnemo/CompileDaemon

WORKDIR /go/src/app
COPY . .

RUN go get -d -v ./...
RUN go install -v ./...

CMD ["CompileDaemon", "-build", "go build -o /go/src/bin/app", "-command", "/go/src/bin/app", "-color"]
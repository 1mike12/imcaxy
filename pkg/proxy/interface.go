package proxy

import (
	"context"
	"io"
)

type ProxyResponseWriter interface {
	WriteOK(reader io.ReadCloser)
	WriteError(code int, message string)
	WriteErrorWithFallback(code int, message string, fallbackImageReader io.ReadCloser)
}

type ProxyService interface {
	Handle(ctx context.Context, requestPath, callerOrigin string, responseWriter ProxyResponseWriter)
}

package main

import (
	"io"
	"net/http"
	"strings"
)

type proxyResponseWriter struct {
	w http.ResponseWriter
}

func (w *proxyResponseWriter) WriteOK(reader io.ReadCloser) {
	io.Copy(w.w, reader)
	reader.Close()
}

func (w *proxyResponseWriter) WriteError(code int, message string) {
	w.w.WriteHeader(code)
	io.Copy(w.w, strings.NewReader(message))
}

func (w *proxyResponseWriter) WriteErrorWithFallback(code int, message string, fallbackImageReader io.ReadCloser) {
	io.Copy(w.w, fallbackImageReader)
	fallbackImageReader.Close()
}

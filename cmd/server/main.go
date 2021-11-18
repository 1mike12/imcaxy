package main

import (
	"context"
	"log"
	"net/http"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log.Println("initializing proxy service")
	proxyService := InitializeProxy(ctx)

	log.Println("registering http handlers")
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		request := r.URL.Path + "?" + r.URL.RawQuery
		log.Printf("processing: %s", request)
		proxyService.Handle(ctx, request, r.Header.Get("Origin"), &proxyResponseWriter{w})
		r.Body.Close()
	})

	log.Println("listening on port 80")
	log.Fatal(http.ListenAndServe(":80", nil))
}

package main

import (
	"context"
	"log"
	"net/http"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log.Println("initializing cache service")
	cacheService := InitializeCache(ctx)

	log.Println("initializing invalidation service")
	invalidationService := InitializeInvalidator(ctx, cacheService)

	log.Println("initializing proxy service")
	proxyService := InitializeProxy(ctx, cacheService)

	log.Println("registering http handlers")
	http.HandleFunc("/", handleRequest(ctx, proxyService))
	http.HandleFunc("/invalidate", handleInvalidationRequest(ctx, invalidationService))
	http.HandleFunc("/lastInvalidation", handleLatestInvalidationInfoRequest(ctx, invalidationService))

	log.Println("listening on port 80")
	log.Fatal(http.ListenAndServe(":80", nil))
}

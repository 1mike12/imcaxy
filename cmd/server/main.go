package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	cacherepositories "github.com/thebartekbanach/imcaxy/pkg/cache/repositories"
)

func marshalAndSendInvalidatedEntries(w http.ResponseWriter, statusCode int, entries []cacherepositories.CachedImageModel) {
	invalidatedEntriesJSON, err := json.Marshal(entries)
	if err != nil {
		log.Printf("error ocurred when marshalling invalidated entries: %s", err)
		w.Write([]byte("error ocurred when marshalling invalidated entries"))
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(statusCode)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(invalidatedEntriesJSON)
	w.WriteHeader(statusCode)
}

func getInvalidateEndpointPath() string {
	securityToken := os.Getenv("IMCAXY_INVALIDATE_SECURITY_TOKEN")

	if securityToken == "" {
		return "/invalidate"
	}

	return fmt.Sprintf("/%s/invalidate", securityToken)
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log.Println("initializing cache service")
	cache := InitializeCache(ctx)

	log.Println("initializing proxy service")
	proxyService := InitializeProxy(ctx, cache)

	log.Println("registering http handlers")
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		request := r.URL.Path + "?" + r.URL.RawQuery
		log.Printf("processing: %s", request)
		proxyService.Handle(ctx, request, r.Header.Get("Origin"), &proxyResponseWriter{w})
		r.Body.Close()
	})

	http.HandleFunc(getInvalidateEndpointPath(), func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(ctx, time.Minute)
		defer cancel()

		if r.Method != http.MethodDelete {
			w.Write([]byte("/invalidate requires DELETE method"))
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		allInvalidatedEntries := []cacherepositories.CachedImageModel{}

		urls := r.URL.Query()["urls"]
		for _, url := range urls {
			log.Printf("invalidating URL: %s", url)
			invalidatedEntries, err := cache.InvalidateAllEntriesForURL(ctx, url)
			allInvalidatedEntries = append(allInvalidatedEntries, invalidatedEntries...)
			if err != nil {
				log.Printf("error ocurred when invalidating entries \"%s\": %s", url, err)
				marshalAndSendInvalidatedEntries(w, http.StatusInternalServerError, allInvalidatedEntries)
				return
			}
		}

		marshalAndSendInvalidatedEntries(w, http.StatusOK, allInvalidatedEntries)
	})

	log.Println("listening on port 80")
	log.Fatal(http.ListenAndServe(":80", nil))
}

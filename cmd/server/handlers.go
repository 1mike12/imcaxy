package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/thebartekbanach/imcaxy/pkg/cache"
	"github.com/thebartekbanach/imcaxy/pkg/proxy"
)

func handleRequest(ctx context.Context, proxyService proxy.ProxyService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		processingCtx, cancel := context.WithTimeout(ctx, time.Minute)
		defer cancel()

		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte("only GET method is allowed"))
			return
		}

		request := r.URL.Path + "?" + r.URL.RawQuery
		log.Printf("processing: %s", request)

		proxyService.Handle(processingCtx, request, r.Header.Get("Origin"), &proxyResponseWriter{w})
		r.Body.Close()
	}
}

func handleInvalidationRequest(ctx context.Context, invalidationService cache.InvalidationService) http.HandlerFunc {
	rawAccessToken := os.Getenv("IMCAXY_INVALIDATE_SECURITY_TOKEN")
	accessToken := fmt.Sprintf("Bearer %s", rawAccessToken)

	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(ctx, time.Minute)
		defer cancel()

		if r.Method != http.MethodDelete {
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte("only DELETE method is allowed"))
			return
		}

		if rawAccessToken != "" && r.Header.Get("Authorization") != accessToken {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("access token authorization failed"))
			return
		}

		projectName := r.URL.Query().Get("projectName")
		if projectName == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("projectName query parameter is required"))
			return
		}

		latestCommitHash := r.URL.Query().Get("latestCommitHash")
		if latestCommitHash == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("latestCommitHash query parameter is required"))
			return
		}

		urls := r.URL.Query()["urls"]
		if len(urls) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("urls query parameter is required"))
			return
		}

		result, invalidationErr := invalidationService.Invalidate(ctx, projectName, latestCommitHash, urls)
		jsonResult, marshalErr := json.Marshal(result)
		if marshalErr != nil {
			log.Printf("error ocurred when marshalling invalidated entries: %s", marshalErr)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("error ocurred when marshalling invalidated entries"))
			return
		}

		if invalidationErr != nil {
			log.Printf("error ocurred when invalidating: %s", invalidationErr)
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusOK)
		}

		w.Write(jsonResult)
	}
}

func handleLatestInvalidationInfoRequest(ctx context.Context, invalidationService cache.InvalidationService) http.HandlerFunc {
	rawAccessToken := os.Getenv("IMCAXY_INVALIDATE_SECURITY_TOKEN")
	accessToken := fmt.Sprintf("Bearer %s", rawAccessToken)

	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(ctx, time.Minute)
		defer cancel()

		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte("only GET method is allowed"))
			return
		}

		if rawAccessToken != "" && r.Header.Get("Authorization") != accessToken {
			w.Write([]byte("access token authorization failed"))
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		projectName := r.URL.Query().Get("projectName")
		if projectName == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("projectName query parameter is required"))
			return
		}

		result, infoGetErr := invalidationService.GetLastKnownInvalidation(ctx, projectName)
		jsonResult, marshalErr := json.Marshal(result)
		if marshalErr != nil {
			log.Printf("error ocurred when marshalling invalidated entries: %s", marshalErr)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("error ocurred when marshalling invalidated entries"))
			return
		}

		if infoGetErr != nil {
			log.Printf("error ocurred when getting last known invalidation: %s", infoGetErr)
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusOK)
		}

		w.Write(jsonResult)
	}
}

package main

import (
	"fmt"
	"imcaxy/cache"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	fmt.Println("Configuring Imcaxy server")

	http.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(rw, "Hello world from Imcaxy! Requested url: %s", r.URL.Path[1:])
	})

	http.HandleFunc("/proxy", func(rw http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(rw, "Requested proxied image parameters are: %s", r.URL.Query())
	})

	http.Handle("/status", promhttp.Handler())

	addStartupInfo(time.Now())
	cache.TryAddingAndFinding()

	panic(http.ListenAndServe(":80", nil))
}

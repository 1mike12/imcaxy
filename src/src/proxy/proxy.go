package proxy

import (
	"io"
	"log"
	"net/http"

	"imcaxy/cache"
)

// Proxy service object, should have inserted all dependencies listed in struct definition
type Proxy struct {
	env struct {
		ImaginaryRequestURL string
	}

	cache cache.ImcaxyCache
}

// HandlePipeline is just simple HTTP handler, handles image processing request and returns processed or cached image
func (proxy *Proxy) HandlePipeline(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	fileName := query.Get("file")

	if fileName == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Request must include non-empty file parameters"))
		return
	}

	query.Del("file")

	isCached, err := proxy.cache.ExistsInCache(fileName, query)
	if err != nil {
		log.Println(err)
	}

	if isCached {
		reader, err := proxy.cache.GetFromCache(fileName, query)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Internal server error when reading from cache"))
			log.Println(err)
		}

		io.Copy(w, reader)
		return
	}

	log.Println("Making request to imaginary service at:", (proxy.env.ImaginaryRequestURL + r.URL.Path))

	imaginaryResponse, err := http.Get(proxy.env.ImaginaryRequestURL + r.URL.Path)
	if err != nil {
		log.Println(err)
		// TODO: return a fallback image
		return
	}

	if imaginaryResponse.StatusCode == 200 {
		cacheWriter, err := proxy.cache.AddToCache(fileName, format, query)
		if err != nil {
			log.Println(err)
			io.Copy(w, imaginaryResponse.Body)
			return
		}

		_, err = io.Copy(io.MultiWriter(cacheWriter, w), imaginaryResponse.Body)
		if err != nil {
			log.Println(err)
		}

		return
	}

	io.Copy(w, imaginaryResponse.Body)
}

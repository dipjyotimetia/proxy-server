package main

import (
	"context"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"time"
)

var backendURLs = map[string]string{
	"/v2/beers":        "api.punkapi.com",
	"/api/v2/products": "api.punkapi.com",
}

func main() {

	// Create a server to handle requests.
	server := &http.Server{
		Addr: ":8080",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get the backend URL for the requested API path.
			backendURL, ok := backendURLs[r.URL.Path]
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			// Create a reverse proxy.
			reverseProxy := httputil.NewSingleHostReverseProxy(&url.URL{
				Scheme: "http",
				Host:   backendURL,
			})
			// Forward the request to the backend server.
			reverseProxy.ServeHTTP(w, r)
		}),
	}

	// Start the server.
	go func() {
		log.Println("Starting proxy server on port 8080...")
		if err := server.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()

	// Handle graceful shutdown.
	done := make(chan bool)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	go func() {
		<-quit
		log.Println("Shutting down proxy server...")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		server.Shutdown(ctx)
		close(done)
	}()

	<-done
	log.Println("Proxy server stopped.")
}

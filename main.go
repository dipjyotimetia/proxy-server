package main

import (
	"context"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"time"
)

const (
	// The maximum number of requests allowed per second.
	requestLimit = 100
)

type rateLimiter struct {
	mu sync.Mutex
	// The number of requests made in the last second.
	count int
	// The time at which the last request was made.
	lastRequest time.Time
}

func (r *rateLimiter) allow() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	if now.Sub(r.lastRequest) > time.Second {
		r.count = 0
	}

	if r.count >= requestLimit {
		return false
	}

	r.count++
	r.lastRequest = now

	return true
}

func main() {
	// Get the URL of the server to proxy.
	backendHost := "http://localhost:8081"

	// Create a rate limiter.
	rateLimiter := &rateLimiter{}

	// Create a reverse proxy.
	reverseProxy := httputil.NewSingleHostReverseProxy(&url.URL{
		Scheme: "http",
		Host:   backendHost,
	})

	// Create a server to handle requests.
	server := &http.Server{
		Addr: ":8080",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if the request is allowed.
			if !rateLimiter.allow() {
				http.Error(w, "Too many requests", http.StatusTooManyRequests)
				return
			}

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

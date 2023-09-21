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
	mu        sync.Mutex
	limit     int
	remaining int
	lastReset time.Time
}

func newRateLimiter(limit int) *rateLimiter {
	return &rateLimiter{
		limit:     limit,
		remaining: limit,
		lastReset: time.Now(),
	}
}

func (rl *rateLimiter) allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	if now.Sub(rl.lastReset) > 1*time.Second {
		rl.remaining = rl.limit
		rl.lastReset = now
	}

	if rl.remaining == 0 {
		return false
	}

	rl.remaining--
	return true
}

var backendURLs = map[string]string{
	"/api/v1/users":    "http://localhost:8081",
	"/api/v2/products": "http://localhost:8082",
}

func main() {
	// Create a rate limiter.
	rateLimiter := newRateLimiter(requestLimit)

	// Create a server to handle requests.
	server := &http.Server{
		Addr: ":8080",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if the request is allowed.
			if !rateLimiter.allow() {
				w.WriteHeader(http.StatusTooManyRequests)
				return
			}
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

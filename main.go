package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// The maximum number of requests allowed per second.
	requestLimit = 100
)

type rateLimiter struct {
	redisClient *redis.Client
	limit       int
}

func newRateLimiter(redisClient *redis.Client, limit int) *rateLimiter {
	return &rateLimiter{
		redisClient: redisClient,
		limit:       limit,
	}
}

func (rl *rateLimiter) allow(apiKey string) bool {
	// Get the current timestamp.
	now := time.Now().Unix()

	// Get the number of requests that have been made in the last second for the given API key.
	count, err := rl.redisClient.ZCard(context.Background(), fmt.Sprintf("rate-limiter:%s:%d:%d", apiKey, rl.limit, now)).Result()
	if err != nil {
		log.Println(err)
		return false
	}

	// If the number of requests is less than the limit, allow the request.
	if count < int64(rl.limit) {
		// Add the request to the Redis set.
		err = rl.redisClient.ZAdd(context.Background(), fmt.Sprintf("rate-limiter:%s:%d:%d", apiKey, rl.limit, now), redis.Z{
			Score:  float64(now),
			Member: "1",
		}).Err()
		if err != nil {
			log.Println(err)
			return false
		}

		return true
	}

	// Otherwise, the request is not allowed.
	return false
}

var backendURLs = map[string]string{
	"/api/v1/users":    "http://localhost:8081",
	"/api/v2/products": "http://localhost:8082",
}

func main() {
	// Create a new Redis connection.
	redisClient := redis.NewClient(&redis.Options{
		Addr: os.Getenv("REDIS_ADDR"),
	})

	// Create a new rate limiter using the Redis connection.
	rateLimiter := newRateLimiter(redisClient, requestLimit)

	// Create a server to handle requests.
	server := &http.Server{
		Addr: ":8080",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if the request is allowed.
			if !rateLimiter.allow(r.Header.Get("X-API-Key")) {
				http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
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

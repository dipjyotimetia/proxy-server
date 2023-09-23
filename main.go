package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

type Subscription struct {
	ClientID string `json:"client_id"`
	Plan     string `json:"plan"`
	Limit    int64  `json:"limit"`
}

type RateLimiter struct {
	redisClient *redis.Client
}

func NewRateLimiter(redisClient *redis.Client) *RateLimiter {
	return &RateLimiter{
		redisClient: redisClient,
	}
}

func (r *RateLimiter) Allow(ctx context.Context, key string, limit int64, duration time.Duration) error {
	// Use the INCRBY command to increment by 1 and get the new count.
	newCount, err := r.redisClient.IncrBy(ctx, key, 1).Result()
	if err != nil {
		return err
	}

	if newCount > limit {
		// Use the EXPIRE command to set the expiration time.
		err := r.redisClient.Expire(ctx, key, duration).Err()
		if err != nil {
			return err
		}
		return fmt.Errorf("rate limit exceeded")
	}

	return nil
}

type ReverseProxyServer struct {
	rateLimiter *RateLimiter
}

func NewReverseProxyServer(rateLimiter *RateLimiter) *ReverseProxyServer {
	return &ReverseProxyServer{
		rateLimiter: rateLimiter,
	}
}

func (s *ReverseProxyServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Check if the client has a valid subscription
	subscription, err := GetSubscription(s.rateLimiter.redisClient, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// Check if the client has exceeded their rate limit
	err = s.rateLimiter.Allow(r.Context(), r.URL.Path, subscription.Limit, time.Second)
	if err != nil {
		http.Error(w, err.Error(), http.StatusTooManyRequests)
		return
	}

	backendURL, ok := backendURLs[r.URL.Path]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	reverseProxy := httputil.NewSingleHostReverseProxy(&url.URL{
		Scheme: "https",
		Host:   backendURL,
	})

	reverseProxy.ServeHTTP(w, r)
}

// GetSubscription GetSubscription()
func GetSubscription(redisClient *redis.Client, r *http.Request) (*Subscription, error) {
	// Get the client ID from the request
	clientID := r.Header.Get("X-Client-ID")

	// Get the subscription from Redis
	subscriptionBytes, err := redisClient.Get(r.Context(), clientID).Bytes()
	if err != nil {
		return nil, err
	}

	// Unmarshal the subscription JSON into a Subscription object
	var subscription Subscription
	err = json.Unmarshal(subscriptionBytes, &subscription)
	if err != nil {
		return nil, err
	}

	return &subscription, nil
}

var backendURLs = map[string]string{
	"/v2/beers":        "api.punkapi.com",
	"/v2/beers/random": "api.punkapi.com",
}

func main() {
	// Create a new Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr: os.Getenv("REDIS_ADDR"),
	})

	// Create a new rate limiter
	rateLimiter := NewRateLimiter(redisClient)
	// Create a new reverse proxy server
	reverseProxyServer := NewReverseProxyServer(rateLimiter)

	// Listen for incoming requests
	log.Println("Listening on port 8080")
	log.Fatal(http.ListenAndServe(":8080", reverseProxyServer))
}

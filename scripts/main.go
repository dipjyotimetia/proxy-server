package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

// Check if the plan is one of the allowed values.
var allowedPlans = map[string]bool{
	"silver":   true,
	"gold":     true,
	"platinum": true,
}

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

func main() {
	// Create a new Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	// Check if there was an error during client initialization
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	rateLimiter := NewRateLimiter(redisClient)
	setSubscriptionRoute := NewSetSubscriptionRoute(rateLimiter)

	http.HandleFunc("/subscriptions", setSubscriptionRoute)
	log.Println("Listening on port 8081")
	log.Fatal(http.ListenAndServe(":8081", nil))
}

// NewSetSubscriptionRoute creates a new route to handle POST requests to `/subscriptions`.
func NewSetSubscriptionRoute(rateLimiter *RateLimiter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Decode the JSON body of the request to get the subscription object.
		var subscription Subscription
		err := json.NewDecoder(r.Body).Decode(&subscription)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Validate the subscription object.
		if subscription.ClientID == "" || subscription.Plan == "" || subscription.Limit <= 0 {
			http.Error(w, "invalid subscription", http.StatusBadRequest)
			return
		}

		if !allowedPlans[subscription.Plan] {
			http.Error(w, "invalid plan", http.StatusBadRequest)
			return
		}

		// Save the subscription to Redis.
		err = SetSubscription(r.Context(), rateLimiter.redisClient, &subscription)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Return a success response to the client.
		w.WriteHeader(http.StatusCreated)
	}
}

func SetSubscription(ctx context.Context, redisClient *redis.Client, subscription *Subscription) error {
	// Marshal the subscription object to JSON
	subscriptionJSON, err := json.Marshal(subscription)
	if err != nil {
		return err
	}

	// Use the SET command to set the subscription in Redis with an expiration time.
	err = redisClient.Set(ctx, subscription.ClientID, subscriptionJSON, time.Minute).Err()
	if err != nil {
		return err
	}

	return nil
}

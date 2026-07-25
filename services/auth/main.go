package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"nodefall/services/auth/db"
	"nodefall/shared/middleware"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	dbURL := requireEnv("NODEFALL_DB_URL")

	database, err := db.Connect(ctx, dbURL)
	if err != nil {
		log.Fatalf("connecting to database: %v", err)
	}
	defer database.Close()

	if err := database.EnsureSchema(ctx); err != nil {
		log.Fatalf("ensuring schema: %v", err)
	}

	h := &handlers{db: database}

	// 1 request/sec sustained, burst of 5 — generous enough for normal
	// use, tight enough to blunt naive brute-force/spam attempts.
	limiter := middleware.NewRateLimiter(1, 5)

	mux := http.NewServeMux()
	mux.Handle("POST /register", limiter.Wrap(http.HandlerFunc(h.handleRegister)))
	mux.Handle("POST /login", limiter.Wrap(http.HandlerFunc(h.handleLogin)))

	httpServer := &http.Server{
		Addr:    ":8082",
		Handler: middleware.WithCORS(mux),
	}

	go func() {
		<-ctx.Done()
		log.Println("shutting down auth service")
		httpServer.Shutdown(context.Background())
	}()

	log.Println("auth service running on :8082")
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

// requireEnv reads an environment variable, failing fast at startup
// if it isn't set rather than limping along and failing confusingly
// on the first request that needs it.
func requireEnv(key string) string {
	val := os.Getenv(key)
	if val == "" {
		log.Fatalf("%s not set", key)
	}
	return val
}

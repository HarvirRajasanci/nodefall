package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"nodefall/shared/config"
	"nodefall/shared/middleware"
	"nodefall/shared/registry"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg := config.Load()
	if cfg.RedisURL == "" {
		log.Fatal("NODEFALL_REDIS_URL not set — matchmaker requires Redis for service discovery")
	}
	reg := registry.New(cfg.RedisURL)

	queue := NewQueue(reg)

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	go queue.Run(ctx, ticker.C)

	h := &handlers{queue: queue}
	mux := http.NewServeMux()
	mux.Handle("POST /queue", middleware.WithAuth(http.HandlerFunc(h.handleJoinQueue)))
	mux.Handle("GET /queue", middleware.WithAuth(http.HandlerFunc(h.handleQueueStatus)))
	mux.Handle("DELETE /queue", middleware.WithAuth(http.HandlerFunc(h.handleLeaveQueue)))

	httpServer := &http.Server{
		Addr:    ":8083",
		Handler: middleware.WithCORS(mux),
	}

	go func() {
		<-ctx.Done()
		log.Println("shutting down matchmaker")
		httpServer.Shutdown(context.Background())
	}()

	log.Println("matchmaker running on :8083")
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

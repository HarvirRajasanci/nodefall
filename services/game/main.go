package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"

	"nodefall/services/game/engine"
	"nodefall/services/game/server"
	"nodefall/shared/middleware"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	e := engine.New()
	go e.Run(ctx)

	srv := server.New(e)
	mux := http.NewServeMux()
	mux.Handle("/ws", middleware.WithAuth(http.HandlerFunc(srv.ServeWS)))

	httpServer := &http.Server{
		Addr:    ":8081",
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		log.Println("shutting down game server")
		httpServer.Shutdown(context.Background())
	}()

	log.Println("game server running on :8081")
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

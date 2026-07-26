package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"

	"nodefall/services/game/matchmanager"
	"nodefall/services/game/server"
	"nodefall/shared/genproto"
	"nodefall/shared/middleware"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	matches := matchmanager.New(ctx)

	srv := server.New(matches)
	mux := http.NewServeMux()
	mux.Handle("/ws", middleware.WithAuth(http.HandlerFunc(srv.ServeWS)))

	httpServer := &http.Server{
		Addr:    ":8081",
		Handler: mux,
	}

	grpcServer := grpc.NewServer()
	genproto.RegisterGameServiceServer(grpcServer, server.NewGRPCServer(matches, "localhost:8081"))

	grpcListener, err := net.Listen("tcp", ":9090")
	if err != nil {
		log.Fatalf("grpc listen: %v", err)
	}

	go func() {
		<-ctx.Done()
		log.Println("shutting down game server")
		httpServer.Shutdown(context.Background())
		grpcServer.GracefulStop()
	}()

	go func() {
		log.Println("game gRPC server running on :9090")
		if err := grpcServer.Serve(grpcListener); err != nil {
			log.Printf("grpc serve error: %v", err)
		}
	}()

	log.Println("game server running on :8081")
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

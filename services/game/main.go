package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"

	"nodefall/services/game/matchmanager"
	"nodefall/services/game/server"
	"nodefall/shared/config"
	"nodefall/shared/genproto"
	"nodefall/shared/middleware"
	"nodefall/shared/registry"
)

const (
	registryTTL       = 5 * time.Second
	heartbeatInterval = 2 * time.Second
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	instanceID := os.Getenv("NODEFALL_INSTANCE_ID")
	if instanceID == "" {
		instanceID = "game-1"
	}

	cfg := config.Load()
	if cfg.RedisURL != "" {
		reg := registry.New(cfg.RedisURL)
		inst := registry.Instance{ID: instanceID, GRPCAddr: instanceID + ":9090"}
		go reg.Heartbeat(ctx, inst, registryTTL, heartbeatInterval)
		log.Printf("registered with Redis registry as %s", instanceID)
	} else {
		log.Println("NODEFALL_REDIS_URL not set — running without service registry")
	}

	matches := matchmanager.New(ctx)

	srv := server.New(matches)
	mux := http.NewServeMux()
	mux.Handle("/ws", middleware.WithAuth(http.HandlerFunc(srv.ServeWS)))

	httpServer := &http.Server{
		Addr:    ":8081",
		Handler: mux,
	}

	grpcServer := grpc.NewServer()
	genproto.RegisterGameServiceServer(grpcServer, server.NewGRPCServer(matches, instanceID))

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
		log.Printf("game gRPC server (%s) running on :9090", instanceID)
		if err := grpcServer.Serve(grpcListener); err != nil {
			log.Printf("grpc serve error: %v", err)
		}
	}()

	log.Printf("game server (%s) running on :8081", instanceID)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

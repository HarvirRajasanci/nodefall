package registry

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
)

func testRegistry(t *testing.T) (*Registry, *miniredis.Miniredis) {
	t.Helper()

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("starting miniredis: %v", err)
	}
	t.Cleanup(mr.Close)

	return New(mr.Addr()), mr
}

func TestRegister_ThenListLive_ReturnsInstance(t *testing.T) {
	r, _ := testRegistry(t)
	ctx := context.Background()

	err := r.Register(ctx, Instance{ID: "game-1", GRPCAddr: "game-1:9090"}, 5*time.Second)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	instances, err := r.ListLive(ctx)
	if err != nil {
		t.Fatalf("ListLive failed: %v", err)
	}
	if len(instances) != 1 || instances[0].ID != "game-1" {
		t.Errorf("got %+v, want one instance game-1", instances)
	}
}

func TestListLive_OmitsExpiredInstance(t *testing.T) {
	r, mr := testRegistry(t)
	ctx := context.Background()

	r.Register(ctx, Instance{ID: "game-1", GRPCAddr: "game-1:9090"}, 1*time.Second)
	mr.FastForward(2 * time.Second) // simulate the TTL elapsing

	instances, err := r.ListLive(ctx)
	if err != nil {
		t.Fatalf("ListLive failed: %v", err)
	}
	if len(instances) != 0 {
		t.Errorf("got %d instances, want 0 — entry should have expired", len(instances))
	}
}

func TestListLive_ReturnsMultipleInstances(t *testing.T) {
	r, _ := testRegistry(t)
	ctx := context.Background()

	r.Register(ctx, Instance{ID: "game-1", GRPCAddr: "game-1:9090"}, 5*time.Second)
	r.Register(ctx, Instance{ID: "game-2", GRPCAddr: "game-2:9090"}, 5*time.Second)

	instances, err := r.ListLive(ctx)
	if err != nil {
		t.Fatalf("ListLive failed: %v", err)
	}
	if len(instances) != 2 {
		t.Errorf("got %d instances, want 2", len(instances))
	}
}

func TestHeartbeat_RegistersImmediatelyOnStart(t *testing.T) {
	r, _ := testRegistry(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go r.Heartbeat(ctx, Instance{ID: "game-1", GRPCAddr: "game-1:9090"}, 5*time.Second, time.Second)

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		instances, _ := r.ListLive(context.Background())
		if len(instances) == 1 {
			return // success — the initial Register landed
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("instance was not registered shortly after Heartbeat started")
}

func TestHeartbeat_RefreshesBeforeTTLExpires(t *testing.T) {
	r, _ := testRegistry(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// TTL comfortably longer than the interval, so a real refresh
	// keeps the entry alive well past what a single un-refreshed
	// registration's TTL would have allowed.
	ttl := 300 * time.Millisecond
	interval := 100 * time.Millisecond
	go r.Heartbeat(ctx, Instance{ID: "game-1", GRPCAddr: "game-1:9090"}, ttl, interval)

	time.Sleep(700 * time.Millisecond) // several intervals, past the original TTL

	instances, err := r.ListLive(context.Background())
	if err != nil {
		t.Fatalf("ListLive failed: %v", err)
	}
	if len(instances) != 1 {
		t.Error("instance expired despite Heartbeat running — refresh did not keep it alive")
	}
}

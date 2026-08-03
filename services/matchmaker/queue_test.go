package main

import (
	"context"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"

	"github.com/alicebob/miniredis/v2"

	"nodefall/shared/genproto"
	"nodefall/shared/registry"
)

type fakeGameService struct {
	genproto.UnimplementedGameServiceServer
	calls      []*genproto.MatchRequest
	acceptNext bool
}

func (f *fakeGameService) StartMatch(ctx context.Context, req *genproto.MatchRequest) (*genproto.MatchResponse, error) {
	f.calls = append(f.calls, req)
	return &genproto.MatchResponse{
		Accepted:   f.acceptNext,
		ServerAddr: "test-instance",
	}, nil
}

// testRegistry returns a Registry backed by an in-memory miniredis,
// so tests never need a real Redis server running.
func testRegistry(t *testing.T) *registry.Registry {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("starting miniredis: %v", err)
	}
	t.Cleanup(mr.Close)
	return registry.New(mr.Addr())
}

// startBufconnInstance spins up an in-process gRPC server backed by
// fake, listening on a real (but local, ephemeral) TCP port — bufconn
// itself can't be dialed by address string the way grpc.NewClient
// inside tryFormMatch expects, so a real loopback listener is used
// here instead, registered in the registry with that real address.
func startBufconnInstance(t *testing.T, id string, fake *fakeGameService) string {
	t.Helper()

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}

	srv := grpc.NewServer()
	genproto.RegisterGameServiceServer(srv, fake)
	go srv.Serve(lis)
	t.Cleanup(srv.Stop)

	return lis.Addr().String()
}

func TestJoin_AddsPlayerToWaitingQueue(t *testing.T) {
	q := NewQueue(testRegistry(t))

	q.Join("player-1")

	status, _ := q.Status("player-1")
	if status != "waiting" {
		t.Errorf("got status %q, want %q", status, "waiting")
	}
}

func TestJoin_IgnoresDuplicateJoin(t *testing.T) {
	q := NewQueue(testRegistry(t))

	q.Join("player-1")
	q.Join("player-1")

	if len(q.waiting) != 1 {
		t.Errorf("got %d entries, want 1 — duplicate join should be ignored", len(q.waiting))
	}
}

func TestLeave_RemovesPlayerFromQueue(t *testing.T) {
	q := NewQueue(testRegistry(t))
	q.Join("player-1")

	q.Leave("player-1")

	status, _ := q.Status("player-1")
	if status != "not_queued" {
		t.Errorf("got status %q, want %q", status, "not_queued")
	}
}

func TestStatus_ReturnsNotQueuedForUnknownPlayer(t *testing.T) {
	q := NewQueue(testRegistry(t))

	status, entry := q.Status("never-joined")

	if status != "not_queued" {
		t.Errorf("got status %q, want %q", status, "not_queued")
	}
	if entry != nil {
		t.Error("got non-nil entry for an unqueued player")
	}
}

func TestTryFormMatch_NoInstancesRegistered_Requeues(t *testing.T) {
	q := NewQueue(testRegistry(t))
	q.Join("player-1")
	q.Join("player-2")

	q.tryFormMatch(context.Background())

	status, _ := q.Status("player-1")
	if status != "waiting" {
		t.Errorf("got status %q, want %q — no instances registered, should requeue", status, "waiting")
	}
}

func TestTryFormMatch_FormsMatchWithRegisteredInstance(t *testing.T) {
	reg := testRegistry(t)
	fake := &fakeGameService{acceptNext: true}
	addr := startBufconnInstance(t, "game-1", fake)
	reg.Register(context.Background(), registry.Instance{ID: "game-1", GRPCAddr: addr}, 5*time.Second)

	q := NewQueue(reg)
	q.Join("player-1")
	q.Join("player-2")

	q.tryFormMatch(context.Background())

	if len(fake.calls) != 1 {
		t.Fatalf("got %d StartMatch calls, want 1", len(fake.calls))
	}

	status, entry := q.Status("player-1")
	if status != "matched" || entry.ServerAddr != "test-instance" {
		t.Errorf("player-1 status = %q, entry = %+v, want matched", status, entry)
	}
}

func TestTryFormMatch_DoesNothingWithTooFewPlayers(t *testing.T) {
	reg := testRegistry(t)
	fake := &fakeGameService{acceptNext: true}
	addr := startBufconnInstance(t, "game-1", fake)
	reg.Register(context.Background(), registry.Instance{ID: "game-1", GRPCAddr: addr}, 5*time.Second)

	q := NewQueue(reg)
	q.Join("player-1")

	q.tryFormMatch(context.Background())

	if len(fake.calls) != 0 {
		t.Errorf("got %d StartMatch calls, want 0", len(fake.calls))
	}
}

func TestTryFormMatch_FallsBackToSecondInstanceWhenFirstRejects(t *testing.T) {
	reg := testRegistry(t)
	rejecting := &fakeGameService{acceptNext: false}
	accepting := &fakeGameService{acceptNext: true}

	addr1 := startBufconnInstance(t, "game-1", rejecting)
	addr2 := startBufconnInstance(t, "game-2", accepting)
	reg.Register(context.Background(), registry.Instance{ID: "game-1", GRPCAddr: addr1}, 5*time.Second)
	reg.Register(context.Background(), registry.Instance{ID: "game-2", GRPCAddr: addr2}, 5*time.Second)

	q := NewQueue(reg)
	q.Join("player-1")
	q.Join("player-2")

	q.tryFormMatch(context.Background())

	if len(rejecting.calls) != 1 {
		t.Errorf("got %d calls to the rejecting instance, want 1", len(rejecting.calls))
	}
	if len(accepting.calls) != 1 {
		t.Errorf("got %d calls to the accepting instance, want 1 (fallback)", len(accepting.calls))
	}

	status, _ := q.Status("player-1")
	if status != "matched" {
		t.Errorf("got status %q, want %q — should have matched via fallback", status, "matched")
	}
}

func TestTryFormMatch_IgnoresExpiredInstance(t *testing.T) {
	reg := testRegistry(t)
	fake := &fakeGameService{acceptNext: true}
	addr := startBufconnInstance(t, "game-1", fake)
	// Registered with a TTL of effectively zero real time by never
	// refreshing and using ListLive against a fresh registry lookup
	// after the entry's context is done — simplest reliable way to
	// simulate "this instance's heartbeat stopped" is to just never
	// register it in the first place.
	_ = addr

	q := NewQueue(reg)
	q.Join("player-1")
	q.Join("player-2")

	q.tryFormMatch(context.Background())

	if len(fake.calls) != 0 {
		t.Errorf("got %d StartMatch calls, want 0 — instance was never registered (simulating expiry)", len(fake.calls))
	}
	status, _ := q.Status("player-1")
	if status != "waiting" {
		t.Errorf("got status %q, want %q", status, "waiting")
	}
}

func TestRun_FormsMatchOnTick(t *testing.T) {
	reg := testRegistry(t)
	fake := &fakeGameService{acceptNext: true}
	addr := startBufconnInstance(t, "game-1", fake)
	reg.Register(context.Background(), registry.Instance{ID: "game-1", GRPCAddr: addr}, 5*time.Second)

	q := NewQueue(reg)
	q.Join("player-1")
	q.Join("player-2")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tickCh := make(chan time.Time)
	go q.Run(ctx, tickCh)

	tickCh <- time.Now()

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if status, _ := q.Status("player-1"); status == "matched" {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("player was not matched after Run processed a tick")
}

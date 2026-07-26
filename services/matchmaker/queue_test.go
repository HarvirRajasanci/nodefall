package main

import (
	"context"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	"nodefall/shared/genproto"
)

// fakeGameService is a minimal genproto.GameServiceServer for testing,
// recording every StartMatch call it receives without needing a real
// game service running.
type fakeGameService struct {
	genproto.UnimplementedGameServiceServer
	calls      []*genproto.MatchRequest
	acceptNext bool
}

func (f *fakeGameService) StartMatch(ctx context.Context, req *genproto.MatchRequest) (*genproto.MatchResponse, error) {
	f.calls = append(f.calls, req)
	return &genproto.MatchResponse{
		Accepted:   f.acceptNext,
		ServerAddr: "test-server:8081",
	}, nil
}

// newTestQueue starts an in-process gRPC server backed by fake, and
// returns a Queue connected to it over an in-memory bufconn listener —
// no real network port needed.
func newTestQueue(t *testing.T, fake *fakeGameService) *Queue {
	t.Helper()

	lis := bufconn.Listen(1024 * 1024)
	srv := grpc.NewServer()
	genproto.RegisterGameServiceServer(srv, fake)

	go srv.Serve(lis)
	t.Cleanup(srv.Stop)

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("dialing test server: %v", err)
	}

	return &Queue{
		matched:    make(map[string]*QueueEntry),
		gameClient: genproto.NewGameServiceClient(conn),
	}
}

func TestJoin_AddsPlayerToWaitingQueue(t *testing.T) {
	q := newTestQueue(t, &fakeGameService{})

	q.Join("player-1")

	status, _ := q.Status("player-1")
	if status != "waiting" {
		t.Errorf("got status %q, want %q", status, "waiting")
	}
}

func TestJoin_IgnoresDuplicateJoin(t *testing.T) {
	q := newTestQueue(t, &fakeGameService{})

	q.Join("player-1")
	q.Join("player-1")

	if len(q.waiting) != 1 {
		t.Errorf("got %d entries in queue, want 1 — duplicate join should be ignored", len(q.waiting))
	}
}

func TestLeave_RemovesPlayerFromQueue(t *testing.T) {
	q := newTestQueue(t, &fakeGameService{})
	q.Join("player-1")

	q.Leave("player-1")

	status, _ := q.Status("player-1")
	if status != "not_queued" {
		t.Errorf("got status %q, want %q", status, "not_queued")
	}
}

func TestStatus_ReturnsNotQueuedForUnknownPlayer(t *testing.T) {
	q := newTestQueue(t, &fakeGameService{})

	status, entry := q.Status("never-joined")

	if status != "not_queued" {
		t.Errorf("got status %q, want %q", status, "not_queued")
	}
	if entry != nil {
		t.Error("got non-nil entry for an unqueued player")
	}
}

func TestTryFormMatch_FormsMatchOnceEnoughPlayersWaiting(t *testing.T) {
	fake := &fakeGameService{acceptNext: true}
	q := newTestQueue(t, fake)

	q.Join("player-1")
	q.Join("player-2")

	q.tryFormMatch(context.Background())

	if len(fake.calls) != 1 {
		t.Fatalf("got %d StartMatch calls, want 1", len(fake.calls))
	}
	if len(fake.calls[0].PlayerIds) != 2 {
		t.Errorf("got %d players in match request, want 2", len(fake.calls[0].PlayerIds))
	}

	status1, entry1 := q.Status("player-1")
	if status1 != "matched" || entry1.ServerAddr != "test-server:8081" {
		t.Errorf("player-1 status = %q, entry = %+v, want matched with server addr", status1, entry1)
	}
}

func TestTryFormMatch_DoesNothingWithTooFewPlayers(t *testing.T) {
	fake := &fakeGameService{acceptNext: true}
	q := newTestQueue(t, fake)

	q.Join("player-1") // only 1, need MinPlayersPerMatch (2)

	q.tryFormMatch(context.Background())

	if len(fake.calls) != 0 {
		t.Errorf("got %d StartMatch calls, want 0 — not enough players waiting", len(fake.calls))
	}
}

func TestTryFormMatch_RequeuesPlayersOnRejection(t *testing.T) {
	fake := &fakeGameService{acceptNext: false} // game service rejects
	q := newTestQueue(t, fake)

	q.Join("player-1")
	q.Join("player-2")

	q.tryFormMatch(context.Background())

	status1, _ := q.Status("player-1")
	if status1 != "waiting" {
		t.Errorf("got status %q, want %q — rejected players should be requeued", status1, "waiting")
	}
}

func TestRun_FormsMatchOnTick(t *testing.T) {
	fake := &fakeGameService{acceptNext: true}
	q := newTestQueue(t, fake)
	q.Join("player-1")
	q.Join("player-2")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tickCh := make(chan time.Time)
	go q.Run(ctx, tickCh)

	tickCh <- time.Now() // manually trigger one matching pass

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if status, _ := q.Status("player-1"); status == "matched" {
			return // success
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("player was not matched after Run processed a tick")
}

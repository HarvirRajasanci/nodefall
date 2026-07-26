package matchmanager

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestCreateMatch_RegistersUnderGivenID(t *testing.T) {
	m := New(context.Background())

	m.CreateMatch("match-1", []string{"player-1"})

	e, ok := m.Get("match-1")
	if !ok {
		t.Fatal("Get returned ok=false for a match ID that was just created")
	}
	if e == nil {
		t.Error("got nil engine for an existing match ID")
	}
}

func TestGet_ReturnsFalseForUnknownMatchID(t *testing.T) {
	m := New(context.Background())

	_, ok := m.Get("nonexistent-match-id")

	if ok {
		t.Error("got ok=true for a match ID that was never created")
	}
}

func TestCreateMatch_EngineOnlyAllowsAssignedPlayers(t *testing.T) {
	m := New(context.Background())

	m.CreateMatch("match-1", []string{"player-1", "player-2"})
	e, _ := m.Get("match-1")

	if ok := e.Join(&fakeClient{id: "player-1"}); !ok {
		t.Error("assigned player was rejected")
	}
	if ok := e.Join(&fakeClient{id: "stranger"}); ok {
		t.Error("unassigned player was allowed to join")
	}
}

func TestCreateMatch_StartsRunningTickLoop(t *testing.T) {
	m := New(context.Background())
	m.CreateMatch("match-1", []string{"player-1"})
	e, _ := m.Get("match-1")

	client := &fakeClient{id: "player-1"}
	e.Join(client)

	time.Sleep(100 * time.Millisecond) // world.TickRate is 50ms

	if client.msgCount() == 0 {
		t.Error("no messages received — engine's Run() loop does not appear to be ticking")
	}
}

func TestRemove_DeletesMatchFromManager(t *testing.T) {
	m := New(context.Background())
	m.CreateMatch("match-1", []string{"player-1"})

	m.Remove("match-1")

	_, ok := m.Get("match-1")
	if ok {
		t.Error("match still retrievable after Remove")
	}
}

// fakeClient is a minimal engine.Client for testing. It's accessed
// from two goroutines at once here — the engine's own Run() loop
// calls Send, while the test's main goroutine reads the message
// count after sleeping — so it needs its own mutex, unlike the
// simpler fakeClient in engine's own tests, which is only ever
// touched from the single test goroutine directly.
type fakeClient struct {
	mu   sync.Mutex
	id   string
	msgs [][]byte
}

func (f *fakeClient) ID() string { return f.id }

func (f *fakeClient) Send(msg []byte) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.msgs = append(f.msgs, msg)
}

func (f *fakeClient) msgCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.msgs)
}

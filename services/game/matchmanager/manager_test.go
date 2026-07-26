package matchmanager

import (
	"context"
	"testing"
	"time"
)

func TestCreateMatch_ReturnsUniqueIDs(t *testing.T) {
	m := New(context.Background())

	id1 := m.CreateMatch([]string{"player-1"})
	id2 := m.CreateMatch([]string{"player-2"})

	if id1 == id2 {
		t.Error("got the same match ID for two different CreateMatch calls, want unique IDs")
	}
}

func TestCreateMatch_RegistersRetrievableEngine(t *testing.T) {
	m := New(context.Background())

	matchID := m.CreateMatch([]string{"player-1"})

	e, ok := m.Get(matchID)
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

	matchID := m.CreateMatch([]string{"player-1", "player-2"})
	e, _ := m.Get(matchID)

	if ok := e.Join(&fakeClient{id: "player-1"}); !ok {
		t.Error("assigned player was rejected")
	}
	if ok := e.Join(&fakeClient{id: "stranger"}); ok {
		t.Error("unassigned player was allowed to join")
	}
}

func TestCreateMatch_StartsRunningTickLoop(t *testing.T) {
	// Confirms the engine's Run() loop is actually ticking on its own
	// goroutine — join a player, wait slightly longer than one real
	// tick, and confirm a snapshot was broadcast without ever calling
	// Tick() manually.
	m := New(context.Background())
	matchID := m.CreateMatch([]string{"player-1"})
	e, _ := m.Get(matchID)

	client := &fakeClient{id: "player-1"}
	e.Join(client)

	time.Sleep(100 * time.Millisecond) // world.TickRate is 50ms

	if len(client.msgs) == 0 {
		t.Error("no messages received — engine's Run() loop does not appear to be ticking")
	}
}

func TestRemove_DeletesMatchFromManager(t *testing.T) {
	m := New(context.Background())
	matchID := m.CreateMatch([]string{"player-1"})

	m.Remove(matchID)

	_, ok := m.Get(matchID)
	if ok {
		t.Error("match still retrievable after Remove")
	}
}

// fakeClient is a minimal engine.Client for testing, duplicated here
// (rather than imported) since engine's fakeClient is unexported and
// test-only within its own package.
type fakeClient struct {
	id   string
	msgs [][]byte
}

func (f *fakeClient) ID() string      { return f.id }
func (f *fakeClient) Send(msg []byte) { f.msgs = append(f.msgs, msg) }

package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"

	"nodefall/services/game/engine"
	"nodefall/shared/jwt"
	"nodefall/shared/middleware"
)

// fakeEngine records every call it receives, so tests can assert on
// server behavior without depending on real game logic.
type fakeEngine struct {
	mu     sync.Mutex
	joined []engine.Client
	left   []engine.Client
	inputs []engine.Input
}

func (f *fakeEngine) Join(c engine.Client) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.joined = append(f.joined, c)
}

func (f *fakeEngine) Leave(c engine.Client) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.left = append(f.left, c)
}

func (f *fakeEngine) HandleInput(c engine.Client, input engine.Input) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.inputs = append(f.inputs, input)
}

func (f *fakeEngine) joinCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.joined)
}

func (f *fakeEngine) leaveCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.left)
}

func (f *fakeEngine) inputCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.inputs)
}

func (f *fakeEngine) lastInput() engine.Input {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.inputs[len(f.inputs)-1]
}

// newTestServer wires a Server (wrapped with middleware.WithAuth) to a
// fake engine behind a real HTTP server, so a real WebSocket client can
// dial in over the network with a real signed JWT.
func newTestServer(t *testing.T, e *fakeEngine) (*httptest.Server, string) {
	t.Helper()

	os.Setenv("NODEFALL_JWT_SECRET", "test-secret-do-not-use-in-prod")

	srv := New(e)
	mux := http.NewServeMux()
	mux.Handle("/ws", middleware.WithAuth(http.HandlerFunc(srv.ServeWS)))

	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)

	token, err := jwt.Sign("player-1")
	if err != nil {
		t.Fatalf("Sign failed: %v", err)
	}

	wsURL := "ws" + ts.URL[len("http"):] + "/ws?token=" + token
	return ts, wsURL
}

// waitFor polls cond until it's true or the timeout elapses, failing
// the test if it never becomes true. Needed because Join/Leave/
// HandleInput happen on goroutines the test doesn't directly control.
func waitFor(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("condition not met before timeout")
}

func TestServeWS_JoinsEngineOnConnect(t *testing.T) {
	e := &fakeEngine{}
	_, wsURL := newTestServer(t, e)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	waitFor(t, time.Second, func() bool { return e.joinCount() == 1 })
}

func TestServeWS_LeavesEngineOnDisconnect(t *testing.T) {
	e := &fakeEngine{}
	_, wsURL := newTestServer(t, e)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}

	waitFor(t, time.Second, func() bool { return e.joinCount() == 1 })

	conn.Close(websocket.StatusNormalClosure, "done")

	waitFor(t, time.Second, func() bool { return e.leaveCount() == 1 })
}

func TestServeWS_ForwardsValidInputToEngine(t *testing.T) {
	e := &fakeEngine{}
	_, wsURL := newTestServer(t, e)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	waitFor(t, time.Second, func() bool { return e.joinCount() == 1 })

	input := engine.Input{DX: 1, DY: 0, Angle: 1.5, Shoot: true}
	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	if err := conn.Write(ctx, websocket.MessageText, data); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	waitFor(t, time.Second, func() bool { return e.inputCount() == 1 })

	got := e.lastInput()
	if got.DX != 1 || got.Angle != 1.5 || !got.Shoot {
		t.Errorf("got input %+v, want %+v", got, input)
	}
}

func TestServeWS_IgnoresMalformedInput(t *testing.T) {
	e := &fakeEngine{}
	_, wsURL := newTestServer(t, e)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	waitFor(t, time.Second, func() bool { return e.joinCount() == 1 })

	if err := conn.Write(ctx, websocket.MessageText, []byte("not json")); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	valid := engine.Input{Shoot: true}
	data, _ := json.Marshal(valid)
	if err := conn.Write(ctx, websocket.MessageText, data); err != nil {
		t.Fatalf("connection appears closed after malformed input: %v", err)
	}

	waitFor(t, time.Second, func() bool { return e.inputCount() == 1 })
}

func TestClient_Send_DropsWhenBufferFull(t *testing.T) {
	client := &Client{id: "player-1", send: make(chan []byte, 2)}

	client.Send([]byte("one"))
	client.Send([]byte("two"))
	client.Send([]byte("three"))

	if len(client.send) != 2 {
		t.Errorf("got %d buffered messages, want 2", len(client.send))
	}
}

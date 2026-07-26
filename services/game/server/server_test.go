package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/coder/websocket"

	"nodefall/services/game/matchmanager"
	"nodefall/shared/jwt"
	"nodefall/shared/middleware"
)

// newTestServer wires a real Server + real matchmanager.Manager behind
// a real HTTP server. Both are pure in-memory Go packages with no
// external dependencies, so using the real things here (rather than
// fakes) is cheap and gives stronger coverage than mocking would.
func newTestServer(t *testing.T) (*httptest.Server, *matchmanager.Manager) {
	t.Helper()

	os.Setenv("NODEFALL_JWT_SECRET", "test-secret-do-not-use-in-prod")

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	matches := matchmanager.New(ctx)
	srv := New(matches)
	mux := http.NewServeMux()
	mux.Handle("/ws", middleware.WithAuth(http.HandlerFunc(srv.ServeWS)))

	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)

	return ts, matches
}

func wsURL(t *testing.T, ts *httptest.Server, matchID, playerID string) string {
	t.Helper()
	token, err := jwt.Sign(playerID)
	if err != nil {
		t.Fatalf("Sign failed: %v", err)
	}
	return "ws" + ts.URL[len("http"):] + "/ws?match=" + matchID + "&token=" + token
}

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

func TestServeWS_JoinsAssignedPlayerToMatch(t *testing.T) {
	ts, matches := newTestServer(t)
	matches.CreateMatch("match-1", []string{"player-1"})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, wsURL(t, ts, "match-1", "player-1"), nil)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	e, _ := matches.Get("match-1")
	waitFor(t, time.Second, func() bool { return e.PlayerCount() == 1 })
}

func TestServeWS_RejectsPlayerNotInMatch(t *testing.T) {
	ts, matches := newTestServer(t)
	matches.CreateMatch("match-2", []string{"player-1"})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, wsURL(t, ts, "match-2", "stranger"), nil)
	if err != nil {
		// A dial-level failure is also an acceptable rejection outcome.
		return
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	if _, _, err := conn.Read(ctx); err == nil {
		t.Error("expected the connection to be closed for a player not in this match")
	}
}

func TestServeWS_RejectsUnknownMatchID(t *testing.T) {
	ts, _ := newTestServer(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, _, err := websocket.Dial(ctx, wsURL(t, ts, "nonexistent-match", "player-1"), nil)
	if err == nil {
		t.Error("expected Dial to fail for an unknown match ID")
	}
}

func TestServeWS_RejectsMissingMatchID(t *testing.T) {
	ts, _ := newTestServer(t)

	token, err := jwt.Sign("player-1")
	if err != nil {
		t.Fatalf("Sign failed: %v", err)
	}
	url := "ws" + ts.URL[len("http"):] + "/ws?token=" + token // no "match" param at all

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, _, err = websocket.Dial(ctx, url, nil)
	if err == nil {
		t.Error("expected Dial to fail when no match id is given")
	}
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

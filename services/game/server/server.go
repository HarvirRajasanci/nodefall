package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/coder/websocket"

	"nodefall/services/game/engine"
	"nodefall/shared/middleware"
)

// MatchManager is the subset of matchmanager.Manager this package
// depends on: looking up an already-created match by ID. Server never
// creates matches itself — that happens via the gRPC StartMatch
// handler in grpc.go.
type MatchManager interface {
	Get(matchID string) (*engine.Engine, bool)
}

// Server upgrades HTTP connections to WebSockets, looks up the
// requested match, and wires the connection to that match's engine.
// It knows nothing about game rules — that lives entirely in
// engine/world.
type Server struct {
	matches MatchManager
}

// New creates a Server that routes connections through the given
// match manager.
func New(m MatchManager) *Server {
	return &Server{matches: m}
}

// ServeWS is the HTTP handler that upgrades a connection and blocks
// for the lifetime of that connection. Must be wrapped with
// middleware.WithAuth — the player ID comes from the verified JWT.
// The target match is given via the "match" query parameter.
func (s *Server) ServeWS(w http.ResponseWriter, r *http.Request) {
	playerID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "missing verified identity", http.StatusUnauthorized)
		return
	}

	matchID := r.URL.Query().Get("match")
	if matchID == "" {
		http.Error(w, "missing match id", http.StatusBadRequest)
		return
	}

	e, ok := s.matches.Get(matchID)
	if !ok {
		http.Error(w, "match not found", http.StatusNotFound)
		return
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		// TODO: restrict to the real frontend origin before shipping.
		InsecureSkipVerify: true,
	})
	if err != nil {
		log.Printf("websocket accept: %v", err)
		return
	}

	client := &Client{
		id:   playerID,
		conn: conn,
		send: make(chan []byte, 16),
	}

	if !e.Join(client) {
		conn.Close(websocket.StatusPolicyViolation, "not part of this match")
		return
	}
	defer e.Leave(client)

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		client.writePump(ctx, cancel)
	}()
	go func() {
		defer wg.Done()
		s.readPump(ctx, cancel, client, e)
	}()
	wg.Wait()

	conn.Close(websocket.StatusNormalClosure, "connection closed")
}

// readPump reads and decodes client input, forwarding it to e. Runs
// until the context is cancelled or the read fails.
func (s *Server) readPump(ctx context.Context, cancel context.CancelFunc, client *Client, e *engine.Engine) {
	defer cancel()
	for {
		_, data, err := client.conn.Read(ctx)
		if err != nil {
			return
		}

		var input engine.Input
		if err := json.Unmarshal(data, &input); err != nil {
			log.Printf("client %s: malformed input: %v", client.id, err)
			continue
		}

		e.HandleInput(client, input)
	}
}

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

// Engine is the subset of the game engine this package depends on.
// Parameters use engine.Client (an interface engine already defines),
// so any concrete *engine.Engine satisfies this without server needing
// to import anything from engine beyond its two small public types.
type Engine interface {
	Join(client engine.Client)
	Leave(client engine.Client)
	HandleInput(client engine.Client, input engine.Input)
}

// Server upgrades HTTP connections to WebSockets and wires each one
// to the engine as a Client. It knows nothing about game rules —
// that lives entirely in engine/world.
type Server struct {
	engine Engine
}

// New creates a Server that forwards connections to the given engine.
func New(e Engine) *Server {
	return &Server{engine: e}
}

// ServeWS is the HTTP handler that upgrades a connection and blocks
// for the lifetime of that connection. Must be wrapped with
// middleware.WithAuth — the player ID comes from the verified JWT,
// never from the request directly.
func (s *Server) ServeWS(w http.ResponseWriter, r *http.Request) {
	playerID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "missing verified identity", http.StatusUnauthorized)
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

	s.engine.Join(client)
	defer s.engine.Leave(client)

	// Connection lifetime is tied to this context — closing it (client
	// disconnect, server shutdown, or an explicit call) tears down both
	// pumps below.
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
		s.readPump(ctx, cancel, client)
	}()
	wg.Wait()

	conn.Close(websocket.StatusNormalClosure, "connection closed")
}

// readPump reads and decodes client input, forwarding it to the engine.
// Runs until the context is cancelled or the read fails.
func (s *Server) readPump(ctx context.Context, cancel context.CancelFunc, client *Client) {
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

		s.engine.HandleInput(client, input)
	}
}

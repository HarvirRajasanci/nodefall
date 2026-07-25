package server

import (
	"context"
	"log"
	"time"

	"github.com/coder/websocket"
)

// Client wraps one player's WebSocket connection. It satisfies
// engine.Client (ID() string, Send([]byte)) without engine ever
// needing to import this package.
type Client struct {
	id   string
	conn *websocket.Conn
	send chan []byte
}

// ID returns the player ID this connection belongs to.
func (c *Client) ID() string { return c.id }

// Send queues a message for delivery without blocking the engine's
// tick loop. If the client is too far behind, the message is dropped
// rather than blocking the broadcast to every other player.
func (c *Client) Send(msg []byte) {
	select {
	case c.send <- msg:
	default:
		log.Printf("client %s: send buffer full, dropping message", c.id)
	}
}

// writePump owns all writes to this connection — coder/websocket
// requires a single writer at a time per connection, so every
// outbound message funnels through this one goroutine.
func (c *Client) writePump(ctx context.Context, cancel context.CancelFunc) {
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-c.send:
			writeCtx, cancelWrite := context.WithTimeout(ctx, 5*time.Second)
			err := c.conn.Write(writeCtx, websocket.MessageText, msg)
			cancelWrite()
			if err != nil {
				return
			}
		}
	}
}

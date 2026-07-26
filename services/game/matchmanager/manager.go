// Package matchmanager holds and manages multiple concurrent matches
// (each its own *engine.Engine), keyed by match ID. This is what makes
// it possible for the game server to run more than one match at a time
// rather than a single perpetual world.
package matchmanager

import (
	"context"
	"sync"

	"github.com/google/uuid"

	"nodefall/services/game/engine"
)

// Manager holds every currently active match.
type Manager struct {
	mu      sync.RWMutex
	matches map[string]*engine.Engine
	ctx     context.Context
}

// New creates an empty Manager. ctx governs the lifetime of every
// match's tick loop — cancelling it stops all matches, e.g. on server
// shutdown.
func New(ctx context.Context) *Manager {
	return &Manager{
		matches: make(map[string]*engine.Engine),
		ctx:     ctx,
	}
}

// CreateMatch creates a new engine restricted to playerIDs, starts its
// tick loop on its own goroutine, registers it under a fresh match ID,
// and returns that ID.
func (m *Manager) CreateMatch(playerIDs []string) string {
	m.mu.Lock()
	defer m.mu.Unlock()

	matchID := uuid.New().String()
	e := engine.NewForPlayers(playerIDs)

	m.matches[matchID] = e
	go e.Run(m.ctx)

	return matchID
}

// Get returns the engine for the given match ID, and whether it exists.
func (m *Manager) Get(matchID string) (*engine.Engine, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	e, ok := m.matches[matchID]
	return e, ok
}

// Remove deletes a match from the manager. The match's own tick loop
// keeps running until ctx is cancelled — Remove only stops the manager
// from routing new connections to it. In practice this project doesn't
// currently call Remove anywhere; matches simply persist for the life
// of the server. Provided for completeness and future cleanup logic
// (e.g. removing matches that have had zero players for some time).
func (m *Manager) Remove(matchID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.matches, matchID)
}

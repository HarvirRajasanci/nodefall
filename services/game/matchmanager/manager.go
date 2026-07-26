package matchmanager

import (
	"context"
	"sync"

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
// tick loop on its own goroutine, and registers it under matchID. The
// caller (the gRPC StartMatch handler) supplies matchID rather than
// this generating its own, since whoever calls StartMatch — the
// matchmaker — needs to already know the ID to hand to players.
func (m *Manager) CreateMatch(matchID string, playerIDs []string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	e := engine.NewForPlayers(playerIDs)
	m.matches[matchID] = e
	go e.Run(m.ctx)
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
// from routing new connections to it. Not currently called anywhere;
// provided for future idle-match cleanup.
func (m *Manager) Remove(matchID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.matches, matchID)
}

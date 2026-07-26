package engine

import (
	"context"
	"encoding/json"
	"math/rand"
	"sync"
	"time"

	"nodefall/services/game/world"
)

// Client is the minimal capability the engine needs from a connected
// player's transport layer: an identity and a way to receive messages.
// Defined here, not imported from the server package, so the engine has
// zero networking knowledge and can be driven directly in tests — any
// type with these two methods (including server.Client) satisfies this.
type Client interface {
	ID() string
	Send(msg []byte)
}

// Input is the decoded shape of one player action sent from the client.
type Input struct {
	DX    float64 `json:"dx"`
	DY    float64 `json:"dy"`
	Angle float64 `json:"angle"`
	Shoot bool    `json:"shoot"`
}

// Phase describes whether the match is still waiting for players
// (no simulation runs, no input is applied) or live.
type Phase string

const (
	PhaseWaiting Phase = "waiting"
	PhaseLive    Phase = "live"
)

// Snapshot is the per-tick state broadcast to every connected client.
type Snapshot struct {
	Phase            Phase           `json:"phase"`
	CountdownSeconds int             `json:"countdown_seconds"`
	Players          []*world.Player `json:"players"`
	Bullets          []*world.Bullet `json:"bullets"`
	Items            []*world.Item   `json:"items"`
	Zone             *world.Zone     `json:"zone"`
}

// Engine holds the live state for a single match and advances it one
// fixed tick at a time. Safe for concurrent use — HandleInput is called
// from each connection's read pump while Run's tick loop runs
// concurrently.
type Engine struct {
	mu      sync.Mutex
	players map[string]*world.Player
	clients map[string]Client
	bullets []*world.Bullet
	items   []*world.Item
	zone    *world.Zone

	respawnAt map[string]time.Time // playerID -> when they respawn

	phase       Phase
	waitStarted time.Time // zero until the first player joins
}

// New creates an engine with a fresh zone and item spawn, ready to
// accept players. Starts in PhaseWaiting — no simulation runs and no
// input is applied until enough real time has passed after the first
// player joins (world.MatchStartDelay).
func New() *Engine {
	return &Engine{
		players:   make(map[string]*world.Player),
		clients:   make(map[string]Client),
		items:     world.SpawnItems(world.ItemCount),
		zone:      world.NewZone(),
		respawnAt: make(map[string]time.Time),
		phase:     PhaseWaiting,
	}
}

// Join adds a new player to the match at a random spawn position and
// registers their client for future broadcasts. If this is the first
// player to join a waiting match, starts the countdown to PhaseLive.
func (e *Engine) Join(client Client) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.phase == PhaseWaiting && len(e.players) == 0 && e.waitStarted.IsZero() {
		e.waitStarted = time.Now()
	}

	x, y := randomSpawn()
	e.players[client.ID()] = world.NewPlayer(client.ID(), x, y)
	e.clients[client.ID()] = client
}

// Leave removes a player and their client from the match.
func (e *Engine) Leave(client Client) {
	e.mu.Lock()
	defer e.mu.Unlock()

	delete(e.players, client.ID())
	delete(e.clients, client.ID())
	delete(e.respawnAt, client.ID())
}

// HandleInput applies one decoded player action: movement, aim angle,
// and an optional shot. Ignored if the client is unknown, the player
// is currently dead, or the match hasn't gone live yet.
func (e *Engine) HandleInput(client Client, input Input) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.phase != PhaseLive {
		return
	}

	player, ok := e.players[client.ID()]
	if !ok || !player.Alive {
		return
	}

	player.Move(input.DX, input.DY)
	player.Angle = input.Angle

	if input.Shoot && player.CanShoot(world.FireRate) {
		player.RecordShot()
		bullet := world.NewBullet(player.ID, player.X, player.Y, input.Angle, world.BulletDamage)
		e.bullets = append(e.bullets, bullet)
	}
}

// Run starts the fixed tick loop and blocks until ctx is cancelled.
func (e *Engine) Run(ctx context.Context) {
	ticker := time.NewTicker(world.TickRate)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			e.Tick()
		}
	}
}

// Tick advances the match by one fixed timestep. While PhaseWaiting,
// only the phase transition and broadcast happen — no bullets move,
// no zone damage applies, no respawns process. Exported so tests can
// drive the engine tick-by-tick without waiting on a real ticker.
func (e *Engine) Tick() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.phase == PhaseWaiting {
		if !e.waitStarted.IsZero() && time.Since(e.waitStarted) >= world.MatchStartDelay {
			e.phase = PhaseLive
		}
		e.broadcast()
		return
	}

	e.moveBullets()
	e.checkBulletCollisions()
	e.checkItemPickups()
	e.zone.Shrink()
	e.applyZoneDamage()
	e.processRespawns()
	e.broadcast()
}

// moveBullets advances every bullet and drops any that expired this tick.
func (e *Engine) moveBullets() {
	alive := e.bullets[:0]
	for _, bullet := range e.bullets {
		if bullet.Move() {
			alive = append(alive, bullet)
		}
	}
	e.bullets = alive
}

// checkBulletCollisions applies damage for any bullet that hit a player,
// removing the bullet and scheduling a respawn if the hit was lethal.
func (e *Engine) checkBulletCollisions() {
	remaining := e.bullets[:0]
	for _, bullet := range e.bullets {
		hit := false
		for _, player := range e.players {
			if bullet.Hits(player) {
				if player.ApplyDamage(bullet.Damage) {
					e.respawnAt[player.ID] = time.Now().Add(world.RespawnDelay)
				}
				hit = true
				break
			}
		}
		if !hit {
			remaining = append(remaining, bullet)
		}
	}
	e.bullets = remaining
}

// checkItemPickups applies any item a player is standing on and removes
// it from the world.
func (e *Engine) checkItemPickups() {
	remaining := e.items[:0]
	for _, item := range e.items {
		picked := false
		for _, player := range e.players {
			if item.CollidesWith(player) {
				player.ApplyPickup(item.Type)
				picked = true
				break
			}
		}
		if !picked {
			remaining = append(remaining, item)
		}
	}
	e.items = remaining
}

// applyZoneDamage damages every player outside the zone, scheduling a
// respawn for anyone it kills.
func (e *Engine) applyZoneDamage() {
	for _, player := range e.players {
		if e.zone.DamageIfOutside(player) {
			e.respawnAt[player.ID] = time.Now().Add(world.RespawnDelay)
		}
	}
}

// processRespawns brings back any player whose respawn timer has elapsed.
func (e *Engine) processRespawns() {
	for id, at := range e.respawnAt {
		if time.Now().Before(at) {
			continue
		}
		x, y := randomSpawn()
		e.players[id].Respawn(x, y)
		delete(e.respawnAt, id)
	}
}

// broadcast sends the current match state to every connected client.
func (e *Engine) broadcast() {
	data, err := json.Marshal(e.snapshot())
	if err != nil {
		return
	}
	for _, client := range e.clients {
		client.Send(data)
	}
}

// snapshot builds the current state to broadcast. Caller must hold e.mu.
func (e *Engine) snapshot() Snapshot {
	players := make([]*world.Player, 0, len(e.players))
	for _, p := range e.players {
		players = append(players, p)
	}

	countdown := 0
	if e.phase == PhaseWaiting && !e.waitStarted.IsZero() {
		remaining := world.MatchStartDelay - time.Since(e.waitStarted)
		if remaining < 0 {
			remaining = 0
		}
		countdown = int(remaining/time.Second) + 1
	}

	return Snapshot{
		Phase:            e.phase,
		CountdownSeconds: countdown,
		Players:          players,
		Bullets:          e.bullets,
		Items:            e.items,
		Zone:             e.zone,
	}
}

// randomSpawn picks a random position within the map bounds, using the
// same edge-clamping margin as world.Player's movement clamping.
func randomSpawn() (float64, float64) {
	x := world.PlayerRadius + rand.Float64()*(world.MapSize-2*world.PlayerRadius)
	y := world.PlayerRadius + rand.Float64()*(world.MapSize-2*world.PlayerRadius)
	return x, y
}

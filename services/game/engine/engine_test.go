package engine

import (
	"math"
	"testing"
	"time"

	"nodefall/services/game/world"
)

// fakeClient is a minimal Client for testing, with no real network
// connection — it just records every message it's sent.
type fakeClient struct {
	id   string
	msgs [][]byte
}

func (f *fakeClient) ID() string      { return f.id }
func (f *fakeClient) Send(msg []byte) { f.msgs = append(f.msgs, msg) }

// makeLive bypasses the real MatchStartDelay wait so tests that need
// immediate simulation don't have to sleep for real time.
func makeLive(e *Engine) {
	e.phase = PhaseLive
}

func TestNew_CreatesEngineWithItemsAndZone(t *testing.T) {
	e := New()

	if len(e.items) != world.ItemCount {
		t.Errorf("got %d items, want %d", len(e.items), world.ItemCount)
	}
	if e.zone == nil {
		t.Error("got nil zone, want initialized zone")
	}
	if e.phase != PhaseWaiting {
		t.Errorf("got phase %v, want %v", e.phase, PhaseWaiting)
	}
}

func TestJoin_AddsPlayerAndClient(t *testing.T) {
	e := New()
	client := &fakeClient{id: "player-1"}

	e.Join(client)

	if _, ok := e.players["player-1"]; !ok {
		t.Error("player was not added after Join")
	}
	if _, ok := e.clients["player-1"]; !ok {
		t.Error("client was not registered after Join")
	}
}

func TestJoin_StartsWaitTimerOnFirstPlayer(t *testing.T) {
	e := New()
	client := &fakeClient{id: "player-1"}

	e.Join(client)

	if e.waitStarted.IsZero() {
		t.Error("waitStarted was not set after first player joined")
	}
}

func TestJoin_DoesNotResetWaitTimerForLateJoiners(t *testing.T) {
	e := New()
	e.Join(&fakeClient{id: "player-1"})
	first := e.waitStarted

	e.Join(&fakeClient{id: "player-2"})

	if e.waitStarted != first {
		t.Error("waitStarted was reset by a second player joining, want unchanged")
	}
}

func TestLeave_RemovesPlayerAndClient(t *testing.T) {
	e := New()
	client := &fakeClient{id: "player-1"}
	e.Join(client)

	e.Leave(client)

	if _, ok := e.players["player-1"]; ok {
		t.Error("player still present after Leave")
	}
	if _, ok := e.clients["player-1"]; ok {
		t.Error("client still registered after Leave")
	}
}

func TestTick_TransitionsToLiveAfterMatchStartDelay(t *testing.T) {
	e := New()
	e.Join(&fakeClient{id: "player-1"})
	e.waitStarted = time.Now().Add(-world.MatchStartDelay - time.Second)

	e.Tick()

	if e.phase != PhaseLive {
		t.Errorf("got phase %v, want %v", e.phase, PhaseLive)
	}
}

func TestTick_StaysWaitingBeforeMatchStartDelay(t *testing.T) {
	e := New()
	e.Join(&fakeClient{id: "player-1"})
	e.waitStarted = time.Now()

	e.Tick()

	if e.phase != PhaseWaiting {
		t.Errorf("got phase %v, want %v", e.phase, PhaseWaiting)
	}
}

func TestHandleInput_IgnoredWhileWaiting(t *testing.T) {
	e := New()
	client := &fakeClient{id: "player-1"}
	e.Join(client)
	startX := e.players["player-1"].X

	e.HandleInput(client, Input{DX: 1, DY: 0})

	if e.players["player-1"].X != startX {
		t.Error("player moved during PhaseWaiting, want no effect")
	}
}

func TestHandleInput_MovesAlivePlayer(t *testing.T) {
	e := New()
	client := &fakeClient{id: "player-1"}
	e.Join(client)
	makeLive(e)
	startX := e.players["player-1"].X

	e.HandleInput(client, Input{DX: 1, DY: 0})

	if e.players["player-1"].X <= startX {
		t.Error("player did not move after HandleInput")
	}
}

func TestHandleInput_IgnoresUnknownClient(t *testing.T) {
	e := New()
	makeLive(e)
	unknown := &fakeClient{id: "ghost"}

	e.HandleInput(unknown, Input{DX: 1, DY: 0})

	if len(e.players) != 0 {
		t.Error("unknown client's input created a player")
	}
}

func TestHandleInput_IgnoresDeadPlayer(t *testing.T) {
	e := New()
	client := &fakeClient{id: "player-1"}
	e.Join(client)
	makeLive(e)
	e.players["player-1"].Alive = false
	startX := e.players["player-1"].X

	e.HandleInput(client, Input{DX: 1, DY: 0})

	if e.players["player-1"].X != startX {
		t.Error("dead player moved after HandleInput, want no effect")
	}
}

func TestHandleInput_ShootingCreatesBullet(t *testing.T) {
	e := New()
	client := &fakeClient{id: "player-1"}
	e.Join(client)
	makeLive(e)

	e.HandleInput(client, Input{Shoot: true})

	if len(e.bullets) != 1 {
		t.Fatalf("got %d bullets, want 1", len(e.bullets))
	}
	if e.bullets[0].OwnerID != "player-1" {
		t.Errorf("got bullet OwnerID %q, want %q", e.bullets[0].OwnerID, "player-1")
	}
}

func TestHandleInput_ShootingRespectsFireRate(t *testing.T) {
	e := New()
	client := &fakeClient{id: "player-1"}
	e.Join(client)
	makeLive(e)

	e.HandleInput(client, Input{Shoot: true})
	e.HandleInput(client, Input{Shoot: true})

	if len(e.bullets) != 1 {
		t.Errorf("got %d bullets, want 1 — second shot should be blocked by FireRate", len(e.bullets))
	}
}

func TestTick_BulletMovesAndCanExpire(t *testing.T) {
	e := New()
	client := &fakeClient{id: "player-1"}
	e.Join(client)
	// Add a second player so checkMatchEnd doesn't immediately end the
	// match with the lone player as winner before the bullet gets a
	// chance to move.
	e.Join(&fakeClient{id: "player-2"})
	makeLive(e)
	e.HandleInput(client, Input{Shoot: true})
	startX := e.bullets[0].X

	e.Tick()

	if len(e.bullets) == 1 && e.bullets[0].X == startX {
		t.Error("bullet did not move after Tick")
	}
}

func TestTick_BulletHitsAndDamagesPlayer(t *testing.T) {
	e := New()
	attacker := &fakeClient{id: "attacker"}
	target := &fakeClient{id: "target"}
	e.Join(attacker)
	e.Join(target)
	makeLive(e)

	e.players["attacker"].X, e.players["attacker"].Y = 100, 100
	e.players["target"].X, e.players["target"].Y = 100, 300
	startHP := e.players["target"].HP

	e.HandleInput(attacker, Input{Angle: math.Pi / 2, Shoot: true})

	for i := 0; i < 30 && e.players["target"].HP == startHP; i++ {
		e.Tick()
	}

	if e.players["target"].HP >= startHP {
		t.Errorf("target HP %d, want less than starting %d — bullet should have hit", e.players["target"].HP, startHP)
	}
}

func TestCheckBulletCollisions_LethalHitSchedulesRespawn(t *testing.T) {
	e := New()
	client := &fakeClient{id: "player-1"}
	e.Join(client)
	e.Join(&fakeClient{id: "player-2"}) // keep 2 alive so checkMatchEnd doesn't end the match
	makeLive(e)
	e.players["player-1"].X, e.players["player-1"].Y = 500, 500

	lethal := world.NewBullet("someone-else", 500, 500, 0, 9999)
	e.bullets = append(e.bullets, lethal)

	e.checkBulletCollisions()

	if e.players["player-1"].Alive {
		t.Error("player still alive after a lethal hit")
	}
	if _, ok := e.respawnAt["player-1"]; !ok {
		t.Error("no respawn scheduled after a lethal hit")
	}
}

func TestProcessRespawns_RevivesPlayerAfterDelayElapses(t *testing.T) {
	e := New()
	client := &fakeClient{id: "player-1"}
	e.Join(client)
	makeLive(e)
	e.players["player-1"].Alive = false
	e.respawnAt["player-1"] = time.Now().Add(-time.Second)

	e.processRespawns()

	if !e.players["player-1"].Alive {
		t.Error("player not revived after respawn delay elapsed")
	}
	if _, ok := e.respawnAt["player-1"]; ok {
		t.Error("respawnAt entry not cleared after respawn")
	}
}

func TestProcessRespawns_DoesNothingBeforeDelayElapses(t *testing.T) {
	e := New()
	client := &fakeClient{id: "player-1"}
	e.Join(client)
	makeLive(e)
	e.players["player-1"].Alive = false
	e.respawnAt["player-1"] = time.Now().Add(time.Hour)

	e.processRespawns()

	if e.players["player-1"].Alive {
		t.Error("player revived before respawn delay elapsed")
	}
}

func TestCheckItemPickups_AppliesEffectAndRemovesItem(t *testing.T) {
	e := New()
	client := &fakeClient{id: "player-1"}
	e.Join(client)
	makeLive(e)
	e.players["player-1"].X, e.players["player-1"].Y = 700, 700
	e.players["player-1"].Armour = 0

	e.items = []*world.Item{world.NewItem("armour", 700, 700)}

	e.checkItemPickups()

	if e.players["player-1"].Armour == 0 {
		t.Error("armour pickup was not applied")
	}
	if len(e.items) != 0 {
		t.Errorf("got %d items remaining, want 0", len(e.items))
	}
}

func TestTick_BroadcastsSnapshotToClients(t *testing.T) {
	e := New()
	client := &fakeClient{id: "player-1"}
	e.Join(client)

	e.Tick()

	if len(client.msgs) != 1 {
		t.Fatalf("got %d messages sent, want 1", len(client.msgs))
	}
}

func TestCheckMatchEnd_DeclaresWinnerWhenOnePlayerRemainsAlive(t *testing.T) {
	e := New()
	e.Join(&fakeClient{id: "winner"})
	e.Join(&fakeClient{id: "loser"})
	makeLive(e)
	e.players["loser"].Alive = false

	e.checkMatchEnd()

	if e.phase != PhaseEnded {
		t.Errorf("got phase %v, want %v", e.phase, PhaseEnded)
	}
	if e.winner != "winner" {
		t.Errorf("got winner %q, want %q", e.winner, "winner")
	}
}

func TestCheckMatchEnd_NoWinnerWhenEveryoneDies(t *testing.T) {
	e := New()
	e.Join(&fakeClient{id: "player-1"})
	e.Join(&fakeClient{id: "player-2"})
	makeLive(e)
	e.players["player-1"].Alive = false
	e.players["player-2"].Alive = false

	e.checkMatchEnd()

	if e.phase != PhaseEnded {
		t.Errorf("got phase %v, want %v", e.phase, PhaseEnded)
	}
	if e.winner != "" {
		t.Errorf("got winner %q, want empty (no survivors)", e.winner)
	}
}

func TestCheckMatchEnd_ContinuesWithMultipleSurvivors(t *testing.T) {
	e := New()
	e.Join(&fakeClient{id: "player-1"})
	e.Join(&fakeClient{id: "player-2"})
	makeLive(e)

	e.checkMatchEnd()

	if e.phase != PhaseLive {
		t.Errorf("got phase %v, want %v — both players still alive", e.phase, PhaseLive)
	}
}

func TestCheckMatchEnd_ResetsImmediatelyWhenNoPlayersConnected(t *testing.T) {
	e := New()
	e.Join(&fakeClient{id: "player-1"})
	makeLive(e)
	e.Leave(&fakeClient{id: "player-1"})

	e.checkMatchEnd()

	if e.phase != PhaseWaiting {
		t.Errorf("got phase %v, want %v — no players connected", e.phase, PhaseWaiting)
	}
}

func TestTickEnded_ResetsAfterDelayElapses(t *testing.T) {
	e := New()
	client := &fakeClient{id: "player-1"}
	e.Join(client)
	e.phase = PhaseEnded
	e.winner = "player-1"
	e.endedAt = time.Now().Add(-world.ResetDelay - time.Second)

	e.Tick()

	if e.phase != PhaseWaiting {
		t.Errorf("got phase %v, want %v", e.phase, PhaseWaiting)
	}
	if e.winner != "" {
		t.Errorf("got winner %q, want cleared after reset", e.winner)
	}
	if !e.players["player-1"].Alive {
		t.Error("connected player not revived after reset")
	}
}

func TestTickEnded_StaysEndedBeforeDelayElapses(t *testing.T) {
	e := New()
	e.Join(&fakeClient{id: "player-1"})
	e.phase = PhaseEnded
	e.endedAt = time.Now()

	e.Tick()

	if e.phase != PhaseEnded {
		t.Errorf("got phase %v, want %v", e.phase, PhaseEnded)
	}
}

func TestResetMatch_GivesFreshZoneAndItems(t *testing.T) {
	e := New()
	e.Join(&fakeClient{id: "player-1"})
	makeLive(e)
	oldZone := e.zone
	e.zone.Radius = 10 // simulate a heavily shrunk zone

	e.resetMatch()

	if e.zone == oldZone {
		t.Error("zone was not replaced on reset")
	}
	if e.zone.Radius != world.ZoneInitialRadius {
		t.Errorf("got Radius %v, want fresh %v", e.zone.Radius, world.ZoneInitialRadius)
	}
	if len(e.items) != world.ItemCount {
		t.Errorf("got %d items, want %d", len(e.items), world.ItemCount)
	}
}

func TestNewForPlayers_AllowsListedPlayer(t *testing.T) {
	e := NewForPlayers([]string{"player-1", "player-2"})
	client := &fakeClient{id: "player-1"}

	ok := e.Join(client)

	if !ok {
		t.Error("got false, want true — player-1 is in the allowed list")
	}
	if _, exists := e.players["player-1"]; !exists {
		t.Error("player-1 was not actually added despite ok=true")
	}
}

func TestNewForPlayers_RejectsUnlistedPlayer(t *testing.T) {
	e := NewForPlayers([]string{"player-1", "player-2"})
	client := &fakeClient{id: "stranger"}

	ok := e.Join(client)

	if ok {
		t.Error("got true, want false — stranger is not in the allowed list")
	}
	if _, exists := e.players["stranger"]; exists {
		t.Error("stranger was added despite not being in the allowed list")
	}
}

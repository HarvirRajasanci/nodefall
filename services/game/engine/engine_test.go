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

func TestNew_CreatesEngineWithItemsAndZone(t *testing.T) {
	e := New()

	if len(e.items) != world.ItemCount {
		t.Errorf("got %d items, want %d", len(e.items), world.ItemCount)
	}
	if e.zone == nil {
		t.Error("got nil zone, want initialized zone")
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

func TestHandleInput_MovesAlivePlayer(t *testing.T) {
	e := New()
	client := &fakeClient{id: "player-1"}
	e.Join(client)
	startX := e.players["player-1"].X

	e.HandleInput(client, Input{DX: 1, DY: 0})

	if e.players["player-1"].X <= startX {
		t.Error("player did not move after HandleInput")
	}
}

func TestHandleInput_IgnoresUnknownClient(t *testing.T) {
	e := New()
	unknown := &fakeClient{id: "ghost"}

	// Should not panic and should have no effect.
	e.HandleInput(unknown, Input{DX: 1, DY: 0})

	if len(e.players) != 0 {
		t.Error("unknown client's input created a player")
	}
}

func TestHandleInput_IgnoresDeadPlayer(t *testing.T) {
	e := New()
	client := &fakeClient{id: "player-1"}
	e.Join(client)
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

	e.HandleInput(client, Input{Shoot: true})
	e.HandleInput(client, Input{Shoot: true}) // immediately again, too soon

	if len(e.bullets) != 1 {
		t.Errorf("got %d bullets, want 1 — second shot should be blocked by FireRate", len(e.bullets))
	}
}

func TestTick_BulletMovesAndCanExpire(t *testing.T) {
	e := New()
	client := &fakeClient{id: "player-1"}
	e.Join(client)
	e.HandleInput(client, Input{Shoot: true})
	startX := e.bullets[0].X

	e.Tick()

	// The bullet may have already hit something and been removed, or
	// still be flying — either way, it must have moved from its start
	// position if it's still present.
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

	e.players["attacker"].X, e.players["attacker"].Y = 100, 100
	e.players["target"].X, e.players["target"].Y = 100, 300
	startHP := e.players["target"].HP

	// Angle pointing straight down (positive Y) toward the target.
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
	e.players["player-1"].Alive = false
	e.respawnAt["player-1"] = time.Now().Add(-time.Second) // already due

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
	e.players["player-1"].Alive = false
	e.respawnAt["player-1"] = time.Now().Add(time.Hour) // far in the future

	e.processRespawns()

	if e.players["player-1"].Alive {
		t.Error("player revived before respawn delay elapsed")
	}
}

func TestCheckItemPickups_AppliesEffectAndRemovesItem(t *testing.T) {
	e := New()
	client := &fakeClient{id: "player-1"}
	e.Join(client)
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

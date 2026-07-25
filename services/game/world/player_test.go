package world

import (
	"testing"
	"time"
)

func TestNewPlayer_SetsStartingFields(t *testing.T) {
	player := NewPlayer("player-1", 100, 200)

	if player.ID != "player-1" {
		t.Errorf("got ID %q, want %q", player.ID, "player-1")
	}
	if player.X != 100 || player.Y != 200 {
		t.Errorf("got position (%v, %v), want (100, 200)", player.X, player.Y)
	}
	if player.HP != PlayerStartHP {
		t.Errorf("got HP %d, want %d", player.HP, PlayerStartHP)
	}
	if player.Armour != PlayerMinArmour {
		t.Errorf("got Armour %d, want %d", player.Armour, PlayerMinArmour)
	}
	if player.Gun != PlayerStartGun {
		t.Errorf("got Gun %q, want %q", player.Gun, PlayerStartGun)
	}
	if !player.Alive {
		t.Error("got Alive false, want true")
	}
}

func TestApplyDamage_NoArmourHitsHPDirectly(t *testing.T) {
	player := NewPlayer("player-1", 0, 0)
	player.Armour = PlayerMinArmour

	died := player.ApplyDamage(30)

	if died {
		t.Error("got died true, want false")
	}
	if player.HP != PlayerStartHP-30 {
		t.Errorf("got HP %d, want %d", player.HP, PlayerStartHP-30)
	}
}

func TestApplyDamage_ArmourFullyAbsorbs(t *testing.T) {
	player := NewPlayer("player-1", 0, 0)
	player.Armour = 50

	died := player.ApplyDamage(30)

	if died {
		t.Error("got died true, want false")
	}
	if player.HP != PlayerStartHP {
		t.Errorf("got HP %d, want unchanged %d", player.HP, PlayerStartHP)
	}
	if player.Armour != 20 {
		t.Errorf("got Armour %d, want 20", player.Armour)
	}
}

func TestApplyDamage_ArmourPartiallyAbsorbs(t *testing.T) {
	player := NewPlayer("player-1", 0, 0)
	player.Armour = 10

	died := player.ApplyDamage(30)

	if died {
		t.Error("got died true, want false")
	}
	if player.Armour != PlayerMinArmour {
		t.Errorf("got Armour %d, want %d", player.Armour, PlayerMinArmour)
	}
	// 10 damage absorbed by armour, remaining 20 hits HP.
	if player.HP != PlayerStartHP-20 {
		t.Errorf("got HP %d, want %d", player.HP, PlayerStartHP-20)
	}
}

func TestApplyDamage_KillsPlayerAndClampsHP(t *testing.T) {
	player := NewPlayer("player-1", 0, 0)

	died := player.ApplyDamage(PlayerStartHP + 50) // way more than enough to kill

	if !died {
		t.Error("got died false, want true")
	}
	if player.HP != PlayerMinHP {
		t.Errorf("got HP %d, want clamped to %d", player.HP, PlayerMinHP)
	}
	if player.Alive {
		t.Error("got Alive true, want false")
	}
}

func TestApplyDamage_NonLethalLeavesPlayerAlive(t *testing.T) {
	player := NewPlayer("player-1", 0, 0)

	died := player.ApplyDamage(10)

	if died {
		t.Error("got died true, want false")
	}
	if !player.Alive {
		t.Error("got Alive false, want true")
	}
}

func TestRespawn_ResetsPlayerFully(t *testing.T) {
	player := NewPlayer("player-1", 0, 0)
	player.ApplyDamage(PlayerStartHP) // kill the player, drain state
	player.Armour = 0
	player.Gun = "shotgun"

	player.Respawn(500, 600)

	if player.X != 500 || player.Y != 600 {
		t.Errorf("got position (%v, %v), want (500, 600)", player.X, player.Y)
	}
	if player.HP != PlayerStartHP {
		t.Errorf("got HP %d, want %d", player.HP, PlayerStartHP)
	}
	if player.Armour != PlayerMinArmour {
		t.Errorf("got Armour %d, want %d", player.Armour, PlayerMinArmour)
	}
	if player.Gun != PlayerStartGun {
		t.Errorf("got Gun %q, want %q", player.Gun, PlayerStartGun)
	}
	if !player.Alive {
		t.Error("got Alive false, want true")
	}
}

func TestCanShoot_TrueAfterFireRateElapsed(t *testing.T) {
	player := NewPlayer("player-1", 0, 0)
	player.LastShot = time.Now().Add(-2 * time.Second)

	if !player.CanShoot(1 * time.Second) {
		t.Error("got CanShoot false, want true")
	}
}

func TestCanShoot_FalseBeforeFireRateElapsed(t *testing.T) {
	player := NewPlayer("player-1", 0, 0)
	player.LastShot = time.Now()

	if player.CanShoot(1 * time.Second) {
		t.Error("got CanShoot true, want false")
	}
}

func TestRecordShot_UpdatesLastShot(t *testing.T) {
	player := NewPlayer("player-1", 0, 0)
	player.LastShot = time.Now().Add(-10 * time.Second)

	player.RecordShot()

	if player.CanShoot(1 * time.Second) {
		t.Error("got CanShoot true immediately after RecordShot, want false")
	}
}

func TestApplyPickup_ArmourAddsUpToMax(t *testing.T) {
	player := NewPlayer("player-1", 0, 0)
	player.Armour = PlayerMaxArmour - 10

	player.ApplyPickup("armour")

	if player.Armour != PlayerMaxArmour {
		t.Errorf("got Armour %d, want clamped to %d", player.Armour, PlayerMaxArmour)
	}
}

func TestApplyPickup_ArmourDoesNotExceedMax(t *testing.T) {
	player := NewPlayer("player-1", 0, 0)
	player.Armour = PlayerMinArmour

	player.ApplyPickup("armour")

	if player.Armour != ArmourPickupAmount {
		t.Errorf("got Armour %d, want %d", player.Armour, ArmourPickupAmount)
	}
}

func TestApplyPickup_GunReplacesCurrentGun(t *testing.T) {
	player := NewPlayer("player-1", 0, 0)

	player.ApplyPickup("shotgun")

	if player.Gun != "shotgun" {
		t.Errorf("got Gun %q, want %q", player.Gun, "shotgun")
	}
}

func TestMove_UpdatesPositionWithinBounds(t *testing.T) {
	player := NewPlayer("player-1", 1500, 1500)

	player.Move(1, 0)

	want := 1500 + PlayerSpeed
	if player.X != want {
		t.Errorf("got X %v, want %v", player.X, want)
	}
	if player.Y != 1500 {
		t.Errorf("got Y %v, want unchanged 1500", player.Y)
	}
}

func TestMove_ClampsAtMinBound(t *testing.T) {
	player := NewPlayer("player-1", 0, 0)

	player.Move(-100, -100) // push hard toward the negative edge

	if player.X != PlayerRadius {
		t.Errorf("got X %v, want clamped to %v", player.X, PlayerRadius)
	}
	if player.Y != PlayerRadius {
		t.Errorf("got Y %v, want clamped to %v", player.Y, PlayerRadius)
	}
}

func TestMove_ClampsAtMaxBound(t *testing.T) {
	player := NewPlayer("player-1", MapSize, MapSize)

	player.Move(100, 100) // push hard toward the positive edge

	want := MapSize - PlayerRadius
	if player.X != want {
		t.Errorf("got X %v, want clamped to %v", player.X, want)
	}
	if player.Y != want {
		t.Errorf("got Y %v, want clamped to %v", player.Y, want)
	}
}

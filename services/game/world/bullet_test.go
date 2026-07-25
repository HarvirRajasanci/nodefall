package world

import (
	"math"
	"testing"
)

func TestNewBullet_SetsVelocityFromAngle(t *testing.T) {
	// Angle 0 should point straight along the positive X axis.
	bullet := NewBullet("player-1", 100, 100, 0, 10)

	if math.Abs(bullet.VX-BulletSpeed) > 0.0001 {
		t.Errorf("got VX %v, want %v", bullet.VX, BulletSpeed)
	}
	if math.Abs(bullet.VY) > 0.0001 {
		t.Errorf("got VY %v, want 0", bullet.VY)
	}
}

func TestNewBullet_SetsStartingFields(t *testing.T) {
	bullet := NewBullet("player-1", 100, 200, 0, 25)

	if bullet.OwnerID != "player-1" {
		t.Errorf("got OwnerID %q, want %q", bullet.OwnerID, "player-1")
	}
	if bullet.X != 100 || bullet.Y != 200 {
		t.Errorf("got position (%v, %v), want (100, 200)", bullet.X, bullet.Y)
	}
	if bullet.Damage != 25 {
		t.Errorf("got Damage %d, want 25", bullet.Damage)
	}
	if bullet.Life != BulletLife {
		t.Errorf("got Life %d, want %d", bullet.Life, BulletLife)
	}
}

func TestMove_AdvancesPosition(t *testing.T) {
	bullet := NewBullet("player-1", 0, 0, 0, 10)
	startX := bullet.X

	bullet.Move()

	if bullet.X <= startX {
		t.Errorf("got X %v, want greater than %v", bullet.X, startX)
	}
}

func TestMove_ExpiresAfterBulletLifeTicks(t *testing.T) {
	bullet := NewBullet("player-1", 0, 0, 0, 10)

	var alive bool
	for i := 0; i < BulletLife; i++ {
		alive = bullet.Move()
	}

	if alive {
		t.Error("bullet still reports alive after BulletLife ticks, want expired")
	}
	if bullet.Life != 0 {
		t.Errorf("got Life %d, want 0", bullet.Life)
	}
}

func TestMove_StaysAliveBeforeLifeRunsOut(t *testing.T) {
	bullet := NewBullet("player-1", 0, 0, 0, 10)

	alive := bullet.Move()

	if !alive {
		t.Error("bullet reports expired on first move, want alive")
	}
}

func TestHits_DetectsCollisionWithinRadius(t *testing.T) {
	bullet := NewBullet("player-1", 100, 100, 0, 10)
	target := NewPlayer("player-2", 100, 100) // same position, definite overlap

	if !bullet.Hits(target) {
		t.Error("Hits returned false for overlapping bullet and player, want true")
	}
}

func TestHits_MissesOutsideRadius(t *testing.T) {
	bullet := NewBullet("player-1", 0, 0, 0, 10)
	target := NewPlayer("player-2", 10000, 10000) // far away

	if bullet.Hits(target) {
		t.Error("Hits returned true for distant bullet and player, want false")
	}
}

func TestHits_NeverHitsOwnOwner(t *testing.T) {
	bullet := NewBullet("player-1", 100, 100, 0, 10)
	owner := NewPlayer("player-1", 100, 100) // same ID as bullet's OwnerID

	if bullet.Hits(owner) {
		t.Error("Hits returned true against the bullet's own owner, want false")
	}
}

func TestHits_NeverHitsDeadPlayer(t *testing.T) {
	bullet := NewBullet("player-1", 100, 100, 0, 10)
	target := NewPlayer("player-2", 100, 100)
	target.Alive = false

	if bullet.Hits(target) {
		t.Error("Hits returned true against a dead player, want false")
	}
}

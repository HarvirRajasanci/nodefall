package world

import (
	"testing"
	"time"
)

func TestNewZone_SetsInitialState(t *testing.T) {
	zone := NewZone()

	center := MapSize / 2
	if zone.X != center || zone.Y != center {
		t.Errorf("got center (%v, %v), want (%v, %v)", zone.X, zone.Y, center, center)
	}
	if zone.Radius != ZoneInitialRadius {
		t.Errorf("got Radius %v, want %v", zone.Radius, ZoneInitialRadius)
	}
}

func TestShrink_DoesNothingBeforeIntervalElapses(t *testing.T) {
	zone := NewZone()
	startRadius := zone.Radius

	shrank := zone.Shrink()

	if shrank {
		t.Error("got shrank true, want false — interval hasn't elapsed")
	}
	if zone.Radius != startRadius {
		t.Errorf("got Radius %v, want unchanged %v", zone.Radius, startRadius)
	}
}

func TestShrink_ShrinksAfterIntervalElapses(t *testing.T) {
	zone := NewZone()
	zone.lastShrink = time.Now().Add(-ZoneShrinkInterval - time.Second)
	startRadius := zone.Radius

	shrank := zone.Shrink()

	if !shrank {
		t.Error("got shrank false, want true — interval has elapsed")
	}
	want := startRadius - ZoneShrinkAmount
	if zone.Radius != want {
		t.Errorf("got Radius %v, want %v", zone.Radius, want)
	}
}

func TestShrink_NeverGoesBelowMinRadius(t *testing.T) {
	zone := NewZone()
	zone.Radius = ZoneMinRadius + 50 // close to the floor
	zone.lastShrink = time.Now().Add(-ZoneShrinkInterval - time.Second)

	zone.Shrink()

	if zone.Radius != ZoneMinRadius {
		t.Errorf("got Radius %v, want clamped to %v", zone.Radius, ZoneMinRadius)
	}
}

func TestShrink_UpdatesLastShrinkTimestamp(t *testing.T) {
	zone := NewZone()
	zone.lastShrink = time.Now().Add(-ZoneShrinkInterval - time.Second)

	zone.Shrink()

	// Immediately calling Shrink again should now report false, since
	// lastShrink was just reset to "now" by the previous call.
	if zone.Shrink() {
		t.Error("got shrank true on second call, want false — lastShrink should have just been reset")
	}
}

func TestIsOutside_TrueBeyondRadius(t *testing.T) {
	zone := NewZone()
	// Place the player far outside the zone's current radius.
	player := NewPlayer("player-1", zone.X+zone.Radius+1000, zone.Y)

	if !zone.IsOutside(player) {
		t.Error("got IsOutside false, want true")
	}
}

func TestIsOutside_FalseWithinRadius(t *testing.T) {
	zone := NewZone()
	player := NewPlayer("player-1", zone.X, zone.Y) // dead center, well inside

	if zone.IsOutside(player) {
		t.Error("got IsOutside true, want false")
	}
}

func TestDamageIfOutside_DamagesPlayerOutsideZone(t *testing.T) {
	zone := NewZone()
	player := NewPlayer("player-1", zone.X+zone.Radius+1000, zone.Y)
	startHP := player.HP

	zone.DamageIfOutside(player)

	if player.HP != startHP-ZoneDamagePerTick {
		t.Errorf("got HP %d, want %d", player.HP, startHP-ZoneDamagePerTick)
	}
}

func TestDamageIfOutside_NoDamageWhenInsideZone(t *testing.T) {
	zone := NewZone()
	player := NewPlayer("player-1", zone.X, zone.Y)
	startHP := player.HP

	zone.DamageIfOutside(player)

	if player.HP != startHP {
		t.Errorf("got HP %d, want unchanged %d", player.HP, startHP)
	}
}

func TestDamageIfOutside_NoDamageWhenPlayerAlreadyDead(t *testing.T) {
	zone := NewZone()
	player := NewPlayer("player-1", zone.X+zone.Radius+1000, zone.Y)
	player.Alive = false
	startHP := player.HP

	zone.DamageIfOutside(player)

	if player.HP != startHP {
		t.Errorf("got HP %d, want unchanged %d", player.HP, startHP)
	}
}

func TestDamageIfOutside_ReturnsTrueOnLethalDamage(t *testing.T) {
	zone := NewZone()
	player := NewPlayer("player-1", zone.X+zone.Radius+1000, zone.Y)
	player.HP = ZoneDamagePerTick // exactly enough for the next hit to kill

	died := zone.DamageIfOutside(player)

	if !died {
		t.Error("got died false, want true")
	}
}

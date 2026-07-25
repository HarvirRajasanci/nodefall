package world

import (
	"math"
	"time"
)

// Zone represents the shrinking safe area players must stay inside.
// It starts covering the whole map and periodically shrinks toward
// its center, dealing damage each tick to any player caught outside it.
type Zone struct {
	X          float64   `json:"x"`
	Y          float64   `json:"y"`
	Radius     float64   `json:"radius"`
	lastShrink time.Time `json:"-"` // excluded from JSON — internal timing only
}

// NewZone creates a zone centered on the map at its initial radius.
func NewZone() *Zone {
	center := MapSize / 2

	return &Zone{
		X:          center,
		Y:          center,
		Radius:     ZoneInitialRadius,
		lastShrink: time.Now(),
	}
}

// Shrink reduces the zone's radius by ZoneShrinkAmount if at least
// ZoneShrinkInterval has passed since it last shrank, never going
// below ZoneMinRadius. Returns true if the zone actually shrank
// this call.
func (zone *Zone) Shrink() bool {
	if time.Since(zone.lastShrink) < ZoneShrinkInterval {
		return false
	}

	zone.Radius = max(ZoneMinRadius, zone.Radius-ZoneShrinkAmount)
	zone.lastShrink = time.Now()
	return true
}

// IsOutside reports whether the given player is currently outside
// the zone's boundary.
func (zone *Zone) IsOutside(player *Player) bool {
	dx := player.X - zone.X
	dy := player.Y - zone.Y
	distance := math.Hypot(dx, dy)

	return distance > zone.Radius
}

// DamageIfOutside applies ZoneDamagePerTick to the player if they are
// outside the zone and still alive. Returns true if the player died
// as a result, matching Player.ApplyDamage's own return convention.
func (zone *Zone) DamageIfOutside(player *Player) bool {
	if !player.Alive || !zone.IsOutside(player) {
		return false
	}

	return player.ApplyDamage(ZoneDamagePerTick)
}

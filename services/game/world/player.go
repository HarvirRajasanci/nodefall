package world

import "time"

// Player represents a connected player in the game world.
// It holds both the game state sent to clients and internal
// timing state excluded from JSON serialization.
type Player struct {
	ID       string    `json:"id"`
	X        float64   `json:"x"`
	Y        float64   `json:"y"`
	HP       int       `json:"hp"`
	Armour   int       `json:"armour"`
	Gun      string    `json:"gun"`
	Alive    bool      `json:"alive"`
	Angle    float64   `json:"angle"`
	LastShot time.Time `json:"-"` // excluded from JSON — internal timing only
}

// NewPlayer creates a player at the given coordinates with full health,
// no armour, and a pistol. The engine is responsible for choosing spawn coordinates.
func NewPlayer(id string, x, y float64) *Player {
	return &Player{
		ID:     id,
		X:      x,
		Y:      y,
		HP:     PlayerStartHP,
		Armour: PlayerMinArmour,
		Gun:    PlayerStartGun,
		Alive:  true,
	}
}

// absorbWithArmour reduces incoming damage by the player's current armour.
// Returns the remaining damage after armour absorption.
func (player *Player) absorbWithArmour(damage int) int {
	if player.Armour <= PlayerMinArmour {
		return damage
	}
	absorbed := min(damage, player.Armour)
	player.Armour -= absorbed
	return damage - absorbed
}

// ApplyDamage applies damage to the player, with armour absorbing first.
// Returns true if the player died as a result of this damage.
func (player *Player) ApplyDamage(damageAmount int) bool {
	damageAmount = player.absorbWithArmour(damageAmount)
	player.HP = max(PlayerMinHP, player.HP-damageAmount)
	if player.HP == PlayerMinHP {
		player.Alive = false
		return true
	}
	return false
}

// Respawn resets the player to full health at the given coordinates.
// The engine is responsible for choosing a safe respawn position.
func (player *Player) Respawn(x, y float64) {
	player.X = x
	player.Y = y
	player.HP = PlayerStartHP
	player.Armour = PlayerMinArmour
	player.Gun = PlayerStartGun
	player.Alive = true
}

// CanShoot returns true if enough time has elapsed since the player's last shot
// to fire again based on the current gun's fire rate.
func (player *Player) CanShoot(fireRate time.Duration) bool {
	return time.Since(player.LastShot) >= fireRate
}

// ApplyPickup applies an item pickup effect to the player.
// Armour pickups add to current armour up to PlayerMaxArmour.
// Gun pickups replace the player's current weapon.
func (player *Player) ApplyPickup(itemType string) {
	switch itemType {
	case "armour":
		player.Armour = min(player.Armour+ArmourPickupAmount, PlayerMaxArmour)
	default:
		player.Gun = itemType
	}
}

// Move updates the player's position by the given delta values,
// clamped to keep the player within map bounds.
func (player *Player) Move(dx, dy float64) {
	player.X = clampPosition(player.X + dx*PlayerSpeed)
	player.Y = clampPosition(player.Y + dy*PlayerSpeed)
}

// RecordShot updates the last shot timestamp to now.
// Call this immediately after confirming the player can shoot.
func (player *Player) RecordShot() {
	player.LastShot = time.Now()
}

// clampPosition constrains a position value to stay within map bounds,
// accounting for the player's radius so the player never overlaps the edge.
func clampPosition(value float64) float64 {
	return max(PlayerRadius, min(value, MapSize-PlayerRadius))
}

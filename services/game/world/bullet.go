package world

import "math"

// Bullet represents a single fired projectile in the game world.
type Bullet struct {
	OwnerID string  `json:"owner_id"`
	X       float64 `json:"x"`
	Y       float64 `json:"y"`
	VX      float64 `json:"vx"`
	VY      float64 `json:"vy"`
	Damage  int     `json:"damage"`
	Life    int     `json:"life"`
}

// NewBullet creates a bullet fired by the given owner from (x, y) at the
// given angle (radians), travelling at BulletSpeed with BulletLife ticks
// remaining before it expires.
func NewBullet(ownerID string, x, y, angle float64, damage int) *Bullet {
	return &Bullet{
		OwnerID: ownerID,
		X:       x,
		Y:       y,
		VX:      math.Cos(angle) * BulletSpeed,
		VY:      math.Sin(angle) * BulletSpeed,
		Damage:  damage,
		Life:    BulletLife,
	}
}

// Move advances the bullet by one tick and decrements its remaining life.
// Returns true if the bullet is still alive and should keep existing;
// returns false once its life has run out and it should be removed.
func (bullet *Bullet) Move() bool {
	bullet.X += bullet.VX
	bullet.Y += bullet.VY
	bullet.Life--
	return bullet.Life > 0
}

// Hits reports whether this bullet is currently colliding with the given
// player. A bullet never hits its own owner, and can't hit a player who
// is already dead.
func (bullet *Bullet) Hits(player *Player) bool {
	if !player.Alive || player.ID == bullet.OwnerID {
		return false
	}

	dx := bullet.X - player.X
	dy := bullet.Y - player.Y
	distance := math.Hypot(dx, dy)

	return distance <= BulletRadius+PlayerRadius
}

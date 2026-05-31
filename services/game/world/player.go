package world

import (
	"time"
)

type Player struct {
	ID       string    `json:"id"`
	X        float64   `json:"x"`
	Y        float64   `json:"y"`
	HP       int       `json:"hp"`
	Armour   int       `json:"armour"`
	Gun      string    `json:"gun"`
	Alive    bool      `json:"alive"`
	Angle    float64   `json:"angle"`
	LastShot time.Time `json:"-"`
}

func NewPlayer(id string, x, y float64) *Player {
	return &Player{
		ID:     id,
		X:      x,
		Y:      y,
		HP:     100,
		Armour: 0,
		Gun:    "pistol",
		Alive:  true,
	}
}

func (player *Player) absorbWithArmour(damage int) int {
	if player.Armour <= 0 {
		return damage
	}

	absorbed := min(damage, player.Armour)
	player.Armour -= absorbed
	return damage - absorbed
}

// Returns true if the player has died
func (player *Player) ApplyDamage(damage int) bool {
	damage = player.absorbWithArmour(damage)
	player.HP -= damage

	if player.HP <= 0 {
		player.HP = 0
		player.Alive = false
		return true
	}
	return false
}

func (player *Player) Respawn(x, y float64) {

}

func (player *Player) CanShoot(fireRate float64) bool {

}

func (player *Player) ApplyPickup(itemType string) {
	switch itemType {
	case "armour":
		player.Armour = min(player.Armour+50, 100)
	default:
		player.Gun = itemType
	}
}

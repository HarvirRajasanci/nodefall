package world

import (
	"math"
	"math/rand"
)

// itemTypes lists every pickup type that can spawn in the world.
// "armour" is handled specially by Player.ApplyPickup; everything else
// is treated as a gun name and replaces the player's current weapon.
var itemTypes = []string{"armour", "shotgun", "rifle"}

// Item represents a pickup lying in the game world: either an armour
// pickup or a weapon pickup, identified by Type.
type Item struct {
	Type string  `json:"type"`
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
}

// NewItem creates an item of the given type at the given coordinates.
func NewItem(itemType string, x, y float64) *Item {
	return &Item{
		Type: itemType,
		X:    x,
		Y:    y,
	}
}

// SpawnItems creates count items of random types at random positions
// within the map bounds, respecting ItemRadius so items never spawn
// overlapping the map edge.
func SpawnItems(count int) []*Item {
	items := make([]*Item, 0, count)
	for i := 0; i < count; i++ {
		itemType := itemTypes[rand.Intn(len(itemTypes))]
		x := ItemRadius + rand.Float64()*(MapSize-2*ItemRadius)
		y := ItemRadius + rand.Float64()*(MapSize-2*ItemRadius)
		items = append(items, NewItem(itemType, x, y))
	}
	return items
}

// CollidesWith reports whether this item is currently overlapping the
// given player, meaning the player is close enough to pick it up.
// An item can't be picked up by a dead player.
func (item *Item) CollidesWith(player *Player) bool {
	if !player.Alive {
		return false
	}

	dx := item.X - player.X
	dy := item.Y - player.Y
	distance := math.Hypot(dx, dy)

	return distance <= ItemRadius+PlayerRadius
}

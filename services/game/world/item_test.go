package world

import (
	"testing"
)

func TestNewItem_SetsFields(t *testing.T) {
	item := NewItem("armour", 100, 200)

	if item.Type != "armour" {
		t.Errorf("got Type %q, want %q", item.Type, "armour")
	}
	if item.X != 100 || item.Y != 200 {
		t.Errorf("got position (%v, %v), want (100, 200)", item.X, item.Y)
	}
}

func TestSpawnItems_ReturnsRequestedCount(t *testing.T) {
	items := SpawnItems(ItemCount)

	if len(items) != ItemCount {
		t.Errorf("got %d items, want %d", len(items), ItemCount)
	}
}

func TestSpawnItems_AllWithinMapBounds(t *testing.T) {
	items := SpawnItems(ItemCount)

	for _, item := range items {
		if item.X < ItemRadius || item.X > MapSize-ItemRadius {
			t.Errorf("item X %v out of bounds [%v, %v]", item.X, ItemRadius, MapSize-ItemRadius)
		}
		if item.Y < ItemRadius || item.Y > MapSize-ItemRadius {
			t.Errorf("item Y %v out of bounds [%v, %v]", item.Y, ItemRadius, MapSize-ItemRadius)
		}
	}
}

func TestSpawnItems_AllTypesAreValid(t *testing.T) {
	items := SpawnItems(ItemCount)

	valid := make(map[string]bool)
	for _, t := range itemTypes {
		valid[t] = true
	}

	for _, item := range items {
		if !valid[item.Type] {
			t.Errorf("item has unexpected Type %q", item.Type)
		}
	}
}

func TestCollidesWith_DetectsCollisionWithinRadius(t *testing.T) {
	item := NewItem("armour", 100, 100)
	player := NewPlayer("player-1", 100, 100) // same position, definite overlap

	if !item.CollidesWith(player) {
		t.Error("CollidesWith returned false for overlapping item and player, want true")
	}
}

func TestCollidesWith_MissesOutsideRadius(t *testing.T) {
	item := NewItem("armour", 0, 0)
	player := NewPlayer("player-1", 10000, 10000) // far away

	if item.CollidesWith(player) {
		t.Error("CollidesWith returned true for distant item and player, want false")
	}
}

func TestCollidesWith_NeverCollidesWithDeadPlayer(t *testing.T) {
	item := NewItem("armour", 100, 100)
	player := NewPlayer("player-1", 100, 100)
	player.Alive = false

	if item.CollidesWith(player) {
		t.Error("CollidesWith returned true against a dead player, want false")
	}
}

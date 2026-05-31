package world

import "time"

const (
	// Map
	MapSize = 3000.0

	// Player
	PlayerRadius    = 16.0
	PlayerSpeed     = 3.5
	PlayerMinHP     = 0
	PlayerStartHP   = 100
	PlayerMinArmour = 0
	PlayerMaxArmour = 100
	PlayerStartGun  = "pistol"

	// Bullet
	BulletSpeed  = 10.0
	BulletRadius = 6.0
	BulletLife   = 120 // ticks before despawn (~6 seconds at 20 ticks/sec)

	// Zone
	ZoneInitialRadius  = MapSize / 2
	ZoneShrinkAmount   = 200.0
	ZoneMinRadius      = 150.0
	ZoneDamagePerTick  = 1
	ZoneShrinkInterval = 20 * time.Second

	// Items
	ItemRadius         = 18.0
	ItemCount          = 35
	ArmourPickupAmount = 50

	// Game loop
	TickRate     = 50 * time.Millisecond // 20 ticks/sec
	RespawnDelay = 5 * time.Second
	ResetDelay   = 8 * time.Second
)

package game

import "testing"

// The source15 route exposed a burning tank beside a blocked gate without
// an overlapping strike marker. A nearby firing position makes waiting still
// unsafe for dismount; ordinary driving must first create room to leave.
func TestDemoDrivesToASaferBurningTankExitWithoutAStrikeMarker(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGame(true)
	g.scene = playScene
	g.player = actor{x: 126, y: 10728, health: 1, inTank: true, tankSlot: 19, tankFuel: 1, tankBurn: 60, tankBurning: true}
	g.cameraY, g.cameraBackY = 10634, 10656
	g.secondaryTerrain = g.cameraY < secondaryMapRows*tileSize
	g.enemies = []actor{{x: 126, y: 10708, kind: 2, slot: 3, health: 1}}
	g.pickups, g.mines = nil, nil
	g.eventIndex = len(g.resources.events)
	g.randomSpawn, g.redSpawn = false, false
	for tick := 0; tick < 20 && g.player.inTank; tick++ {
		g.stepPlay([2]playerControls{})
	}
	if g.player.inTank || g.player.x == 126 && g.player.y == 10728 || g.player.deadTicks != 0 || g.lives != 5 || g.ammunition != 99 || g.grenades != 50 {
		t.Fatalf("safe ordinary dismount failed: player%+v lives%d ammo%d grenades%d", g.player, g.lives, g.ammunition, g.grenades)
	}
}

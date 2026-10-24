package game

import "testing"

// The complete native run exposed this passage beside a parked tank. With
// no ammunition, entering its boarding rectangle while walking does not
// require a context press or a retreat back out of that same passage.
func TestUnarmedDemoWalksPastAnUnusedTank(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGame(true)
	g.scene = playScene
	g.player = actor{x: 98, y: 11666, health: 1}
	g.cameraY, g.cameraBackY = 11552, 11552
	g.secondaryTerrain = g.cameraY < secondaryMapRows*tileSize
	g.ammunition, g.grenades = 0, 0
	g.enemies = []actor{{x: 98, y: 11646, kind: 4, slot: 19, health: 1, tankFuel: 8}}
	g.pickups, g.mines = nil, nil
	g.eventIndex = len(g.resources.events)
	g.randomSpawn, g.redSpawn = false, false
	g.buildDemoPath()
	for range 64 {
		g.stepPlay([2]playerControls{})
	}
	if g.player.y > 11622 || g.player.inTank || g.player.deadTicks != 0 || g.lives != 5 || g.ammunition != 0 || g.grenades != 0 {
		t.Fatalf("unused tank blocked ordinary progress: player %+v lives %d ammo %d grenades %d", g.player, g.lives, g.ammunition, g.grenades)
	}
	if len(g.enemies) != 1 || g.enemies[0].kind != 4 || g.enemies[0].health != 1 || g.enemies[0].x != 98 || g.enemies[0].y != 11646 {
		t.Fatal("walking past the tank changed its ownership or position")
	}
}

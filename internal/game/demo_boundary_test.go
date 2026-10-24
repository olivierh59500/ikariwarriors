package game

import "testing"

// An ordinary late-level death can exhaust the original horizontal respawn
// search and place the hero beyond the walking range. The wall to the left is
// solid at this Y; normal upward movement reaches a gap before returning left.
func TestDemoReturnsThroughTheGapAfterAnOriginalEdgeRespawn(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGame(true)
	g.scene = playScene
	g.player = actor{x: 208, y: 2446, health: 1, invulnerable: 76}
	g.cameraY, g.cameraBackY = 2270, 2272
	g.secondaryTerrain = g.cameraY < secondaryMapRows*tileSize
	g.lives = 4
	g.enemies, g.pickups, g.mines = nil, nil, nil
	g.eventIndex = len(g.resources.events)
	g.randomSpawn, g.redSpawn = false, false
	for tick := 0; tick < 32 && g.player.x > 195; tick++ {
		g.stepPlay([2]playerControls{})
	}
	if g.player.x > 195 || g.player.y > 2432 || g.player.deadTicks != 0 || g.lives != 4 || g.ammunition != 99 || g.grenades != 50 {
		t.Fatalf("ordinary edge recovery failed: player%+v lives%d ammo%d grenades%d", g.player, g.lives, g.ammunition, g.grenades)
	}
}

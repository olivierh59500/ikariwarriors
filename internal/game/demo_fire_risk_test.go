package game

import "testing"

// These two original-map standoffs were found by the complete native demo.
// Its old player waited 22-23 pixels below a visible firing soldier until a
// newly born bullet was too close to dodge. No future random draw is supplied.
func TestDemoCreatesRoomBeforeStoppedInfantryFires(t *testing.T) {
	for _, position := range []struct{ x, y, camera, back, enemyX, enemyY float64 }{
		{130, 3322, 3210, 3232, 132, 3299},
		{134, 1094, 1000, 1024, 142, 1072},
	} {
		g, err := New(Options{Mute: true})
		if err != nil {
			t.Fatal(err)
		}
		g.startGame(true)
		g.scene = playScene
		g.player = actor{x: position.x, y: position.y, health: 1}
		g.cameraY, g.cameraBackY = position.camera, position.back
		g.secondaryTerrain = g.cameraY < secondaryMapRows*tileSize
		g.enemies = []actor{{x: position.enemyX, y: position.enemyY, kind: 0, slot: 9, health: 1, shots: 3, marchY: 1}}
		g.pickups, g.mines = nil, nil
		g.eventIndex = len(g.resources.events)
		g.randomSpawn, g.redSpawn = false, false
		g.ammunition, g.grenades = 0, 0
		for range 24 {
			g.stepPlay([2]playerControls{})
		}
		if g.player.x == position.x && g.player.y == position.y || g.player.deadTicks != 0 || g.lives != 5 || g.ammunition != 0 || g.grenades != 0 {
			t.Fatalf("firing standoff%.0f/%.0f was not escaped through ordinary movement: player%+v lives%d", position.x, position.y, g.player, g.lives)
		}
	}
}

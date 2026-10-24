package game

import "testing"

// Original normally played run12 alternated boarding/dismounting at this exact
// gate, with zero ammunition and41 grenades. All movement and context actions
// here use the shared game loop and its original boarding collision rectangle.
func TestDemoLeavesTheOriginalExhaustedTankBeforeThrowing(t *testing.T) {
	for _, boarded := range []bool{false, true} {
		g, err := New(Options{Mute: true})
		if err != nil {
			t.Fatal(err)
		}
		g.startGame(true)
		g.scene = playScene
		g.cameraY, g.cameraBackY = 10644, 10656
		g.secondaryTerrain = g.cameraY < secondaryMapRows*tileSize
		g.player = actor{x: 126, y: 10728, health: 1, inTank: boarded, tankSlot: 19, tankFuel: 25}
		g.lives, g.ammunition, g.grenades = 4, 0, 41
		g.enemies = nil
		if !boarded {
			g.enemies = []actor{{x: 126, y: 10728, kind: 4, slot: 19, health: 1, tankFuel: 25}}
		}
		g.eventIndex = len(g.resources.events)
		g.randomSpawn, g.redSpawn = false, false
		for tick := 0; tick < 32 && (g.player.inTank || g.boardingTankAt(0, g.player.x, g.player.y) >= 0); tick++ {
			g.stepPlay([2]playerControls{})
		}
		if g.player.inTank || g.boardingTankAt(0, g.player.x, g.player.y) >= 0 || g.player.deadTicks != 0 || g.lives != 4 || g.ammunition != 0 || g.grenades != 41 {
			t.Fatalf("boarded%v: exhausted-tank escape failed: player%+v lives%d ammo%d grenades%d", boarded, g.player, g.lives, g.ammunition, g.grenades)
		}
	}
}

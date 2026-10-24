package game

import "testing"

// A marching soldier enters the shell's horizontal rectangle during its
// first enemy update. The owned shell has already passed that position, so
// predicting the soldier one update early invents an immediate interception.
func TestDemoCannonLandingUsesThePreMovementEnemyPose(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGame(false)
	g.player = actor{x: 72, y: 14970, health: 1, inTank: true, direction: 0}
	g.cameraY, g.cameraBackY = 14812, 14816
	g.secondaryTerrain = g.cameraY < secondaryMapRows*tileSize
	g.enemies = []actor{{x: 90, y: 14974, health: 1, kind: 0, slot: 5, marchX: 1, marchTime: 20}}
	g.projectiles, g.mines = nil, nil
	shot := g.projectileFor(&g.player, false, 0, false, 6)
	g.prepareDemoForecast()
	predicted, predictedTick, landed := g.demoShotLanding(shot)
	g.demoForecastActive = false
	if !landed || predictedTick <= 1 {
		t.Fatalf("predicted an interception before the soldier moved: tick %d landed %v", predictedTick, landed)
	}
	flightLimit := shot.life
	for tick := 1; tick <= flightLimit; tick++ {
		moved, flying := g.advanceOwnedProjectile(shot)
		if !flying {
			if tick != predictedTick || moved.x != predicted.x || moved.y != predicted.y {
				t.Fatalf("predicted landing %.0f/%.0f at %d; actual %.0f/%.0f at %d", predicted.x, predicted.y, predictedTick, moved.x, moved.y, tick)
			}
			return
		}
		shot = moved
		g.advanceInfantryMarch(&g.enemies[0])
	}
	t.Fatal("the real shell did not reach its predicted landing")
}

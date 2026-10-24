package game

import "testing"

// These original-map gunner positions put the target at the edge of the
// projectile rectangle. Two pixels of horizontal input before firing decide
// whether the real shot hits; rotating the sprite alone is insufficient.
func TestDemoAimsFromItsActualPostHorizontalMuzzle(t *testing.T) {
	for _, fixture := range []struct {
		name      string
		x         float64
		direction int
		move      playerControls
		canHit    bool
	}{
		{"turn-misses", 74, 0, playerControls{}, false},
		{"turn-hits", 76, 0, playerControls{}, true},
		{"locked-hits", 74, 7, playerControls{}, true},
		{"locked-step-misses", 74, 7, playerControls{x: -1}, false},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			g, err := New(Options{Mute: true})
			if err != nil {
				t.Fatal(err)
			}
			g.startGame(false)
			g.scene = playScene
			g.player = actor{x: fixture.x, y: 14906, direction: fixture.direction, health: 1}
			g.cameraY, g.cameraBackY = 14812, 14816
			g.secondaryTerrain = g.cameraY < secondaryMapRows*tileSize
			g.enemies = []actor{{x: 52, y: 14860, kind: 2, slot: 3, health: 1}}
			g.projectiles, g.mines = nil, nil
			aim, found := g.demoAimForMove(fixture.move)
			if found != fixture.canHit {
				t.Fatalf("aim %d found %v, expected a hittable trajectory: %v", aim, found, fixture.canHit)
			}
			if !found {
				return
			}
			control := fixture.move
			g.directionLock[0] = g.player.direction == aim
			if !g.directionLock[0] {
				control.x, control.y = directionVector(aim)
			}
			control.fire = true
			g.stepPlayer(0, control)
			// Keep this gunner stationary, matching the planning observation.
			// Advance the real owned projectile and damage routines unchanged.
			for range 13 {
				g.advancePlayerProjectiles(0)
			}
			if g.enemies[0].health != 0 || g.ammunition != 98 || g.score != 100 {
				t.Fatalf("selected ordinary shot missed: enemy %+v ammo %d score %d", g.enemies[0], g.ammunition, g.score)
			}
		})
	}
}

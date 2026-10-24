package game

import "testing"

// This controlled two-player case creates the real finite ten-tick bullet
// through ordinary input, then reaches its moving impact phase at the upper
// screen edge. It is a source-order regression, not a normally played win.
func prepareLatePrunePlayers(t *testing.T) (*Game, float64) {
	t.Helper()
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGamePlayers(false, 2)
	g.scene, g.demo, g.randomSpawn, g.redSpawn = playScene, false, false, false
	g.eventIndex = len(g.resources.events)
	g.cameraY, g.cameraBackY = 9000, 9024
	g.enemies, g.projectiles, g.pickups, g.mines = nil, nil, nil, nil
	g.player.y, g.player2.y = g.cameraY+21, g.cameraY+94
	g.player.direction = 0
	g.bulletPower[0] = 2
	found := false
	for x := 14.0; x <= 192; x += 2 {
		probe := g.player2
		probe.x = x
		g.moveActorStep(&probe, 0, -2)
		if probe.y < g.player2.y {
			g.player2.x, found = x, true
			break
		}
	}
	if !found {
		t.Fatal("the original map has no ordinary upward second-player step")
	}
	camera := g.cameraY
	for tick := 0; tick < 10; tick++ {
		g.stepPlay([2]playerControls{{fire: tick == 0}, {}})
	}
	if g.cameraY != camera || len(g.projectiles) != 1 {
		t.Fatalf("finite warmup changed the camera or shot pool: camera %.0f shots %+v", g.cameraY, g.projectiles)
	}
	shot := g.projectiles[0]
	if shot.enemy || shot.owner != 0 || shot.life != 0 || shot.impact != 3 || shot.vy != -6 || shot.y != camera-35 || g.ammunition != 98 {
		t.Fatalf("ordinary bullet did not reach its source impact phase: shot %+v ammo %d", shot, g.ammunition)
	}
	return g, camera
}

// $FEC4 advances P1's pool before P2 input. $916C calls the $1224A bounds
// pass after both inputs. P2's actual upward scroll makes oldY-41 become
// finalY-39, so that still-active impact image must survive this frame.
func TestOwnedProjectileBoundsUseTheCameraAfterBothPlayers(t *testing.T) {
	for _, scroll := range []bool{false, true} {
		name := "stationary-camera-clears"
		if scroll {
			name = "second-player-scroll-retains"
		}
		t.Run(name, func(t *testing.T) {
			g, camera := prepareLatePrunePlayers(t)
			controls := [2]playerControls{}
			if scroll {
				controls[1].y = -1
			}
			g.stepPlay(controls)
			if !scroll {
				if g.cameraY != camera || len(g.projectiles) != 0 {
					t.Fatalf("late bounds did not clear the stationary-camera shot: camera %.0f shots %+v", g.cameraY, g.projectiles)
				}
				return
			}
			if g.cameraY != camera-2 {
				t.Fatalf("ordinary P2 upward input did not scroll two pixels: %.0f", g.cameraY)
			}
			if len(g.projectiles) != 1 {
				t.Fatalf("P1 shot was cleared before P2's scroll: count %d, old camera %.0f, final camera %.0f; expected live phase2 at worldY %.0f/final screenY -39", len(g.projectiles), camera, g.cameraY, camera-41)
			}
			shot := g.projectiles[0]
			if shot.y != camera-41 || shot.y-g.cameraY != -39 || shot.impact != 2 || shot.life != 0 || shot.owner != 0 {
				t.Fatalf("retained moving impact differs: %+v camera %.0f", shot, g.cameraY)
			}
		})
	}
}

// The owned damage query precedes the late small-object painter as well.
// An active edge target must be hit even when the moved shot's old-camera
// coordinate is outside the drawing rectangle before P2 changes that camera.
func TestOwnedProjectileDamagePrecedesTheLateScreenBounds(t *testing.T) {
	g, camera := prepareLatePrunePlayers(t)
	shot := g.projectiles[0]
	g.enemies = []actor{{x: shot.x, y: camera - 41, health: 1, kind: 2, slot: 3}}
	g.stepPlay([2]playerControls{{}, {y: -1}})
	if len(g.enemies) != 1 || g.enemies[0].health > 0 || g.score != 100 || g.score2 != 0 {
		t.Fatalf("the owned damage pass skipped its edge target: enemies %+v scores %d/%d", g.enemies, g.score, g.score2)
	}
	if g.ammunition != 98 || g.ammunition2 != 99 || g.grenades != 50 || g.grenades2 != 50 || g.lives != 5 || g.lives2 != 5 {
		t.Fatal("the bounded edge check changed finite resources or reserves")
	}
}

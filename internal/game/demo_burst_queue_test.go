package game

import "testing"

// A normally played loss exposed this shared-pool consequence: the moving
// fragments miss the player, but pending vehicle cleanup creates a second
// contact quad before the same update's enemy-blast pass. These controlled
// scenes compare the forecast with the actual production gameplay loop.
func TestDemoAnticipatesTheImmediateQuadWhenVehicleFragmentsAreQueued(t *testing.T) {
	for _, occupied := range []bool{false, true} {
		for _, damaged := range []bool{false, true} {
			name := "free"
			if occupied {
				name = "occupied"
			}
			if damaged {
				name += "/pending-cleanup"
			} else {
				name += "/incoming-fragment"
			}
			t.Run(name, func(t *testing.T) {
				g := newGrenadeTimingScene(t, "infantry", false)
				lives, ammunition, grenades := g.lives, g.ammunition, g.grenades
				g.player = actor{x: 104, y: 3906, health: 1}
				g.cameraY, g.cameraBackY = 3812, 3840
				g.secondaryTerrain = g.cameraY < secondaryMapRows*tileSize
				g.enemies, g.projectiles = nil, nil
				g.enemyVehicles[0] = enemyVehicle{x: 82, y: 3904, alive: [3]bool{true, true, true}, owner: -1}
				if damaged {
					g.enemyVehicles[0].damaged = 1
				}
				if occupied {
					g.burst = radialBurst{active: true, remaining: 1, owner: -1, parts: [8]navPoint{
						{82, 3927}, {122, 3927}, {162, 3927}, {82, 3967},
						{162, 3967}, {82, 4007}, {122, 4007}, {162, 4007},
					}}
				}
				forecast := g.demoThreatAt(104, 3904, 1) > 0
				if forecast != occupied {
					t.Fatalf("first-update quad threat%t, occupied pool%t", forecast, occupied)
				}
				g.stepPlay([2]playerControls{{y: -1}})
				if hit := g.player.deadTicks > 0; hit != forecast {
					t.Fatalf("actual first-update hit%t, predicted%t; player%+v", hit, forecast, g.player)
				}
				if g.lives != lives || g.ammunition != ammunition || g.grenades != grenades {
					t.Fatal("controlled contact changed finite resources or consumed a reserve early")
				}
			})
		}
	}
}

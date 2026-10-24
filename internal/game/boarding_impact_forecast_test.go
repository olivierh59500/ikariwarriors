package game

import "testing"

func newControlledImpactForecast(t *testing.T) *Game {
	t.Helper()
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGame(false)
	g.scene = playScene
	g.cameraY, g.cameraBackY = 11104, 11136
	g.player = actor{x: 104, y: 11280, health: 1, inTank: true, tankFuel: 17}
	g.enemies, g.projectiles, g.pickups, g.mines = nil, nil, nil, nil
	g.eventIndex, g.randomSpawn, g.redSpawn = len(g.resources.events), false, false
	g.grenadePower[0] = 2
	return g
}

// This is a controlled native prediction check, not an additional original
// G2 observation. An old foot-impact timer is distinct from an owned quad.
func TestBoardedFootImpactDoesNotForecastGrenadeUpgradeFragments(t *testing.T) {
	g := newControlledImpactForecast(t)
	shot := projectile{x: 32, y: 11210, vy: -6, owner: 0, slot: 0, power: 1, frame: 1, impact: 3}
	g.projectiles = []projectile{shot}
	if _, fragments := g.demoFragmentStart(shot); fragments {
		t.Fatal("a boarded foot-impact cleanup was mistaken for an upgraded quad expiry")
	}
	for tick := 1; tick <= 3; tick++ {
		g.stepPlay([2]playerControls{})
		if g.blasts[0].impact > 0 || g.burst.active || g.sceneryBurst.active || g.player.health <= 0 || g.lives != 5 {
			t.Fatalf("old foot-impact timer created a new explosion at tick%d", tick)
		}
	}
	if len(g.projectiles) != 0 {
		t.Fatal("the finite foot-impact timer did not clear")
	}
}

// Current grenade power still controls an actual owned quad's expiry after
// dismount. Its hostile fragments must remain predicted and physically real.
func TestOwnedCannonQuadStillForecastsFragmentsAfterDismount(t *testing.T) {
	g := newControlledImpactForecast(t)
	g.player = actor{x: 80, y: 11208, health: 1}
	quad := projectile{x: 80, y: 11200, owner: 0, slot: 0, cannon: true, frame: 7, impact: 2}
	g.blasts[0] = quad
	first, fragments := g.demoFragmentStart(quad)
	if !fragments || first != 2 || !g.demoFutureFragmentsHit(quad, g.player, 2) {
		t.Fatal("dismount hid the actual owned quad's future hostile fragments")
	}
	g.stepPlay([2]playerControls{})
	if g.player.health <= 0 || g.burst.active {
		t.Fatal("the owned quad's thrower exemption or remaining phase was lost")
	}
	g.stepPlay([2]playerControls{})
	if !g.burst.active || g.burst.remaining != 5 || g.burst.owner != -1 || g.player.health != 0 || g.player.deadTicks != 23 {
		t.Fatal("the predicted hostile first fragment did not occur in complete gameplay")
	}
}

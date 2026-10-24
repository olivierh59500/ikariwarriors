package game

import "testing"

// The source reuse corpus independently establishes this short timer. These
// controlled scenes isolate its effect on AI predictions from the old quad.
func newSlotCleanupScene(t *testing.T, remaining int, mounted bool) *Game {
	t.Helper()
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGame(false)
	g.scene = playScene
	g.player = actor{x: 104, y: 10000, direction: 0, health: 1, inTank: mounted, tankFuel: 64}
	g.cameraY, g.cameraBackY = 9904, 9936
	g.secondaryTerrain = g.cameraY < secondaryMapRows*tileSize
	g.enemies, g.projectiles, g.pickups, g.mines = nil, nil, nil, nil
	g.eventIndex, g.randomSpawn, g.redSpawn = len(g.resources.events), false, false
	g.pendingFootImpact[0] = remaining
	for row := range g.destroyed {
		for column := range g.destroyed[row] {
			g.destroyed[row][column] = true
		}
	}
	return g
}

func TestDemoDoesNotForecastAnExpiryAfterTheKnownSlotCleanup(t *testing.T) {
	g := newSlotCleanupScene(t, 2, true)
	shot := g.projectileFor(&g.player, false, 0, false, 6)
	if _, _, lands := g.demoShotLanding(shot); lands {
		t.Fatal("forecast invented a later quad from a shell that clears in two updates")
	}
	g.stepPlay([2]playerControls{{fire: true}})
	g.stepPlay([2]playerControls{})
	if len(g.projectiles) != 0 || g.blasts[0].impact != 0 || g.ammunition != 98 {
		t.Fatalf("ordinary shell did not clear without a quad: shots%+v blast%+v ammo%d", g.projectiles, g.blasts[0], g.ammunition)
	}
}

func TestDemoIncludesCollisionBeforeTheSameFrameSlotCleanup(t *testing.T) {
	g := newSlotCleanupScene(t, 1, true)
	g.player.direction = 4
	g.enemies = []actor{{x: 104, y: 10020, health: 1, kind: 2, slot: 3}}
	shot := g.projectileFor(&g.player, false, 0, false, 6)
	landing, tick, lands := g.demoShotLanding(shot)
	if !lands || tick != 1 {
		t.Fatal("cleanup suppressed the preceding first-update actor contact")
	}
	g.stepPlay([2]playerControls{{fire: true}})
	if g.enemies[0].health > 0 || g.blasts[0].impact != 3 || g.blasts[0].x != landing.x || g.blasts[0].y != landing.y || g.ammunition != 98 {
		t.Fatalf("actual contact did not precede cleanup: enemy%+v blast%+v", g.enemies[0], g.blasts[0])
	}
}

func TestDemoDoesNotCountABulletHitBeyondTheKnownSlotCleanup(t *testing.T) {
	g := newSlotCleanupScene(t, 2, false)
	g.player.direction = 2
	g.enemies = []actor{{x: 142, y: 9990, health: 1, kind: 2, slot: 3}}
	// Occupy the normal first choice so the real allocator uses the affected slot.
	g.projectiles = []projectile{{x: -40, y: 9910, vx: -6, life: 10, owner: 0, slot: 1}}
	shot := g.projectileFor(&g.player, false, 0, false, 6)
	if shot.slot != 0 {
		t.Fatal("controlled allocator did not select the reused first slot")
	}
	if _, hit := g.demoBulletTarget(shot); hit {
		t.Fatal("forecast counted a hit after the real first-slot cleanup")
	}
	g.stepPlay([2]playerControls{{fire: true}})
	g.stepPlay([2]playerControls{})
	g.stepPlay([2]playerControls{})
	if g.enemies[0].health != 1 || g.score != 0 || g.ammunition != 98 {
		t.Fatalf("controlled bullet actually hit the supposedly uncovered target: enemies%+v score%d ammo%d", g.enemies, g.score, g.ammunition)
	}
}

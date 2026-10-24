package game

import "testing"

func pendingShotGame(t *testing.T) *Game {
	t.Helper()
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGame(false)
	g.scene = playScene
	g.projectiles, g.mines = nil, nil
	return g
}

func TestDemoWaitsForItsExistingLethalBullet(t *testing.T) {
	g := pendingShotGame(t)
	g.enemies = []actor{{x: 78, y: 14930, kind: 2, slot: 3, health: 1}}
	g.projectiles = []projectile{{x: 78, y: 14956, vy: -6, owner: 0, power: 1, life: 7}}
	if mask := g.demoPendingBulletTargets(); mask != 1 {
		t.Fatalf("lethal shot forecast mask %b, expected the existing gunner", mask)
	}
	if _, found := g.demoAim(); found {
		t.Fatal("the pilot requested another shot at an already covered target")
	}
	ammunition := g.ammunition
	g.advancePlayerProjectiles(0)
	if g.enemies[0].health != 0 || g.score != 100 || g.ammunition != ammunition {
		t.Fatal("the actual in-flight bullet did not finish its predicted target")
	}
}

func TestDemoPendingShotStopsAtTheActualFirstActor(t *testing.T) {
	g := pendingShotGame(t)
	g.enemies = []actor{
		{x: 78, y: 14942, kind: 2, slot: 3, health: 1},
		{x: 78, y: 14930, kind: 2, slot: 4, health: 1},
	}
	g.projectiles = []projectile{{x: 78, y: 14968, vy: -6, owner: 0, power: 1, life: 9}}
	if mask := g.demoPendingBulletTargets(); mask != 1 {
		t.Fatalf("one bullet incorrectly covered actors behind its first hit: %b", mask)
	}
	g.advancePlayerProjectiles(0)
	if g.enemies[0].health != 0 || g.enemies[1].health != 1 || len(g.projectiles) != 0 {
		t.Fatal("the real projectile did not stop at its first predicted actor")
	}
}

func TestDemoDoesNotTreatANonlethalBulletAsACompletedKill(t *testing.T) {
	g := pendingShotGame(t)
	g.enemies = []actor{{x: 78, y: 14930, kind: 2, slot: 3, health: 2}}
	g.projectiles = []projectile{{x: 78, y: 14956, vy: -6, owner: 0, power: 1, life: 7}}
	if g.demoPendingBulletTargets() != 0 {
		t.Fatal("a nonlethal impact suppressed necessary follow-up fire")
	}
	g.advancePlayerProjectiles(0)
	if g.enemies[0].health != 1 || len(g.projectiles) != 0 {
		t.Fatal("the actual projectile did not leave the expected surviving actor")
	}
}

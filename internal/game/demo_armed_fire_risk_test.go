package game

import (
	"reflect"
	"testing"
)

// These are controlled native policy diagnostics, not original-game fixtures.
// They leave all model rules intact and require ordinary real projectile hits
// to justify a forecast kill; finite ammunition alone is not a completed hit.
func armedFiringLaneGame(t *testing.T) *Game {
	t.Helper()
	g := pendingShotGame(t)
	g.player = actor{x: 74, y: 14964, health: 1}
	g.enemies = []actor{{x: 78, y: 14930, kind: 2, slot: 3, health: 1}}
	g.projectiles, g.mines = nil, nil
	g.ammunition, g.grenades = 99, 50
	g.bulletPower[0] = 1
	return g
}

func guardFiringLaneRandom(t *testing.T, g *Game) {
	t.Helper()
	g.options.RandomSource = func(index uint64) uint32 { t.Fatalf("controller queried live future random word%d", index); return 0 }
}

func TestDemoArmedFiringLaneRemainsSoftAndReadOnly(t *testing.T) {
	g := armedFiringLaneGame(t)
	guardFiringLaneRandom(t, g)
	player, enemies, random, index, lives, ammo, grenades := g.player, append([]actor(nil), g.enemies...), g.random, g.randomSourceIndex, g.lives, g.ammunition, g.grenades
	armed := g.demoPossibleShotCost(74, 14964, 1)
	if armed <= 0 {
		t.Fatal("ammunition concealed the visible firing lane before any owned hit")
	}
	if flank := g.demoPossibleShotCost(160, 14964, 1); flank >= armed {
		t.Fatal("a clear flank did not cost less than the firing lane")
	}
	if hard := g.demoThreatAt(74, 14964, 1); hard != 0 {
		t.Fatal("possible future fire was converted into an existing hard collision")
	}
	g.ammunition = 0
	unarmed := g.demoPossibleShotCost(74, 14964, 1)
	g.ammunition = ammo
	if unarmed != armed {
		t.Fatalf("carried bullets changed an enemy's unresolved firing risk: armed%g unarmed%g", armed, unarmed)
	}
	if g.player != player || !reflect.DeepEqual(g.enemies, enemies) || g.random != random || g.randomSourceIndex != index || g.lives != lives || g.ammunition != ammo || g.grenades != grenades || len(g.projectiles) != 0 {
		t.Fatal("read-only risk changed mechanics, resources or random state")
	}
}

func TestDemoFiringLaneRespectsRealProtectionAndReadiness(t *testing.T) {
	g := armedFiringLaneGame(t)
	guardFiringLaneRandom(t, g)
	g.player.invulnerable = 5
	if g.demoPossibleShotCost(74, 14964, 4) != 0 || g.demoPossibleShotCost(74, 14964, 5) <= 0 {
		t.Fatal("risk ignored the exact expiry of existing protection")
	}
	g.player.invulnerable = 0
	g.player.inTank = true
	if g.demoPossibleShotCost(74, 14964, 5) != 0 {
		t.Fatal("infantry bullet standoff cost applies inside an intact tank")
	}
	g.player.inTank = false
	g.enemies[0].kind = 0
	g.enemies[0].shots = 0
	if g.demoPossibleShotCost(78, 14964, 1) != 0 {
		t.Fatal("marching infantry with no firing pose acquired a fictitious shot")
	}
	g.enemies[0].shots = 2
	g.enemies[0].marchTime = 3
	if g.demoPossibleShotCost(78, 14964, 2) != 0 {
		t.Fatal("a still-timed marching step was classified as an immediate firing pose")
	}
	g.enemies[0].marchTime = 0
	if g.demoPossibleShotCost(78, 14964, 2) <= 0 {
		t.Fatal("stopped, armed infantry lost its visible firing warning")
	}
}

func TestDemoVerifiedHitClearsOnlyItsFutureFiringPass(t *testing.T) {
	g := armedFiringLaneGame(t)
	g.ammunition = 0 // The policy must retain the same hit timing even without spare rounds.
	shot := projectile{x: 78, y: 14968, vy: -6, life: 9, owner: 0, power: 1, slot: 1}
	guardFiringLaneRandom(t, g)
	g.prepareDemoForecast()
	target, hit, lethal := g.demoBulletImpact(shot)
	if target != 0 || !lethal || hit < 2 {
		t.Fatalf("controlled ordinary shot has no delayed lethal trajectory: target%d hit%d lethal%v", target, hit, lethal)
	}
	g.demoEnemyDefeat[target] = hit
	if g.demoPossibleShotCost(74, 14964, hit-1) <= 0 {
		t.Fatal("future owned hit erased an earlier possible enemy firing pass")
	}
	if g.demoPossibleShotCost(74, 14964, hit) != 0 || g.demoPossibleShotCost(74, 14964, hit+1) != 0 {
		t.Fatal("verified owned hit did not clear the same frame's later enemy firing pass")
	}
	if g.demoThreatAt(74, 14950, hit) <= 0 {
		t.Fatal("owned hit cancelled the preceding same-frame body contact")
	}
	if g.demoThreatAt(74, 14950, hit+1) != 0 {
		t.Fatal("dead target retained body contact on the following frame")
	}
	// The unchanged model, rather than the policy, must perform the actual hit.
	g.options.RandomSource = nil
	g.demoForecastActive = false
	g.projectiles = []projectile{shot}
	for i := 1; i < hit; i++ {
		g.advancePlayerProjectiles(0)
		if g.enemies[0].health <= 0 {
			t.Fatal("target died before its predicted ordinary impact")
		}
	}
	g.advancePlayerProjectiles(0)
	if g.enemies[0].health > 0 || g.ammunition != 0 {
		t.Fatal("ordinary owned shot did not make its forecast kill without new ammunition")
	}
}

func TestDemoVerifiedHitDoesNotEraseExistingHostileBullet(t *testing.T) {
	g := armedFiringLaneGame(t)
	guardFiringLaneRandom(t, g)
	g.projectiles = []projectile{{x: 74, y: 14932, vy: 6, enemy: true, owner: -1, life: 15}}
	g.prepareDemoForecast()
	g.demoEnemyDefeat[0] = 3
	if g.demoPossibleShotCost(74, 14950, 4) != 0 {
		t.Fatal("controlled verified kill did not clear future possible fire")
	}
	if g.demoThreatAt(74, 14950, 4) <= 0 {
		t.Fatal("killing a shooter concealed an already flying ordinary hostile bullet")
	}
}

func TestDemoNonlethalOwnedHitKeepsTheFiringWarning(t *testing.T) {
	g := ordinaryStandoffGame(t)
	guardFiringLaneRandom(t, g)
	g.enemies[0].health = 2
	g.prepareDemoForecast()
	before := g.demoPossibleShotCost(g.player.x, g.player.y, 1)
	attack := playerControls{y: -1, fire: true}
	_, base := g.demoPlanMove(navPoint{126, 9912}, &attack)
	_, predicted := g.demoPlanAttack(navPoint{126, 9912}, &attack, 0)
	if before <= 0 || predicted != base {
		t.Fatal("nonlethal shot concealed unresolved firing risk or invented a completed kill")
	}
	if g.enemies[0].health != 2 || g.ammunition != 54 {
		t.Fatal("planning applied real damage or spent finite ammunition")
	}
}

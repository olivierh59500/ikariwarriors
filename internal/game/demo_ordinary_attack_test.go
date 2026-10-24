package game

import "testing"

// These positions reproduce the armed standoff observed in the original at
// tick3650. The ordinary update still owns contact order and ammunition use.
func ordinaryStandoffGame(t *testing.T) *Game {
	t.Helper()
	g := pendingShotGame(t)
	g.player = actor{x: 126, y: 9928, direction: 4, health: 1}
	g.cameraY, g.cameraBackY = 9794, 9794
	g.secondaryTerrain = g.cameraY < secondaryMapRows*tileSize
	g.ammunition, g.grenades = 54, 50
	g.bulletPower[0], g.grenadePower[0] = 1, 1
	g.enemies = []actor{{x: 123, y: 9905, kind: 0, slot: 8, health: 1, shots: 2}}
	g.pickups = nil
	g.eventIndex = len(g.resources.events)
	g.randomSpawn, g.redSpawn = false, false
	return g
}

func TestDemoOrdinaryFireForecastClearsStoppedInfantry(t *testing.T) {
	g := ordinaryStandoffGame(t)
	target := navPoint{126, 9912}
	attack := playerControls{y: -1, fire: true}
	g.prepareDemoForecast()
	player, enemy, lives, ammo, grenades, random := g.player, g.enemies[0], g.lives, g.ammunition, g.grenades, g.random
	_, withoutKill := g.demoPlanMove(target, &attack)
	_, withKill := g.demoPlanAttack(target, &attack, 0)
	if withKill <= withoutKill+100000 {
		t.Fatalf("the ordinary lethal shot did not clear the future route: without%g with%g", withoutKill, withKill)
	}
	if g.player != player || g.enemies[0] != enemy || g.lives != lives || g.ammunition != ammo || g.grenades != grenades || g.random != random || len(g.projectiles) != 0 {
		t.Fatal("planning changed the physical scene or finite resources")
	}
	g.demoForecastActive = false
	chosen := g.demoControls()
	if !chosen.fire || chosen.action || chosen.x != 0 || chosen.y != -1 {
		t.Fatalf("the real demo hook did not select the safe north shot: %+v", chosen)
	}
	g.stepPlay([2]playerControls{chosen})
	if g.player.deadTicks != 0 || g.player.health != 1 || g.lives != lives || g.ammunition != ammo-1 {
		t.Fatalf("actual ordinary shot failed: player%+v lives%d ammo%d", g.player, g.lives, g.ammunition)
	}
	for _, enemy := range g.enemies {
		if enemy.slot == 8 && enemy.health > 0 {
			t.Fatal("the predicted infantry target survived the real shot")
		}
	}
}

func TestDemoOrdinaryFireCannotCancelItsFirstInputContact(t *testing.T) {
	g := ordinaryStandoffGame(t)
	g.enemies[0].y = g.player.y - 18
	attack := playerControls{y: -1, fire: true}
	g.prepareDemoForecast()
	_, score := g.demoPlanAttack(navPoint{126, 9912}, &attack, 0)
	if score != demoBlockedScore {
		t.Fatalf("a future kill excused an existing body contact: score%g", score)
	}
	g.demoForecastActive = false
	g.stepPlay([2]playerControls{attack})
	if g.player.deadTicks == 0 {
		t.Fatal("the real pre-projectile body contact unexpectedly became safe")
	}
}

func TestDemoOrdinaryKillKeepsAlreadyFiredHostileBullets(t *testing.T) {
	g := ordinaryStandoffGame(t)
	g.projectiles = []projectile{{x: 126, y: 9952, vy: -6, life: 12, enemy: true, owner: -1, slot: 0}}
	attack := playerControls{y: -1, fire: true}
	g.prepareDemoForecast()
	_, score := g.demoPlanAttack(navPoint{126, 9912}, &attack, 0)
	if score >= float64(demoHorizon-1)*10000 {
		t.Fatalf("killing the shooter concealed its existing bullet: score%g", score)
	}
	g.demoForecastActive = false
	g.stepPlay([2]playerControls{attack})
	if g.player.deadTicks != 0 {
		t.Fatal("the controlled hostile bullet hit before the first shot could kill its owner")
	}
	for _, enemy := range g.enemies {
		if enemy.slot == 8 && enemy.health > 0 {
			t.Fatal("the real ordinary shot did not kill the hostile bullet's owner")
		}
	}
	g.stepPlay([2]playerControls{})
	if g.player.deadTicks == 0 {
		t.Fatal("killing the shooter removed its real in-flight projectile")
	}
}

func TestDemoOrdinaryAttackDoesNotInventNonlethalOrInterceptedKills(t *testing.T) {
	for _, intercepted := range []bool{false, true} {
		g := ordinaryStandoffGame(t)
		if intercepted {
			g.enemies = append([]actor{{x: 123, y: 9906, kind: 2, slot: 3, health: 2}}, g.enemies...)
		} else {
			g.enemies[0].health = 2
		}
		attack := playerControls{y: -1, fire: true}
		g.prepareDemoForecast()
		_, baseline := g.demoPlanMove(navPoint{126, 9912}, &attack)
		_, forecast := g.demoPlanAttack(navPoint{126, 9912}, &attack, 0)
		if forecast != baseline {
			t.Fatalf("nonlethal first impact changed body survival, intercepted%v: baseline%g forecast%g", intercepted, baseline, forecast)
		}
		g.demoForecastActive = false
		g.stepPlay([2]playerControls{attack})
		if g.enemies[0].health != 1 {
			t.Fatalf("real first target has health%d, expected one remaining", g.enemies[0].health)
		}
		if intercepted && g.enemies[1].health != 1 {
			t.Fatal("a bullet also killed the actor behind its first impact")
		}
	}
}

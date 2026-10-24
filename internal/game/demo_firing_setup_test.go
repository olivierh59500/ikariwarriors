package game

import (
	"reflect"
	"testing"
)

// This native policy fixture is the actual failed river bank at tick188.
// It is not an Amiga oracle; model rules and existing gun/mine data stay intact.
func knownGunSetupScene(t *testing.T) *Game {
	t.Helper()
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGame(false)
	g.scene = playScene
	g.player = actor{x: 194, y: 8969, direction: 0, health: 1, wet: true}
	g.cameraY, g.cameraBackY = 8874, 8896
	g.secondaryTerrain = g.cameraY < secondaryMapRows*tileSize
	g.ammunition, g.grenades = 6, 45
	g.enemies = []actor{{x: 170, y: 8920, kind: 2, slot: 3, health: 1}, {x: 92, y: 8890, kind: 2, slot: 4, health: 1}}
	g.projectiles = []projectile{{x: 144, y: 8958, vx: 4, vy: 4, enemy: true, owner: -1, slot: 3, life: 1}}
	g.mines = []landmine{{x: 166, y: 8994, kind: 19, slot: 83}, {x: 172, y: 9004, kind: 19, slot: 84}}
	g.pickups = nil
	g.eventIndex, g.randomSpawn, g.redSpawn = len(g.resources.events), false, false
	return g
}

func TestKnownGunSetupKeepsAllThreatsAndLiveState(t *testing.T) {
	g := knownGunSetupScene(t)
	guardFiringLaneRandom(t, g)
	g.prepareDemoForecast()
	player, enemies, shots, mines := g.player, append([]actor(nil), g.enemies...), append([]projectile(nil), g.projectiles...), append([]landmine(nil), g.mines...)
	ammo, grenades, lives, random, index, defeats := g.ammunition, g.grenades, g.lives, g.random, g.randomSourceIndex, g.demoEnemyDefeat
	normal := g.demoPlanMoveDetails(navPoint{164, 9011}, nil)
	control, locked, found := g.demoKnownGunFiringSetup(170, 8920, normal)
	if !found || control.fire || control.action || control.x == 0 && control.y == 0 || locked {
		t.Fatalf("no ordinary uncommitted firing setup was found:%+v locked%v found%v", control, locked, found)
	}
	x, y, valid := g.demoStep(g.player.x, g.player.y, control.x, control.y)
	probe := g.player
	probe.x, probe.y = x, y
	if !valid || g.demoThreatForMove(probe, g.player.y, 1) > 0 {
		t.Fatal("setup discarded a known first-input hazard")
	}
	if g.player != player || !reflect.DeepEqual(g.enemies, enemies) || !reflect.DeepEqual(g.projectiles, shots) || !reflect.DeepEqual(g.mines, mines) || g.ammunition != ammo || g.grenades != grenades || g.lives != lives || g.random != random || g.randomSourceIndex != index || g.demoEnemyDefeat != defeats {
		t.Fatal("setup changed live world/resources/RNG or promised an uncommitted future kill")
	}
}

func TestKnownGunSetupDoesNotRepeatARealPendingLethalShot(t *testing.T) {
	g := knownGunSetupScene(t)
	g.player.x, g.player.y, g.player.direction = 193, 8967, 7
	shot := g.projectileFor(&g.player, false, 0, false, 6)
	g.projectiles = []projectile{shot}
	guardFiringLaneRandom(t, g)
	g.prepareDemoForecast()
	if g.demoPendingBulletTargets()&1 == 0 {
		t.Fatal("controlled real owned trajectory does not cover the gun")
	}
	beforeAmmo, beforeShots := g.ammunition, append([]projectile(nil), g.projectiles...)
	normal := g.demoPlanMoveDetails(navPoint{164, 8967}, nil)
	if _, _, found := g.demoKnownGunFiringSetup(170, 8920, normal); found {
		t.Fatal("an already covered gun requested another setup or shot")
	}
	if g.ammunition != beforeAmmo || !reflect.DeepEqual(g.projectiles, beforeShots) {
		t.Fatal("pending-shot decision spent ammunition or changed its flight")
	}
}

func TestKnownGunSetupCannotExcuseAnExistingImmediateBullet(t *testing.T) {
	g := knownGunSetupScene(t)
	g.projectiles = []projectile{{x: g.player.x, y: g.player.y, vy: 4, enemy: true, owner: -1, life: 15}}
	guardFiringLaneRandom(t, g)
	g.prepareDemoForecast()
	normal := g.demoPlanMoveDetails(navPoint{164, 9011}, nil)
	if _, _, found := g.demoKnownGunFiringSetup(170, 8920, normal); found {
		t.Fatal("an eventual firing pose concealed an existing immediate bullet contact")
	}
	if g.player.health != 1 || g.lives != 5 {
		t.Fatal("query applied a real hit or changed protection")
	}
}

package game

import "testing"

// This controlled native roof-objective scene comes from the observed second-cycle
// stall at88/12910. It is not an original-game oracle. It preserves the real
// map, roof building, hostile bullet and finite source-style resources.
func TestStalledCombatClearsTheObservedRoofThroughRealScenery(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGame(true)
	g.scene = playScene
	g.player = actor{x: 88, y: 12910, direction: 2, health: 1}
	g.cameraY, g.cameraBackY = 12816, 12832
	g.secondaryTerrain = g.cameraY < secondaryMapRows*tileSize
	g.ammunition, g.grenades = 27, 96
	g.enemies = []actor{{x: 52, y: 12850, kind: 2, slot: 3, health: 1}, {x: 108, y: 12880, kind: 6, slot: 32, health: 1, bob: 99040}}
	g.projectiles = []projectile{{x: 68, y: 12912, vx: -4, vy: 4, enemy: true, owner: -1, slot: 0, life: 6}}
	g.mines, g.pickups = nil, nil
	g.eventIndex, g.randomSpawn, g.redSpawn = len(g.resources.events), false, false
	g.idleArtilleryTicks = 120
	row, column, _, found := g.sceneryBounds(116, 12876)
	if !found {
		t.Fatal("observed rooftop gun is not tied to a real composite")
	}
	spent := 0
	for tick := 0; tick < 300; tick++ {
		before := g.grenades
		g.tick++
		g.stepPlay([2]playerControls{})
		if g.grenades < before {
			spent++
		}
		if g.lives != 5 || g.player.health != 1 || g.player.deadTicks != 0 || g.player.invulnerable != 0 {
			t.Fatalf("stalled combat response lost ordinary safety at%d: player%+v", tick, g.player)
		}
		roofAlive := false
		for _, enemy := range g.enemies {
			roofAlive = roofAlive || enemy.kind == 6 && enemy.health > 0
		}
		if g.destroyed[row][column] && !roofAlive {
			if spent == 0 {
				t.Fatal("roof disappeared without spending an actual finite grenade")
			}
			return
		}
	}
	t.Fatalf("observed roof combat remains blocked: hero%.0f/%.0f ammo%d grenades%d target%d/%d scenery%v path%d/%d", g.player.x, g.player.y, g.ammunition, g.grenades, g.demoSceneryRow, g.demoSceneryCol, g.demoPlanScenery, g.demoPathIndex, len(g.demoPath))
}

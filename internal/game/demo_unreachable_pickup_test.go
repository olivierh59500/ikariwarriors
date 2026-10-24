package game

import "testing"

// Normally played original run14 exposed this weapon pickup across an
// impassable wall. The original terrain still allows progress on the right;
// its optional upgrade must not strand an otherwise healthy tank indefinitely.
func TestDemoContinuesPastAnUnreachableOriginalWeaponPickup(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGame(true)
	g.scene = playScene
	g.player = actor{x: 128, y: 11510, health: 1, inTank: true, tankSlot: 19, tankFuel: 57}
	g.cameraY, g.cameraBackY = 11380, 11392
	g.secondaryTerrain = g.cameraY < secondaryMapRows*tileSize
	g.enemies, g.pickups, g.mines = nil, []pickup{{x: 98, y: 11510, kind: 6}}, []landmine{{x: 124, y: 11598, kind: 18, slot: 81}, {x: 184, y: 11584, kind: 18, slot: 82}, {x: 100, y: 11500, kind: 19, slot: 83}}
	g.eventIndex = len(g.resources.events)
	g.randomSpawn, g.redSpawn = false, false
	g.buildDemoPath()
	if len(g.demoPath) < 2 || g.demoPath[len(g.demoPath)-1].y >= g.player.y-32 {
		t.Fatalf("unreachable pickup blocked the open route: path%+v", g.demoPath)
	}
	initialY := g.player.y
	for range 64 {
		g.stepPlay([2]playerControls{})
	}
	if g.player.y >= initialY-32 || g.player.deadTicks != 0 || g.lives != 5 || g.bulletPower[0] != 1 {
		t.Fatalf("ordinary bypass failed: player%+v lives%d power%d pickups%+v", g.player, g.lives, g.bulletPower[0], g.pickups)
	}
}

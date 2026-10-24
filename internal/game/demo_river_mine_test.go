package game

import "testing"

// The second natural attract cycle respawned below this live mine, while the
// two covering guns remained alive. This controlled encounter exercises the same map,
// gun pair and mine positions without invulnerability or altered physics.
func TestRiverMineDetourAfterRespawn(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGame(true)
	g.scene = playScene
	g.player = actor{x: 105, y: 9327, direction: 0, health: 1}
	g.cameraY, g.cameraBackY = 9186, 9216
	g.secondaryTerrain = g.cameraY < secondaryMapRows*tileSize
	g.enemies = []actor{{x: 172, y: 9220, kind: 2, slot: 3, health: 1, direction: 5}, {x: 112, y: 9190, kind: 2, slot: 4, health: 1, direction: 4}}
	g.projectiles, g.pickups = nil, nil
	g.mines = []landmine{{x: 44, y: 9412, kind: 19, slot: 83}, {x: 104, y: 9314, kind: 19, slot: 84}}
	g.eventIndex, g.randomSpawn, g.redSpawn = len(g.resources.events), false, false
	for tick := 0; tick < 180; tick++ {
		g.stepPlay([2]playerControls{})
		if g.player.health != 1 || g.player.deadTicks != 0 || g.player.invulnerable != 0 || g.lives != 5 {
			t.Fatalf("detour entered mine/crossfire: tick%d player%+v lives%d", tick, g.player, g.lives)
		}
		minePresent := false
		for _, mine := range g.mines {
			if mine.slot == 84 {
				minePresent = true
			}
		}
		if !minePresent {
			t.Fatalf("detour triggered or removed the actual blocking mine at tick%d", tick)
		}
		gunAlive := false
		for _, enemy := range g.enemies {
			if enemy.kind == 2 && enemy.health > 0 {
				gunAlive = true
			}
		}
		if !gunAlive {
			if g.player.y < 9260 || g.ammunition != 97 || g.grenades != 50 || g.score != 200 {
				t.Fatalf("covering guns not cleared safely with two ordinary bullets: tick%d player%+v ammo%d grenades%d", tick, g.player, g.ammunition, g.grenades)
			}
			t.Logf("both guns cleared after %d real updates at %.0f/%.0f; mine remained intact", tick+1, g.player.x, g.player.y)
			return
		}
	}
	t.Fatalf("live mine blocked the remembered approach: player%+v enemies%+v", g.player, g.enemies)
}

package game

import "testing"

// This is a controlled native strategy regression on the recovered map. The
// original constructor and its hostile effects have separate numerical oracles.
func TestOpeningBunkerIsClearedWithOneOrdinaryGrenade(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGame(true)
	g.scene = playScene
	g.player = actor{x: 104, y: 14912, direction: 0, health: 1}
	g.cameraY, g.cameraBackY = 14820, 14848
	g.secondaryTerrain = g.cameraY < secondaryMapRows*tileSize
	g.enemies = []actor{{x: 94, y: 14846, slot: 32, kind: 6, health: 1}}
	g.pickups, g.mines = nil, nil
	g.eventIndex, g.randomSpawn, g.redSpawn = len(g.resources.events), false, false
	seenScenery, seenHostileRadial := false, false
	for tick := 1; tick <= 40; tick++ {
		g.stepPlay([2]playerControls{})
		if g.player.health != 1 || g.player.deadTicks != 0 || g.player.invulnerable != 0 || g.lives != 5 {
			t.Fatalf("ordinary bunker attack was unsafe at tick%d: player%+v reserves%d", tick, g.player, g.lives)
		}
		seenScenery = seenScenery || g.sceneryBurst.active
		seenHostileRadial = seenHostileRadial || g.burst.active && g.burst.owner == -1
		if tick == 17 && len(g.enemies) != 1 {
			t.Fatal("the gun disappeared before the ordinary grenade landed")
		}
		if tick == 18 && (len(g.enemies) != 0 || !g.destroyed[926][3] || !g.sceneryBurst.active || g.sceneryBurst.remaining != 7) {
			t.Fatalf("ordinary landing did not clear the bunker and its gun: enemies%+v scenery%+v", g.enemies, g.sceneryBurst)
		}
	}
	for row := 926; row <= 928; row++ {
		for column := 3; column <= 7; column++ {
			if !g.destroyed[row][column] {
				t.Fatalf("ordinary explosive left bunker cell%d/%d intact", row, column)
			}
		}
	}
	if !seenScenery || !seenHostileRadial || g.sceneryBurst.active || g.burst.active || len(g.burstQueue) != 0 {
		t.Fatal("the complete hostile scenery and radial sequence was not observed")
	}
	if g.grenades != 49 || g.ammunition != 99 || g.score != 1000 {
		t.Fatalf("one ordinary grenade must clear scenery without a direct gun kill bonus: ammo%d grenades%d score%d", g.ammunition, g.grenades, g.score)
	}
}

func TestOpeningBunkerGuideClearsBeforeEnteringTheGunRange(t *testing.T) {
	g, err := New(Options{Demo: true, Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	seenGun, cleared := false, false
	bunkerThrows := 0
	for g.gameplayTicks < 600 {
		beforeGrenades := g.grenades
		if err = g.Update(); err != nil {
			t.Fatal(err)
		}
		// Ordinary combat before reaching the bunker can spend a grenade too.
		// Count the guide's actual bunker throw, separately from those attacks.
		if g.grenades < beforeGrenades && g.demoPlanScenery && g.demoSceneryRow == 926 && g.demoSceneryCol == 3 {
			bunkerThrows++
		}
		gunPresent := false
		for _, enemy := range g.enemies {
			gunPresent = gunPresent || enemy.slot == 32 && enemy.kind == 6 && enemy.x == 94 && enemy.y == 14846
		}
		seenGun = seenGun || gunPresent
		if seenGun && !gunPresent && !cleared {
			cleared = true
			if !g.destroyed[926][3] || bunkerThrows != 1 || g.player.y < 14910 || g.lives != 5 || g.player.health != 1 || g.player.deadTicks != 0 || g.player.invulnerable != 0 {
				t.Fatalf("guide entered the firing lane before its ordinary bunker clearance: player%+v lives%d grenades%d", g.player, g.lives, g.grenades)
			}
		}
	}
	if !cleared || g.lives != 5 || g.player.deadTicks != 0 || g.player.y > 14200 {
		t.Fatalf("remembered opening route failed to continue: cleared%t lives%d player%+v", cleared, g.lives, g.player)
	}
}

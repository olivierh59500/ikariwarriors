package game

import "testing"

// The first dry gun covers a narrow corridor. Clear it before committing to
// that corridor; backing away inside it leaves no lateral bullet escape.
func TestRememberedRiverEntryClearsTheNarrowApproachGun(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGame(true)
	g.scene = playScene
	g.player = actor{x: 130, y: 9310, health: 1}
	g.cameraY, g.cameraBackY = 9216, 9226
	g.secondaryTerrain = g.cameraY < secondaryMapRows*tileSize
	g.enemies = []actor{{x: 112, y: 9190, kind: 2, slot: 4, health: 1}}
	g.projectiles, g.pickups, g.mines = nil, nil, nil
	g.eventIndex, g.randomSpawn, g.redSpawn = len(g.resources.events), false, false
	for tick := 0; tick < 180; tick++ {
		g.stepPlay([2]playerControls{})
		if g.player.deadTicks != 0 || g.player.health != 1 || g.lives != 5 || g.player.invulnerable != 0 {
			t.Fatalf("approach entered the gun's crossfire: player %+v", g.player)
		}
		if len(g.enemies) == 0 || g.enemies[0].health <= 0 {
			if g.score != 100 || g.ammunition != 98 || g.grenades != 50 || g.player.y < 9260 {
				t.Fatalf("gun was not cleared from the dry approach with one ordinary shot: player %+v score%d ammo%d grenades%d", g.player, g.score, g.ammunition, g.grenades)
			}
			return
		}
	}
	t.Fatalf("approach did not clear the original fixed gun: player %+v enemies %+v", g.player, g.enemies)
}

// This controlled encounter uses the real map, gun positions, mine pair and
// enemy firing rules. The remembered approach must clear both guns before
// committing to the slow crossing, using only two ordinary shots.
func TestRememberedRiverApproachClearsTheGunsBeforeCrossing(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGame(true)
	g.scene = playScene
	g.player = actor{x: 130, y: 9012, health: 1}
	g.cameraY, g.cameraBackY = 8918, 8928
	g.secondaryTerrain = g.cameraY < secondaryMapRows*tileSize
	g.ammunition, g.grenades = 6, 45
	g.enemies = []actor{
		{x: 170, y: 8920, kind: 2, slot: 3, health: 1},
		{x: 92, y: 8890, kind: 2, slot: 4, health: 1},
	}
	g.projectiles, g.pickups = nil, nil
	g.mines = []landmine{{x: 74, y: 8998, kind: 19, slot: 83}, {x: 144, y: 8996, kind: 19, slot: 84}}
	g.eventIndex = len(g.resources.events)
	g.randomSpawn, g.redSpawn = false, false
	for tick := 0; tick < 240; tick++ {
		g.stepPlay([2]playerControls{})
		if g.player.deadTicks != 0 || g.player.health != 1 || g.lives != 5 || g.player.invulnerable != 0 || len(g.mines) != 2 {
			t.Fatalf("remembered approach entered the crossfire or mine pair: player %+v lives %d mines %+v", g.player, g.lives, g.mines)
		}
		alive := false
		for _, enemy := range g.enemies {
			alive = alive || enemy.health > 0
		}
		if !alive {
			if g.score != 200 || g.ammunition != 4 || g.grenades != 45 {
				t.Fatalf("two-gun approach changed finite resources: score %d ammo %d grenades %d", g.score, g.ammunition, g.grenades)
			}
			return
		}
	}
	t.Fatalf("remembered approach did not clear both guns: player %+v enemies %+v", g.player, g.enemies)
}

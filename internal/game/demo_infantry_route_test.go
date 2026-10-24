package game

import "testing"

// This is a naturally reached native standoff. The original terrain is used;
// no actor, reserve, protection, ammunition or grenade grant is made by the
// planner. A controlled continuation isolates the local detour from spawns.
func TestUnarmedDemoFlanksStoppedGrenadier(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGame(true)
	g.scene = playScene
	g.player = actor{x: 44, y: 1292, health: 1}
	g.cameraY, g.cameraBackY = 1198, 1216
	g.secondaryTerrain = g.cameraY < secondaryMapRows*tileSize
	g.ammunition, g.grenades = 0, 57
	g.bulletPower[0], g.grenadePower[0] = 2, 2
	g.enemies = []actor{{x: 52, y: 1270, health: 1, kind: 3, slot: 0, direction: 4}}
	g.pickups = nil
	g.mines = []landmine{{x: 64, y: 1390, kind: 18, slot: 82}, {x: 104, y: 1294, kind: 20, slot: 85}}
	g.eventIndex = len(g.resources.events)
	g.randomSpawn, g.redSpawn = false, false
	g.buildDemoPath()
	for tick := 0; tick < 128; tick++ {
		g.stepPlay([2]playerControls{})
		if g.player.deadTicks > 0 || g.lives != 5 || g.player.invulnerable != 0 {
			t.Fatalf("ordinary flank lost health at%d: player%+v lives%d", tick, g.player, g.lives)
		}
		if g.player.y < 1238 {
			t.Logf("flanked after%d updates at%.0f/%.0f, ammo%d grenades%d", tick+1, g.player.x, g.player.y, g.ammunition, g.grenades)
			return
		}
	}
	t.Fatalf("stationary soldier still blocks the route at%.0f/%.0f with ammo%d grenades%d", g.player.x, g.player.y, g.ammunition, g.grenades)
}

func TestDemoInfantryRouteMarginAllowsOnlyEscape(t *testing.T) {
	bodies := []navPoint{{50, 50}}
	for _, tc := range []struct {
		name         string
		x, y, nx, ny float64
		clear        bool
	}{
		{"enter", 50, 72, 50, 70, false}, {"retreat", 50, 70, 50, 72, true}, {"sidestep", 50, 70, 48, 70, true}, {"deeper", 50, 70, 50, 68, false}, {"outside", 24, 72, 24, 70, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := demoRouteClearsInfantry(bodies, tc.x, tc.y, tc.nx, tc.ny); got != tc.clear {
				t.Fatalf("clear=%v want%v", got, tc.clear)
			}
		})
	}
}

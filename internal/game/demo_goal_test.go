package game

import "testing"

func controlledDemoGoalBoundary(t *testing.T) *Game {
	t.Helper()
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGame(true)
	g.player = actor{x: 178, y: 12900, health: 1, direction: 0}
	g.enemies = []actor{{x: 98, y: 12782, kind: 4, slot: 20, health: 1, tankFuel: 64}}
	g.ammunition, g.grenades = 91, 47
	g.demoPlanScenery, g.demoSceneryRow, g.demoSceneryCol = true, 803, 4
	g.demoTarget, g.demoTargetKind = navPoint{98, 12782}, 1
	return g
}

// The naturally retained second cycle alternated at these ordinary positions:
// the same useful tank is 198 then 200 pixels away. A valid firing detour must
// retain its selected goal when an ordinary step crosses that boundary.
func TestSceneryApproachKeepsItsValidTankAcrossTheSelectionRadius(t *testing.T) {
	g := controlledDemoGoalBoundary(t)
	target, ok := g.demoDestination()
	if !ok || target != g.demoTarget {
		t.Fatal("useful tank was not selected inside the acquisition radius")
	}
	g.stepPlayer(0, playerControls{y: 1})
	if g.player.y != 12902 {
		t.Fatalf("ordinary two-pixel detour changed: %+v", g.player)
	}
	target, ok = g.demoDestination()
	if !ok || target != g.demoTarget {
		t.Fatal("two-pixel firing detour dropped its valid selected tank")
	}
	g.demoPlanScenery = false
	if _, ok := g.demoDestination(); ok {
		t.Fatal("completed approach enlarged the ordinary acquisition range")
	}
	g.demoPlanScenery = true
	g.enemies[0].health = 0
	if _, ok := g.demoDestination(); ok {
		t.Fatal("approach retained an unavailable tank")
	}
	g.enemies[0].health = 1
	g.player.y = 13064
	if _, ok := g.demoDestination(); ok {
		t.Fatal("approach retained a tank beyond its bounded detour range")
	}
}

func TestDemoActuallyClearsTheGateBeforeItsOptionalTank(t *testing.T) {
	g := controlledDemoGoalBoundary(t)
	// Isolate the observed geometry from unrelated earlier event waves. This
	// is a controlled actual-Update regression, not a complete ordinary run.
	g.scene = playScene
	g.cameraY, g.cameraBackY = 12806, 12832
	g.secondaryTerrain = g.cameraY < secondaryMapRows*tileSize
	g.eventIndex, g.randomSpawn, g.redSpawn = len(g.resources.events), false, false
	g.demoPath = nil
	lives, ammunition, grenades := g.lives, g.ammunition, g.grenades
	for tick := 0; tick < 120 && !g.destroyed[803][4]; tick++ {
		if err := g.Update(); err != nil {
			t.Fatal(err)
		}
		if g.player.deadTicks > 0 || g.lives != lives {
			t.Fatalf("controlled approach became unsafe: %+v", g.player)
		}
	}
	if !g.destroyed[803][4] || g.ammunition != ammunition || g.grenades != grenades-1 {
		t.Fatalf("did not clear the actual gate with one ordinary grenade: player%+v ammunition%d grenades%d", g.player, g.ammunition, g.grenades)
	}
}

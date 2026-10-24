package game

import (
	"encoding/json"
	"os"
	"testing"
)

func TestOriginalTerrainProbeRange(t *testing.T) {
	data, err := os.ReadFile("testdata/terrain-bounds.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct{ RawX, WorldX, WorldY, Code int }
	if err = json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, original := range cases {
		if got := g.terrainCode(float64(original.WorldX), float64(original.WorldY)); got != byte(original.Code) {
			t.Fatalf("original rawX%d returned%d, native%d", original.RawX, original.Code, got)
		}
		if original.RawX < 48 || original.RawX > 250 {
			if g.solidAt(float64(original.WorldX), float64(original.WorldY)) {
				t.Fatal("outside the original terrain lookup became an artificial wall")
			}
		}
	}
}

func TestOriginalProjectileCanCrossTheMapEdgeBeforeLeavingTheDisplay(t *testing.T) {
	data, err := os.ReadFile("testdata/projectile-bounds.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Kind                    string
		Tick, Range, QuadActive int
		Shot                    struct{ X, Y, Active int }
	}
	if err = json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	var g *Game
	kind := ""
	for _, original := range cases {
		if kind != original.Kind {
			kind = original.Kind
			g, err = New(Options{Mute: true})
			if err != nil {
				t.Fatal(err)
			}
			g.cameraY = 10000
			velocity := 4.0
			if kind == "left" {
				velocity = -4
			}
			g.projectiles = []projectile{{x: float64(original.Shot.X - 56), y: g.cameraY + float64(original.Shot.Y), vx: velocity, life: original.Range / 4, enemy: true, owner: -1, grenade: true, frame: 9}}
		}
		if original.Tick > 0 {
			g.updateProjectiles()
		}
		var flight, blast *projectile
		for i := range g.projectiles {
			flight = &g.projectiles[i]
		}
		if g.blasts[2].impact > 0 {
			blast = &g.blasts[2]
		}
		if (flight != nil) != (original.Shot.Active != 0) || (blast != nil) != (original.QuadActive != 0) {
			t.Fatalf("%s tick%d original flight/blast lifetime differs", kind, original.Tick)
		}
		if flight != nil && (int(flight.x+56) != original.Shot.X || int(flight.y-g.cameraY) != original.Shot.Y || flight.life*4 != original.Range) {
			t.Fatalf("%s tick%d original projectile position differs", kind, original.Tick)
		}
	}
}

func TestDepartingSoldierLeavesTheOriginalMapAndDisplay(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.cameraY = 10000
	g.enemies = []actor{{x: 204, y: g.cameraY + 20, kind: 0, health: 1, slot: 5, marchX: 2, marchTime: 60}}
	g.updateEnemies()
	g.pruneOffscreenEnemies()
	if len(g.enemies) != 1 || g.enemies[0].x != 206 {
		t.Fatal("a departing platoon hit an invented wall at the map boundary")
	}
	g.enemies[0].x = 270
	g.updateEnemies()
	g.pruneOffscreenEnemies()
	if len(g.enemies) != 1 || g.enemies[0].x != 272 {
		t.Fatal("the original rawX320 boundary was removed early")
	}
	g.updateEnemies()
	g.pruneOffscreenEnemies()
	if len(g.enemies) != 0 {
		t.Fatal("the departing soldier remained active beyond original rawX320")
	}
}

func TestOriginalPlayerMovementBoundsPreserveTheCurrentPosition(t *testing.T) {
	data, err := os.ReadFile("testdata/player-bounds.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct{ Player, StartRawX, Direction, ResultRawX, Progress int }
	if err = json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGamePlayers(false, 2)
	for _, original := range cases {
		value := &g.player
		if original.Player == 1 {
			value = &g.player2
		}
		value.x, value.y, value.health = float64(original.StartRawX-48), float64(levelHeight-16-original.Progress), 1
		g.moveActor(value, float64(original.Direction*2), 0)
		if got := int(value.x + 48); got != original.ResultRawX {
			t.Fatalf("player%d rawX%d input%d became%d, original%d", original.Player, original.StartRawX, original.Direction, got, original.ResultRawX)
		}
	}
}

func TestDemoReturnsFromAnOriginalBoardingPositionOutsideTheWalkingRange(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGame(true)
	g.player.x = 212
	g.player.inTank, g.player.tankFuel = true, 64
	g.enemies = nil
	for range 12 {
		g.stepPlay([2]playerControls{})
	}
	if g.player.x > 194 || g.player.health != 1 || g.lives != 5 {
		t.Fatal("the demonstration could not return normally from an outside boarding position")
	}
}

func TestNavigationDoesNotTurnAnOriginalBlockedInputIntoAnOutsideRoute(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGame(true)
	g.player.x = 194
	g.enemies, g.mines = nil, nil
	if _, _, advances := g.demoStep(g.player.x, g.player.y, 1, 0); advances {
		t.Fatal("the original blocked right input became a fictitious route edge")
	}
	g.buildDemoPath()
	for _, point := range g.demoPath {
		if point.x < 13 || point.x > 195 {
			t.Fatal("the demonstration planned outside the actual player movement range")
		}
	}
}

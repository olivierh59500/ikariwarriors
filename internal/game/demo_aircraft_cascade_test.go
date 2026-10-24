package game

import (
	"encoding/json"
	"os"
	"testing"
)

// The recovered runtime rules permit a detached helicopter tail to be
// struck by the queued front-section fragments. This native gameplay fixture
// isolates that observed deterministic chain; no future random state is stored.
func loadAircraftCascadeState(t *testing.T) *Game {
	return loadAircraftCascadeFixture(t, "testdata/demo-aircraft-cascade.json")
}

func loadAircraftCascadeFixture(t *testing.T, path string) *Game {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var source struct {
		IncludeInfantry             bool
		Enemies                     []unsafeAttackActor
		GameplayTicks               int
		CameraY, CameraBackY        float64
		Lives, Ammunition, Grenades int
		Player                      unsafeAttackActor
		BulletPower, GrenadePower   [2]int
		Projectiles                 []unsafeAttackShot
		Mines                       []struct {
			X, Y       float64
			Kind, Slot int
		}
		DestroyedCells [][2]int
		Aircraft       struct {
			X, Y         float64
			Phase, Owner int
			Flying       bool
			Alive        [5]bool
			Damaged      byte
		}
		Burst struct {
			Active           bool
			Owner, Remaining int
			Parts            [8]struct{ X, Y float64 }
		}
		BurstQueue []struct {
			X, Y  float64
			Owner int
		}
		SelectedWaypoint struct{ X, Y float64 }
		DemoPath         []struct{ X, Y float64 }
		DemoPathIndex    int
		DemoTarget       struct{ X, Y float64 }
		DemoTargetKind   int
	}
	if err := json.Unmarshal(data, &source); err != nil {
		t.Fatal(err)
	}
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGame(true)
	g.scene, g.gameplayTicks = playScene, int(source.GameplayTicks)
	g.player = source.Player.actor()
	g.cameraY, g.cameraBackY = source.CameraY, source.CameraBackY
	g.secondaryTerrain = g.cameraY < secondaryMapRows*tileSize
	g.lives, g.ammunition, g.grenades = source.Lives, source.Ammunition, source.Grenades
	g.bulletPower, g.grenadePower = source.BulletPower, source.GrenadePower
	g.enemies, g.projectiles, g.pickups, g.mines = nil, nil, nil, nil
	if source.IncludeInfantry {
		for _, enemy := range source.Enemies {
			g.enemies = append(g.enemies, enemy.actor())
		}
	}
	g.eventIndex, g.randomSpawn, g.redSpawn = len(g.resources.events), false, false
	g.aircraft = helicopter{x: source.Aircraft.X, y: source.Aircraft.Y, phase: source.Aircraft.Phase,
		flying: source.Aircraft.Flying, alive: source.Aircraft.Alive, damaged: source.Aircraft.Damaged, owner: source.Aircraft.Owner}
	g.burst = radialBurst{active: source.Burst.Active, owner: source.Burst.Owner, remaining: source.Burst.Remaining}
	for i, part := range source.Burst.Parts {
		g.burst.parts[i] = navPoint{part.X, part.Y}
	}
	for _, request := range source.BurstQueue {
		g.burstQueue = append(g.burstQueue, burstRequest{x: request.X, y: request.Y, owner: request.Owner})
	}
	for _, shot := range source.Projectiles {
		g.projectiles = append(g.projectiles, projectile{x: shot.X, y: shot.Y, vx: shot.VX, vy: shot.VY,
			life: shot.Life, impact: shot.Impact, owner: shot.Owner, slot: shot.Slot, frame: shot.Frame,
			power: shot.Power, enemy: shot.Enemy, grenade: shot.Grenade, cannon: shot.Cannon, fragments: shot.Fragments})
	}
	for _, mine := range source.Mines {
		g.mines = append(g.mines, landmine{x: mine.X, y: mine.Y, kind: mine.Kind, slot: mine.Slot})
	}
	for _, cell := range source.DestroyedCells {
		g.destroyed[cell[0]][cell[1]] = true
	}
	g.demoPath = []navPoint{{source.SelectedWaypoint.X, source.SelectedWaypoint.Y}}
	g.demoPathIndex, g.demoPlanMines, g.demoPlanVehicle = 0, len(g.mines), false
	if source.IncludeInfantry {
		g.demoPath = nil
		for _, point := range source.DemoPath {
			g.demoPath = append(g.demoPath, navPoint{point.X, point.Y})
		}
		g.demoPathIndex = source.DemoPathIndex
		g.demoTarget, g.demoTargetKind = navPoint{source.DemoTarget.X, source.DemoTarget.Y}, source.DemoTargetKind
	}
	g.demoLastX, g.demoLastY = g.player.x-1, g.player.y
	return g
}

// The five original birth pairs establish one eight-pixel move at owned
// expiry. In these historical native fixtures the grenade lands at 5013 and
// its owned quad expires at 5016. The first burst moves six times through 5021,
// the queued front burst starts at 5022, then moves at 5023 and 5024.
func checkRecordedSingleBirthAircraftCascade(t *testing.T, g *Game, lives int) {
	t.Helper()
	if g.aircraft.alive != ([5]bool{false, false, false, true, false}) || g.aircraft.flying || g.aircraft.y != 9084 {
		t.Fatalf("single-birth cascade did not retain the stopped tail: %+v", g.aircraft)
	}
	if g.player.health != 1 || g.player.deadTicks != 0 || g.player.invulnerable != 0 || g.lives != lives || g.player.x != 130 || g.player.y != 9083 {
		t.Fatalf("historical controls lost their extra safe frame: player %+v lives %d", g.player, g.lives)
	}
	if g.blasts[2].impact != 0 || g.blasts[2].x != 152 || g.blasts[2].y != 9059 || !g.burst.active || g.burst.remaining != 5 || len(g.burstQueue) != 0 {
		t.Fatalf("front quad or first queued movement differs: quad %+v burst %+v queue %+v", g.blasts[2], g.burst, g.burstQueue)
	}
	// The front was hit before its last two-pixel move: 9082-30+7=9059.
	// These are the source's eight directions after exactly one eight-pixel move.
	want := [8]navPoint{{144, 9051}, {152, 9051}, {160, 9051}, {144, 9059}, {160, 9059}, {144, 9067}, {152, 9067}, {160, 9067}}
	if g.burst.parts != want {
		t.Fatalf("first queued movement %+v, want %+v", g.burst.parts, want)
	}
	// On its second movement the south fragment reaches 9059+16=9075,
	// inside the surviving tail at 9084-6=9078. Cleanup starts a quad at
	// 9078+7=9085 while the shared pool is busy, so damage remains immediate.
	// This cleanup follows the original fall pass, so the new counter remains at 24.
	g.stepPlay([2]playerControls{{y: 1}})
	if g.aircraft.alive[3] || g.player.health != 0 || g.player.deadTicks != 24 || g.lives != lives || g.blasts[2].impact != 3 || g.blasts[2].x != 152 || g.blasts[2].y != 9085 {
		t.Fatalf("actual delayed tail/quad chain differs: tail %v player %+v quad %+v", g.aircraft.alive[3], g.player, g.blasts[2])
	}
}

func TestRecordedAircraftCascadeUsesTheSingleBirthSchedule(t *testing.T) {
	g := loadAircraftCascadeState(t)
	g.demo = false
	lives := g.lives
	for step := 0; step < 16; step++ {
		control := playerControls{}
		if step == 0 {
			control.fire = true
		}
		if step >= 10 {
			control.y = 1
		}
		g.stepPlay([2]playerControls{control})
		if g.player.deadTicks > 0 {
			t.Fatalf("historical controls were hit before the corrected tail cleanup at %d: %+v", g.gameplayTicks, g.player)
		}
	}
	checkRecordedSingleBirthAircraftCascade(t, g, lives)
}

// Isolate the historical aircraft mechanism from the newly recovered moving
// mines. Complete cold demonstrations separately exercise all hazards together.
func TestRecordedGrenadeLaunchCreatesTheAircraftTrap(t *testing.T) {
	g := loadAircraftCascadeFixture(t, "testdata/demo-aircraft-cascade-launch.json")
	g.mines = nil
	// Keep independent helicopter fire out of this controlled grenade-chain
	// mechanism; complete cold runs retain the ordinary random state.
	g.random = 1
	g.demo = false
	lives := g.lives
	for tick := 4996; tick <= 5023; tick++ {
		control := playerControls{}
		if tick <= 5006 {
			control.y = -1
		}
		control.action = tick == 4996
		control.fire = tick == 4997 || tick == 4999 || tick == 5008
		if tick >= 5018 {
			control.y = 1
		}
		g.stepPlay([2]playerControls{control})
		if g.player.deadTicks > 0 {
			t.Fatalf("historical pre-throw controls were hit before the corrected tail cleanup at %d: %+v", tick, g.player)
		}
	}
	checkRecordedSingleBirthAircraftCascade(t, g, lives)
}

// This is a controlled mechanism check, not a normally played source oracle.
// A known damaged tail meets an already busy shared pool in actual stepPlay.
func TestAircraftFragmentsCanCreateAnImmediateHostileQuad(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGame(false)
	g.demo, g.randomSpawn, g.redSpawn = false, false, false
	g.eventIndex = len(g.resources.events)
	g.cameraY, g.cameraBackY = 8976, 8992
	g.secondaryTerrain = g.cameraY < secondaryMapRows*tileSize
	g.player = actor{x: 130, y: 9078, health: 1}
	g.enemies, g.projectiles, g.pickups, g.mines = nil, nil, nil, nil
	g.aircraft = helicopter{x: 152, y: 9084, alive: [5]bool{false, false, false, true, false}, damaged: 8, owner: -1}
	// A distant ordinary burst keeps the shared pool occupied without
	// contacting the player or aircraft before the requested cleanup.
	g.createBurst(20, g.cameraY-100, -1)
	lives, ammunition, grenades := g.lives, g.ammunition, g.grenades
	g.stepPlay([2]playerControls{})
	if g.aircraft.alive[3] || g.aircraft.damaged != 0 || !g.burst.active || g.burst.remaining != 5 || len(g.burstQueue) != 1 {
		t.Fatalf("busy cleanup did not queue its tail burst: aircraft %+v pool %+v queue %+v", g.aircraft, g.burst, g.burstQueue)
	}
	if request := g.burstQueue[0]; request.x != 152 || request.y != 9085 || request.owner != -1 {
		t.Fatalf("tail burst queued at %+v, want 152/9085 owner -1", request)
	}
	if g.blasts[2].impact != 3 || g.blasts[2].x != 152 || g.blasts[2].y != 9085 || g.player.health != 0 || g.player.deadTicks != 24 {
		t.Fatalf("busy tail cleanup did not cause its immediate hostile quad: player %+v quad %+v", g.player, g.blasts[2])
	}
	if g.lives != lives || g.ammunition != ammunition || g.grenades != grenades || g.player.invulnerable != 99 {
		t.Fatal("controlled cleanup changed reserves/weapons or the original first-hit protection")
	}
}

func TestDemoKeepsClearOfTheRecordedAircraftCascade(t *testing.T) {
	// At tick5008 the grenade has already committed a slow water crossing
	// to the source-correct delayed chain. Start at the genuine tick4996 launch decision instead:
	// ordinary anticipation must prevent the trap before the grenade flies.
	g := loadAircraftCascadeFixture(t, "testdata/demo-aircraft-cascade-launch.json")
	lives, ammunition, grenades := g.lives, g.ammunition, g.grenades
	for range 48 {
		g.stepPlay([2]playerControls{})
		if g.player.deadTicks != 0 || g.lives != lives || g.player.health != 1 {
			t.Fatalf("demonstration entered the visible helicopter chain at tick %d: player %+v", g.gameplayTicks, g.player)
		}
	}
	if g.ammunition > ammunition || g.grenades > grenades {
		t.Fatal("avoiding the explosion increased finite weapon resources")
	}
}

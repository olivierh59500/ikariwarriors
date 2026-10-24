package game

import (
	"encoding/json"
	"os"
	"testing"
)

type unsafeAttackActor struct {
	X, Y                                         float64
	Direction, Health, DeadTicks, Invulnerable   int
	Kind, Slot, MarchX, MarchY, MarchTime, Shots int
	TankSlot, TankFuel, TankBurn, TankDriver     int
	InTank, TankBurning, Wet                     bool
	Upper, Lower                                 struct{ Step, Delay, Frame int }
}

func (source unsafeAttackActor) actor() actor {
	return actor{x: source.X, y: source.Y, direction: source.Direction, health: source.Health,
		deadTicks: source.DeadTicks, invulnerable: source.Invulnerable, kind: source.Kind, slot: source.Slot,
		marchX: source.MarchX, marchY: source.MarchY, marchTime: source.MarchTime, shots: source.Shots,
		inTank: source.InTank, tankBurning: source.TankBurning, tankSlot: source.TankSlot,
		tankFuel: source.TankFuel, tankBurn: source.TankBurn, tankDriver: source.TankDriver, wet: source.Wet,
		upper: walkAnimation{step: source.Upper.Step, delay: source.Upper.Delay, frame: source.Upper.Frame},
		lower: walkAnimation{step: source.Lower.Step, delay: source.Lower.Delay, frame: source.Lower.Frame}}
}

type unsafeAttackShot struct {
	X, Y, VX, VY, TerrainBias               float64
	Life, Impact, Owner, Slot, Frame, Power int
	Enemy, Grenade, Cannon, Fragments       bool
}

// This fixture records native deterministic gameplay immediately before the
// old controller chose an unsafe diagonal at tick5239. It is an AI regression
// state, not an independent Amiga oracle. No random-generator state is stored.
func loadUnsafeAttackState(t *testing.T) (*Game, navPoint) {
	t.Helper()
	data, err := os.ReadFile("testdata/demo-unsafe-attack.json")
	if err != nil {
		t.Fatal(err)
	}
	var source struct {
		Player      unsafeAttackActor
		Enemies     []unsafeAttackActor
		Projectiles []unsafeAttackShot
		Mines       []struct {
			X, Y       float64
			Kind, Slot int
		}
		CameraY, CameraBackY           float64
		Lives, Ammunition, Grenades    int
		BulletPower, GrenadePower      [2]int
		RangeBonus                     [2]int
		DirectionLock                  [2]bool
		DestroyedCells                 [][2]int
		DemoPath                       []struct{ X, Y float64 }
		DemoPathIndex, DemoTargetKind  int
		DemoTarget                     struct{ X, Y float64 }
		DemoPlanScenery                bool
		DemoSceneryRow, DemoSceneryCol int
		SelectedWaypoint               struct{ X, Y float64 }
	}
	if err := json.Unmarshal(data, &source); err != nil {
		t.Fatal(err)
	}
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGame(true)
	g.scene = playScene
	g.player = source.Player.actor()
	g.cameraY, g.cameraBackY = source.CameraY, source.CameraBackY
	g.secondaryTerrain = g.cameraY < secondaryMapRows*tileSize
	g.lives, g.ammunition, g.grenades = source.Lives, source.Ammunition, source.Grenades
	g.bulletPower, g.grenadePower, g.rangeBonus = source.BulletPower, source.GrenadePower, source.RangeBonus
	g.directionLock = source.DirectionLock
	g.enemies, g.projectiles, g.pickups, g.mines = nil, nil, nil, nil
	for _, enemy := range source.Enemies {
		g.enemies = append(g.enemies, enemy.actor())
	}
	for _, shot := range source.Projectiles {
		g.projectiles = append(g.projectiles, projectile{x: shot.X, y: shot.Y, vx: shot.VX, vy: shot.VY,
			terrainBias: shot.TerrainBias, life: shot.Life, impact: shot.Impact, owner: shot.Owner,
			slot: shot.Slot, frame: shot.Frame, power: shot.Power, enemy: shot.Enemy,
			grenade: shot.Grenade, cannon: shot.Cannon, fragments: shot.Fragments})
	}
	for _, mine := range source.Mines {
		g.mines = append(g.mines, landmine{x: mine.X, y: mine.Y, kind: mine.Kind, slot: mine.Slot})
	}
	for _, cell := range source.DestroyedCells {
		g.destroyed[cell[0]][cell[1]] = true
	}
	g.demoPath = g.demoPath[:0]
	for _, point := range source.DemoPath {
		g.demoPath = append(g.demoPath, navPoint{point.X, point.Y})
	}
	g.demoPathIndex, g.demoTargetKind = source.DemoPathIndex, source.DemoTargetKind
	g.demoTarget = navPoint{source.DemoTarget.X, source.DemoTarget.Y}
	g.demoPlanScenery, g.demoSceneryRow, g.demoSceneryCol = source.DemoPlanScenery, source.DemoSceneryRow, source.DemoSceneryCol
	g.demoPlanMines, g.demoPlanVehicle = len(g.mines), g.player.inTank
	g.demoLastX, g.demoLastY = g.player.x-1, g.player.y
	// The fixture was captured after the old decision, before its physics.
	// Tick5238 did not fire; restore that preceding input bookkeeping so the
	// test actually exercises attack selection rather than its held-fire guard.
	g.demoFireHeld = false
	return g, navPoint{source.SelectedWaypoint.X, source.SelectedWaypoint.Y}
}

func TestDemoDoesNotChooseAnAttackWhenItsFirstStepIsUnsafe(t *testing.T) {
	g, target := loadUnsafeAttackState(t)
	g.prepareDemoForecast()
	_, normalScore := g.demoPlanMove(target, nil)
	attack := playerControls{x: 1, y: -1, fire: true}
	_, attackScore := g.demoPlanMove(target, &attack)
	g.demoForecastActive = false
	if normalScore != demoBlockedScore || attackScore != demoBlockedScore {
		t.Fatalf("recorded trap no longer rejects both moves: ordinary%g attack%g", normalScore, attackScore)
	}
	if _, _, ok := g.demoStep(g.player.x, g.player.y, attack.x, attack.y); ok {
		t.Fatal("recorded diagonal unexpectedly enters the mine safety margin")
	}
	control := g.demoControls()
	if control.fire || control.action || control.x != 0 || control.y != 0 {
		t.Fatalf("blocked-score tie selected an unsafe attack: %+v", control)
	}
}

func TestUnsafeAttackCanMoveBeforeTheOrdinaryBulletContact(t *testing.T) {
	g, _ := loadUnsafeAttackState(t)
	g.demo = false
	beforeX, beforeAmmo := g.player.x, g.ammunition
	g.stepPlayer(0, playerControls{x: 1, y: -1, fire: true})
	if g.player.x != beforeX+1 || g.player.deadTicks == 0 || g.ammunition != beforeAmmo-1 {
		t.Fatalf("recorded ordinary partial movement/contact changed: player%+v ammunition%d", g.player, g.ammunition)
	}
	// Rejecting the attack is a strategy change. The original horizontal,
	// weapon, contact and vertical phases remain ordinary shared physics.
}

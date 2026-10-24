package game

import (
	"encoding/json"
	"os"
	"testing"
)

type demoFanActor struct {
	unsafeAttackActor
	UpperFalling, LowerFalling, Hidden bool
	BlinkPhase, MoveDirection          int
	TankFlame                          struct{ Step, Delay, Frame int }
}

func (source demoFanActor) actor() actor {
	value := source.unsafeAttackActor.actor()
	value.upperFalling, value.lowerFalling = source.UpperFalling, source.LowerFalling
	value.hidden, value.blinkPhase, value.moveDirection = source.Hidden, uint8(source.BlinkPhase), source.MoveDirection
	value.tankFlame = walkAnimation{step: source.TankFlame.Step, delay: source.TankFlame.Delay, frame: source.TankFlame.Frame}
	return value
}

func loadDemoLateFanField(t *testing.T) *Game {
	t.Helper()
	data, err := os.ReadFile("testdata/demo-late-fan-field.json")
	if err != nil {
		t.Fatal(err)
	}
	var source struct {
		Player, Player2                    demoFanActor
		PlayerCount, GameplayTicks         int
		CameraY, CameraBackY               float64
		Lives, Ammunition, Grenades, Score int
		Enemies                            []demoFanActor
		Projectiles                        []unsafeAttackShot
		Mines                              []struct {
			X, Y       float64
			Kind, Slot int
		}
		Pickups []struct {
			X, Y      float64
			Kind, Bob int
		}
		FanTraps []struct {
			X, Y     float64
			Cooldown int
			Active   bool
		}
		DestroyedCells                 [][2]int
		BulletPower, GrenadePower      [2]int
		RangeBonus, FireDelay          [2]int
		TankFuelPhase                  [2]uint8
		DirectionLock, AirStrikeBusy   [2]bool
		IdleArtilleryTicks             uint32
		EventIndex, LastSpawnScroll    int
		RandomSpawn, RedSpawn          bool
		DemoPath                       []struct{ X, Y float64 }
		DemoPathIndex, DemoTargetKind  int
		DemoTarget                     struct{ X, Y float64 }
		DemoPlanScenery                bool
		DemoSceneryRow, DemoSceneryCol int
		DemoFireHeld                   bool
		SceneryBurst                   struct{ Phase int }
	}
	if err := json.Unmarshal(data, &source); err != nil {
		t.Fatal(err)
	}
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGame(true)
	g.scene, g.gameplayTicks, g.playerCount = playScene, source.GameplayTicks, source.PlayerCount
	g.player, g.player2 = source.Player.actor(), source.Player2.actor()
	g.cameraY, g.cameraBackY = source.CameraY, source.CameraBackY
	g.secondaryTerrain = g.cameraY < secondaryMapRows*tileSize
	g.lives, g.ammunition, g.grenades, g.score = source.Lives, source.Ammunition, source.Grenades, source.Score
	g.bulletPower, g.grenadePower, g.rangeBonus, g.fireDelay = source.BulletPower, source.GrenadePower, source.RangeBonus, source.FireDelay
	g.tankFuelPhase, g.directionLock, g.airStrikeBusy = source.TankFuelPhase, source.DirectionLock, source.AirStrikeBusy
	g.idleArtilleryTicks = source.IdleArtilleryTicks
	g.eventIndex, g.lastSpawnScroll, g.randomSpawn, g.redSpawn = source.EventIndex, source.LastSpawnScroll, source.RandomSpawn, source.RedSpawn
	g.enemies, g.projectiles, g.mines, g.pickups = nil, nil, nil, nil
	for _, value := range source.Enemies {
		g.enemies = append(g.enemies, value.actor())
	}
	for _, shot := range source.Projectiles {
		g.projectiles = append(g.projectiles, projectile{x: shot.X, y: shot.Y, vx: shot.VX, vy: shot.VY, terrainBias: shot.TerrainBias,
			life: shot.Life, impact: shot.Impact, owner: shot.Owner, slot: shot.Slot, frame: shot.Frame, power: shot.Power,
			enemy: shot.Enemy, grenade: shot.Grenade, cannon: shot.Cannon, fragments: shot.Fragments})
	}
	for _, mine := range source.Mines {
		g.mines = append(g.mines, landmine{x: mine.X, y: mine.Y, kind: mine.Kind, slot: mine.Slot})
	}
	for _, item := range source.Pickups {
		g.pickups = append(g.pickups, pickup{x: item.X, y: item.Y, kind: item.Kind, bob: item.Bob})
	}
	if len(source.FanTraps) != len(g.fanTraps) {
		t.Fatal("missing fan states in the native regression")
	}
	for index, trap := range source.FanTraps {
		g.fanTraps[index] = fanTrap{x: trap.X, y: trap.Y, cooldown: trap.Cooldown, active: trap.Active}
	}
	for _, cell := range source.DestroyedCells {
		g.destroyed[cell[0]][cell[1]] = true
	}
	g.sceneryBurst.phase = source.SceneryBurst.Phase
	g.demoPath = nil
	for _, point := range source.DemoPath {
		g.demoPath = append(g.demoPath, navPoint{point.X, point.Y})
	}
	g.demoPathIndex, g.demoTargetKind = source.DemoPathIndex, source.DemoTargetKind
	g.demoTarget = navPoint{source.DemoTarget.X, source.DemoTarget.Y}
	g.demoPlanScenery, g.demoSceneryRow, g.demoSceneryCol = source.DemoPlanScenery, source.DemoSceneryRow, source.DemoSceneryCol
	g.demoPlanVehicle, g.demoPlanMines, g.demoFireHeld = g.player.inTank, len(g.mines), source.DemoFireHeld
	g.demoLastX, g.demoLastY = g.player.x-1, g.player.y
	return g
}

func TestRememberedFanFiringLinesRemainBeyondTheirBullets(t *testing.T) {
	for index := 1; index < len(fanTrapPositions); index++ {
		g := loadDemoLateFanField(t)
		g.demo, g.randomSpawn, g.redSpawn = false, false, false
		g.eventIndex, g.idleArtilleryTicks = len(g.resources.events), 0
		g.enemies, g.projectiles, g.mines = nil, nil, nil
		for other := range g.fanTraps {
			g.fanTraps[other].active = other == index
		}
		trap := g.fanTraps[index]
		g.player = actor{x: trap.x, y: trap.y + 74, health: 1}
		g.cameraY, g.cameraBackY = g.player.y-96, g.player.y-96
		g.secondaryTerrain = g.cameraY < secondaryMapRows*tileSize
		lives, ammunition, grenades := g.lives, g.ammunition, g.grenades
		for range 80 {
			g.stepPlay([2]playerControls{})
			if g.player.deadTicks != 0 || g.lives != lives || g.player.invulnerable != 0 || g.player.health != 1 {
				t.Fatalf("fan%d reaches its remembered firing line: %+v", index, g.player)
			}
		}
		if g.ammunition != ammunition || g.grenades != grenades {
			t.Fatal("stationary fan range check changed finite weapon resources")
		}
	}
}

func TestRememberedDemoClearsTheLateFanFieldWithoutAnotherFall(t *testing.T) {
	g := loadDemoLateFanField(t)
	lives, ammunition, grenades := g.lives, g.ammunition, g.grenades
	for tick := 0; tick < 1600; tick++ {
		g.stepPlay([2]playerControls{})
		if g.player.deadTicks != 0 || g.player.health != 1 || g.lives != lives {
			t.Fatalf("late fan continuation lost a life at%d, player %+v ammo%d grenades%d", g.gameplayTicks, g.player, g.ammunition, g.grenades)
		}
		cleared := true
		for index := 1; index < len(g.fanTraps); index++ {
			cleared = cleared && !g.fanTraps[index].active
		}
		if cleared && g.player.y < fanTrapPositions[4].y-37 && !g.sceneryBurst.active && !g.burst.active {
			if g.ammunition > ammunition || g.grenades != grenades-4 || g.player.invulnerable != 0 {
				t.Fatalf("remembered route altered original resources/protection: ammo%d grenades%d protection%d", g.ammunition, g.grenades, g.player.invulnerable)
			}
			t.Logf("cleared all four launchers in%d ordinary ticks at%g/%g with%d reserve lives, ammo%d and grenades%d", tick+1, g.player.x, g.player.y, g.lives, g.ammunition, g.grenades)
			return
		}
	}
	t.Fatalf("late fan continuation did not finish: player %+v fans %+v", g.player, g.fanTraps)
}

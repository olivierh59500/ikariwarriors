package game

import (
	"encoding/json"
	"os"
	"testing"
)

type demoShorelineShot struct {
	unsafeAttackShot
	AirStrike int
}

func (s demoShorelineShot) projectile() projectile {
	return projectile{x: s.X, y: s.Y, vx: s.VX, vy: s.VY, terrainBias: s.TerrainBias, life: s.Life, impact: s.Impact, owner: s.Owner, slot: s.Slot, frame: s.Frame, power: s.Power, enemy: s.Enemy, grenade: s.Grenade, cannon: s.Cannon, fragments: s.Fragments, airStrike: uint8(s.AirStrike)}
}

// The ordinary second attract cycle reached this shoreline with fuel 38 and
// waited 546 updates for fuel exhaustion. Keep its actual shots, blast, event
// flags and fuel phase; no future random sequence is stored or restored.
func loadDemoWaterShoreline(t *testing.T) *Game {
	t.Helper()
	data, err := os.ReadFile("testdata/demo-water-shoreline.json")
	if err != nil {
		t.Fatal(err)
	}
	var source struct {
		Player, Player2                                                    demoFanActor
		PlayerCount, GameplayTicks                                         int
		CameraY, CameraBackY                                               float64
		Lives, Lives2, Ammunition, Ammunition2, Grenades, Grenades2, Score int
		Enemies                                                            []demoFanActor
		Projectiles                                                        []demoShorelineShot
		Blasts                                                             [3]demoShorelineShot
		Mines                                                              []struct {
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
		DestroyedCells                                           [][2]int
		BulletPower, GrenadePower, RangeBonus, FireDelay         [2]int
		TankFuelPhase                                            [2]uint8
		DirectionLock, AirStrikeBusy                             [2]bool
		IdleArtilleryTicks                                       uint32
		EventIndex, LastSpawnScroll                              int
		RandomSpawn, RedSpawn                                    bool
		CannonReservation                                        [2]int
		DemoPath                                                 []struct{ X, Y float64 }
		DemoPathIndex, DemoTargetKind                            int
		DemoTarget                                               struct{ X, Y float64 }
		DemoPlanScenery, DemoPlanVehicle, DemoFireHeld           bool
		DemoSceneryRow, DemoSceneryCol, DemoPlanMines, DemoStall int
		DemoLastX, DemoLastY                                     float64
		Boss, Aircraft, Final, Ending                            struct{ Active, Flying bool }
		EnemyVehicles                                            [2]struct{ Active bool }
		Burst, SceneryBurst                                      struct {
			Active bool
			Phase  int
		}
		BurstQueue []struct{ X, Y float64 }
	}
	if err = json.Unmarshal(data, &source); err != nil {
		t.Fatal(err)
	}
	if source.Boss.Active || source.Aircraft.Flying || source.Final.Active || source.Ending.Active || source.Burst.Active || source.SceneryBurst.Active || len(source.BurstQueue) > 0 {
		t.Fatal("shoreline fixture requires additional active composite state")
	}
	for _, v := range source.EnemyVehicles {
		if v.Active {
			t.Fatal("shoreline fixture has an active vehicle")
		}
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
	g.lives, g.lives2, g.ammunition, g.ammunition2, g.grenades, g.grenades2, g.score = source.Lives, source.Lives2, source.Ammunition, source.Ammunition2, source.Grenades, source.Grenades2, source.Score
	g.bulletPower, g.grenadePower, g.rangeBonus, g.fireDelay = source.BulletPower, source.GrenadePower, source.RangeBonus, source.FireDelay
	g.tankFuelPhase, g.directionLock, g.airStrikeBusy = source.TankFuelPhase, source.DirectionLock, source.AirStrikeBusy
	g.idleArtilleryTicks, g.eventIndex, g.lastSpawnScroll, g.randomSpawn, g.redSpawn = source.IdleArtilleryTicks, source.EventIndex, source.LastSpawnScroll, source.RandomSpawn, source.RedSpawn
	g.cannonReservation = source.CannonReservation
	g.enemies, g.projectiles, g.mines, g.pickups = nil, nil, nil, nil
	for _, a := range source.Enemies {
		g.enemies = append(g.enemies, a.actor())
	}
	for _, s := range source.Projectiles {
		g.projectiles = append(g.projectiles, s.projectile())
	}
	for i, s := range source.Blasts {
		g.blasts[i] = s.projectile()
	}
	for _, m := range source.Mines {
		g.mines = append(g.mines, landmine{x: m.X, y: m.Y, kind: m.Kind, slot: m.Slot})
	}
	for _, p := range source.Pickups {
		g.pickups = append(g.pickups, pickup{x: p.X, y: p.Y, kind: p.Kind, bob: p.Bob})
	}
	if len(source.FanTraps) != len(g.fanTraps) {
		t.Fatal("missing shoreline fan state")
	}
	for i, f := range source.FanTraps {
		g.fanTraps[i] = fanTrap{x: f.X, y: f.Y, cooldown: f.Cooldown, active: f.Active}
	}
	for _, c := range source.DestroyedCells {
		g.destroyed[c[0]][c[1]] = true
	}
	g.sceneryBurst.phase = source.SceneryBurst.Phase
	g.demoPath = nil
	for _, p := range source.DemoPath {
		g.demoPath = append(g.demoPath, navPoint{p.X, p.Y})
	}
	g.demoPathIndex, g.demoTargetKind, g.demoTarget = source.DemoPathIndex, source.DemoTargetKind, navPoint{source.DemoTarget.X, source.DemoTarget.Y}
	g.demoPlanScenery, g.demoPlanVehicle, g.demoPlanMines = source.DemoPlanScenery, source.DemoPlanVehicle, source.DemoPlanMines
	g.demoSceneryRow, g.demoSceneryCol, g.demoFireHeld, g.demoStall = source.DemoSceneryRow, source.DemoSceneryCol, source.DemoFireHeld, source.DemoStall
	g.demoLastX, g.demoLastY = source.DemoLastX, source.DemoLastY
	return g
}

func TestDemoLeavesTheObservedHealthyTankAtWaterWithoutReboarding(t *testing.T) {
	g := loadDemoWaterShoreline(t)
	lives, ammo, grenades := g.lives, g.ammunition, g.grenades
	if !g.player.inTank || g.player.tankFuel != 38 || g.terrainCode(44, 5632) != 139 {
		t.Fatal("observed shoreline state changed")
	}
	leftTank := false
	for tick := 0; tick < 48 && g.player.y > 5610; tick++ {
		if err := g.Update(); err != nil {
			t.Fatal(err)
		}
		if !g.player.inTank {
			leftTank = true
		} else if leftTank {
			t.Fatal("reboarded the water-blocked parked tank")
		}
		if g.player.deadTicks > 0 || g.lives != lives {
			t.Fatalf("ordinary shoreline exit was unsafe: %+v", g.player)
		}
	}
	if !leftTank || g.player.y > 5610 || g.player.tankFuel < 30 || g.ammunition > ammo || g.grenades > grenades {
		t.Fatalf("did not cross water promptly using ordinary inputs: player%+v ammo%d grenades%d", g.player, g.ammunition, g.grenades)
	}
}

func TestDemoDoesNotRouteBackToItsWaterBlockedParkedTank(t *testing.T) {
	g := loadDemoWaterShoreline(t)
	// A normal context action parks the actual healthy tank without changing
	// health or resources. The next production decision must leave it parked.
	g.stepPlayer(0, playerControls{action: true})
	if g.player.inTank || g.boardingTankAt(0, g.player.x, g.player.y) < 0 {
		t.Fatal("ordinary dismount did not leave an overlapping tank")
	}
	if target, ok := g.demoDestination(); ok && target == (navPoint{44, 5640}) {
		t.Fatal("route selected the same water-blocked tank")
	}
	if err := g.Update(); err != nil {
		t.Fatal(err)
	}
	if g.player.inTank {
		t.Fatal("automatic boarding ignored the impassable shoreline")
	}
}

func TestDemoKeepsTankProtectionWhenTheShoreExitIsUnsafe(t *testing.T) {
	g := loadDemoWaterShoreline(t)
	// This controlled immediate contact distinguishes a safe ordinary egress
	// decision from merely noticing water in the static map.
	g.projectiles = append(g.projectiles, projectile{x: g.player.x - 4, y: g.player.y, vx: 0, vy: 4, life: 6, enemy: true, owner: -1, slot: 0, power: 1})
	if g.demoTankExitSafe() {
		t.Fatal("controlled immediate infantry contact was not detected")
	}
	if err := g.Update(); err != nil {
		t.Fatal(err)
	}
	if !g.player.inTank || g.player.deadTicks > 0 {
		t.Fatalf("discarded tank protection during actual contact: %+v", g.player)
	}
}

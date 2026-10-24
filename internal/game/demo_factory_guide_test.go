package game

import (
	"encoding/json"
	"os"
	"testing"
)

// This fixture was reached by an ordinary native run from the beginning.
// It retains the real idle-artillery and event flags, unlike the earlier
// visible-only diagnostic. No future random state is read or restored.
func loadDemoFactoryApproach(t *testing.T) *Game {
	t.Helper()
	data, err := os.ReadFile("testdata/demo-factory-guns.json")
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
		DestroyedCells                [][2]int
		BulletPower, GrenadePower     [2]int
		RangeBonus, FireDelay         [2]int
		TankFuelPhase                 [2]uint8
		DirectionLock, AirStrikeBusy  [2]bool
		IdleArtilleryTicks            uint32
		EventIndex, LastSpawnScroll   int
		RandomSpawn, RedSpawn         bool
		CannonReservation             [2]int
		Boss, Aircraft, Final, Ending struct{ Active, Flying bool }
		EnemyVehicles                 [2]struct{ Active bool }
		Burst, SceneryBurst           struct {
			Active bool
			Phase  int
		}
		BurstQueue []struct{ X, Y float64 }
		Blasts     [3]struct{ Impact int }
	}
	if err = json.Unmarshal(data, &source); err != nil {
		t.Fatal(err)
	}
	if source.Boss.Active || source.Aircraft.Flying || source.Final.Active || source.Ending.Active || source.Burst.Active || source.SceneryBurst.Active || len(source.BurstQueue) != 0 {
		t.Fatal("factory fixture requires additional active composite state")
	}
	for _, vehicle := range source.EnemyVehicles {
		if vehicle.Active {
			t.Fatal("factory fixture has an active vehicle")
		}
	}
	for _, blast := range source.Blasts {
		if blast.Impact != 0 {
			t.Fatal("factory fixture has an active blast")
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
	g.lives, g.ammunition, g.grenades, g.score = source.Lives, source.Ammunition, source.Grenades, source.Score
	g.bulletPower, g.grenadePower, g.rangeBonus, g.fireDelay = source.BulletPower, source.GrenadePower, source.RangeBonus, source.FireDelay
	g.tankFuelPhase, g.directionLock, g.airStrikeBusy = source.TankFuelPhase, source.DirectionLock, source.AirStrikeBusy
	g.idleArtilleryTicks = source.IdleArtilleryTicks
	g.eventIndex, g.lastSpawnScroll, g.randomSpawn, g.redSpawn = source.EventIndex, source.LastSpawnScroll, source.RandomSpawn, source.RedSpawn
	g.cannonReservation = source.CannonReservation
	g.enemies, g.projectiles, g.mines, g.pickups = nil, nil, nil, nil
	for _, enemy := range source.Enemies {
		g.enemies = append(g.enemies, enemy.actor())
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
		t.Fatal("missing factory fixture fan state")
	}
	for index, trap := range source.FanTraps {
		g.fanTraps[index] = fanTrap{x: trap.X, y: trap.Y, cooldown: trap.Cooldown, active: trap.Active}
	}
	for _, cell := range source.DestroyedCells {
		g.destroyed[cell[0]][cell[1]] = true
	}
	g.sceneryBurst.phase = source.SceneryBurst.Phase
	return g
}

func TestRememberedFactoryGrenadesClearBothGunsAndTheirFragments(t *testing.T) {
	g := loadDemoFactoryApproach(t)
	lives, ammunition, grenades := g.lives, g.ammunition, g.grenades
	sawFragments := false
	for tick := 0; tick < 240; tick++ {
		g.stepPlay([2]playerControls{})
		sawFragments = sawFragments || g.burst.active
		if g.player.deadTicks != 0 || g.player.health != 1 || g.player.invulnerable != 0 || g.lives != lives {
			t.Fatalf("factory route lost a life or altered protection at %d: %+v", g.gameplayTicks, g.player)
		}
		alive := false
		for _, enemy := range g.enemies {
			alive = alive || enemy.kind == 2 && enemy.health > 0 && enemy.deadTicks == 0 && enemy.y == 4990 && (enemy.x == 22 || enemy.x == 192)
		}
		if !alive && !g.demoFactoryExplosionPending() {
			if g.ammunition != ammunition || g.grenades != grenades-2 || !sawFragments || g.player.y < 5060 {
				t.Fatalf("factory clearance did not conserve its two throws and safe hold: player %+v ammo %d grenades %d fragments %v", g.player, g.ammunition, g.grenades, sawFragments)
			}
			t.Logf("cleared both factory guns and upgraded fragments in %d ordinary ticks at %.0f/%.0f with %d reserve lives and %d grenades", tick+1, g.player.x, g.player.y, g.lives, g.grenades)
			return
		}
	}
	t.Fatalf("factory route did not clear: player %+v enemies %+v grenades %d", g.player, g.enemies, g.grenades)
}

func TestFactoryGuideDecisionsPreservePlayerAndResources(t *testing.T) {
	g := loadDemoFactoryApproach(t)
	player, lives, ammunition, grenades := g.player, g.lives, g.ammunition, g.grenades
	if _, active := g.demoFactoryGuns(); !active {
		t.Fatal("the recorded firing approach did not activate its remembered guide")
	}
	if g.player != player || g.lives != lives || g.ammunition != ammunition || g.grenades != grenades {
		t.Fatal("a factory decision changed player state or finite resources")
	}
}

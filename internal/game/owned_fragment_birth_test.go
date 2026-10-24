package game

import (
	"encoding/json"
	"os"
	"testing"
)

type originalOwnedFragmentPoint struct{ X, Y float64 }

type originalOwnedFragmentShot struct {
	unsafeAttackShot
	AirStrike uint8
}

func (source originalOwnedFragmentShot) projectile() projectile {
	return projectile{x: source.X, y: source.Y, vx: source.VX, vy: source.VY, terrainBias: source.TerrainBias,
		life: source.Life, impact: source.Impact, owner: source.Owner, slot: source.Slot, frame: source.Frame, power: source.Power,
		enemy: source.Enemy, grenade: source.Grenade, cannon: source.Cannon, fragments: source.Fragments, airStrike: source.AirStrike}
}

// These five normally played adjacent-frame pairs include three upgraded
// tank shells and two foot grenades. Their expected counters and all forty
// positions come from actual RAM, independently of the native functions.
func TestNormallyPlayedOriginalOwnedFragmentBirth(t *testing.T) {
	data, err := os.ReadFile("testdata/owned-fragment-birth.json")
	if err != nil {
		t.Fatal(err)
	}
	var original struct {
		Cases []struct {
			Name, BeforeSHA256, AfterSHA256                          string
			BeforeTick, AfterTick                                    int
			Player                                                   unsafeAttackActor
			CameraY, CameraBackY                                     float64
			Lives, Ammunition, Grenades                              int
			BulletPower, GrenadePower, RangeBonus, CannonReservation int
			DirectionLock                                            bool
			TankFuelPhase                                            [2]uint8
			IdleArtilleryTicks                                       uint32
			Enemies                                                  []unsafeAttackActor
			Projectiles                                              []originalOwnedFragmentShot
			Mines                                                    []struct {
				X, Y       float64
				Kind, Slot int
			}
			Pickups []struct {
				X, Y float64
				Kind int
			}
			FanTraps []struct {
				X, Y     float64
				Cooldown int
				Active   bool
			}
			DestroyedCells [][2]int
			OwnedQuad      originalOwnedFragmentShot
			SceneryBurst   struct {
				Active                         bool
				Remaining, Phase, Frame, Owner int
				Enabled                        byte
				Origin, Center                 originalOwnedFragmentPoint
				Parts                          [8]originalOwnedFragmentPoint
			}
			Control struct {
				X, Y         float64
				Fire, Action bool
			}
			Expected struct {
				Active           bool
				Remaining, Owner int
				Parts            [8]originalOwnedFragmentPoint
			}
		}
	}
	if err := json.Unmarshal(data, &original); err != nil {
		t.Fatal(err)
	}
	if len(original.Cases) != 5 {
		t.Fatal("incomplete normally played owned-fragment observations")
	}
	for _, source := range original.Cases {
		t.Run(source.Name, func(t *testing.T) {
			if source.AfterTick != source.BeforeTick+1 || len(source.BeforeSHA256) != 64 || len(source.AfterSHA256) != 64 {
				t.Fatal("original birth observations lack consecutive-state provenance")
			}
			g, err := New(Options{Mute: true})
			if err != nil {
				t.Fatal(err)
			}
			g.startGame(false)
			g.scene, g.demo, g.playerCount = playScene, false, 1
			// Capture ordinals start at zero after the first completed game tick.
			g.gameplayTicks = source.BeforeTick + 1
			g.player, g.player2 = source.Player.actor(), actor{}
			g.cameraY, g.cameraBackY = source.CameraY, source.CameraBackY
			g.lives, g.ammunition, g.grenades = source.Lives, source.Ammunition, source.Grenades
			g.bulletPower[0], g.grenadePower[0], g.rangeBonus[0] = source.BulletPower, source.GrenadePower, source.RangeBonus
			g.directionLock[0], g.cannonReservation[0] = source.DirectionLock, source.CannonReservation
			g.tankFuelPhase, g.idleArtilleryTicks = source.TankFuelPhase, source.IdleArtilleryTicks
			g.enemies, g.projectiles, g.mines, g.pickups = nil, nil, nil, nil
			for _, value := range source.Enemies {
				g.enemies = append(g.enemies, value.actor())
			}
			for _, value := range source.Projectiles {
				g.projectiles = append(g.projectiles, value.projectile())
			}
			for _, value := range source.Mines {
				g.mines = append(g.mines, landmine{x: value.X, y: value.Y, kind: value.Kind, slot: value.Slot})
			}
			for _, value := range source.Pickups {
				g.pickups = append(g.pickups, pickup{x: value.X, y: value.Y, kind: value.Kind})
			}
			if len(source.FanTraps) != len(g.fanTraps) {
				t.Fatal("incomplete actual fan observations")
			}
			for index, value := range source.FanTraps {
				g.fanTraps[index] = fanTrap{x: value.X, y: value.Y, cooldown: value.Cooldown, active: value.Active}
			}
			for _, cell := range source.DestroyedCells {
				g.destroyed[cell[0]][cell[1]] = true
			}
			g.blasts = [3]projectile{source.OwnedQuad.projectile()}
			value := source.SceneryBurst
			g.sceneryBurst = sceneryBurst{active: value.Active, remaining: value.Remaining, phase: value.Phase, frame: value.Frame, owner: value.Owner, enabled: value.Enabled,
				origin: navPoint{value.Origin.X, value.Origin.Y}, center: navPoint{value.Center.X, value.Center.Y}}
			for index, point := range value.Parts {
				g.sceneryBurst.parts[index] = navPoint{point.X, point.Y}
			}
			// This single-frame collision regression retains observed actors, shots,
			// mines, scenery and finite resources. New scripted/random births happen
			// outside the owned-quad phase and are excluded; no future RNG is stored.
			g.eventIndex, g.randomSpawn, g.redSpawn = len(g.resources.events), false, false
			if g.burst.active || g.blasts[0].impact != 1 || g.grenadePower[0] != 2 {
				t.Fatal("actual source state does not isolate a new upgraded owned burst")
			}
			control := playerControls{x: source.Control.X, y: source.Control.Y, fire: source.Control.Fire, action: source.Control.Action}
			g.stepPlay([2]playerControls{control})
			if g.burst.active != source.Expected.Active || g.burst.remaining != source.Expected.Remaining || g.burst.owner != source.Expected.Owner {
				t.Fatalf("complete-tick fragment pool active%t remaining%d owner%d; original active%t remaining%d owner%d", g.burst.active, g.burst.remaining, g.burst.owner, source.Expected.Active, source.Expected.Remaining, source.Expected.Owner)
			}
			for index, point := range source.Expected.Parts {
				if actual := g.burst.parts[index]; actual != (navPoint{point.X, point.Y}) {
					t.Errorf("part%d native%+v original%.0f/%.0f", index, actual, point.X, point.Y)
				}
			}
		})
	}
}

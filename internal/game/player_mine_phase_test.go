package game

import (
	"encoding/json"
	"os"
	"testing"
)

// The original checks mines after horizontal input and before vertical input.
// This adjacent normally played pair lies exactly on that inclusive edge.
func TestNormallyPlayedOriginalMineContactPrecedesVerticalMovement(t *testing.T) {
	data, err := os.ReadFile("testdata/player-mine-phases.json")
	if err != nil {
		t.Fatal(err)
	}
	var original struct {
		Cases []struct {
			Name, BeforeSHA256, AfterSHA256                                    string
			BeforeTick, AfterTick                                              int
			Player                                                             unsafeAttackActor
			CameraY, CameraBackY                                               float64
			Lives, Ammunition, Grenades, BulletPower, GrenadePower, RangeBonus int
			DirectionLock                                                      bool
			TankFuelPhase                                                      [2]uint8
			IdleArtilleryTicks                                                 uint32
			Enemies                                                            []unsafeAttackActor
			Projectiles                                                        []originalOwnedFragmentShot
			Mines                                                              []struct {
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
			Control        struct {
				X, Y         float64
				Fire, Action bool
			}
			Expected struct {
				Active           bool
				Remaining, Owner int
				Parts            [8]originalOwnedFragmentPoint
				MineSlots        []int
				Player           struct {
					X, Y                                                         float64
					Health, DeadTicks, Invulnerable, Lives, Ammunition, Grenades int
				}
			}
		}
	}
	if err = json.Unmarshal(data, &original); err != nil {
		t.Fatal(err)
	}
	if len(original.Cases) != 1 {
		t.Fatal("incomplete original mine phase observation")
	}
	for _, source := range original.Cases {
		t.Run(source.Name, func(t *testing.T) {
			if source.AfterTick != source.BeforeTick+1 || len(source.BeforeSHA256) != 64 || len(source.AfterSHA256) != 64 {
				t.Fatal("missing adjacent original-state provenance")
			}
			g, err := New(Options{Mute: true})
			if err != nil {
				t.Fatal(err)
			}
			g.startGame(false)
			g.scene, g.demo, g.playerCount = playScene, false, 1
			g.gameplayTicks = source.BeforeTick + 1 // Source completed-frame indices start at zero.
			g.player, g.player2 = source.Player.actor(), actor{}
			g.cameraY, g.cameraBackY = source.CameraY, source.CameraBackY
			g.lives, g.ammunition, g.grenades = source.Lives, source.Ammunition, source.Grenades
			g.bulletPower[0], g.grenadePower[0], g.rangeBonus[0] = source.BulletPower, source.GrenadePower, source.RangeBonus
			g.directionLock[0], g.tankFuelPhase, g.idleArtilleryTicks = source.DirectionLock, source.TankFuelPhase, source.IdleArtilleryTicks
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
				t.Fatal("missing actual fan records")
			}
			for index, value := range source.FanTraps {
				g.fanTraps[index] = fanTrap{x: value.X, y: value.Y, cooldown: value.Cooldown, active: value.Active}
			}
			for _, cell := range source.DestroyedCells {
				g.destroyed[cell[0]][cell[1]] = true
			}
			// The original before state has no active large encounter or blast pool.
			// Retain existing physical objects, while excluding new random/scripted
			// births outside this observed input-contact phase. No RNG state is stored.
			g.eventIndex, g.randomSpawn, g.redSpawn = len(g.resources.events), false, false
			g.stepPlay([2]playerControls{{x: source.Control.X, y: source.Control.Y, fire: source.Control.Fire, action: source.Control.Action}})
			want := source.Expected.Player
			if g.player.x != want.X || g.player.y != want.Y || g.player.health != want.Health || g.player.deadTicks != want.DeadTicks || g.player.invulnerable != want.Invulnerable {
				t.Errorf("native player%.0f/%.0f health%d fall%d protection%d; original%+v", g.player.x, g.player.y, g.player.health, g.player.deadTicks, g.player.invulnerable, want)
			}
			if g.lives != want.Lives || g.ammunition != want.Ammunition || g.grenades != want.Grenades {
				t.Error("ordinary source resources changed")
			}
			if len(g.mines) != len(source.Expected.MineSlots) {
				t.Errorf("native retained%d mines; original%d", len(g.mines), len(source.Expected.MineSlots))
			}
			for _, mine := range g.mines {
				found := false
				for _, slot := range source.Expected.MineSlots {
					found = found || mine.slot == slot
				}
				if !found {
					t.Errorf("source mine slot%d should be inactive", mine.slot)
				}
			}
			if g.burst.active != source.Expected.Active || g.burst.remaining != source.Expected.Remaining || g.burst.owner != source.Expected.Owner {
				t.Errorf("native burst%+v; original active%t remaining%d owner%d", g.burst, source.Expected.Active, source.Expected.Remaining, source.Expected.Owner)
			}
			for index, point := range source.Expected.Parts {
				if g.burst.parts[index] != (navPoint{point.X, point.Y}) {
					t.Errorf("part%d native%+v original%.0f/%.0f", index, g.burst.parts[index], point.X, point.Y)
				}
			}
		})
	}
}

func newMineInputScene(t *testing.T, players int) *Game {
	t.Helper()
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGamePlayers(false, players)
	g.scene = playScene
	g.cameraY, g.cameraBackY = 9904, 9936
	g.player = actor{x: 104, y: 10000, health: 1}
	g.player2 = actor{x: 104, y: 10000, health: 1}
	g.enemies, g.projectiles, g.mines, g.pickups = nil, nil, nil, nil
	g.eventIndex, g.randomSpawn, g.redSpawn = len(g.resources.events), false, false
	return g
}

// The remaining cases isolate static original BF40/BF54 control flow. They
// are controlled native regressions, not additional original RAM captures.
func TestMineInputStopsAtTheFirstOriginalSlot(t *testing.T) {
	g := newMineInputScene(t, 1)
	g.mines = []landmine{{x: 106, y: 10000, kind: 18, slot: 82}, {x: 104, y: 10000, kind: 18, slot: 81}}
	g.stepPlay([2]playerControls{})
	if len(g.mines) != 1 || g.mines[0].slot != 82 || len(g.burstQueue) != 0 || g.burst.remaining != 5 || g.burst.parts[1] != (navPoint{104, 9992}) {
		t.Fatalf("one entered player must consume only the first original slot: mines%+v burst%+v queue%+v", g.mines, g.burst, g.burstQueue)
	}
}

func TestMineInputRunsOnceForEachEnteredPlayer(t *testing.T) {
	g := newMineInputScene(t, 2)
	g.mines = []landmine{{x: 106, y: 10000, kind: 18, slot: 82}, {x: 104, y: 10000, kind: 18, slot: 81}}
	g.stepPlay([2]playerControls{})
	if len(g.mines) != 0 || len(g.burstQueue) != 1 || g.burstQueue[0] != (burstRequest{106, 10000, -1}) || g.burst.remaining != 5 || g.burst.parts[1] != (navPoint{104, 9992}) {
		t.Fatalf("P1 then P2 must each consume one original mine: mines%+v burst%+v queue%+v", g.mines, g.burst, g.burstQueue)
	}
}

func TestBusyFirstAirStrikeMarkerStillEndsTheInputSearch(t *testing.T) {
	g := newMineInputScene(t, 1)
	g.airStrikeBusy = [2]bool{true, false}
	g.mines = []landmine{{x: 104, y: 10000, kind: 20, slot: 86}, {x: 104, y: 10000, kind: 20, slot: 85}}
	g.stepPlay([2]playerControls{})
	if g.airStrikeBusy != ([2]bool{true, false}) || len(g.projectiles) != 0 || len(g.mines) != 2 {
		t.Fatalf("busy original marker85 must prevent marker86 in the same input: busy%v shots%+v", g.airStrikeBusy, g.projectiles)
	}
}

func TestFallingPlayerDoesNotEnterTheMineInputPhase(t *testing.T) {
	g := newMineInputScene(t, 1)
	g.player.health, g.player.deadTicks, g.player.invulnerable = 0, 12, 50
	g.mines = []landmine{{x: 104, y: 10000, kind: 18, slot: 81}}
	g.stepPlay([2]playerControls{})
	if len(g.mines) != 1 || g.burst.active {
		t.Fatal("a pre-existing falling player entered the input mine pass")
	}
}

func TestHitDuringEnteredInputStillChecksItsMine(t *testing.T) {
	for player := 0; player < 2; player++ {
		t.Run(string(rune('1'+player)), func(t *testing.T) {
			g := newMineInputScene(t, player+1)
			if player == 1 {
				g.player.x = 40
			}
			g.projectiles = []projectile{{x: 104, y: 10000, vy: 4, enemy: true, owner: -1, slot: 0, life: 15}}
			g.mines = []landmine{{x: 104, y: 10000, kind: 18, slot: 81}}
			g.stepPlay([2]playerControls{})
			value := g.player
			if player == 1 {
				value = g.player2
			}
			if len(g.mines) != 0 || !g.burst.active || g.burst.remaining != 5 || value.deadTicks != 23 {
				t.Fatalf("player%d's already-entered input must retain actor activity after its bullet hit: mine%+v burst%+v player%+v", player+1, g.mines, g.burst, value)
			}
		})
	}
}

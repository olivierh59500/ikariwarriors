package game

import (
	"encoding/json"
	"os"
	"testing"
)

type latePlayerObservation struct {
	Camera, X, Y, Progress, Dead, Fall, Protection, Blink, Active int
	Upper, Lower, UpperStep, UpperDelay, LowerStep, LowerDelay    int
	Lives, Ammo, Grenades                                         int
	Wet                                                           bool
}

type latePhaseObservations struct {
	Falls []struct {
		StartTick, SampleTick, RespawnTick int
		Start, Sample                      latePlayerObservation
		Respawn                            struct {
			X, Y, Progress, Lives, Ammunition, Grenades int
			Dead                                        bool
		}
	}
	HelicopterFreshShots []struct {
		Tick, Camera, Phase, HeliRawX, HeliRawY                           int
		Active                                                            [5]bool
		ShotSlot, ShotRawX, ShotRawY, VX, VY, SourceLife, DirectionChoice int
	}
	RedDeathDuringFinal []struct {
		Tick, Ending, Final, Camera int
		Enemy                       struct {
			Slot, X, Y, Dead, Fall, Upper, Lower, UpperStep, UpperDelay, LowerStep, LowerDelay int
			Active                                                                             bool
		}
	}
}

func readLatePhaseObservations(t *testing.T) latePhaseObservations {
	t.Helper()
	b, err := os.ReadFile("testdata/source-late-phases.json")
	if err != nil {
		t.Fatal(err)
	}
	var rows latePhaseObservations
	if err := json.Unmarshal(b, &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows.Falls) != 4 || len(rows.HelicopterFreshShots) != 3 || len(rows.RedDeathDuringFinal) != 2 {
		t.Fatal("incomplete source phase observations")
	}
	return rows
}

// This isolated replay supplies one actually recorded first-fall boundary.
// Later falling samples and reappearance supplies come only from Source22.
// It does not restore the original enemies, input polling or full world.
func supplyRecordedFall(g *Game, s latePlayerObservation) {
	g.scene, g.demo, g.playerCount = playScene, false, 1
	g.cameraY, g.cameraBackY = float64(s.Camera), float64(s.Camera)
	g.secondaryTerrain = g.cameraY < secondaryMapRows*tileSize
	g.enemies, g.projectiles, g.pickups, g.mines = nil, nil, nil, nil
	g.randomSpawn, g.redSpawn, g.eventIndex = false, false, len(g.resources.events)
	lower := (s.Lower - 0x49ac8) / overlayBobBytes
	if s.Lower == 0x4a1a8 {
		lower = -2
	} else if s.Lower == 0x4a248 {
		lower = -3
	}
	g.player = actor{x: float64(s.X - 48), y: float64(s.Camera + s.Y - 4), health: 0, deadTicks: s.Fall, invulnerable: s.Protection, blinkPhase: uint8(s.Blink), hidden: s.Active == 0, wet: s.Wet,
		upper: walkAnimation{frame: (s.Upper - 0x47cc8) / actorBobBytes, step: s.UpperStep, delay: s.UpperDelay},
		lower: walkAnimation{frame: lower, step: s.LowerStep, delay: s.LowerDelay}, upperFalling: true, lowerFalling: true}
	g.lives, g.ammunition, g.grenades = s.Lives, s.Ammo, s.Grenades
}

func compareRecordedFall(t *testing.T, g *Game, s latePlayerObservation) {
	t.Helper()
	p := g.player
	u, l := p.playerArtOffsets(0)
	if int(p.x+48) != s.X || int(p.y-g.cameraY+4) != s.Y || p.deadTicks != s.Fall || p.invulnerable != s.Protection || int(p.blinkPhase) != s.Blink || p.hidden != (s.Active == 0) || u+0x42b88 != s.Upper || l+0x42b88 != s.Lower || p.upper.step != s.UpperStep || p.upper.delay != s.UpperDelay || p.lower.step != s.LowerStep || p.lower.delay != s.LowerDelay {
		t.Fatalf("recorded fall differs: native%+v images%x/%x, source%+v", p, u+0x42b88, l+0x42b88, s)
	}
}

func TestRecordedFallSubphaseObservations(t *testing.T) {
	for _, row := range readLatePhaseObservations(t).Falls {
		g, err := New(Options{Mute: true})
		if err != nil {
			t.Fatal(err)
		}
		supplyRecordedFall(g, row.Start)
		compareRecordedFall(t, g, row.Start)
		for tick := row.StartTick + 1; tick <= row.SampleTick; tick++ {
			g.advancePlayerFall(0)
			g.advancePlayerProtection(0)
		}
		compareRecordedFall(t, g, row.Sample)
		for tick := row.SampleTick + 1; tick <= row.RespawnTick; tick++ {
			g.advancePlayerFall(0)
			g.advancePlayerProtection(0)
		}
		want := row.Respawn
		if int(g.player.x+48) != want.X || int(g.player.y-g.cameraY+4) != want.Y || g.player.deadTicks != 0 || g.player.health != 1 || g.lives != want.Lives || g.ammunition != want.Ammunition || g.grenades != want.Grenades {
			t.Fatalf("actual source reappearance at%d differs: native%+v lives%d ammo%d grenades%d, source%+v", row.RespawnTick, g.player, g.lives, g.ammunition, g.grenades, want)
		}
	}
}

// Actual complete source frames contain lifetime60 shots at the helicopter
// muzzle. Valid controlled gate/vector choices reproduce their observed launch
// direction; this is a phase diagnostic, not a reconstructed full input frame.
func TestRecordedAircraftFreshShotPhase(t *testing.T) {
	for _, row := range readLatePhaseObservations(t).HelicopterFreshShots {
		word := func(index uint64) uint32 {
			if index == 0 {
				return 0
			}
			return uint32(row.DirectionChoice)
		}
		g, err := New(Options{Mute: true, RandomSource: word})
		if err != nil {
			t.Fatal(err)
		}
		g.cameraY = float64(row.Camera)
		g.aircraft = helicopter{x: float64(row.HeliRawX - 32 - 48), y: float64(row.Camera + row.HeliRawY + 26 - 2), phase: (row.Phase + 1) % 3, flying: true, alive: row.Active}
		g.advanceEnemyProjectiles()
		g.advanceAircraft()
		g.fireAircraft()
		if len(g.projectiles) != 1 {
			t.Fatal("recorded aircraft launch missing")
		}
		shot := g.projectiles[0]
		if int(shot.x+56) != row.ShotRawX || int(shot.y-g.cameraY) != row.ShotRawY || int(shot.vx) != row.VX || int(shot.vy) != row.VY || shot.life*4 != row.SourceLife {
			t.Fatalf("source frame%d fresh aircraft shot native%+v original%+v", row.Tick, shot, row)
		}
	}
}

// A red death observed during the final encounter completes while the final
// flag remains set. This checks completion, not the unrecorded random drop kind.
func TestRecordedRedDeathContinuesDuringFinal(t *testing.T) {
	rows := readLatePhaseObservations(t).RedDeathDuringFinal
	before, after := rows[0], rows[1]
	if before.Final != 1 || after.Final != 1 || before.Enemy.Fall != 7 || after.Enemy.Dead != 0 || after.Enemy.Fall != 0 || after.Enemy.Active {
		t.Fatal("source red-death qualification changed")
	}
	g, err := New(Options{Mute: true, RandomSource: func(uint64) uint32 { return 0 }})
	if err != nil {
		t.Fatal(err)
	}
	g.cameraY = float64(before.Camera)
	g.final.active = true
	s := before.Enemy
	g.enemies = []actor{{slot: s.Slot, kind: 5, health: 0, deadTicks: s.Fall, x: float64(s.X - 48), y: g.cameraY + float64(s.Y-4), upper: walkAnimation{frame: (s.Upper - 0x51e08) / actorBobBytes, step: s.UpperStep, delay: s.UpperDelay}, lower: walkAnimation{frame: (s.Lower - 0x54608) / overlayBobBytes, step: s.LowerStep, delay: s.LowerDelay}}}
	for range after.Tick - before.Tick {
		g.advanceEnemyDeaths()
	}
	if len(g.enemies) != 0 {
		t.Fatalf("red source death remains frozen during final: %+v", g.enemies)
	}
}

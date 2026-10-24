package game

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"testing"
)

type continuousSourceState struct {
	RawX      int      `json:"x"`
	RawY      int      `json:"y"`
	Progress  int      `json:"p"`
	Lives     int      `json:"l"`
	Ammo      int      `json:"a"`
	Grenades  int      `json:"g"`
	Dead      bool     `json:"d"`
	Tank      bool     `json:"t"`
	Camera    int      `json:"c"`
	Room      int      `json:"r"`
	Fuel      int      `json:"f"`
	FuelPhase [2]uint8 `json:"z"`
	Score     int      `json:"s"`
}

type continuousSourceActor struct {
	Slot       int `json:"s"`
	X          int `json:"x"`
	Y          int `json:"y"`
	Upper      int `json:"u"`
	Lower      int `json:"l"`
	UpperStep  int `json:"up"`
	UpperDelay int `json:"ud"`
	LowerStep  int `json:"lp"`
	LowerDelay int `json:"ld"`
}

type continuousSourceOwnedShot struct {
	Slot int `json:"s"`
	X    int `json:"x"`
	Y    int `json:"y"`
}

type continuousSourceFixture struct {
	Version int      `json:"v"`
	Code    string   `json:"code"`
	Source  string   `json:"source"`
	Entropy string   `json:"entropy"`
	Words   []uint32 `json:"words"`
	Stages  []int    `json:"stages"`
	Sites   []uint32 `json:"sites"`
	Initial struct {
		Hash     string                  `json:"h"`
		RNGCount uint64                  `json:"rng"`
		Seed     uint32                  `json:"seed"`
		State    continuousSourceState   `json:"v"`
		Event    int                     `json:"e"`
		Player   continuousSourceActor   `json:"p"`
		Enemies  []continuousSourceActor `json:"en"`
	} `json:"initial"`
	Frames []struct {
		Index      int                         `json:"i"`
		Direction  uint8                       `json:"b"`
		Fire       bool                        `json:"f"`
		Key        int                         `json:"k"`
		Toggle     bool                        `json:"t"`
		LockBefore bool                        `json:"lb"`
		LockAfter  bool                        `json:"la"`
		RNGBefore  uint64                      `json:"rb"`
		RNGAfter   uint64                      `json:"re"`
		SeedBefore uint32                      `json:"sb"`
		SeedAfter  uint32                      `json:"sa"`
		Hash       string                      `json:"h"`
		State      continuousSourceState       `json:"v"`
		Owned      []continuousSourceOwnedShot `json:"o"`
	} `json:"frames"`
}

func readContinuousSourceFixture(t *testing.T) continuousSourceFixture {
	t.Helper()
	f, err := os.Open("testdata/original-continuous-512.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	z, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	defer z.Close()
	var source continuousSourceFixture
	if err := json.NewDecoder(z).Decode(&source); err != nil {
		t.Fatal(err)
	}
	if source.Version != 1 || len(source.Frames) != 512 || len(source.Words) != 3629 || len(source.Stages) != len(source.Words) || len(source.Sites) != len(source.Words) || source.Initial.RNGCount != 12 {
		t.Fatal("incomplete original continuous source fixture")
	}
	if len(source.Code) != 64 || len(source.Source) != 64 || len(source.Entropy) != 64 || len(source.Initial.Hash) != 64 {
		t.Fatal("missing original source provenance")
	}
	for i := range int(source.Initial.RNGCount) {
		if source.Stages[i] != -1 {
			t.Fatal("original landing random stage differs")
		}
	}
	return source
}

func compareContinuousSourceState(t *testing.T, label string, g *Game, source continuousSourceState) {
	t.Helper()
	got := continuousSourceState{
		RawX: int(g.player.x + 48), RawY: int(g.player.y-g.cameraY) + 4,
		Progress: levelHeight - 16 - int(g.player.y), Lives: g.lives,
		Ammo: g.ammunition, Grenades: g.grenades, Dead: g.player.deadTicks > 0,
		Tank: g.player.inTank, Camera: int(g.cameraY), Room: int(g.cameraBackY - g.cameraY),
		Fuel: g.player.tankFuel, FuelPhase: g.tankFuelPhase, Score: g.score,
	}
	if got != source {
		t.Fatalf("%s native state%+v, original%+v", label, got, source)
	}
}

// replayContinuousOriginal starts from normal New/startGame and 116 landing
// updates. It imports no original actor, map, camera, clock, animation or
// resource state. Recorded random words are environmental inputs; accepted
// late actions pass through ordinary shared controls. Observers receive each
// completed game only after core, owned-shot and player-image comparisons.
// This 512-update one-player trace contains no falls and does not establish
// whole-game equality or original host input polling behavior.
func replayContinuousOriginal(t *testing.T, afterFrame func(int, *Game)) {
	t.Helper()
	var poses []struct{ Upper, Lower, Turret int }
	blob, err := os.ReadFile("testdata/original-continuous-player-poses.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(blob, &poses); err != nil {
		t.Fatal(err)
	}
	if len(poses) != 512 {
		t.Fatal("incomplete original player-image observations")
	}
	source := readContinuousSourceFixture(t)
	word := func(index uint64) uint32 {
		if index >= uint64(len(source.Words)) {
			panic("continuous original random input exhausted")
		}
		return source.Words[index]
	}
	g, err := New(Options{Mute: true, RandomSource: word})
	if err != nil {
		t.Fatal(err)
	}
	g.startGame(false)
	for range 116 {
		if err := g.Update(); err != nil {
			t.Fatal(err)
		}
	}
	if g.scene != playScene || g.randomSourceIndex != source.Initial.RNGCount || g.random != source.Initial.Seed || g.eventIndex != source.Initial.Event {
		t.Fatal("cold landing did not reproduce the original event/random boundary")
	}
	compareContinuousSourceState(t, "landing", g, source.Initial.State)
	p := source.Initial.Player
	upper, lower := g.player.playerArtOffsets(0)
	if upper+0x42b88 != p.Upper || lower+0x42b88 != p.Lower || g.player.upper.step != p.UpperStep || g.player.upper.delay != p.UpperDelay || g.player.lower.step != p.LowerStep || g.player.lower.delay != p.LowerDelay {
		t.Fatal("cold landing player images or animation phases differ")
	}
	if len(g.enemies) != len(source.Initial.Enemies) {
		t.Fatal("cold landing original actor count differs")
	}
	for _, original := range source.Initial.Enemies {
		found := false
		for _, enemy := range g.enemies {
			if enemy.slot != original.Slot {
				continue
			}
			found = true
			upper, lower := enemy.infantryArtAddresses()
			if int(enemy.x) != original.X || int(enemy.y) != original.Y || upper != original.Upper || lower != original.Lower || enemy.upper.step != original.UpperStep || enemy.upper.delay != original.UpperDelay || enemy.lower.step != original.LowerStep || enemy.lower.delay != original.LowerDelay {
				t.Fatalf("cold landing actor%d differs from original", original.Slot)
			}
		}
		if !found {
			t.Fatalf("cold landing actor%d missing", original.Slot)
		}
	}
	hold, toggles, attempted := 0, 0, 0
	for index, input := range source.Frames {
		if input.Index != index || len(input.Hash) != 64 || input.RNGBefore != g.randomSourceIndex || input.SeedBefore != g.random || input.LockBefore != g.directionLock[0] {
			t.Fatalf("frame%d original entry boundary differs", index)
		}
		for i := input.RNGBefore; i < input.RNGAfter; i++ {
			if i >= uint64(len(source.Words)) || source.Stages[i] != index || source.Sites[i] == 0 {
				t.Fatalf("frame%d original random stage is incomplete", index)
			}
		}
		if input.Fire {
			hold++
		} else {
			hold = 0
		}
		control := playerControls{fire: hold == 1, action: heldGrenade(hold), toggleLock: input.Toggle}
		if input.Direction&0x10 != 0 {
			control.y = -1
		} else if input.Direction&0x20 != 0 {
			control.y = 1
		}
		if input.Direction&0x40 != 0 {
			control.x = -1
		} else if input.Direction&0x80 != 0 {
			control.x = 1
		}
		g.tick++
		g.sceneTick++
		g.stepPlay([2]playerControls{control})
		if input.Key == 0x20 {
			attempted++
		}
		if input.Toggle {
			if input.Key != 0x20 {
				t.Fatal("recorded late toggle lacks a Space action")
			}
			// The ordinary shared control path accepts this late action.
			toggles++
		}
		if g.randomSourceIndex != input.RNGAfter || g.random != input.SeedAfter || g.directionLock[0] != input.LockAfter {
			t.Fatalf("frame%d random sequence or accepted action differs", index)
		}
		compareContinuousSourceState(t, fmt.Sprintf("frame%d", index), g, input.State)
		if !g.player.inTank {
			upper, lower := g.player.playerArtOffsets(0)
			if upper+0x42b88 != poses[index].Upper || lower+0x42b88 != poses[index].Lower {
				t.Fatalf("frame%d native player image addresses %x/%x, original %x/%x", index, upper+0x42b88, lower+0x42b88, poses[index].Upper, poses[index].Lower)
			}
		} else if !g.player.tankBurning {
			turret := 0x57628 + ((8-g.player.direction)&7)*vehicleBobBytes
			if turret != poses[index].Turret {
				t.Fatalf("frame%d native turret %x, original %x", index, turret, poses[index].Turret)
			}
		}
		got := make([]continuousSourceOwnedShot, 0, 3)
		for _, shot := range g.projectiles {
			if !shot.enemy && shot.owner == 0 {
				got = append(got, continuousSourceOwnedShot{Slot: shot.slot, X: int(shot.x), Y: int(shot.y)})
			}
		}
		slices.SortFunc(got, func(a, b continuousSourceOwnedShot) int { return a.Slot - b.Slot })
		if !slices.Equal(got, input.Owned) {
			t.Fatalf("frame%d native owned positions%v, original%v", index, got, input.Owned)
		}
		if afterFrame != nil {
			afterFrame(index, g)
		}
	}
	if toggles != 16 || attempted != 39 || g.randomSourceIndex != uint64(len(source.Words)) {
		t.Fatal("original input sequence did not complete")
	}
}

func TestOriginalContinuousCoreWithRecordedRandomness(t *testing.T) {
	replayContinuousOriginal(t, nil)
}

func TestIndexedRandomSourceKeepsForecastConsumptionLocal(t *testing.T) {
	word := func(index uint64) uint32 { return uint32(index)*17 + 3 }
	g, err := New(Options{Mute: true, RandomSource: word})
	if err != nil {
		t.Fatal(err)
	}
	if g.randomValue() != 3 {
		t.Fatal("indexed source did not begin at zero")
	}
	forecast := *g
	if forecast.randomValue() != 20 || forecast.randomValue() != 37 {
		t.Fatal("copied forecast lost its indexed source position")
	}
	if g.randomSourceIndex != 1 || g.random != 3 || g.randomValue() != 20 {
		t.Fatal("forecast consumed the live game's random inputs")
	}
	g.startGame(false)
	if g.randomSourceIndex != 2 || g.random != 20 || g.randomValue() != 37 {
		t.Fatal("new game reset an existing random input sequence")
	}
	if _, err := json.Marshal(g.options); err != nil {
		t.Fatal("random source leaked into serialized options:", err)
	}
}

//go:build rendercheck

package game

import (
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"os"
	"slices"
	"testing"
)

type ruinsColdSourceState struct {
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

type ruinsColdSourceActor struct {
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

type ruinsColdSourceOwnedShot struct {
	Slot int `json:"s"`
	X    int `json:"x"`
	Y    int `json:"y"`
}

type ruinsColdSourceFixture struct {
	Version int      `json:"v"`
	Code    string   `json:"code"`
	Source  string   `json:"source"`
	Entropy string   `json:"entropy"`
	Words   []uint32 `json:"words"`
	Stages  []int    `json:"stages"`
	Sites   []uint32 `json:"sites"`
	Initial struct {
		Hash     string                 `json:"h"`
		RNGCount uint64                 `json:"rng"`
		Seed     uint32                 `json:"seed"`
		State    ruinsColdSourceState   `json:"v"`
		Event    int                    `json:"e"`
		Player   ruinsColdSourceActor   `json:"p"`
		Enemies  []ruinsColdSourceActor `json:"en"`
	} `json:"initial"`
	Frames []struct {
		Index      int                        `json:"i"`
		Direction  uint8                      `json:"b"`
		Fire       bool                       `json:"f"`
		Key        int                        `json:"k"`
		Toggle     bool                       `json:"t"`
		LockBefore bool                       `json:"lb"`
		LockAfter  bool                       `json:"la"`
		RNGBefore  uint64                     `json:"rb"`
		RNGAfter   uint64                     `json:"re"`
		SeedBefore uint32                     `json:"sb"`
		SeedAfter  uint32                     `json:"sa"`
		Hash       string                     `json:"h"`
		State      ruinsColdSourceState       `json:"v"`
		Owned      []ruinsColdSourceOwnedShot `json:"o"`
	} `json:"frames"`
}

func readRuinsColdSourceFixture(t *testing.T) ruinsColdSourceFixture {
	t.Helper()
	f, err := os.Open("testdata/final-ruins-inputs.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	z, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	defer z.Close()
	var source ruinsColdSourceFixture
	if err := json.NewDecoder(z).Decode(&source); err != nil {
		t.Fatal(err)
	}
	if source.Version != 1 || len(source.Frames) != 12683 || len(source.Words) != 65700 || len(source.Stages) != len(source.Words) || len(source.Sites) != len(source.Words) || source.Initial.RNGCount != 12 {
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

func compareRuinsColdSourceState(t *testing.T, label string, g *Game, source ruinsColdSourceState) {
	t.Helper()
	got := ruinsColdSourceState{
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

// This regression starts normally and replays the recorded original controls
// and returned random words through the complete level. It compares every
// visible pixel in the 168 source frames surrounding the final terrain bake.
// No original actor, map, pose or intermediate mechanical state is imported.
// Accepted late Space events normalize the original keyboard poll.
func TestOriginalRetainedFinalRuinsColdPictures(t *testing.T) {
	var pictures struct {
		Version, Start, End, ClipTopAtBake int
		Palette                            []byte
		InputsSHA256                       string
		Samples                            []struct {
			Index                      int
			SourceSHA256, PixelsSHA256 string
			Pixels                     []byte
		}
	}
	file, err := os.Open("testdata/final-ruins-source.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	packed, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	defer packed.Close()
	if err := json.NewDecoder(packed).Decode(&pictures); err != nil {
		t.Fatal(err)
	}
	if pictures.Version != 1 || pictures.Start != 12515 || pictures.End != 12682 || len(pictures.Samples) != 168 || len(pictures.Palette) != 32 {
		t.Fatal("incomplete original retained-ruins picture window")
	}
	inputs, err := os.ReadFile("testdata/final-ruins-inputs.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprintf("%x", sha256.Sum256(inputs)) != pictures.InputsSHA256 {
		t.Fatal("original cold input fixture changed")
	}
	source := readRuinsColdSourceFixture(t)
	g, err := New(Options{Mute: true, RandomSource: func(index uint64) uint32 {
		if index >= uint64(len(source.Words)) {
			panic("original random fixture exhausted")
		}
		return source.Words[index]
	}})
	if err != nil {
		t.Fatal(err)
	}
	for index := 0; index < 16; index++ {
		value := uint16(pictures.Palette[index*2])<<8 | uint16(pictures.Palette[index*2+1])
		r, gc, b, _ := g.resources.colors[index].RGBA()
		if uint8(r>>8) != uint8(value>>8&15)*17 || uint8(gc>>8) != uint8(value>>4&15)*17 || uint8(b>>8) != uint8(value&15)*17 {
			t.Fatal("source/native palettes differ")
		}
	}
	g.startGame(false)
	for range 116 {
		if err := g.Update(); err != nil {
			t.Fatal(err)
		}
	}
	if g.scene != playScene || g.randomSourceIndex != source.Initial.RNGCount || g.random != source.Initial.Seed {
		t.Fatal("normal landing did not reproduce original random boundary")
	}
	compareRuinsColdSourceState(t, "landing", g, source.Initial.State)
	target := ebiten.NewImage(ScreenWidth, ScreenHeight)
	background := make([]byte, ScreenWidth*ScreenHeight*4)
	hold, compared := 0, 0
	for index, input := range source.Frames {
		if input.Index != index || g.randomSourceIndex != input.RNGBefore || g.random != input.SeedBefore || g.directionLock[0] != input.LockBefore {
			t.Fatalf("frame%d entry differs", index)
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
		if g.randomSourceIndex != input.RNGAfter || g.random != input.SeedAfter || g.directionLock[0] != input.LockAfter {
			t.Fatalf("frame%d random/action boundary differs", index)
		}
		compareRuinsColdSourceState(t, fmt.Sprintf("frame%d", index), g, input.State)
		owned := make([]ruinsColdSourceOwnedShot, 0, 3)
		for _, shot := range g.projectiles {
			if !shot.enemy && shot.owner == 0 {
				owned = append(owned, ruinsColdSourceOwnedShot{Slot: shot.slot, X: int(shot.x), Y: int(shot.y)})
			}
		}
		slices.SortFunc(owned, func(a, b ruinsColdSourceOwnedShot) int { return a.Slot - b.Slot })
		if !slices.Equal(owned, input.Owned) {
			t.Fatalf("frame%d owned projectiles differ", index)
		}
		if index < pictures.Start || index > pictures.End {
			continue
		}
		sample := pictures.Samples[compared]
		if sample.Index != index || sample.SourceSHA256 != input.Hash || len(sample.Pixels) != 64000 || fmt.Sprintf("%x", sha256.Sum256(sample.Pixels)) != sample.PixelsSHA256 {
			t.Fatal("original picture/input provenance differs")
		}
		if index == pictures.Start && (!g.final.defeated || g.final.ruinsClipTop != pictures.ClipTopAtBake) {
			t.Fatal("normal defeat did not capture the independently observed retained world origin")
		}
		check := &originalSmallDrawCheck{g: g, target: target, background: background, expected: sample.Pixels}
		check.draw = func() { g.Draw(target) }
		smallDrawScenes <- check
		if err := <-smallDrawResults; err != nil {
			t.Fatalf("frame%d complete retained-ruins picture: %v", index, err)
		}
		compared++
	}
	if compared != 168 || g.randomSourceIndex != uint64(len(source.Words)) {
		t.Fatal("original late picture/input sequence did not complete")
	}
	t.Logf("All %d complete 320x200 original ruins pictures match after 12683 normal updates and 65700 original random returns", compared)
}

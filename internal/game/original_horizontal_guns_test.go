package game

import (
	"encoding/json"
	"os"
	"testing"
)

type sourceHorizontalGuns struct {
	Bands  []struct{ Start, RawX, MuzzleProgress, Direction int }
	Births []struct {
		Band, Frame, Progress int
		RandomWord            uint32
		Shot                  struct{ Slot, X, Y, VX, VY, Range, Image int }
	}
}

func readSourceHorizontalGuns(t *testing.T) sourceHorizontalGuns {
	t.Helper()
	data, err := os.ReadFile("testdata/original-horizontal-guns-source.json")
	if err != nil {
		t.Fatal(err)
	}
	var source sourceHorizontalGuns
	if err := json.Unmarshal(data, &source); err != nil {
		t.Fatal(err)
	}
	if len(source.Bands) != 4 || len(source.Births) != 8 {
		t.Fatal("missing original four horizontal-gun tables and eight actual births")
	}
	return source
}

// These eight births are actual full-game source frames, not native outputs.
func TestOriginalHorizontalGunBirths(t *testing.T) {
	for _, source := range readSourceHorizontalGuns(t).Births {
		g, err := New(Options{Mute: true, RandomSource: func(uint64) uint32 { return source.RandomWord }})
		if err != nil {
			t.Fatal(err)
		}
		g.playerCount = 1
		g.player = actor{y: float64(levelHeight - 16 - source.Progress), health: 1}
		// Earlier occupied source slots force the same first-free allocation.
		for slot := 0; slot < source.Shot.Slot; slot++ {
			g.projectiles = append(g.projectiles, projectile{enemy: true, owner: -1, slot: slot, life: 5})
		}
		before := g.randomSourceIndex
		g.updateHorizontalGuns()
		if g.randomSourceIndex != before+1 || len(g.projectiles) != source.Shot.Slot+1 {
			t.Fatalf("source frame%d gate/allocation differs", source.Frame)
		}
		shot := g.projectiles[len(g.projectiles)-1]
		if shot.slot != source.Shot.Slot || int(shot.x) != source.Shot.X || int(shot.y) != source.Shot.Y || int(shot.vx) != source.Shot.VX || int(shot.vy) != source.Shot.VY || shot.life*4 != source.Shot.Range || 0x4a2e8+g.projectileFrame(shot)*overlayBobBytes != source.Shot.Image {
			t.Fatalf("source frame%d actual horizontal birth differs: %+v", source.Frame, shot)
		}
	}
}

// E562 uses start<progress<=start+30 and consumes chance before pool access.
func TestOriginalHorizontalGunBandEdgesAndFullPool(t *testing.T) {
	for _, band := range readSourceHorizontalGuns(t).Bands {
		for _, offset := range []int{0, 1, 30, 31} {
			g, err := New(Options{Mute: true, RandomSource: func(uint64) uint32 { return 0 }})
			if err != nil {
				t.Fatal(err)
			}
			g.playerCount = 1
			g.player = actor{y: float64(levelHeight - 16 - band.Start - offset), health: 1}
			for slot := 0; slot < 12; slot++ {
				g.projectiles = append(g.projectiles, projectile{enemy: true, owner: -1, slot: slot, life: 5})
			}
			before := g.randomSourceIndex
			g.updateHorizontalGuns()
			want := uint64(0)
			if offset == 1 || offset == 30 {
				want = 1
			}
			if g.randomSourceIndex != before+want || len(g.projectiles) != 12 {
				t.Fatalf("band%d offset%d full-pool gate differs", band.Start, offset)
			}
		}
	}
}

func TestHorizontalGunScansBothEnteredProgressValues(t *testing.T) {
	g, err := New(Options{Mute: true, RandomSource: func(uint64) uint32 { return 0 }})
	if err != nil {
		t.Fatal(err)
	}
	g.playerCount = 2
	g.player = actor{y: float64(levelHeight - 16 - 11660), health: 0, deadTicks: 5, hidden: true}
	g.player2 = g.player
	before := g.randomSourceIndex
	g.updateHorizontalGuns()
	if g.randomSourceIndex != before+2 || len(g.projectiles) != 2 {
		t.Fatal("falling/hidden bodies suppressed an original progress scan")
	}
}

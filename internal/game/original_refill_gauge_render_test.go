//go:build rendercheck

package game

import (
	"github.com/hajimehoshi/ebiten/v2"
	"testing"
)

// Actual source crops distinguish the full refill bitmap from the ordinary
// depleted gauge. The same tank remains boarded while its fuel timer advances.
func TestOriginalRefillGaugeKeepsFullBitmapUntilConsumption(t *testing.T) {
	var frames []struct {
		Frame, Fuel, Phase int
		Pixels             []byte
		SourceSHA256       string
	}
	readOriginalPaintFixture(t, "testdata/original-refill-gauge.json.gz", &frames)
	if len(frames) != 4 {
		t.Fatal("incomplete original refill sequence")
	}
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.player = actor{health: 1, inTank: true, tankFuel: frames[0].Fuel}
	g.tankFuelPhase[0] = uint8(frames[0].Phase)
	compare := func(index int) {
		source := frames[index]
		if len(source.Pixels) != 16*64 || len(source.SourceSHA256) != 64 || g.player.tankFuel != source.Fuel || int(g.tankFuelPhase[0]) != source.Phase {
			t.Fatalf("source frame%d gauge entry differs", source.Frame)
		}
		expected := make([]byte, ScreenWidth*ScreenHeight)
		for y := range 64 {
			copy(expected[(80+y)*ScreenWidth+32:(80+y)*ScreenWidth+48], source.Pixels[y*16:(y+1)*16])
		}
		target := ebiten.NewImage(ScreenWidth, ScreenHeight)
		check := &originalSmallDrawCheck{g: g, target: target, background: make([]byte, len(expected)*4), expected: expected}
		check.draw = func() { g.drawFuelGauges(target) }
		smallDrawScenes <- check
		if err := <-smallDrawResults; err != nil {
			t.Fatalf("source frame%d refill gauge: %v", source.Frame, err)
		}
	}
	compare(0)
	g.collectPickup(0, 2)
	g.advanceTankFuel(0)
	compare(1)
	for range 7 {
		g.advanceTankFuel(0)
	}
	compare(2)
	g.advanceTankFuel(0)
	compare(3)
}

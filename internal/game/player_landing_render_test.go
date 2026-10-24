//go:build rendercheck

package game

import (
	"fmt"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// These 116 complete pictures follow the original first landing painter through
// the ordinary gameplay handover. Frame 0 precedes that painter and retains
// previous-screen residue; it is retained separately in the private evidence.
// No pixels are masked, and every expected pixel comes from the original.
func TestOriginalCompletePlayerLandingPictures(t *testing.T) {
	var rows []struct {
		Tick         int
		Pixels       []byte
		SourceSHA256 string
	}
	readOriginalPlayerAnimation(t, "testdata/player-landing-painter.json.gz", &rows)
	if len(rows) != 116 {
		t.Fatal("incomplete original landing pictures")
	}
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGamePlayers(false, 2)
	for i, row := range rows {
		if row.Tick != i+1 || len(row.Pixels) != ScreenWidth*ScreenHeight || len(row.SourceSHA256) != 64 {
			t.Fatal("invalid complete original landing picture")
		}
		g.updateIntro()
		t.Run(fmt.Sprintf("ordinary-landing-%03d", row.Tick), func(t *testing.T) {
			target := ebiten.NewImage(ScreenWidth, ScreenHeight)
			check := &originalSmallDrawCheck{g: g, target: target, background: make([]byte, ScreenWidth*ScreenHeight*4), expected: row.Pixels}
			check.draw = func() { g.Draw(target) }
			smallDrawScenes <- check
			if err := <-smallDrawResults; err != nil {
				t.Fatal(err)
			}
		})
	}
}

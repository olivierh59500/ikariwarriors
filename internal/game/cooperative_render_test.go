//go:build rendercheck

package game

import (
	"github.com/hajimehoshi/ebiten/v2"
	"testing"
)

// Original framebuffer pixels are retained exactly under the original player
// masks. This qualifies both native player painters, not complete scene replay.
func TestOriginalCooperativePlayerPainting(t *testing.T) {
	var rows []struct {
		Name         string
		Camera       int
		Players      [2]struct{ X, Y, Upper, Lower int }
		Pixels       []byte
		SourceSHA256 string
	}
	readOriginalPaintFixture(t, "testdata/cooperative-player-pixels.json.gz", &rows)
	if len(rows) != 64 {
		t.Fatal("incomplete original co-op player pictures")
	}
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, source := range rows {
		t.Run(source.Name, func(t *testing.T) {
			g.cameraY = float64(source.Camera)
			for i, p := range source.Players {
				value := actor{x: float64(p.X), y: float64(p.Y), health: 1}
				value.upper.frame = (p.Upper - 0x42b88 - i*22*actorBobBytes) / actorBobBytes
				value.lower.frame = (p.Lower - 0x46288) / overlayBobBytes
				if i == 0 {
					g.player = value
				} else {
					g.player2 = value
				}
			}
			screen := ebiten.NewImage(ScreenWidth, ScreenHeight)
			check := &originalSmallDrawCheck{g: g, target: screen, background: make([]byte, ScreenWidth*ScreenHeight*4), expected: source.Pixels}
			check.draw = func() {
				g.playfield.Clear()
				g.drawActor(g.playfield, &g.player2, 1)
				g.drawActor(g.playfield, &g.player, 0)
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Translate(playfieldX, playfieldY)
				screen.DrawImage(g.playfield, op)
			}
			smallDrawScenes <- check
			if err := <-smallDrawResults; err != nil {
				t.Fatal(err)
			}
		})
	}
}

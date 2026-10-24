//go:build rendercheck

package game

import (
	"github.com/hajimehoshi/ebiten/v2"
	"testing"
)

// Expected pixels and score changes come from three independent original
// E126 calls. Their controlled backgrounds are unchanged original screen data.
func TestOriginalVictoryMessageAndBonus(t *testing.T) {
	var rows []struct {
		Name                    string
		Players                 [2]bool
		ScoreBefore, ScoreAfter [2]int
		Background, Pixels      []byte
		SourceSHA256            string
	}
	readOriginalPaintFixture(t, "testdata/victory-screen.json.gz", &rows)
	if len(rows) != 3 {
		t.Fatal("incomplete original victory observations")
	}
	for _, row := range rows {
		t.Run(row.Name, func(t *testing.T) {
			g, err := New(Options{Mute: true})
			if err != nil {
				t.Fatal(err)
			}
			g.scene = playScene
			g.playerCount = 2
			g.lives, g.lives2 = -1, -1
			if row.Players[0] {
				g.lives = 1
			}
			if row.Players[1] {
				g.lives2 = 1
			}
			g.score, g.score2 = row.ScoreBefore[0], row.ScoreBefore[1]
			g.enterVictory()
			if g.score != row.ScoreAfter[0] || g.score2 != row.ScoreAfter[1] {
				t.Fatal("original per-player victory bonus differs")
			}
			g.enterVictory()
			if g.score != row.ScoreAfter[0] || g.score2 != row.ScoreAfter[1] {
				t.Fatal("victory bonus was awarded twice")
			}
			background := make([]byte, len(row.Background)*4)
			for i, index := range row.Background {
				r, b, c, a := g.resources.colors[index].RGBA()
				background[i*4], background[i*4+1], background[i*4+2], background[i*4+3] = byte(r>>8), byte(b>>8), byte(c>>8), byte(a>>8)
			}
			screen := ebiten.NewImage(ScreenWidth, ScreenHeight)
			check := &originalSmallDrawCheck{g: g, target: screen, background: background, expected: row.Pixels, draw: func() { g.drawVictoryMessage(screen) }}
			smallDrawScenes <- check
			if err := <-smallDrawResults; err != nil {
				t.Fatal(err)
			}
		})
	}
}

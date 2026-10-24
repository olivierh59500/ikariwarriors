//go:build rendercheck

package game

import "testing"

// Original subroutine calls supply the active positions and resulting images.
// Expected pixels decode those original BOBs; the small-object painter itself
// is also covered by complete original framebuffer observations elsewhere.
func TestOriginalMineWarningsUseTheirObservedImages(t *testing.T) {
	var rows []struct {
		Name         string
		Mines        []struct{ X, Y, Kind, Slot int }
		Animations   []struct{ Step, Delay, Frame int }
		Pixels       []byte
		SourceSHA256 string
	}
	readOriginalPaintFixture(t, "testdata/mine-warnings-painter.json.gz", &rows)
	if len(rows) != 35 {
		t.Fatalf("unexpected original mine raster count%d", len(rows))
	}
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.cameraY = 13904
	for _, row := range rows {
		t.Run(row.Name, func(t *testing.T) {
			g.mines = nil
			for _, m := range row.Mines {
				g.mines = append(g.mines, landmine{x: float64(m.X), y: float64(m.Y), kind: m.Kind, slot: m.Slot})
			}
			for i, a := range row.Animations {
				g.mineAnimations[i] = walkAnimation{step: a.Step, delay: a.Delay, frame: a.Frame}
			}
			check := &originalSmallDrawCheck{g: g, background: make([]byte, playfieldWidth*playfieldHeight*4), expected: row.Pixels, draw: g.drawMines}
			smallDrawScenes <- check
			if err := <-smallDrawResults; err != nil {
				t.Fatal(err)
			}
		})
	}
}

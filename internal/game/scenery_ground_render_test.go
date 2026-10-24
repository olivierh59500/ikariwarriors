//go:build rendercheck

package game

import "testing"

// Actual original 11776/1224A calls supply every raster and changed code byte.
// Synthetic rectangles cover all 19 sizes, impact parity and visible boundaries;
// their explicit source probe coordinates are normalized before this fixture.
func TestOriginalControlledSceneryGroundPainting(t *testing.T) {
	var cases []struct {
		Name                                                    string
		Code, Row, Column, Rows, Columns, X, Y, Camera, ClipTop int
		Cleared                                                 [][2]int
		Before, After                                           []byte
		SourceD0, SourceD1, SourceD2                            int
		BeforeSHA256, AfterSHA256                               string
	}
	readOriginalPaintFixture(t, "testdata/scenery-ground-painter.json.gz", &cases)
	if len(cases) != 192 {
		t.Fatal("incomplete original scenery painter observations")
	}
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, source := range cases {
		t.Run(source.Name, func(t *testing.T) {
			if source.SourceD0 != 14992-source.Y || source.SourceD1 != source.X+48 || source.SourceD2 != source.Y-source.Camera+26 {
				t.Fatal("source terrain-probe coordinate convention was lost")
			}
			g.destroyed = [mapRows][mapColumns]bool{}
			g.sceneryPaint = nil
			g.cameraY, g.cameraBackY = float64(source.Camera), float64(source.ClipTop+32)
			for row := 516; row < 536; row++ {
				for column := range mapColumns {
					g.resources.levelCodes[row][column] = 0
				}
			}
			for row := source.Row; row < source.Row+source.Rows; row++ {
				for column := source.Column; column < source.Column+source.Columns; column++ {
					g.resources.levelCodes[row][column] = byte(source.Code)
				}
			}
			if !g.destroyTileAt(float64(source.X), float64(source.Y)) {
				t.Fatal("native refused the observed scenery clear")
			}
			want := map[[2]int]bool{}
			for _, p := range source.Cleared {
				want[p] = true
			}
			for r := 516; r < 536; r++ {
				for c := range mapColumns {
					if g.destroyed[r][c] != want[[2]int{r, c}] {
						t.Fatalf("cleared%d/%d=%v, original%v", r, c, g.destroyed[r][c], want[[2]int{r, c}])
					}
				}
			}
			if len(source.Before) != playfieldWidth*playfieldHeight || len(source.After) != len(source.Before) || len(source.BeforeSHA256) != 64 || len(source.AfterSHA256) != 64 {
				t.Fatal("invalid original scenery raster provenance")
			}
			background := make([]byte, len(source.Before)*4)
			for i, index := range source.Before {
				r, green, b, _ := g.resources.colors[index].RGBA()
				copy(background[i*4:], []byte{byte(r >> 8), byte(green >> 8), byte(b >> 8), 255})
			}
			check := &originalSmallDrawCheck{g: g, background: background, expected: source.After}
			check.draw = func() { g.drawSceneryGround(g.cameraY) }
			smallDrawScenes <- check
			if err := <-smallDrawResults; err != nil {
				t.Fatal(err)
			}
		})
	}
}

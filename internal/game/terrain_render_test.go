//go:build rendercheck

package game

import (
	"compress/gzip"
	"encoding/json"
	"os"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// These are unmodified, normally played original framebuffers at ten positions
// along the river and first barricade. Original actor rectangles are explicitly
// excluded; every remaining terrain pixel uses the shared native GPU painter.
func TestNormallyPlayedOriginalTerrainRendering(t *testing.T) {
	file, err := os.Open("testdata/played-terrain.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	packed, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	defer packed.Close()
	var scenes []struct {
		Name      string
		CameraY   int
		Destroyed [][2]int
		Pixels    []byte
	}
	if err := json.NewDecoder(packed).Decode(&scenes); err != nil {
		t.Fatal(err)
	}
	if len(scenes) != 10 {
		t.Fatal("incomplete original terrain observations")
	}
	compared := 0
	for _, source := range scenes {
		t.Run(source.Name, func(t *testing.T) {
			if len(source.Pixels) != playfieldWidth*playfieldHeight {
				t.Fatal("invalid original terrain dimensions")
			}
			g, err := New(Options{Mute: true})
			if err != nil {
				t.Fatal(err)
			}
			for _, tile := range source.Destroyed {
				g.destroyed[tile[0]][tile[1]] = true
			}
			ignored := make([]bool, len(source.Pixels))
			for i, index := range source.Pixels {
				ignored[i] = index == 255
				if index != 255 {
					compared++
					if index > 15 {
						t.Fatal("invalid original terrain color")
					}
				}
			}
			screen := ebiten.NewImage(ScreenWidth, ScreenHeight)
			check := &originalSmallDrawCheck{g: g, background: make([]byte, len(source.Pixels)*4), expected: source.Pixels, ignore: ignored}
			check.draw = func() { g.drawBackdrop(screen, float64(source.CameraY)) }
			smallDrawScenes <- check
			if err := <-smallDrawResults; err != nil {
				t.Fatal(err)
			}
		})
	}
	t.Logf("compared%d unobscured original terrain pixels", compared)
}

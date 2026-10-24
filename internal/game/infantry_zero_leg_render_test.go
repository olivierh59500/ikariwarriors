//go:build rendercheck

package game

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

// These 24 controlled poses were drawn by the actual original $11f32 painter.
// A zero leg pointer selects its complete 24-row body format, whereas nonzero
// pointers select the ordinary 16-row torso followed by eight separate legs.
func TestOriginalControlledZeroLegInfantryPainting(t *testing.T) {
	file, err := os.Open("testdata/infantry-zero-leg-painter.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	packed, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	defer packed.Close()
	var original []struct {
		Batch int
		Scene []struct {
			Kind, SourceSlot, X, Y, Upper, Lower int
		}
		Before, After []uint8
	}
	if err := json.NewDecoder(packed).Decode(&original); err != nil {
		t.Fatal(err)
	}
	if len(original) != 4 {
		t.Fatal("incomplete original zero-leg painter scenes")
	}
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, source := range original {
		t.Run(fmt.Sprintf("scene-%02d", source.Batch), func(t *testing.T) {
			if len(source.Scene) != 6 || len(source.Before) != playfieldWidth*playfieldHeight || len(source.After) != len(source.Before) {
				t.Fatal("invalid original zero-leg painter observation")
			}
			background := make([]byte, len(source.Before)*4)
			for i, index := range source.Before {
				r, green, b, _ := g.resources.colors[index].RGBA()
				copy(background[i*4:], []byte{byte(r >> 8), byte(green >> 8), byte(b >> 8), 255})
			}
			actors := make([]actor, 0, len(source.Scene))
			for _, pose := range source.Scene {
				value := originalRetainedInfantryActor(pose.Kind, originalRetainedInfantryState{Upper: pose.Upper, Lower: pose.Lower})
				value.x, value.y, value.slot = float64(pose.X-48), float64(pose.Y-4), pose.SourceSlot
				upper, lower := value.infantryArtAddresses()
				if upper != pose.Upper || lower != pose.Lower {
					t.Fatal("native pose does not address the measured original artwork")
				}
				actors = append(actors, value)
			}
			scene := &originalSmallDrawCheck{g: g, background: background, expected: source.After}
			scene.draw = func() {
				for i := range actors {
					g.drawActor(g.playfield, &actors[i], -1)
				}
			}
			smallDrawScenes <- scene
			if err := <-smallDrawResults; err != nil {
				t.Fatal(err)
			}
		})
	}
}

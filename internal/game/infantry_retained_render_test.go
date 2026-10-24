//go:build rendercheck

package game

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

// Retained death phases can select half-frame byte offsets after a walking
// table's first terminator. These twelve controlled poses were rendered by
// the original $11f32 painter, including normal/red banks and water overlays.
func TestOriginalControlledRetainedInfantryPainting(t *testing.T) {
	file, err := os.Open("testdata/infantry-retained-painter.json.gz")
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
			Kind, SourceSlot, X, Y, Upper, Lower         int
			UpperStep, UpperDelay, LowerStep, LowerDelay int
		}
		Before, After []uint8
	}
	if err := json.NewDecoder(packed).Decode(&original); err != nil {
		t.Fatal(err)
	}
	if len(original) != 2 {
		t.Fatalf("original retained-phase painter scenes: got %d, want 2", len(original))
	}
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	poses := 0
	for _, source := range original {
		t.Run(fmt.Sprintf("scene-%02d", source.Batch), func(t *testing.T) {
			if len(source.Scene) != 6 || len(source.Before) != playfieldWidth*playfieldHeight || len(source.After) != len(source.Before) {
				t.Fatal("invalid original retained-phase painter observation")
			}
			background := make([]byte, len(source.Before)*4)
			for i, index := range source.Before {
				r, green, b, _ := g.resources.colors[index].RGBA()
				copy(background[i*4:], []byte{byte(r >> 8), byte(green >> 8), byte(b >> 8), 255})
			}
			actors := make([]actor, 0, len(source.Scene))
			for _, pose := range source.Scene {
				value := originalRetainedInfantryActor(pose.Kind, originalRetainedInfantryState{Upper: pose.Upper, Lower: pose.Lower, UpperStep: pose.UpperStep, UpperDelay: pose.UpperDelay, LowerStep: pose.LowerStep, LowerDelay: pose.LowerDelay})
				value.x, value.y, value.slot = float64(pose.X-48), float64(pose.Y-4), pose.SourceSlot
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
			poses += len(actors)
		})
	}
	if poses != 12 {
		t.Fatalf("original retained-phase painter poses: got %d, want 12", poses)
	}
}

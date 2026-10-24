//go:build rendercheck

package game

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

// The original $11f32 painter rendered these 72 poses after controlled
// wet/dry $fb82 calls. Both shared water leg images are covered for normal
// and red soldiers, alongside their healthy and death torso images.
func TestOriginalControlledWetInfantryPainting(t *testing.T) {
	file, err := os.Open("testdata/infantry-wet-painter.json.gz")
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
			Kind, SourceSlot, X, Y, Upper, Lower, Dead, Fall int
			UpperStep, UpperDelay, LowerStep, LowerDelay     int
		}
		Before, After []uint8
	}
	if err := json.NewDecoder(packed).Decode(&original); err != nil {
		t.Fatal(err)
	}
	if len(original) != 12 {
		t.Fatalf("controlled original painter scenes: got %d, want 12", len(original))
	}
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	poses := 0
	for _, source := range original {
		t.Run(fmt.Sprintf("scene-%02d", source.Batch), func(t *testing.T) {
			if len(source.Scene) != 6 || len(source.Before) != playfieldWidth*playfieldHeight || len(source.After) != len(source.Before) {
				t.Fatal("invalid controlled original painter observation")
			}
			background := make([]byte, len(source.Before)*4)
			for i, index := range source.Before {
				r, green, b, _ := g.resources.colors[index].RGBA()
				copy(background[i*4:], []byte{byte(r >> 8), byte(green >> 8), byte(b >> 8), 255})
			}
			actors := make([]actor, 0, len(source.Scene))
			for _, pose := range source.Scene {
				value := originalWetInfantryActor(pose.Kind, originalWetInfantryState{Upper: pose.Upper, Lower: pose.Lower, Step: pose.LowerStep, Delay: pose.LowerDelay})
				value.x, value.y = float64(pose.X-48), float64(pose.Y-4)
				value.slot, value.deadTicks = pose.SourceSlot, pose.Fall
				value.upper.step, value.upper.delay = pose.UpperStep, pose.UpperDelay
				if pose.Dead != 0 {
					value.health = 0
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
			poses += len(actors)
		})
	}
	if poses != 72 {
		t.Fatalf("controlled original painter poses: got %d, want 72", poses)
	}
}

//go:build rendercheck

package game

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

// These complete indexed rasters were measured from the original $11f32
// painter. All 50 normal/red torso and leg combinations were observed during
// consecutive, normally played Amiga frames, including both death sequences
// and images retained through blocked steps, turns and reused actor slots.
func TestOriginalInfantryPainting(t *testing.T) {
	file, err := os.Open("testdata/infantry-painter.json.gz")
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
			Tick, SourceSlot, Kind, X, Y, Upper, Lower, Dead, Fall int
			UpperStep, UpperDelay, LowerStep, LowerDelay           int
		}
		Before, After []uint8
	}
	if err := json.NewDecoder(packed).Decode(&original); err != nil {
		t.Fatal(err)
	}
	if len(original) != 9 {
		t.Fatalf("original infantry painter scenes: got %d, want 9", len(original))
	}
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	poses := 0
	for _, source := range original {
		t.Run(fmt.Sprintf("scene-%02d", source.Batch), func(t *testing.T) {
			if len(source.Scene) == 0 || len(source.Before) != playfieldWidth*playfieldHeight || len(source.After) != len(source.Before) {
				t.Fatal("invalid original infantry painter observation")
			}
			background := make([]byte, len(source.Before)*4)
			for i, index := range source.Before {
				if int(index) >= len(g.resources.colors) {
					t.Fatal("invalid original infantry background color")
				}
				r, green, b, _ := g.resources.colors[index].RGBA()
				copy(background[i*4:], []byte{byte(r >> 8), byte(green >> 8), byte(b >> 8), 255})
			}
			actors := make([]actor, 0, len(source.Scene))
			for _, pose := range source.Scene {
				value := actor{
					x: float64(pose.X - 48), y: float64(pose.Y - 4),
					kind: pose.Kind, slot: pose.SourceSlot, health: 1, deadTicks: pose.Fall,
					upper: walkAnimation{step: pose.UpperStep, delay: pose.UpperDelay},
					lower: walkAnimation{step: pose.LowerStep, delay: pose.LowerDelay},
				}
				if pose.Dead != 0 {
					value.health = 0
				}
				upperBase, lowerBase := 0x4cb88, 0x50288
				if pose.Kind == 5 {
					upperBase, lowerBase = 0x51e08, 0x54608
				}
				value.upper.frame = (pose.Upper - upperBase) / actorBobBytes
				switch pose.Lower {
				case 0:
					value.lower.frame = -1
				case 0x4a1a8:
					value.lower.frame = -2
				case 0x4a248:
					value.lower.frame = -3
				default:
					value.lower.frame = (pose.Lower - lowerBase) / overlayBobBytes
				}
				upper, lower := value.infantryArtAddresses()
				if upper != pose.Upper || lower != pose.Lower {
					t.Fatalf("original frame %d, slot %d: graphic addresses %x/%x, want %x/%x", pose.Tick, pose.SourceSlot, upper, lower, pose.Upper, pose.Lower)
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
	if poses != 50 {
		t.Fatalf("original infantry poses: got %d, want 50", poses)
	}
}

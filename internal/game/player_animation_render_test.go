//go:build rendercheck

package game

import (
	"fmt"
	"testing"
)

// The original painter produces all expected pixels, including bank-crossing
// torso offsets and water images. Both native player painters use those exact
// recovered offsets rather than limiting a player to its usual image slice.
func TestOriginalControlledRetainedPlayerPainting(t *testing.T) {
	var scenes []struct {
		Batch int
		Scene []struct {
			Player, X, Y, Upper, Lower, UpperStep, UpperDelay, LowerStep, LowerDelay int
		}
		Before, After []byte
	}
	readOriginalPlayerAnimation(t, "testdata/player-retained-painter.json.gz", &scenes)
	if len(scenes) != 4 {
		t.Fatal("incomplete original player painter scenes")
	}
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.cameraY = 0
	poses := 0
	for _, source := range scenes {
		t.Run(fmt.Sprintf("scene-%d", source.Batch), func(t *testing.T) {
			if len(source.Scene) != 6 || len(source.Before) != playfieldWidth*playfieldHeight || len(source.After) != len(source.Before) {
				t.Fatal("invalid original player painter observation")
			}
			background := make([]byte, len(source.Before)*4)
			for i, index := range source.Before {
				r, green, b, _ := g.resources.colors[index].RGBA()
				copy(background[i*4:], []byte{byte(r >> 8), byte(green >> 8), byte(b >> 8), 255})
			}
			actors := make([]actor, len(source.Scene))
			for i, p := range source.Scene {
				actors[i] = observedPlayerAnimation(p.Player, originalPlayerAnimationState{Upper: p.Upper, Lower: p.Lower, UpperStep: p.UpperStep, UpperDelay: p.UpperDelay, LowerStep: p.LowerStep, LowerDelay: p.LowerDelay})
				actors[i].x, actors[i].y = float64(p.X-48), float64(p.Y-4)
			}
			check := &originalSmallDrawCheck{g: g, background: background, expected: source.After}
			check.draw = func() {
				for i, p := range source.Scene {
					g.drawActor(g.playfield, &actors[i], p.Player)
				}
			}
			smallDrawScenes <- check
			if err := <-smallDrawResults; err != nil {
				t.Fatal(err)
			}
			poses += len(actors)
		})
	}
	if poses != 24 {
		t.Fatal("incomplete source player painter poses")
	}
}

//go:build rendercheck

package game

import "testing"

// The original $11f32 painter ran after the actual protection block. Compare
// the first player's measured rectangle for hidden/visible drawActor output.
// The controlled second-player art pointer was outside its ordinary bank,
// so its pixels are excluded; both players have separate transition proof.
func TestOriginalPlayerBlinkPainting(t *testing.T) {
	original := readOriginalPlayerBlink(t)
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, source := range original.Scenes {
		background := make([]byte, len(source.Before)*4)
		for i, index := range source.Before {
			r, green, b, _ := g.resources.colors[index].RGBA()
			copy(background[i*4:], []byte{byte(r >> 8), byte(green >> 8), byte(b >> 8), 255})
		}
		ignore := make([]bool, len(source.Before))
		for i := range ignore {
			ignore[i] = true
		}
		for _, pose := range source.Scene {
			if pose.Player != 0 {
				continue
			}
			for y := pose.Y - 8; y < pose.Y-8+actorBobHeight+overlayBobHeight; y++ {
				for x := pose.X - 64; x < pose.X-64+actorBobWidth; x++ {
					ignore[y*playfieldWidth+x] = false
				}
			}
		}
		scene := &originalSmallDrawCheck{g: g, background: background, expected: source.After, ignore: ignore}
		scene.draw = func() {
			for _, pose := range source.Scene {
				if pose.Player != 0 {
					continue
				}
				value := actor{x: float64(pose.X - 48), y: float64(pose.Y - 4), health: 1, invulnerable: source.Case.Protection, blinkPhase: uint8(source.Case.Phase)}
				if source.Case.Mode == "inactive" {
					value.health = 0
				} else if source.Case.Mode == "falling" {
					value.health, value.deadTicks = 0, 1
				}
				value.upper.frame = (pose.Upper - 0x42b88 - pose.Player*22*actorBobBytes) / actorBobBytes
				value.lower.frame = (pose.Lower - 0x46288) / overlayBobBytes
				value.advancePlayerBlink(source.Case.Protection > 0)
				g.drawActor(g.playfield, &value, pose.Player)
			}
		}
		smallDrawScenes <- scene
		if err := <-smallDrawResults; err != nil {
			t.Fatalf("original blink scene %d: %v", source.Index, err)
		}
	}
}

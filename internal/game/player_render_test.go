//go:build rendercheck

package game

import (
	"compress/gzip"
	"encoding/json"
	"os"
	"testing"
)

// The before/after pixels come from the original $11f32 wide-bob painter.
// Poses include both players' observed walking and newly falling parts, so
// this verifies the native GPU compositor rather than just graphic indices.
func TestOriginalPlayerControlPainting(t *testing.T) {
	file, err := os.Open("testdata/player-control-painter.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	packed, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	defer packed.Close()
	var original struct {
		Scene         []struct{ Frame, Player, X, Y, Upper, Lower, Fall int }
		Before, After []uint8
	}
	if err := json.NewDecoder(packed).Decode(&original); err != nil {
		t.Fatal(err)
	}
	if len(original.Scene) == 0 || len(original.Before) != playfieldWidth*playfieldHeight || len(original.After) != len(original.Before) {
		t.Fatal("invalid original player painter observations")
	}
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	background := make([]byte, len(original.Before)*4)
	for i, index := range original.Before {
		r, green, b, _ := g.resources.colors[index].RGBA()
		copy(background[i*4:], []byte{byte(r >> 8), byte(green >> 8), byte(b >> 8), 255})
	}
	scene := &originalSmallDrawCheck{g: g, background: background, expected: original.After}
	scene.draw = func() {
		for _, source := range original.Scene {
			value := actor{x: float64(source.X - 48), y: float64(source.Y - 4), health: 1, deadTicks: source.Fall}
			upperBase := 0x42b88 + source.Player*22*actorBobBytes
			lowerBase := 0x46288
			if source.Upper >= 0x47cc8 {
				upperBase, value.upperFalling = 0x47cc8, true
			}
			if source.Lower >= 0x49ac8 {
				lowerBase, value.lowerFalling = 0x49ac8, true
			}
			value.upper.frame = (source.Upper - upperBase) / actorBobBytes
			value.lower.frame = (source.Lower - lowerBase) / overlayBobBytes
			g.drawActor(g.playfield, &value, source.Player)
		}
	}
	smallDrawScenes <- scene
	if err := <-smallDrawResults; err != nil {
		t.Fatal(err)
	}
}

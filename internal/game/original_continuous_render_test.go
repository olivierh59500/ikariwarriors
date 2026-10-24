//go:build rendercheck

package game

import (
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// The corpus contains 512 consecutive original 320x200 palette-index frames in order.
// Its source-RAM identities match the core replay; no native image is a golden.
func readContinuousOriginalPixels(t *testing.T) []byte {
	t.Helper()
	manifestData, err := os.ReadFile("testdata/original-continuous-pixels-512.json")
	if err != nil {
		t.Fatal(err)
	}
	var manifest []struct {
		Frame                      int
		SourceSHA256, PixelsSHA256 string
	}
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open("testdata/original-continuous-pixels-512.bin.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	z, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	defer z.Close()
	pixels, err := io.ReadAll(z)
	if err != nil {
		t.Fatal(err)
	}
	source := readContinuousSourceFixture(t)
	const frameBytes = ScreenWidth * ScreenHeight
	if len(manifest) != len(source.Frames) || len(pixels) != len(manifest)*frameBytes {
		t.Fatal("incomplete original continuous pixel corpus")
	}
	for frame, row := range manifest {
		original := pixels[frame*frameBytes : (frame+1)*frameBytes]
		if row.Frame != frame || row.SourceSHA256 != source.Frames[frame].Hash || row.PixelsSHA256 != fmt.Sprintf("%x", sha256.Sum256(original)) {
			t.Fatalf("frame%d original pixel/source provenance differs", frame)
		}
		for _, index := range original {
			if index >= 16 {
				t.Fatalf("frame%d original palette index exceeds 16 colors", frame)
			}
		}
	}
	return pixels
}

// The shared cold replay checks the source controls, resources, owned shots,
// randomness and every player/turret image before this full-frame observer.
// Every original pixel is compared; there are no ignored regions or PNG writes.
func TestOriginalContinuousFullRaster(t *testing.T) {
	pixels := readContinuousOriginalPixels(t)
	target := ebiten.NewImage(ScreenWidth, ScreenHeight)
	background := make([]byte, ScreenWidth*ScreenHeight*4)
	replayContinuousOriginal(t, func(frame int, g *Game) {
		first := frame * ScreenWidth * ScreenHeight
		check := &originalSmallDrawCheck{g: g, target: target, background: background, expected: pixels[first : first+ScreenWidth*ScreenHeight]}
		check.draw = func() { g.Draw(target) }
		smallDrawScenes <- check
		if err := <-smallDrawResults; err != nil {
			t.Fatalf("frame%d: %v", frame, err)
		}
	})
}

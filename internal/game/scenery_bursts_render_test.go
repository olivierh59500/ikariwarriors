//go:build rendercheck

package game

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

// Four complete eight-bob scenes come from the original $1224a painter. They
// retain its real background and independently recovered 16-color indices.
func TestOriginalSceneryBurstCompositing(t *testing.T) {
	file, err := os.Open("testdata/scenery-bursts-raster.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	stream, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	var scenes []struct {
		Phase         int
		Scene         originalSceneryState
		Before, After []uint8
	}
	if err = json.NewDecoder(stream).Decode(&scenes); err != nil {
		t.Fatal(err)
	}
	if len(scenes) != 4 {
		t.Fatal("missing four original scenery raster phases")
	}
	for _, source := range scenes {
		t.Run(fmt.Sprint(source.Phase), func(t *testing.T) {
			g, err := New(Options{Mute: true})
			if err != nil {
				t.Fatal(err)
			}
			source.Scene.supply(g)
			g.sceneryBurst.active, g.sceneryBurst.enabled, g.sceneryBurst.frame = true, 0xff, 49+source.Phase*4
			rgba := make([]byte, len(source.Before)*4)
			for i, index := range source.Before {
				r, gc, b, _ := g.resources.colors[index].RGBA()
				rgba[i*4], rgba[i*4+1], rgba[i*4+2], rgba[i*4+3] = byte(r>>8), byte(gc>>8), byte(b>>8), 255
			}
			check := &originalSmallDrawCheck{g: g, background: rgba, expected: source.After, draw: g.drawSceneryBurst}
			smallDrawScenes <- check
			if err := <-smallDrawResults; err != nil {
				t.Fatal(err)
			}
		})
	}
}

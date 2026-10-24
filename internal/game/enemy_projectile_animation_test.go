package game

import (
	"encoding/json"
	"os"
	"testing"
)

// All expected images come from uninterrupted original gameplay snapshots.
func TestOriginalEnemyProjectileImageTransitions(t *testing.T) {
	var cases []struct {
		Frame, Slot, RangeBefore, RangeAfter, Kind, ImageBefore, ImageAfter int
		BeforeSHA256, AfterSHA256                                           string
	}
	blob, err := os.ReadFile("testdata/enemy-projectile-art-transitions.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(blob, &cases); err != nil {
		t.Fatal(err)
	}
	if len(cases) != 39 {
		t.Fatal("missing original projectile transitions")
	}
	for _, source := range cases {
		if source.RangeBefore-source.RangeAfter != 4 || len(source.BeforeSHA256) != 64 || len(source.AfterSHA256) != 64 {
			t.Fatal("incomplete original transition provenance")
		}
		shot := projectile{enemy: true, grenade: true, frame: (source.ImageBefore - 0x4a2e8) / overlayBobBytes, life: source.RangeAfter / 4}
		advanceEnemyProjectileImage(&shot)
		if got := 0x4a2e8 + shot.frame*overlayBobBytes; got != source.ImageAfter {
			t.Fatalf("original frame%d slot%d image%x, native%x", source.Frame, source.Slot, source.ImageAfter, got)
		}
	}
}

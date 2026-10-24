package game

import (
	"encoding/json"
	"os"
	"testing"
)

// Original frame945 retains three active, already-hit infantry BOBs. At946
// collecting kind5 adds300 and clears that pickup. C238-C264 tests activity
// alone; its mark-and-score pass does not restart an existing fall animation.
func TestOriginalSmartBombScoresActiveFallingInfantry(t *testing.T) {
	var source struct {
		Frames []struct {
			Score    int `json:"score"`
			Infantry []struct {
				Slot   int `json:"slot"`
				Active int `json:"active"`
				Hit    int `json:"hit"`
			} `json:"infantry"`
		} `json:"frames"`
	}
	data, err := os.ReadFile("testdata/smart-bomb-active-falls.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &source); err != nil {
		t.Fatal(err)
	}
	if len(source.Frames) != 2 {
		t.Fatal("missing independent original frame pair")
	}
	for _, player := range []int{0, 1} {
		g, err := New(Options{Mute: true})
		if err != nil {
			t.Fatal(err)
		}
		g.startGamePlayers(false, 2)
		g.enemies = nil
		for _, value := range source.Frames[0].Infantry {
			if value.Active != 1 || value.Hit != 1 {
				t.Fatal("original actor is not active and already hit")
			}
			g.enemies = append(g.enemies, actor{slot: value.Slot, health: 0, deadTicks: 5,
				upper: walkAnimation{step: 2, delay: 1, frame: 38}})
		}
		// The adjacent large-object pools fall outside the original0..18 scan.
		g.enemies = append(g.enemies, actor{slot: 19, health: 1}, actor{slot: 32, health: 1})
		before := append([]actor(nil), g.enemies...)
		g.score, g.score2 = source.Frames[0].Score, source.Frames[0].Score
		g.collectPickup(player, 5)
		got, other := g.score, g.score2
		if player == 1 {
			got, other = other, got
		}
		if got != source.Frames[1].Score || other != source.Frames[0].Score {
			t.Fatalf("smart bomb score%d, original%d; other%d", got, source.Frames[1].Score, other)
		}
		for index, enemy := range g.enemies {
			if enemy != before[index] {
				t.Fatal("smart bomb restarted a fall or changed an excluded actor")
			}
		}
	}
}

package game

import (
	"encoding/json"
	"os"
	"testing"
)

func TestOriginalFinalGrenadeQuadrantAwards(t *testing.T) {
	var source struct {
		Frames []struct {
			Score, Camera int
			FinalActive   bool
			Parts         []struct {
				Slot, X, Y  int
				Active, Hit bool
			}
			Shot struct{ X, Y int }
		}
	}
	data, err := os.ReadFile("testdata/original-final-quadrant-source.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &source); err != nil {
		t.Fatal(err)
	}
	if len(source.Frames) != 2 || !source.Frames[0].FinalActive || source.Frames[1].FinalActive {
		t.Fatal("missing actual original final defeat boundary")
	}
	before, after := source.Frames[0], source.Frames[1]
	for player := 0; player < 2; player++ {
		g, err := New(Options{Mute: true})
		if err != nil {
			t.Fatal(err)
		}
		g.cameraY = float64(before.Camera)
		g.final = finalEncounter{x: float64(before.Parts[0].X), y: float64(before.Parts[0].Y), active: true}
		g.score, g.score2 = before.Score, before.Score
		g.damageFinalEncounter(projectile{x: float64(after.Shot.X), y: float64(after.Shot.Y), owner: player, grenade: true}, false)
		got, other := g.score, g.score2
		if player == 1 {
			got, other = other, got
		}
		if got != after.Score || other != before.Score || g.final.active || !g.final.defeated {
			t.Fatalf("player%d final quadrant score/state differs: score%d/%d state%+v", player, got, other, g.final)
		}
		// A miss has no award or defeat, guarding against a fixed completion bonus.
		g.final.active, g.final.defeated = true, false
		g.score, g.score2 = before.Score, before.Score
		g.damageFinalEncounter(projectile{x: float64(after.Shot.X - 300), y: float64(after.Shot.Y), owner: player, grenade: true}, false)
		if g.score != before.Score || g.score2 != before.Score || !g.final.active || g.final.defeated {
			t.Fatal("a missed final grenade awarded points or defeated the encounter")
		}
	}
}

package game

import (
	"encoding/json"
	"os"
	"testing"
)

// These isolated original weapon updates start at the bridge reached during
// ordinary play. Both players' grenades and tank shells were measured on
// either side of its tile boundaries; the visible quad is not a terrain mask.
func TestOriginalSceneryImpactProbe(t *testing.T) {
	data, err := os.ReadFile("testdata/scenery-impact-probes.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Owner, X, Y, ScoreDelta int
		Weapon                  string
		Changed                 []struct{ Row, Column, Before, After int }
	}
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	if len(cases) == 0 {
		t.Fatal("missing independent original scenery observations")
	}
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	for i, original := range cases {
		g.startGamePlayers(false, 2)
		g.enemies = nil
		g.damageGrenadeArea(projectile{x: float64(original.X), y: float64(original.Y), owner: original.Owner, grenade: original.Weapon == "grenade", cannon: original.Weapon == "cannon"})
		var expected [mapRows][mapColumns]bool
		for _, cell := range original.Changed {
			if cell.After != 0 || g.resources.levelCodes[cell.Row][cell.Column] != byte(cell.Before) {
				t.Fatalf("case%d original scenery does not match the recovered map", i)
			}
			expected[cell.Row][cell.Column] = true
		}
		if g.destroyed != expected {
			t.Fatalf("case%d player%d %s at%d/%d changed a different terrain footprint", i, original.Owner+1, original.Weapon, original.X, original.Y)
		}
		score, other := g.score, g.score2
		if original.Owner == 1 {
			score, other = other, score
		}
		if score != original.ScoreDelta || other != 0 {
			t.Fatalf("case%d terrain score%d/%d differs from the original%d", i, score, other, original.ScoreDelta)
		}
	}
}

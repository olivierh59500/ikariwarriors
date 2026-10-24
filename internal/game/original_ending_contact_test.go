package game

import (
	"encoding/json"
	"os"
	"testing"
)

func TestOriginalEndingContactAwardsOnceWithoutAFall(t *testing.T) {
	var source struct {
		Frames []struct {
			Camera, Score int
			Player        struct{ X, Y int }
			Parts         []struct {
				Slot, X, Y  int
				Active, Hit bool
			}
		}
	}
	data, err := os.ReadFile("testdata/original-ending-contact-source.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &source); err != nil {
		t.Fatal(err)
	}
	if len(source.Frames) != 3 {
		t.Fatal("missing actual ending contact/retained-hit frames")
	}
	before, after := source.Frames[0], source.Frames[1]
	for player := 0; player < 2; player++ {
		g, err := New(Options{Mute: true})
		if err != nil {
			t.Fatal(err)
		}
		g.playerCount = 2
		g.cameraY = float64(before.Camera)
		g.ending = endingSequence{x: float64(before.Parts[0].X), y: float64(before.Parts[0].Y), active: true}
		g.player = actor{x: float64(before.Player.X), y: float64(before.Player.Y), health: 1}
		g.player2 = g.player
		g.score, g.score2 = before.Score, before.Score
		g.hitPlayerWithInfantry(player)
		got, other := g.score, g.score2
		if player == 1 {
			got, other = other, got
		}
		if got != after.Score || other != before.Score || g.ending.hitParts != 1<<9 || g.player.deadTicks != 0 || g.player2.deadTicks != 0 || !g.ending.active {
			t.Fatal("original ending contact award, mark or immunity differs")
		}
		g.hitPlayerWithInfantry(player)
		if g.score+g.score2 != before.Score+after.Score {
			t.Fatal("retained ending-piece mark awarded a second contact")
		}
	}
}

func TestEndingHitMarksAreSharedWithWeapons(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.cameraY = 36
	g.ending = endingSequence{x: 80, y: 40, active: true}
	g.player = actor{x: 70, y: 130, health: 1}
	if !g.damageBulletTarget(projectile{x: 80, y: 116, owner: 0, power: 1}) || g.score != 100 || g.ending.hitParts != 1<<9 {
		t.Fatal("ending bullet did not mark its original piece")
	}
	g.hitPlayerWithInfantry(0)
	if g.score != 100 {
		t.Fatal("body contact ignored the ending piece's existing weapon hit")
	}
	g.ending.hitParts = 0
	g.score = 0
	g.damageFinalEncounter(projectile{x: 80, y: 128, enemy: true, owner: -1, grenade: true}, false)
	if g.score != 0 || g.ending.hitParts != 1<<9 {
		t.Fatal("enemy ending blast awarded points or lost its shared hit mark")
	}
}

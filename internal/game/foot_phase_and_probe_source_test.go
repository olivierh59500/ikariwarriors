package game

import (
	"encoding/json"
	"os"
	"testing"
)

type sourceFootContact struct {
	Before, After, HorizontalBirth struct {
		Camera, Score int
		Player        struct{ X, Y, Progress int }
		Shot          struct {
			X, Y, Range, Progress, Image int
			Active                       bool
		}
		Enemy11 struct {
			X, Y        int
			Active, Hit bool
		}
	}
}

func readSourceFootContact(t *testing.T) sourceFootContact {
	t.Helper()
	data, err := os.ReadFile("testdata/foot-contact-source.json")
	if err != nil {
		t.Fatal(err)
	}
	var source sourceFootContact
	if err := json.Unmarshal(data, &source); err != nil {
		t.Fatal(err)
	}
	return source
}

// Original3690 hits slot11 as its range reaches zero. The first slot is then
// reactivated by104BE/10B1A; the secondary impact tails never reactivate a hit.
func TestOriginalFootContactAtImpactBirth(t *testing.T) {
	source := readSourceFootContact(t)
	for player := 0; player < 2; player++ {
		for slot := 0; slot < 3; slot++ {
			g, err := New(Options{Mute: true})
			if err != nil {
				t.Fatal(err)
			}
			g.playerCount = 2
			g.cameraY = float64(source.Before.Camera)
			g.enemies = []actor{{slot: 11, x: float64(source.Before.Enemy11.X), y: float64(source.Before.Enemy11.Y), health: 1}}
			g.score, g.score2 = source.Before.Score, source.Before.Score
			shot := projectile{x: float64(source.Before.Shot.X), y: float64(source.Before.Shot.Y), vx: -6, vy: -6, owner: player, slot: slot, power: 2, life: source.Before.Shot.Range / 4, frame: 2}
			moved, active := g.advanceOwnedProjectile(shot)
			if active != (slot == 0) || moved.impact != 3 || int(moved.x) != source.After.Shot.X || int(moved.y) != source.After.Shot.Y {
				t.Fatalf("player%d slot%d collision/impact precedence differs: active%t moved%+v", player, slot, active, moved)
			}
			if g.enemies[0].health > 0 {
				t.Fatal("retained first-slot impact skipped its actor hit")
			}
			got, other := g.score, g.score2
			if player == 1 {
				got, other = other, got
			}
			if got != source.After.Score || other != source.Before.Score {
				t.Fatal("impact birth changed the original actor award")
			}
			pending := *g.pendingOwnedFootImpact(player, slot)
			if slot == 0 && pending != 0 || slot != 0 && pending != 3 {
				t.Fatal("impact visibility disagrees with its independent cleanup timer")
			}
		}
	}
}

// Original4824's horizontal first-slot progress is6133-6=6127. Its visible
// bullet origin is four pixels below that independent terrain probe.
func TestOriginalHorizontalFirstSlotTerrainOrigin(t *testing.T) {
	source := readSourceFootContact(t)
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	row := source.HorizontalBirth
	g.cameraY = float64(row.Camera)
	g.player = actor{x: float64(row.Player.X), y: float64(row.Player.Y), health: 1, direction: 6}
	// The original allocator chooses slot1 before0; keep that earlier slot busy.
	g.projectiles = []projectile{{owner: 0, slot: 1, life: 5}}
	shot := g.projectileFor(&g.player, false, 0, false, 6)
	if shot.slot != 0 {
		t.Fatal("original first-slot allocation was not exercised")
	}
	shot.x, shot.y = shot.x+shot.vx, shot.y+shot.vy
	if int(shot.x) != row.Shot.X || int(shot.y) != row.Shot.Y {
		t.Fatal("source horizontal projectile coordinates differ")
	}
	got := shot.y - 4 + shot.terrainBias
	want := float64(levelHeight - 16 - row.Shot.Progress)
	if got != want {
		t.Fatalf("horizontal terrain probeY%v, original%v", got, want)
	}
}

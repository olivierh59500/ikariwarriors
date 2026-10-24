package game

import (
	"encoding/json"
	"os"
	"strconv"
	"testing"
)

// These are complete original gameplay frames, not calls to the projectile
// update alone. They distinguish one owned damage pulse from three enemy
// pulses and the different start times of upgraded and heavy-shell fragments.
func TestOriginalCompleteGrenadeImpactTimelines(t *testing.T) {
	data, err := os.ReadFile("testdata/impacts.json")
	if err != nil {
		t.Fatal(err)
	}
	type part struct {
		X, Y, Active int
		Bob          string
	}
	var cases []struct {
		Kind                               string
		Tick, BossHealth, QuadActive       int
		BurstActive, BurstRemaining, Owner int
		CameraY, Score                     int
		Actors, Burst                      []part
	}
	if err = json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	var g *Game
	kind := ""
	for _, original := range cases {
		if kind != original.Kind {
			kind = original.Kind
			g, err = New(Options{Mute: true})
			if err != nil {
				t.Fatal(err)
			}
			g.cameraY = float64(original.CameraY)
			g.score = original.Score
			g.playerCount = 1
			g.player = actor{x: 52, y: g.cameraY + 78, health: 1, inTank: true}
			g.spawnMiddleBoss(levelEvent{kind: 30, progress: 8200})
			g.boss.y = g.cameraY + 28
			g.grenadePower[0] = 1
			enemy := kind == "enemy" || kind == "heavy"
			owner, frame := 0, 0
			if enemy {
				owner, frame = -1, 9
			}
			if kind == "heavy" {
				frame = 8
			}
			if kind == "super" {
				g.grenadePower[0] = 2
			}
			g.projectiles = []projectile{{x: 94, y: g.cameraY + 60, vy: 4, life: 1, enemy: enemy, owner: owner, grenade: true, frame: frame, fragments: kind == "heavy"}}
		}
		if original.Tick > 0 {
			g.updateProjectiles()
			g.updateMiddleBoss()
			g.boss.applyDamage()
		}
		if g.boss.health != original.BossHealth {
			t.Fatalf("%s tick%d damage rounds: health%d, original%d", kind, original.Tick, g.boss.health, original.BossHealth)
		}
		if g.score != original.Score {
			t.Fatalf("%s tick%d score%d, original%d", kind, original.Tick, g.score, original.Score)
		}
		var blast *projectile
		pool := 0
		if kind == "enemy" || kind == "heavy" {
			pool = 2
		}
		if g.blasts[pool].impact > 0 {
			blast = &g.blasts[pool]
		}
		if (blast != nil) != (original.QuadActive != 0) {
			t.Fatalf("%s tick%d blast lifetime differs", kind, original.Tick)
		}
		if blast != nil {
			for i, object := range original.Actors {
				address, err := strconv.ParseInt(object.Bob, 0, 32)
				if err != nil {
					t.Fatal(err)
				}
				x, y := blast.x+48+float64(i%2*16), blast.y-g.cameraY-8+float64(i/2*16)
				if int(x) != object.X || int(y) != object.Y || 0x4a2e8+grenadeImpactFrame(blast.impact, i)*160 != int(address) {
					t.Fatalf("%s tick%d quadrant%d differs from the original", kind, original.Tick, i)
				}
			}
		}
		if g.burst.active != (original.BurstActive != 0) {
			t.Fatalf("%s tick%d fragment start differs", kind, original.Tick)
		}
		if g.burst.active {
			if g.burst.remaining != original.BurstRemaining || g.burst.owner != -1 || original.Owner != 0 {
				t.Fatalf("%s tick%d fragment lifetime or source ownership differs", kind, original.Tick)
			}
			for i, object := range original.Burst {
				point := g.burst.parts[i]
				if int(point.x+56) != object.X || int(point.y-g.cameraY) != object.Y {
					t.Fatalf("%s tick%d fragment%d position differs", kind, original.Tick, i)
				}
			}
		}
	}
}

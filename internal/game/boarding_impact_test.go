package game

import (
	"compress/gzip"
	"encoding/json"
	"os"
	"testing"
)

// Controlled original calls first derive a real foot-impact image. Ordinary
// original boarding leaves its late image timer independent of cannon updates.
func TestOriginalBoardingDuringAnExistingFootImpact(t *testing.T) {
	file, err := os.Open("testdata/boarding-during-impact.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	packed, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	defer packed.Close()
	type originalState struct {
		RawX, RawY, Art, Range, Progress, Mask, ImpactPhase, QuadPhase, Score int
		Active, ImpactActive, QuadActive, Mounted, TargetHit                  bool
	}
	var original struct {
		Cases []struct {
			Name, Scenario                         string
			Owner, Phase                           int
			Before, Boarded, AfterOwned, AfterLate originalState
			Players                                []struct{ RawX, RawY int }
			Tank                                   struct{ RawX, RawY, Fuel int }
			Target                                 struct {
				RawX, RawY int
				Active     bool
			}
			SourceSHA256 map[string]string
			QuadParts    []struct {
				RawX, RawY, Art int
				Active          bool
			}
		}
	}
	if err = json.NewDecoder(packed).Decode(&original); err != nil {
		t.Fatal(err)
	}
	if len(original.Cases) != 18 {
		t.Fatal("incomplete original boarding-impact matrix")
	}
	for _, source := range original.Cases {
		t.Run(source.Name, func(t *testing.T) {
			if len(source.SourceSHA256) != 5 || !source.Before.ImpactActive || !source.Boarded.Mounted || source.Before.Mounted {
				t.Fatal("missing original impact derivation or actual boarding provenance")
			}
			g, err := New(Options{Mute: true})
			if err != nil {
				t.Fatal(err)
			}
			g.startGamePlayers(false, 2)
			g.scene = playScene
			// Normalize the controlled source progress probe to the same recovered
			// world cell. Raw relative sprite/contact coordinates remain unchanged.
			g.cameraY = float64(levelHeight - 16 - source.Before.Progress + 4 - source.Before.RawY)
			g.cameraBackY = g.cameraY
			for player, position := range source.Players {
				value := actor{x: float64(position.RawX - 48), y: g.cameraY + float64(position.RawY-4), health: 1}
				if player == 0 {
					g.player = value
				} else {
					g.player2 = value
				}
			}
			g.enemies = []actor{{x: float64(source.Tank.RawX - 48), y: g.cameraY + float64(source.Tank.RawY-4), kind: 4, slot: 19, health: 1, tankFuel: source.Tank.Fuel}}
			if source.Target.Active {
				g.enemies = append(g.enemies, actor{x: float64(source.Target.RawX - 48), y: g.cameraY + float64(source.Target.RawY-4), health: 1, kind: 2, slot: 3})
			}
			g.pickups, g.mines = nil, nil
			g.ammunition, g.ammunition2, g.grenades, g.grenades2 = 17, 17, 23, 23
			g.eventIndex, g.randomSpawn, g.redSpawn = len(g.resources.events), false, false
			// The finite impact timer represents the source's wrapped range; it
			// must still advance after a same-input ordinary context action.
			g.projectiles = []projectile{{x: float64(source.Before.RawX - 56), y: g.cameraY + float64(source.Before.RawY), vx: 0, vy: -6, owner: source.Owner, slot: 0, power: 1, frame: 1, impact: 3 - source.Phase + 1}}
			controls := [2]playerControls{}
			controls[source.Owner].action = true
			g.stepPlay(controls)
			value := g.player
			score := g.score
			if source.Owner == 1 {
				value, score = g.player2, g.score2
			}
			if !value.inTank || g.ammunition != 17 || g.ammunition2 != 17 || g.grenades != 23 || g.grenades2 != 23 {
				t.Fatal("ordinary context boarding did not preserve finite resources")
			}
			want := source.AfterLate
			if int(g.ownedShotMask(source.Owner)) != want.Mask {
				t.Errorf("owned mask native%x original%x", g.ownedShotMask(source.Owner), want.Mask)
			}
			var visible *projectile
			for index := range g.projectiles {
				shot := &g.projectiles[index]
				if !shot.enemy && shot.owner == source.Owner && shot.slot == 0 {
					visible = shot
				}
			}
			if (visible != nil) != want.Active {
				t.Errorf("shot active native%t original%t", visible != nil, want.Active)
			}
			if visible != nil && (int(visible.x+56) != want.RawX || int(visible.y-g.cameraY) != want.RawY || 0x4a2e8+g.projectileFrame(*visible)*160 != want.Art) {
				t.Errorf("retained late foot impact native%+v original%+v", *visible, want)
			}
			quad := g.blasts[source.Owner]
			if (quad.impact > 0) != want.QuadActive {
				t.Errorf("quad native%+v original active%t phase%d", quad, want.QuadActive, want.QuadPhase)
			}
			if want.QuadActive {
				if quad.impact != 4-want.QuadPhase {
					t.Errorf("quad remaining native%d original%d", quad.impact, 4-want.QuadPhase)
				}
				for index, point := range source.QuadParts {
					if int(quad.x+48)+index%2*16 != point.RawX || int(quad.y-g.cameraY)-8+index/2*16 != point.RawY || 0x4a2e8+grenadeImpactFrame(quad.impact, index)*160 != point.Art {
						t.Error("ordinary quad geometry differs from original")
					}
				}
			}
			hit := false
			for _, enemy := range g.enemies {
				if enemy.slot == 3 && enemy.health <= 0 {
					hit = true
				}
			}
			if hit != want.TargetHit || score != want.Score {
				t.Errorf("target native%t/%d original%t/%d", hit, score, want.TargetHit, want.Score)
			}
		})
	}
}

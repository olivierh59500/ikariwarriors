package game

import (
	"compress/gzip"
	"encoding/json"
	"os"
	"testing"
)

type originalImpactReuseState struct {
	RawX, RawY, Art, Range, Progress, Mask, ImpactPhase, QuadPhase, Ammo, Score int
	Active, ImpactActive, QuadActive                                            bool
}

// Original allocation preserves an older first-slot foot-impact timer even
// after that old shot created a quad. The new cannon can lose its slot early.
func TestOriginalNewCannonRetainsAnOlderFootImpactTimer(t *testing.T) {
	file, err := os.Open("testdata/boarding-impact-cannon-reuse.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	packed, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	defer packed.Close()
	var original struct {
		Cases []struct {
			Name                             string
			Owner, Phase, OwnerProgress      int
			Before, AfterBoarding, Allocated originalImpactReuseState
			Frames                           []originalImpactReuseState
			Players                          []struct{ RawX, RawY int }
			Tank                             struct{ RawX, RawY, Fuel int }
			SourceSHA256                     map[string]string
			DestroyedCells                   [][2]int
		}
	}
	if err = json.NewDecoder(packed).Decode(&original); err != nil {
		t.Fatal(err)
	}
	if len(original.Cases) != 6 {
		t.Fatal("incomplete original cannon-slot reuse observations")
	}
	for _, source := range original.Cases {
		t.Run(source.Name, func(t *testing.T) {
			if len(source.SourceSHA256) != 9 || len(source.Frames) != 3 {
				t.Fatal("missing original allocation or subsequent frame provenance")
			}
			g, err := New(Options{Mute: true})
			if err != nil {
				t.Fatal(err)
			}
			g.startGamePlayers(false, 2)
			g.scene = playScene
			// The controlled source retains the owner's logical world progress.
			// Align that ordinary new-cannon muzzle, preserving the older shot's
			// separately controlled terrain progress through its retained bias.
			g.cameraY = float64(levelHeight - 16 - source.OwnerProgress - source.Players[source.Owner].RawY + 4)
			g.cameraBackY = g.cameraY
			for player, point := range source.Players {
				value := actor{x: float64(point.RawX - 48), y: g.cameraY + float64(point.RawY-4), health: 1}
				if player == 0 {
					g.player = value
				} else {
					g.player2 = value
				}
			}
			for _, cell := range source.DestroyedCells {
				g.destroyed[cell[0]][cell[1]] = true
			}
			g.enemies = []actor{{x: float64(source.Tank.RawX - 48), y: g.cameraY + float64(source.Tank.RawY-4), kind: 4, slot: 19, health: 1, tankFuel: source.Tank.Fuel}}
			g.pickups, g.mines = nil, nil
			g.ammunition, g.ammunition2, g.grenades, g.grenades2 = 17, 17, 23, 23
			g.eventIndex, g.randomSpawn, g.redSpawn = len(g.resources.events), false, false
			shot := projectile{x: float64(source.Before.RawX - 56), y: g.cameraY + float64(source.Before.RawY), vx: 0, vy: -6, owner: source.Owner, slot: 0, power: 1, frame: 1, impact: 4 - source.Phase}
			shot.terrainBias = float64(levelHeight-12-source.Before.Progress) - shot.y
			g.projectiles = []projectile{shot}
			controls := [2]playerControls{}
			controls[source.Owner].action = true
			g.stepPlay(controls)
			if g.ownedShotMask(source.Owner) != uint8(source.AfterBoarding.Mask) || g.blasts[source.Owner].impact != 4-source.AfterBoarding.QuadPhase {
				t.Fatal("the ordinary boarding wall quad was not reconstructed")
			}
			for frame, want := range source.Frames {
				controls = [2]playerControls{}
				controls[source.Owner].fire = frame == 0
				g.stepPlay(controls)
				var visible *projectile
				for index := range g.projectiles {
					value := &g.projectiles[index]
					if !value.enemy && value.owner == source.Owner && value.slot == 0 {
						visible = value
					}
				}
				if (visible != nil) != want.Active || int(g.ownedShotMask(source.Owner)) != want.Mask {
					t.Errorf("frame%d first-slot visibility/mask native%t/%x original%t/%x", frame+1, visible != nil, g.ownedShotMask(source.Owner), want.Active, want.Mask)
				}
				if visible != nil && (int(visible.x+56) != want.RawX || int(visible.y-g.cameraY) != want.RawY || visible.life*4 != want.Range || 0x4a2e8+g.projectileFrame(*visible)*160 != want.Art) {
					t.Errorf("frame%d reused cannon native%+v original%+v", frame+1, *visible, want)
				}
				quad := g.blasts[source.Owner]
				if (quad.impact > 0) != want.QuadActive || want.QuadActive && quad.impact != 4-want.QuadPhase {
					t.Errorf("frame%d original independent quad timing differs", frame+1)
				}
				ammo, score := g.ammunition, g.score
				if source.Owner == 1 {
					ammo, score = g.ammunition2, g.score2
				}
				if ammo != want.Ammo || score != want.Score || g.grenades != 23 || g.grenades2 != 23 {
					t.Errorf("frame%d ordinary finite resources differ from the original", frame+1)
				}
			}
		})
	}
}

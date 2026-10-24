package game

import (
	"compress/gzip"
	"encoding/json"
	"os"
	"testing"
)

type originalLiveShotState struct {
	RawX, RawY, Art, Range, Progress, Mask                              int
	Active, Mounted, FlyingGrenade, QuadActive, ImpactActive, TargetHit bool
	BulletPower, GrenadePower, QuadPhase, ImpactPhase, Score            int
}

type originalLiveWeaponCase struct {
	Name                                     string
	Owner, Slot                              int
	Before, AfterOwnedPhase, AfterLateImpact originalLiveShotState
	ContactsBefore                           originalPickupState
	TargetBefore                             struct {
		X, Y   float64
		Active bool
	}
	PickupKinds                                 []int
	RawVX, RawVY, CellRow, CellColumn, CellCode int
	QuadParts                                   []struct {
		RawX, RawY, Art int
		Active          bool
	}
}

// Each expected owned phase and complete late-impact phase was independently
// executed by the original executable. These are controlled fixtures, not
// ordinary play, and the native implementation never contributes goldens.
func TestOriginalControlledLiveOwnedWeaponPolicy(t *testing.T) {
	f, err := os.Open("testdata/pickup-and-live-power.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	z, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	defer z.Close()
	var fixture struct{ WeaponCases []originalLiveWeaponCase }
	if err = json.NewDecoder(z).Decode(&fixture); err != nil {
		t.Fatal(err)
	}
	if len(fixture.WeaponCases) != 30 {
		t.Fatal("missing independent original weapon timelines")
	}
	for _, source := range fixture.WeaponCases {
		t.Run(source.Name, func(t *testing.T) {
			g, err := New(Options{Mute: true})
			if err != nil {
				t.Fatal(err)
			}
			g.startGamePlayers(false, 2)
			// The source range-progress word selects the terrain probe. Translate its
			// unchanged raw coordinate origin to the same ordinary native world cell.
			g.cameraY = float64(levelHeight - 16 - source.Before.Progress + 4 - source.Before.RawY)
			for player, p := range source.ContactsBefore.Players {
				value := actor{x: p.X, y: g.cameraY + p.Y, health: 1}
				if player == source.Owner {
					value.inTank = source.Before.Mounted
					value.tankFuel = 17
				}
				if player == 0 {
					g.player = value
				} else {
					g.player2 = value
				}
			}
			g.ammunition, g.ammunition2 = source.ContactsBefore.Ammo[0], source.ContactsBefore.Ammo[1]
			g.grenades, g.grenades2 = source.ContactsBefore.Grenades[0], source.ContactsBefore.Grenades[1]
			g.score, g.score2 = source.ContactsBefore.Score[0], source.ContactsBefore.Score[1]
			g.bulletPower, g.grenadePower = source.ContactsBefore.BulletPower, source.ContactsBefore.GrenadePower
			g.pickups = nil
			for _, p := range source.ContactsBefore.Items {
				g.pickups = append(g.pickups, pickup{x: p.X, y: g.cameraY + p.Y, kind: p.Kind})
			}
			g.enemies = nil
			if source.TargetBefore.Active {
				g.enemies = append(g.enemies, actor{x: source.TargetBefore.X, y: g.cameraY + source.TargetBefore.Y, health: 1, kind: 2, slot: 3})
			}
			frame := (source.Before.Art - 0x4a2e8) / 160
			bornCannon := frame >= 7 && frame <= 9 && !source.Before.FlyingGrenade
			power := 1
			if source.Before.FlyingGrenade {
				power = 2
				if frame >= 35 {
					power = 3
				}
			} else if frame == 2 || frame == 9 {
				power = 2
			}
			savedFrame := frame
			if source.Before.FlyingGrenade {
				savedFrame = 0
			}
			shot := projectile{x: float64(source.Before.RawX - 56), y: g.cameraY + float64(source.Before.RawY), vx: float64(source.RawVX), vy: float64(source.RawVY), owner: source.Owner, slot: source.Slot, life: source.Before.Range / 4, power: power, frame: savedFrame, cannon: bornCannon, grenade: source.Before.FlyingGrenade}
			g.projectiles = []projectile{shot}
			// C0F6/C282's pickup scan belongs to the input phase; no shot is allocated
			// here, so this specifically checks the already flying owned projectile.
			g.stepPlayer(source.Owner, playerControls{})
			g.advancePlayerProjectiles(source.Owner)
			g.advanceBlast(source.Owner)
			expected := source.AfterLateImpact
			if g.ownedShotMask(source.Owner) != uint8(expected.Mask) {
				t.Fatalf("owned occupancy native%x original%x", g.ownedShotMask(source.Owner), expected.Mask)
			}
			var visible *projectile
			for i := range g.projectiles {
				p := &g.projectiles[i]
				if !p.enemy && p.owner == source.Owner && p.slot == source.Slot {
					visible = p
				}
			}
			if (visible != nil) != expected.Active {
				t.Fatalf("owned visibility native%v original%v", visible != nil, expected.Active)
			}
			if visible != nil {
				if int(visible.x+56) != expected.RawX || int(visible.y-g.cameraY) != expected.RawY {
					t.Fatalf("owned position native%+v original%+v", *visible, expected)
				}
				art := 0x4a2e8 + g.projectileFrame(*visible)*160
				if art != expected.Art {
					t.Fatalf("complete-frame retained art native%x original%x", art, expected.Art)
				}
				if !expected.ImpactActive && visible.life*4 != expected.Range {
					t.Fatalf("remaining range native%d original%d", visible.life*4, expected.Range)
				}
			}
			blast := g.blasts[source.Owner]
			if (blast.impact > 0) != expected.QuadActive {
				t.Fatalf("quad active native%d original%v", blast.impact, expected.QuadActive)
			}
			if expected.QuadActive {
				if blast.impact != 4-expected.QuadPhase {
					t.Fatalf("quad phase native%d original%d", blast.impact, expected.QuadPhase)
				}
				for i, p := range source.QuadParts {
					if int(blast.x+48)+i%2*16 != p.RawX || int(blast.y-g.cameraY)-8+i/2*16 != p.RawY || 0x4a2e8+grenadeImpactFrame(blast.impact, i)*160 != p.Art {
						t.Fatal("actual complete-frame original quad differs")
					}
				}
			}
			hit := false
			if source.TargetBefore.Active {
				hit = g.enemies[0].health <= 0
			}
			if hit != expected.TargetHit {
				t.Fatalf("target hit native%v original%v", hit, expected.TargetHit)
			}
			score := g.score
			if source.Owner == 1 {
				score = g.score2
			}
			if score != expected.Score {
				t.Fatalf("score native%d original%d", score, expected.Score)
			}
		})
	}
}

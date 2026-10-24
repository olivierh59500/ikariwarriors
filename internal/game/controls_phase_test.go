package game

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

type originalControlState struct {
	X, Y, Progress, Camera                       int
	Dead                                         bool
	Fall, Protection, Ammo, Grenades, Score      int
	Tank                                         bool
	Upper, Lower                                 int
	UpperStep, UpperDelay, LowerStep, LowerDelay int
	Facing                                       [2]int
}

// Complete original frames exercise the ordering between each movement axis,
// weapon birth, existing bullet/infantry contacts and the late fall counters.
// Fixture actors are explicitly seeded; this is independent of the demo pilot.
func TestOriginalCompleteControlPhases(t *testing.T) {
	data, err := os.ReadFile("testdata/controls-phases.json")
	if err != nil {
		t.Fatal(err)
	}
	var frames []struct {
		Case struct {
			Group, Weapon                string
			Owner, Direction, Protection int
			Locked                       bool
			Enemies                      []struct{ DX, DY int }
		}
		Before, After originalControlState
		Shots         []struct{ Slot, X, Y, Bob, Remaining int }
		Eliminated    []bool
	}
	if err := json.Unmarshal(data, &frames); err != nil {
		t.Fatal(err)
	}
	if len(frames) == 0 {
		t.Fatal("missing original complete control frames")
	}
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	for index, frame := range frames {
		t.Run(fmt.Sprintf("%03d_%s_p%d", index, frame.Case.Group, frame.Case.Owner+1), func(t *testing.T) {
			original, owner := frame.Before, frame.Case.Owner
			g.startGamePlayers(false, owner+1)
			g.scene = playScene
			g.enemies = nil
			g.eventIndex = len(g.resources.events)
			g.randomSpawn, g.redSpawn = false, false
			g.cameraY = float64(original.Camera)
			g.cameraBackY = levelHeight - float64((int(levelHeight-g.cameraY)-1)&^31)
			value, lives := &g.player, &g.lives
			if owner == 1 {
				g.player = actor{x: 132, y: g.cameraY + 146, health: 1}
				g.lives = 5
				value, lives = &g.player2, &g.lives2
			}
			*lives = 5
			*value = actor{x: float64(original.X - 48), y: float64(levelHeight - 16 - original.Progress), health: 1, invulnerable: original.Protection, inTank: original.Tank, tankFuel: 64, tankSlot: 19 + owner}
			upperBase := 0x42b88 + owner*22*actorBobBytes
			upperBytes := actorBobBytes
			if original.Tank {
				upperBase, upperBytes = 0x5b408+owner*24*vehicleBobBytes, vehicleBobBytes
			}
			value.upper = walkAnimation{step: original.UpperStep, delay: original.UpperDelay, frame: (original.Upper - upperBase) / upperBytes}
			value.lower = walkAnimation{step: original.LowerStep, delay: original.LowerDelay, frame: max(0, (original.Lower-0x46288)/overlayBobBytes)}
			g.directionLock[owner] = frame.Case.Locked
			g.ammunition, g.ammunition2, g.grenades, g.grenades2 = 99, 99, 50, 50
			for i, enemy := range frame.Case.Enemies {
				kind, slot := 2, 3+i
				if frame.Case.Group == "boarding" {
					kind, slot = 4, 19
				}
				g.enemies = append(g.enemies, actor{x: value.x + float64(enemy.DX), y: value.y + float64(enemy.DY), health: 1, kind: kind, slot: slot, tankFuel: 64})
			}
			if frame.Case.Group == "coupled" {
				g.projectiles = append(g.projectiles, projectile{x: float64(120 - 56), y: g.cameraY + 104, enemy: true, owner: -1, life: 15})
			}
			control := playerControls{fire: frame.Case.Weapon == "bullet" || frame.Case.Weapon == "cannon", action: frame.Case.Weapon == "grenade" || frame.Case.Weapon == "context"}
			if frame.Case.Direction&0x10 != 0 {
				control.y = -1
			} else if frame.Case.Direction&0x20 != 0 {
				control.y = 1
			}
			if frame.Case.Direction&0x40 != 0 {
				control.x = -1
			} else if frame.Case.Direction&0x80 != 0 {
				control.x = 1
			}
			var controls [2]playerControls
			controls[owner] = control
			g.stepPlay(controls)
			expected := frame.After
			ammo, grenades, score := g.ammunition, g.grenades, g.score
			if owner == 1 {
				ammo, grenades, score = g.ammunition2, g.grenades2, g.score2
			}
			if int(value.x+48) != expected.X || int(value.y-g.cameraY+4) != expected.Y || levelHeight-16-int(value.y) != expected.Progress || int(g.cameraY) != expected.Camera || (value.deadTicks > 0) != expected.Dead || value.deadTicks != expected.Fall || value.invulnerable != expected.Protection || value.inTank != expected.Tank || ammo != expected.Ammo || grenades != expected.Grenades || score != expected.Score {
				t.Errorf("state: raw%g/%g progress%g camera%g fall%d protection%d tank%v ammo%d grenades%d score%d; original%+v", value.x+48, value.y-g.cameraY+4, levelHeight-16-value.y, g.cameraY, value.deadTicks, value.invulnerable, value.inTank, ammo, grenades, score, expected)
			}
			upper, lower := value.playerArtOffsets(owner)
			if lower >= 0 {
				lower += 0x42b88
			} else {
				lower = 0
			}
			if upper+0x42b88 != expected.Upper || lower != expected.Lower || value.upper.step != expected.UpperStep || value.upper.delay != expected.UpperDelay || value.lower.step != expected.LowerStep || value.lower.delay != expected.LowerDelay {
				t.Errorf("pose: upper%06x/%d/%d lower%06x/%d/%d, original upper%06x/%d/%d lower%06x/%d/%d", upper+0x42b88, value.upper.step, value.upper.delay, lower, value.lower.step, value.lower.delay, expected.Upper, expected.UpperStep, expected.UpperDelay, expected.Lower, expected.LowerStep, expected.LowerDelay)
			}
			for i, eliminated := range frame.Eliminated {
				alive := false
				for _, enemy := range g.enemies {
					alive = alive || enemy.slot == 3+i && enemy.health > 0
				}
				if alive == eliminated {
					t.Errorf("infantry slot%d alive%v, original eliminated%v", 3+i, alive, eliminated)
				}
			}
			count := 0
			for _, shot := range g.projectiles {
				if shot.enemy || shot.owner != owner {
					continue
				}
				count++
				matched := false
				for _, reference := range frame.Shots {
					if shot.slot == reference.Slot {
						matched = int(shot.x+56) == reference.X && int(shot.y-g.cameraY) == reference.Y && shot.life*4 == reference.Remaining && 0x4a2e8+g.projectileFrame(shot)*bobBytes == reference.Bob
					}
				}
				if !matched {
					t.Errorf("shot%+v has no matching original slot/position/range/image in%+v", shot, frame.Shots)
				}
			}
			if count != len(frame.Shots) {
				t.Errorf("owned shots%d, original%d", count, len(frame.Shots))
			}
		})
	}
}

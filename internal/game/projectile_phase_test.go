package game

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

// These independent original sequences exercise moving targets and both player
// input phases. Each sequence continues for four complete gameplay frames.
func TestOriginalCompleteProjectilePhases(t *testing.T) {
	data, err := os.ReadFile("testdata/projectile-phases.json")
	if err != nil {
		t.Fatal(err)
	}
	type playerState struct {
		X, Y, Progress, Fall, Protection, Score int
		Dead                                    bool
	}
	type frameState struct {
		Camera  int
		Players []playerState
		Enemy   struct {
			X, Y, Active, Dead int
			March              [3]int
		}
		Shots []struct{ Slot, X, Y, Bob, Remaining int }
	}
	var sequences []struct {
		Case struct {
			Group                             string
			Owner, March, DX, Direction, Kind int
		}
		Before frameState
		Frames []frameState
		Inputs []int
	}
	if err := json.Unmarshal(data, &sequences); err != nil {
		t.Fatal(err)
	}
	if len(sequences) != 114 {
		t.Fatalf("original projectile sequence count %d, expected 114", len(sequences))
	}
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	for index, sequence := range sequences {
		t.Run(fmt.Sprintf("%03d_%s_p%d", index, sequence.Case.Group, sequence.Case.Owner+1), func(t *testing.T) {
			c := sequence.Case
			count := c.Owner + 1
			if c.Group == "friend-blast" {
				count = 2
			}
			g.startGamePlayers(false, count)
			// Controlled source fixtures activate players after a one-player
			// foundation; their absent second player has no retained scroll request.
			g.cooperativeScroll = [2]cooperativeScrollState{{up: true, claimed: true}, {}}
			g.cooperativeDirection = 0
			g.scene = playScene
			g.enemies = nil
			g.eventIndex = len(g.resources.events)
			g.randomSpawn, g.redSpawn = false, false
			g.cameraY = float64(sequence.Before.Camera)
			g.cameraBackY = levelHeight - float64((int(levelHeight-g.cameraY)-1)&^31)
			for player := range count {
				before := sequence.Before.Players[player]
				value := &g.player
				if player == 1 {
					value = &g.player2
				}
				*value = actor{x: float64(before.X - 48), y: float64(levelHeight - 16 - before.Progress), health: 1, upper: walkAnimation{frame: 8}, lower: walkAnimation{frame: 12}}
			}
			switch c.Group {
			case "marching":
				g.enemies = []actor{{x: 120 - 48, y: g.cameraY + 60 - 4, health: 1, kind: 0, slot: 5, marchX: c.March, marchTime: 16}}
				g.projectiles = []projectile{{x: float64(120 + c.DX - 56), y: g.cameraY + 60, vx: 6, life: 10, power: 1, owner: c.Owner}}
			case "friend-blast":
				g.projectiles = []projectile{{x: 120 - 56, y: g.cameraY + 94, grenade: true, life: 1, power: 2, owner: c.Owner}}
			case "flying-contact":
				g.projectiles = []projectile{{x: 120 - 56, y: g.cameraY + 98, enemy: true, owner: -1, life: 15, power: 1, grenade: c.Kind > 0, fragments: c.Kind > 1}}
				if c.Kind > 0 {
					g.projectiles[0].frame = 9
				}
			}
			var controls [2]playerControls
			target := c.Owner
			if c.Group == "friend-blast" {
				target = 1 - target
			}
			for tick, original := range sequence.Frames {
				// The original hardware reader can release a seeded direction
				// between frames. Replay its observed input rather than holding it.
				direction := sequence.Inputs[tick]
				if target == 1 {
					direction <<= 4
				}
				control := playerControls{}
				if direction&0x10 != 0 {
					control.y = -1
				} else if direction&0x20 != 0 {
					control.y = 1
				}
				if direction&0x40 != 0 {
					control.x = -1
				} else if direction&0x80 != 0 {
					control.x = 1
				}
				controls[target] = control
				g.stepPlay(controls)
				for player := range count {
					value, score := g.player, g.score
					if player == 1 {
						value, score = g.player2, g.score2
					}
					expected := original.Players[player]
					if int(value.x+48) != expected.X || int(value.y-g.cameraY+4) != expected.Y || levelHeight-16-int(value.y) != expected.Progress || value.deadTicks != expected.Fall || value.invulnerable != expected.Protection || score != expected.Score || int(g.cameraY) != original.Camera {
						t.Errorf("tick%d player%d raw%g/%g progress%g fall%d protection%d score%d camera%g; original%+v camera%d", tick, player, value.x+48, value.y-g.cameraY+4, levelHeight-16-value.y, value.deadTicks, value.invulnerable, score, g.cameraY, expected, original.Camera)
					}
				}
				if c.Group == "marching" {
					if len(g.enemies) != 1 {
						t.Fatalf("tick%d enemies%d, expected one original slot", tick, len(g.enemies))
					}
					value, expected := g.enemies[0], original.Enemy
					if int(value.x+48) != expected.X || int(value.y-g.cameraY+4) != expected.Y || (value.health <= 0) != (expected.Dead != 0) || [3]int{value.marchY, value.marchX, value.marchTime} != expected.March {
						t.Errorf("tick%d infantry raw%g/%g dead%v march%d/%d/%d; original%+v", tick, value.x+48, value.y-g.cameraY+4, value.health <= 0, value.marchY, value.marchX, value.marchTime, expected)
					}
				}
				if len(g.projectiles) != len(original.Shots) {
					t.Errorf("tick%d shots%d, original%d", tick, len(g.projectiles), len(original.Shots))
				}
				for _, shot := range g.projectiles {
					slot := 34 + shot.owner*3 + shot.slot
					if shot.enemy {
						slot = 65 + shot.slot
					}
					matched := false
					for _, expected := range original.Shots {
						if slot == expected.Slot {
							matched = int(shot.x+56) == expected.X && int(shot.y-g.cameraY) == expected.Y && shot.life*4 == expected.Remaining && 0x4a2e8+g.projectileFrame(shot)*bobBytes == expected.Bob
						}
					}
					if !matched {
						t.Errorf("tick%d shot%+v has no original slot/position/range/image match in%+v", tick, shot, original.Shots)
					}
				}
			}
		})
	}
}

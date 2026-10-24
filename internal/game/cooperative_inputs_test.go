package game

import (
	"compress/gzip"
	"encoding/json"
	"os"
	"testing"
)

type cooperativeObservedPlayer struct {
	X, Y, Progress, Lives, Fall, Protection, Upper, Lower, UpperStep, UpperDelay, LowerStep, LowerDelay int
	Dead, Wet, Tank, Up, Down, Claimed                                                                  bool
}
type cooperativeObservedFrame struct {
	Camera, Room, EventIndex int
	DirectionLatch           uint8
	Players                  [2]cooperativeObservedPlayer
	Enemies                  []struct {
		X, Y                                                            float64
		Slot, Kind, Health, DeadTicks, MarchX, MarchY, MarchTime, Shots int
	}
	Shots []struct {
		X, Y, VX, VY float64
		Life, Slot   int
		Grenade      bool
	}
	Pickups []struct {
		X, Y float64
		Kind int
	}
}

// Each ordinary source frame supplies its actual starting state and both
// verified joystick reads. This checks coupled movement and animation, rather
// than claiming an uninterrupted replay of the whole original game.
func TestOriginalCooperativeInputTransitions(t *testing.T) {
	runOriginalCooperativeInputs(t, false)
}

// Replay the same complete short original input sequence from its observed
// initial state, without restoring any intermediate native state.
func TestOriginalCooperativeInputSequence(t *testing.T) {
	runOriginalCooperativeInputs(t, true)
}

func runOriginalCooperativeInputs(t *testing.T, continuous bool) {
	f, err := os.Open("testdata/cooperative-inputs.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	z, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	defer z.Close()
	var rows []struct {
		Name                      string
		Directions                [2]int
		Before, After             cooperativeObservedFrame
		BeforeSHA256, AfterSHA256 string
	}
	if err := json.NewDecoder(z).Decode(&rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 64 {
		t.Fatal("incomplete ordinary two-player source observations")
	}
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range rows {
		t.Run(row.Name, func(t *testing.T) {
			if !continuous || row.Name == "ordinary-coop-000" {
				g.startGamePlayers(false, 2)
				g.scene = playScene
				g.eventIndex = row.Before.EventIndex
				g.cameraY, g.cameraBackY = float64(row.Before.Camera), float64(row.Before.Camera+row.Before.Room)
				g.cooperativeDirection = row.Before.DirectionLatch
				g.randomSpawn, g.redSpawn = false, false
				g.enemies, g.projectiles, g.pickups, g.mines = nil, nil, nil, nil
				for i, p := range row.Before.Players {
					value := actor{x: float64(p.X), y: float64(p.Y), health: 1, invulnerable: p.Protection, wet: p.Wet, inTank: p.Tank}
					if p.Dead {
						value.health = 0
						value.deadTicks = p.Fall
					}
					value.upper = walkAnimation{frame: (p.Upper - 0x42b88 - i*22*actorBobBytes) / actorBobBytes, step: p.UpperStep, delay: p.UpperDelay}
					value.lower = walkAnimation{frame: (p.Lower - 0x46288) / overlayBobBytes, step: p.LowerStep, delay: p.LowerDelay}
					if i == 0 {
						g.player = value
						g.lives = p.Lives
					} else {
						g.player2 = value
						g.lives2 = p.Lives
					}
					g.cooperativeScroll[i] = cooperativeScrollState{up: p.Up, down: p.Down, claimed: p.Claimed}
				}
				for _, e := range row.Before.Enemies {
					g.enemies = append(g.enemies, actor{x: e.X, y: e.Y, kind: e.Kind, slot: e.Slot, health: e.Health, deadTicks: e.DeadTicks, marchX: e.MarchX, marchY: e.MarchY, marchTime: e.MarchTime, shots: e.Shots})
				}
				for _, s := range row.Before.Shots {
					g.projectiles = append(g.projectiles, projectile{x: s.X, y: s.Y, vx: s.VX, vy: s.VY, life: s.Life, slot: s.Slot, enemy: true, owner: -1, grenade: s.Grenade})
				}
				for _, p := range row.Before.Pickups {
					g.pickups = append(g.pickups, pickup{x: p.X, y: p.Y, kind: p.Kind})
				}

			}
			var controls [2]playerControls
			for i, d := range row.Directions {
				if d&0x10 != 0 {
					controls[i].y = -1
				} else if d&0x20 != 0 {
					controls[i].y = 1
				}
				if d&0x40 != 0 {
					controls[i].x = -1
				} else if d&0x80 != 0 {
					controls[i].x = 1
				}
			}
			g.stepPlay(controls)
			if g.cameraY != float64(row.After.Camera) || g.eventIndex != row.After.EventIndex || g.cooperativeDirection != row.After.DirectionLatch {
				t.Errorf("camera/event/latch %.0f/%d/%d, original%d/%d/%d", g.cameraY, g.eventIndex, g.cooperativeDirection, row.After.Camera, row.After.EventIndex, row.After.DirectionLatch)
			}
			for i, p := range row.After.Players {
				value := g.player
				if i == 1 {
					value = g.player2
				}
				upper := 0x42b88 + i*22*actorBobBytes + value.upper.frame*actorBobBytes
				lower := 0x46288 + value.lower.frame*overlayBobBytes
				state := g.cooperativeScroll[i]
				if value.x != float64(p.X) || value.y != float64(p.Y) || upper != p.Upper || lower != p.Lower || value.upper.step != p.UpperStep || value.upper.delay != p.UpperDelay || value.lower.step != p.LowerStep || value.lower.delay != p.LowerDelay || state.up != p.Up || state.down != p.Down || state.claimed != p.Claimed {
					t.Errorf("player%d pose%.0f/%.0f %x/%x phases%+v/%+v state%+v, original%+v", i, value.x, value.y, upper, lower, value.upper, value.lower, state, p)
				}
			}
		})
	}
}

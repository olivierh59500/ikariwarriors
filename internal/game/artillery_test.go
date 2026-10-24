package game

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"testing"
)

type originalArtilleryShot struct {
	Slot, X, Y, VX, VY, Range, Type int
	Bob                             uint32
}

type originalArtilleryState struct {
	Busy  [2]bool
	Shots []originalArtilleryShot
	Quad  []struct {
		X, Y int
		Bob  uint32
	}
	QuadType       int
	Burst          []struct{ X, Y int }
	BurstRemaining int
}

func (s originalArtilleryState) supply(g *Game) {
	g.cameraY = 0
	g.player.health, g.player2.health = 0, 0
	g.airStrikeBusy = s.Busy
	g.projectiles, g.burstQueue = nil, nil
	g.burst, g.blasts = radialBurst{}, [3]projectile{}
	for _, v := range s.Shots {
		shot := projectile{x: float64(v.X - 56), y: float64(v.Y), vx: float64(v.VX), vy: float64(v.VY),
			life: v.Range / 4, frame: int(v.Bob-0x4a2e8) / 160, enemy: true, owner: -1, grenade: v.Type != 0, fragments: v.Type >= 2, slot: v.Slot}
		if v.Range == 0 {
			shot.life = 64
		}
		if v.Type >= 3 {
			shot.airStrike = uint8(v.Type - 2)
		}
		g.projectiles = append(g.projectiles, shot)
	}
	if len(s.Quad) != 0 {
		first := s.Quad[0]
		phase := int(first.Bob-0x4c188) / 640
		shot := projectile{x: float64(first.X - 48), y: float64(first.Y + 8), enemy: true, owner: -1,
			grenade: true, fragments: s.QuadType >= 2, impact: 4 - phase, frame: 9}
		if s.QuadType >= 3 {
			shot.airStrike = uint8(s.QuadType - 2)
			shot.frame = 17
		}
		g.blasts[2] = shot
	}
}

func (s originalArtilleryState) compare(t *testing.T, g *Game) {
	t.Helper()
	if g.airStrikeBusy != s.Busy || len(g.projectiles) != len(s.Shots) {
		t.Fatalf("strike locks/shots native%v/%d original%v/%d", g.airStrikeBusy, len(g.projectiles), s.Busy, len(s.Shots))
	}
	for i, want := range s.Shots {
		v := g.projectiles[i]
		kind := 0
		if v.grenade {
			kind = 1
		}
		if v.fragments {
			kind = 2
		}
		if v.airStrike != 0 {
			kind = int(v.airStrike) + 2
		}
		got := originalArtilleryShot{Slot: v.slot, X: int(v.x + 56), Y: int(v.y - g.cameraY), VX: int(v.vx), VY: int(v.vy), Range: v.life * 4, Type: kind, Bob: uint32(0x4a2e8 + v.frame*160)}
		if got != want {
			t.Fatalf("source shot%d differs: native%+v original%+v", i, got, want)
		}
	}
	quad := g.blasts[2]
	if (quad.impact > 0) != (len(s.Quad) > 0) {
		t.Fatal("source artillery quad lifetime differs")
	}
	for i, want := range s.Quad {
		if int(quad.x+48+float64(i%2*16)) != want.X || int(quad.y-8+float64(i/2*16)) != want.Y || uint32(0x4a2e8+grenadeImpactFrame(quad.impact, i)*160) != want.Bob {
			t.Fatalf("source artillery quad%d differs", i)
		}
	}
	if g.burst.active != (len(s.Burst) > 0) {
		t.Fatal("source artillery fragment visibility differs")
	}
	if g.burst.active {
		if g.burst.remaining != s.BurstRemaining {
			t.Fatal("source artillery fragment lifetime differs")
		}
		for i, want := range s.Burst {
			p := g.burst.parts[i]
			if int(p.x+56) != want.X || int(p.y) != want.Y {
				t.Fatalf("source artillery shard%d differs", i)
			}
		}
	}
}

func TestOriginalArtilleryEmission(t *testing.T) {
	data, err := os.ReadFile("testdata/artillery-emission.json")
	if err != nil {
		t.Fatal(err)
	}
	var rows []struct {
		Kind           string
		Launcher, X, Y int
		Before, After  originalArtilleryState
	}
	if err = json.Unmarshal(data, &rows); err != nil {
		t.Fatal(err)
	}
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range rows {
		t.Run(row.Kind, func(t *testing.T) {
			row.Before.supply(g)
			g.fireAirStrike(float64(row.X-56), float64(row.Y), row.Launcher)
			row.After.compare(t, g)
		})
	}
}

func TestOriginalArtilleryLockAfterQuadReplacement(t *testing.T) {
	data, err := os.ReadFile("testdata/artillery-expiry.json")
	if err != nil {
		t.Fatal(err)
	}
	var rows []struct {
		Kind          string
		Before, After originalArtilleryState
	}
	if err = json.Unmarshal(data, &rows); err != nil {
		t.Fatal(err)
	}
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range rows {
		t.Run(row.Kind, func(t *testing.T) {
			row.Before.supply(g)
			g.advanceBlast(2)
			row.After.compare(t, g)
		})
	}
}

// These are full original gameplay frames, including the shared quad and burst.
// The idle target samples and damage-counter resets come from the observations;
// the native game calculates every trajectory, expiry and marker lock itself.
func TestOriginalLiveArtilleryTimelines(t *testing.T) {
	data, err := os.ReadFile("testdata/artillery-lived.json")
	if err != nil {
		t.Fatal(err)
	}
	var rows []struct {
		Kind                                           string
		Tick, Idle, TargetX, TargetY, PlayerX, PlayerY int
		Dead                                           bool
		Mines                                          []struct{ X, Y, Slot int }
		State                                          originalArtilleryState
	}
	if err = json.Unmarshal(data, &rows); err != nil {
		t.Fatal(err)
	}
	var g *Game
	var previousIdle int
	kind := ""
	for index, row := range rows {
		if kind != row.Kind {
			kind = row.Kind
			g, err = New(Options{Mute: true})
			if err != nil {
				t.Fatal(err)
			}
			g.playerCount = 1
			row.State.supply(g)
		} else {
			// No test actor absorbs the observed impact or changes its trajectory.
			g.player.health, g.player2.health = 0, 0
			previous := rows[index-1]
			if len(previous.Mines) > 0 {
				g.player = actor{x: float64(previous.PlayerX - 48), y: float64(previous.PlayerY - 4), health: 1, inTank: true, tankFuel: 64}
				if previous.Dead {
					g.player.health = 0
				}
				g.mines = nil
				for _, m := range previous.Mines {
					mineKind := 20
					if m.Slot < 85 {
						mineKind = 18
					}
					g.mines = append(g.mines, landmine{x: float64(m.X - 56), y: float64(m.Y), kind: mineKind, slot: m.Slot})
				}
				g.updateMines()
				g.player.health = 0
			}
			g.updateProjectiles()
			if strings.HasPrefix(kind, "idle") {
				g.idleArtilleryTicks = uint32(previousIdle + 1)
				// Damage probes are a separate observed part of this live scene.
				if row.Idle == 0 {
					g.idleArtilleryTicks = 0
				}
				if g.idleArtilleryTicks > 600 {
					found := false
					for seed := uint32(1); seed < 1<<20; seed++ {
						g.random = seed
						x, y := int(g.randomValue()&127)+64, int(g.randomValue()&127)+64
						if x == row.TargetX && y == row.TargetY {
							g.random = seed
							found = true
							break
						}
					}
					if !found {
						t.Fatal("cannot supply the independently observed random target")
					}
				}
				g.updateIdleArtillery()
			}
			g.pruneOffscreenProjectiles()
		}
		t.Run(row.Kind+"/"+strconv.Itoa(row.Tick), func(t *testing.T) { row.State.compare(t, g) })
		previousIdle = row.Idle
	}
}

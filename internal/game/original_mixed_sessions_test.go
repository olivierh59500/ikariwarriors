package game

import (
	"compress/gzip"
	"encoding/json"
	"os"
	"testing"
)

type mixedSourcePlayer struct {
	X, RawY, Progress, Upper, Lower, UpperStep, UpperDelay, LowerStep, LowerDelay int
	Active, Up, Down, Claimed                                                     bool
}

type mixedSourceFrame struct {
	Scroll, Camera int
	DirectionLatch uint8
	Players        [2]mixedSourcePlayer
	SourceSHA256   string
}

func TestOriginalMixedPlayerSessions(t *testing.T) {
	f, err := os.Open("testdata/mixed-player-sessions.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	z, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	defer z.Close()
	var sessions []struct {
		Session, Players int
		Menu             mixedSourceFrame
		Landing          []mixedSourceFrame
		Moves            []struct {
			Directions    [2]int
			Before, After mixedSourceFrame
		}
	}
	if err := json.NewDecoder(z).Decode(&sessions); err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 4 {
		t.Fatal("incomplete mixed-player source sessions")
	}
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	// Start with the normal application constructor. The source menu is an
	// independent assertion; no original state is imported into the game.
	if g.cooperativeDirection != sessions[0].Menu.DirectionLatch {
		t.Fatal("initial direction differs from the original title/menu")
	}
	for i, p := range sessions[0].Menu.Players {
		value := g.player
		if i == 1 {
			value = g.player2
		}
		state := g.cooperativeScroll[i]
		if value.upper.step != p.UpperStep || value.upper.delay != p.UpperDelay || value.lower.step != p.LowerStep || value.lower.delay != p.LowerDelay || state.up != p.Up || state.down != p.Down || state.claimed != p.Claimed {
			t.Fatal("initial player counters differ from the original title/menu")
		}
	}
	for _, session := range sessions {
		if len(session.Landing) != 117 || len(session.Moves) != 32 {
			t.Fatal("incomplete original landing or input interval")
		}
		g.Back()
		g.startGamePlayers(false, session.Players)
		for tick, frame := range session.Landing {
			if tick > 0 {
				g.updateIntro()
			}
			if g.arrival.scroll != frame.Scroll {
				t.Errorf("session%d landing%d scroll%d original%d", session.Session, tick, g.arrival.scroll, frame.Scroll)
			}
			for i := range session.Players {
				p := frame.Players[i]
				upper, lower := g.arrival.upper[i], g.arrival.lower[i]
				if g.arrival.playerY(i) != p.RawY || upper.step != p.UpperStep || upper.delay != p.UpperDelay || lower.step != p.LowerStep || lower.delay != p.LowerDelay {
					t.Errorf("session%d landing%d player%d y%d phases%+v/%+v original%+v", session.Session, tick, i, g.arrival.playerY(i), upper, lower, p)
				}
			}
		}
		if g.scene != playScene {
			t.Fatal("landing did not enter the ordinary game")
		}
		for tick, move := range session.Moves {
			var controls [2]playerControls
			for i, mask := range move.Directions {
				if mask&0x10 != 0 {
					controls[i].y = -1
				} else if mask&0x20 != 0 {
					controls[i].y = 1
				}
				if mask&0x40 != 0 {
					controls[i].x = -1
				} else if mask&0x80 != 0 {
					controls[i].x = 1
				}
			}
			g.stepPlay(controls)
			frame := move.After
			if g.cameraY != float64(frame.Camera) || g.cooperativeDirection != frame.DirectionLatch {
				t.Errorf("session%d frame%d camera/latch %.0f/%d original%d/%d", session.Session, tick, g.cameraY, g.cooperativeDirection, frame.Camera, frame.DirectionLatch)
			}
			for i, p := range frame.Players {
				value := g.player
				if i == 1 {
					value = g.player2
				}
				if value.upper.step != p.UpperStep || value.upper.delay != p.UpperDelay || value.lower.step != p.LowerStep || value.lower.delay != p.LowerDelay {
					t.Errorf("session%d frame%d player%d retained phases%+v/%+v original%+v", session.Session, tick, i, value.upper, value.lower, p)
				}
				if i < session.Players {
					upper, lower := value.playerArtOffsets(i)
					if value.x != float64(p.X) || 14992-value.y != float64(p.Progress) || value.y-g.cameraY+4 != float64(p.RawY) || upper+0x42b88 != p.Upper || lower+0x42b88 != p.Lower {
						t.Errorf("session%d frame%d player%d pose %.0f/%.0f images%x/%x original%+v", session.Session, tick, i, value.x, value.y-g.cameraY+4, upper+0x42b88, lower+0x42b88, p)
					}
				}
			}
		}
	}
}

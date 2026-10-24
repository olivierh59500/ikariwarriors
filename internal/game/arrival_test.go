package game

import (
	"encoding/json"
	"os"
	"strconv"
	"testing"
)

func TestOriginalArrivalPositionsAndScroll(t *testing.T) {
	data, err := os.ReadFile("testdata/arrival.json")
	if err != nil {
		t.Fatal(err)
	}
	var rows []struct {
		Players, Tick, Scroll int
		Finished              bool
		Poses                 []struct{ X, Y int }
	}
	if err := json.Unmarshal(data, &rows); err != nil {
		t.Fatal(err)
	}
	var a arrivalSequence
	for _, source := range rows {
		if source.Tick == 0 {
			a = newArrival(source.Players)
		} else {
			a.advance()
		}
		if a.tick != source.Tick || a.scroll != source.Scroll || (a.tick == 116) != source.Finished {
			t.Fatalf("%d players tick%d native tick%d scroll%d, original scroll%d finished%v", source.Players, source.Tick, a.tick, a.scroll, source.Scroll, source.Finished)
		}
		for player, pose := range source.Poses {
			if 120+60*player != pose.X || a.playerY(player) != pose.Y {
				t.Fatalf("%d players tick%d player%d native%d/%d original%d/%d", source.Players, source.Tick, player, 120+60*player, a.playerY(player), pose.X, pose.Y)
			}
		}
	}
}

// The original keeps its previous animation phase through a menu restart.
// Seed this independent source run with its recorded phase, then compare
// both graphic addresses and both pairs of animation counters at every step.
func TestOriginalArrivalAnimation(t *testing.T) {
	data, err := os.ReadFile("testdata/arrival.json")
	if err != nil {
		t.Fatal(err)
	}
	var rows []struct {
		Players, Tick int
		Poses         []struct {
			Bob, Legs                      string
			Step, Delay, LegStep, LegDelay int
		}
	}
	if err := json.Unmarshal(data, &rows); err != nil {
		t.Fatal(err)
	}
	a := newArrival(1)
	for _, source := range rows {
		if source.Players != 1 {
			continue
		}
		pose := source.Poses[0]
		if source.Tick == 0 {
			a.upper[0] = walkAnimation{step: pose.Step, delay: pose.Delay, frame: 8}
			a.lower[0] = walkAnimation{step: pose.LegStep, delay: pose.LegDelay, frame: 24}
		} else {
			a.advance()
		}
		body, err := strconv.ParseInt(pose.Bob, 0, 32)
		if err != nil {
			t.Fatal(err)
		}
		legs, err := strconv.ParseInt(pose.Legs, 0, 32)
		if err != nil {
			t.Fatal(err)
		}
		upper, lower := a.upper[0], a.lower[0]
		if 0x42b88+upper.frame*actorBobBytes != int(body) || 0x46288+lower.frame*overlayBobBytes != int(legs) || upper.step != pose.Step || upper.delay != pose.Delay || lower.step != pose.LegStep || lower.delay != pose.LegDelay {
			t.Fatalf("arrival tick%d native%+v/%+v original%+v", source.Tick, upper, lower, pose)
		}
	}
}

func TestArrivalHandsOverToTheOrdinaryGame(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGamePlayers(false, 2)
	before, second := g.player, g.player2
	for range 115 {
		g.updateIntro()
	}
	if g.scene != introScene || g.player != before || g.player2 != second {
		t.Fatal("arrival changed the ready gameplay state or ended early")
	}
	g.updateIntro()
	if g.scene != playScene || g.arrival.scroll != 132 || g.cameraY != levelHeight-132 || g.lives != 5 || g.ammunition != 99 || g.grenades != 50 {
		t.Fatal("arrival did not hand over the original ordinary starting state")
	}
	for player, value := range []actor{g.player, g.player2} {
		if g.arrival.upper[player] != value.upper || g.arrival.lower[player] != value.lower {
			t.Fatal("arrival animation jumped when ordinary controls took over")
		}
	}
}

package game

import (
	"compress/gzip"
	"encoding/json"
	"os"
	"testing"
)

// These are actual ordinary automatic-landing boundaries, including the
// initial one-update separation and the last retained scroll claim.
func TestOriginalCooperativeLandingDecisions(t *testing.T) {
	f, err := os.Open("testdata/cooperative-landing.json.gz")
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
		Tick, Scroll int
		Y            [2]int
		States       [2]struct{ Up, Down, Claimed bool }
		SourceSHA256 string
	}
	if err := json.NewDecoder(z).Decode(&rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 117 {
		t.Fatal("incomplete source landing observations")
	}
	a := newArrival(2)
	for i, p := range rows[0].States {
		a.cooperative[i] = cooperativeScrollState{up: p.Up, down: p.Down, claimed: p.Claimed}
	}
	for _, source := range rows {
		if source.Tick > 0 {
			a.advance()
		}
		if a.tick != source.Tick || a.scroll != source.Scroll {
			t.Fatalf("landing%d scroll%d, original%d", a.tick, a.scroll, source.Scroll)
		}
		for i, p := range source.States {
			if a.playerY(i) != source.Y[i] || a.cooperative[i].up != p.Up || a.cooperative[i].down != p.Down || a.cooperative[i].claimed != p.Claimed {
				t.Fatalf("landing%d player%d y%d state%+v, original%d/%+v", a.tick, i, a.playerY(i), a.cooperative[i], source.Y[i], p)
			}
		}
	}
}

// The direction latch survives an ordinary F1/menu restart. Released gates
// produce the other initial scroll leader, as recorded in that source run.
func TestCooperativeRestartRetainsSoftwareState(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.cooperativeDirection = 1
	g.cooperativeScroll = [2]cooperativeScrollState{}
	g.startGamePlayers(false, 2)
	for i := 0; i < 116; i++ {
		g.updateIntro()
	}
	if g.scene != playScene || g.cooperativeDirection != 1 || !g.cooperativeScroll[0].claimed || g.cooperativeScroll[1].claimed || g.cameraY != 14876 {
		t.Fatal("ordinary restart lost its retained direction or changed the observed scroll leader")
	}
}

func TestNewCooperativeGameUsesTheObservedInitialLeader(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGamePlayers(false, 2)
	for i := 0; i < 116; i++ {
		g.updateIntro()
	}
	if g.cooperativeScroll[0].claimed || !g.cooperativeScroll[1].claimed || g.cameraY != 14876 {
		t.Fatal("new two-player game lost the observed first scroll leader")
	}
	before := g.player
	g.stepPlay([2]playerControls{{y: -1}, {}})
	if g.player.x != before.x || g.player.y != before.y || g.cameraY != 14876 {
		t.Fatal("initial peer scroll claim failed to suppress the first upward step")
	}
}

// The original11186/111B0departure paths explicitly release the player's gate.
func TestDepartedCooperativePlayerReleasesTheScroll(t *testing.T) {
	for player := 0; player < 2; player++ {
		g, err := New(Options{Mute: true})
		if err != nil {
			t.Fatal(err)
		}
		g.startGamePlayers(false, 2)
		g.scene = playScene
		g.cooperativeScroll[player].claimed = true
		value, lives := &g.player, &g.lives
		if player == 1 {
			value, lives = &g.player2, &g.lives2
		}
		value.health, value.deadTicks, *lives = 0, 1, 0
		g.advancePlayerLife(player)
		if *lives != -1 || g.cooperativeScroll[player].claimed {
			t.Fatal("departed player retained the original scroll lock")
		}
	}
}

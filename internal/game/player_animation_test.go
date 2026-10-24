package game

import (
	"compress/gzip"
	"encoding/json"
	"os"
	"testing"
)

func readOriginalPlayerAnimation(t *testing.T, path string, value any) {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	z, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	defer z.Close()
	if err := json.NewDecoder(z).Decode(value); err != nil {
		t.Fatal(err)
	}
}

type originalPlayerAnimationState struct {
	Upper, Lower                                 int
	UpperStep, UpperDelay, LowerStep, LowerDelay int
}

// Convert only the recorded initial images; expected phases and image offsets
// always come from the independently observed original routine or frame.
func observedPlayerAnimation(player int, p originalPlayerAnimationState) actor {
	a := actor{health: 1}
	a.upper = walkAnimation{step: p.UpperStep, delay: p.UpperDelay, frame: infantryFrameForOffset(p.Upper-0x42b88-player*22*actorBobBytes, actorBobBytes)}
	a.lower = walkAnimation{step: p.LowerStep, delay: p.LowerDelay, frame: infantryFrameForOffset(p.Lower-0x46288, overlayBobBytes)}
	if p.Lower == 0x4a1a8 {
		a.lower.frame = -2
	} else if p.Lower == 0x4a248 {
		a.lower.frame = -3
	}
	return a
}

func checkOriginalPlayerAnimation(t *testing.T, player int, value actor, expected originalPlayerAnimationState) {
	t.Helper()
	upper, lower := value.playerArtOffsets(player)
	if upper+0x42b88 != expected.Upper || lower+0x42b88 != expected.Lower || value.upper.step != expected.UpperStep || value.upper.delay != expected.UpperDelay || value.lower.step != expected.LowerStep || value.lower.delay != expected.LowerDelay {
		t.Fatalf("player%d art%x/%x phases%+v/%+v, original%+v", player, upper+0x42b88, lower+0x42b88, value.upper, value.lower, expected)
	}
}

// All 117 boundaries are ordinary original landing frames. This also checks the
// retained title-wave phase, rather than resetting the native phase to a fit.
func TestOriginalOrdinaryPlayerLandingAnimation(t *testing.T) {
	var fixture struct {
		Landing []struct {
			Tick         int
			Players      [2]originalPlayerAnimationState
			SourceSHA256 string
		}
		Restart struct {
			Before, After [2]originalPlayerAnimationState
		}
	}
	readOriginalPlayerAnimation(t, "testdata/player-landing-animation.json.gz", &fixture)
	if len(fixture.Landing) != 117 {
		t.Fatal("incomplete ordinary source landing")
	}
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGamePlayers(false, 2)
	for _, frame := range fixture.Landing {
		if frame.Tick > 0 {
			g.updateIntro()
		}
		for player, p := range frame.Players {
			value := actor{upper: g.arrival.upper[player], lower: g.arrival.lower[player]}
			checkOriginalPlayerAnimation(t, player, value, p)
		}
	}
	for player, p := range fixture.Landing[116].Players {
		value := g.player
		if player == 1 {
			value = g.player2
		}
		checkOriginalPlayerAnimation(t, player, value, p)
	}
	// Actual source F1/menu observations retain independent upper/lower phases.
	g.player = observedPlayerAnimation(0, fixture.Restart.Before[0])
	g.player2 = observedPlayerAnimation(1, fixture.Restart.Before[1])
	g.scene = playScene
	g.Back()
	g.startGamePlayers(false, 2)
	for range 116 {
		g.updateIntro()
	}
	checkOriginalPlayerAnimation(t, 0, g.player, fixture.Restart.After[0])
	checkOriginalPlayerAnimation(t, 1, g.player2, fixture.Restart.After[1])
}

// These are controlled original FB82/FB2A calls, with explicit wet flags and
// retained phases from walking, waving and falling. They qualify animation
// state transitions, rather than claiming ordinary water traversal or play.
func TestOriginalControlledRetainedPlayerAnimation(t *testing.T) {
	var rows []struct {
		Player, Direction int
		Wet               []int
		States            []originalPlayerAnimationState
		SourceSHA256      string
	}
	readOriginalPlayerAnimation(t, "testdata/player-retained-animation.json.gz", &rows)
	if len(rows) != 256 {
		t.Fatal("incomplete controlled original player sequences")
	}
	transitions := 0
	for index, row := range rows {
		if len(row.States) != len(row.Wet)+1 {
			t.Fatal("incomplete original animation interval")
		}
		value := observedPlayerAnimation(row.Player, row.States[0])
		for tick, wet := range row.Wet {
			advancePlayerWalk(&value.upper, &value.lower, row.Player, row.Direction, wet != 0, false)
			upper, lower := value.playerArtOffsets(row.Player)
			expected := row.States[tick+1]
			if upper+0x42b88 != expected.Upper || lower+0x42b88 != expected.Lower || value.upper.step != expected.UpperStep || value.upper.delay != expected.UpperDelay || value.lower.step != expected.LowerStep || value.lower.delay != expected.LowerDelay {
				t.Fatalf("case%d tick%d: player%d art%x/%x phases%+v/%+v, original%+v", index, tick, row.Player, upper+0x42b88, lower+0x42b88, value.upper, value.lower, expected)
			}
			transitions++
		}
	}
	if transitions != 2048 {
		t.Fatal("incomplete source transitions")
	}
}

package game

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

type originalPlayerBlinkState struct {
	Protection, Phase, Falling, Active int
}

type originalPlayerBlinkCase struct {
	Protection, Phase int
	Mode              string
}

type originalPlayerBlinkObservations struct {
	Cases []struct {
		Index         int
		Case          originalPlayerBlinkCase
		Before, After [2]originalPlayerBlinkState
	}
	Scenes []struct {
		Index         int
		Case          originalPlayerBlinkCase
		Scene         []struct{ Player, X, Y, Upper, Lower int }
		State         [2]originalPlayerBlinkState
		Before, After []uint8
	}
	ColdPhases            [2]uint8
	MenuBefore, MenuAfter [2]originalPlayerBlinkState
}

func readOriginalPlayerBlink(t *testing.T) originalPlayerBlinkObservations {
	t.Helper()
	file, err := os.Open("testdata/player-blink.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	packed, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	defer packed.Close()
	var original originalPlayerBlinkObservations
	if err := json.NewDecoder(packed).Decode(&original); err != nil {
		t.Fatal(err)
	}
	if len(original.Cases) != 135 || len(original.Scenes) != 4 {
		t.Fatal("incomplete original player-blink observations")
	}
	return original
}

// These two-player transitions were measured at the actual original late
// protection block, including its last protected tick and byte underflow.
// The expected phase and visibility do not come from native counter parity.
func TestOriginalPlayerBlinkCycle(t *testing.T) {
	original := readOriginalPlayerBlink(t)
	for _, source := range original.Cases {
		t.Run(fmt.Sprintf("case-%03d-%s", source.Index, source.Case.Mode), func(t *testing.T) {
			g := &Game{playerCount: 2}
			for player, before := range source.Before {
				value := &g.player
				if player == 1 {
					value = &g.player2
				}
				value.invulnerable, value.blinkPhase = before.Protection, uint8(before.Phase)
				if before.Active != 0 {
					value.health = 1
				}
				if before.Falling != 0 {
					// The isolated source block starts after the fall pass.
					// The late protection block does not advance this existing fall.
					value.health, value.deadTicks = 0, 1
				}
				g.advancePlayerProtection(player)
				after := source.After[player]
				if value.invulnerable != after.Protection || int(value.blinkPhase) != after.Phase || value.hidden != (after.Active == 0) {
					t.Errorf("player %d: native protection%d phase%d hidden%v; original %+v", player, value.invulnerable, value.blinkPhase, value.hidden, after)
				}
			}
		})
	}
}

func TestOriginalPlayerBlinkMenuRetention(t *testing.T) {
	original := readOriginalPlayerBlink(t)
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	if g.player.blinkPhase != original.ColdPhases[0] || g.player2.blinkPhase != original.ColdPhases[1] {
		t.Fatal("cold player blink cycle differs from the original executable")
	}
	g.player.blinkPhase, g.player2.blinkPhase = uint8(original.MenuBefore[0].Phase), uint8(original.MenuBefore[1].Phase)
	g.player.hidden, g.player2.hidden = true, true
	g.startGame(false)
	if int(g.player.blinkPhase) != original.MenuAfter[0].Phase || int(g.player2.blinkPhase) != original.MenuAfter[1].Phase || g.player.hidden {
		t.Fatal("new game failed to retain the original visual phases and restore the first player")
	}
}

package game

import (
	"encoding/json"
	"os"
	"testing"
)

func TestOriginalBurningTankContextAndRepeatedHit(t *testing.T) {
	data, err := os.ReadFile("testdata/tank-context.json")
	if err != nil {
		t.Fatal(err)
	}
	var original []struct {
		Stage                              string
		InTank, Burning, BurnTicks, P2Tank int
	}
	if err = json.Unmarshal(data, &original); err != nil {
		t.Fatal(err)
	}
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGamePlayers(false, 2)
	g.player = actor{x: 104, y: 10000, health: 1, inTank: true, tankSlot: 19, tankFuel: 64}
	g.player2 = actor{x: 104, y: 10000, health: 1}
	for _, frame := range original {
		switch frame.Stage {
		case "burned":
			g.hitPlayer(0)
		case "hit_again":
			g.player.tankBurn = 0
			g.hitPlayer(0)
		case "dismounted":
			if !g.useTank(0) {
				t.Fatal("the original burning vehicle could not be left")
			}
			g.enemies = append(g.enemies, actor{x: 104, y: 10000, health: 1, kind: 4, slot: 20, tankFuel: 64})
		case "own_board_blocked":
			if g.useTank(0) {
				t.Fatal("the driver boarded another tank while the original burning context was active")
			}
		case "other_player_boarded":
			// The observed original did NOT board: slot19 was encountered
			// before the eligible slot20, and its burning owner blocked it.
			if g.useTank(1) {
				t.Fatal("the second player skipped the first original overlapping tank slot")
			}
		case "respawned":
			continue
		}
		if g.player.inTank != (frame.InTank != 0) || g.player2.inTank != (frame.P2Tank != 0) {
			t.Fatalf("original context%s changed vehicle ownership", frame.Stage)
		}
		burning, remaining := g.player.inTank && g.player.tankBurning, g.player.tankBurn
		for _, tank := range g.enemies {
			if tank.kind == 4 && tank.tankBurning && tank.tankDriver == 0 {
				burning, remaining = true, tank.tankBurn
			}
		}
		if burning != (frame.Burning != 0) || remaining != frame.BurnTicks {
			t.Fatalf("original context%s changed the burning interval", frame.Stage)
		}
	}
}

func TestOriginalTankSelectionUsesSlotOrderInsteadOfInsertionOrder(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGame(false)
	g.enemies = []actor{
		{x: g.player.x, y: g.player.y, health: 1, kind: 4, slot: 21, tankFuel: 64},
		{x: g.player.x, y: g.player.y, health: 1, kind: 4, slot: 19, tankFuel: 40},
	}
	if !g.useTank(0) || g.player.tankSlot != 19 || g.player.tankFuel != 40 {
		t.Fatal("the context action ignored the original slot19/20/21 order")
	}
}

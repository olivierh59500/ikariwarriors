package game

import (
	"encoding/json"
	"os"
	"strconv"
	"testing"
)

// Original player probes distinguish river tiles that infantry can cross
// from the same tiles that stop even a burning, occupied tank.
func TestOriginalTankWaterProbes(t *testing.T) {
	data, err := os.ReadFile("testdata/tank-water.json")
	if err != nil {
		t.Fatal(err)
	}
	var rows []struct {
		Player, Code              int
		Boarded, Burning, Allowed bool
	}
	if err = json.Unmarshal(data, &rows); err != nil {
		t.Fatal(err)
	}
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.cameraY = 13356
	for _, row := range rows {
		mode := "infantry"
		if row.Boarded {
			mode = "tank"
		}
		if row.Burning {
			mode = "burning"
		}
		t.Run(strconv.Itoa(row.Player)+"/"+mode+"/"+strconv.Itoa(row.Code), func(t *testing.T) {
			value := &g.player
			if row.Player == 1 {
				value = &g.player2
			}
			*value = actor{x: 96, y: 13432, health: 1, inTank: row.Boarded, tankBurning: row.Burning}
			g.resources.levelCodes[839][6] = byte(row.Code)
			if got := g.playerTerrainPassable(value, 96, 13424); got != row.Allowed {
				t.Fatalf("native passability %v, original %v", got, row.Allowed)
			}
			if row.Player == 0 {
				_, _, allowed := g.demoStep(96, 13432, 0, -1)
				if allowed != row.Allowed {
					t.Fatalf("demo passability %v, original %v", allowed, row.Allowed)
				}
			}
		})
	}
}

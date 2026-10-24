package game

import (
	"compress/gzip"
	"encoding/json"
	"os"
	"testing"
)

type originalMineState struct {
	Players []struct {
		X, Y   int
		Active bool
		Lives  int
	}
	Mines      []struct{ Slot, Kind, X, Y int }
	Animations []struct{ Step, Delay, Frame int }
	Alarm      int
	Terrain    [][3]int
	SHA256     string
}

// These independent original subroutine observations cover motion gates, both
// player choices, water boundaries, persistent animation phases and alarms.
func TestOriginalHomingMineMotionAndWarnings(t *testing.T) {
	var rows []struct {
		Name, Routine string
		Draws         []uint32
		Effects       []int
		Before, After originalMineState
	}
	f, err := os.Open("testdata/homing-mines.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	z, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	defer z.Close()
	if err = json.NewDecoder(z).Decode(&rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 115 {
		t.Fatal("incomplete original mine oracle")
	}
	for _, row := range rows {
		t.Run(row.Name, func(t *testing.T) {
			g, err := New(Options{Mute: true})
			if err != nil {
				t.Fatal(err)
			}
			g.playerCount = 1
			if row.Before.Players[1].Active {
				g.playerCount = 2
			}
			g.player = actor{x: float64(row.Before.Players[0].X), y: float64(row.Before.Players[0].Y), health: 1}
			g.player2 = actor{x: float64(row.Before.Players[1].X), y: float64(row.Before.Players[1].Y), health: 1}
			g.lives, g.lives2 = row.Before.Players[0].Lives, row.Before.Players[1].Lives
			g.cameraY = 13904
			g.cameraBackY = 13920
			g.mineAlarmTicks = uint8(row.Before.Alarm)
			for i, a := range row.Before.Animations {
				g.mineAnimations[i] = walkAnimation{step: a.Step, delay: a.Delay, frame: a.Frame}
			}
			for _, mine := range row.Before.Mines {
				g.mines = append(g.mines, landmine{x: float64(mine.X), y: float64(mine.Y), kind: mine.Kind, slot: mine.Slot})
			}
			for _, cell := range row.Before.Terrain {
				g.resources.levelCodes[cell[0]][cell[1]] = byte(cell[2])
			}
			used := 0
			if row.Routine == "motion" {
				g.moveHomingMines(func() uint32 {
					if used >= len(row.Draws) {
						t.Fatal("extra random draw")
					}
					v := row.Draws[used]
					used++
					return v
				})
			} else {
				g.animateMines()
			}
			if used != len(row.Draws) {
				t.Fatalf("random draws%d, original%d", used, len(row.Draws))
			}
			if len(g.mines) != len(row.After.Mines) {
				t.Fatal("active mine count changed")
			}
			for i, want := range row.After.Mines {
				got := g.mines[i]
				if got.x != float64(want.X) || got.y != float64(want.Y) || got.slot != want.Slot {
					t.Fatalf("mine%+v, original%+v", got, want)
				}
			}
			for i, want := range row.After.Animations {
				got := g.mineAnimations[i]
				if got.step != want.Step || got.delay != want.Delay || got.frame != want.Frame {
					t.Fatalf("slot%d animation%+v, original%+v", i+81, got, want)
				}
			}
			if int(g.mineAlarmTicks) != row.After.Alarm {
				t.Fatalf("alarm%d, original%d", g.mineAlarmTicks, row.After.Alarm)
			}
		})
	}
}

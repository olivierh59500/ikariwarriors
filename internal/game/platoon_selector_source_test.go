package game

import (
	"encoding/json"
	"os"
	"testing"
)

func TestOriginalPlatoonPaintAndInitializerBoundaries(t *testing.T) {
	var source struct {
		Cases []struct {
			Frame, Kind, Camera, ExpectedPaintSelector, ExpectedSpawnPath int
			Tank                                                          bool
			Player, Last                                                  struct {
				Slot, X, Y, Upper, Lower int
				Active                   bool
			}
		}
	}
	data, err := os.ReadFile("testdata/platoon-paint-source-cases.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &source); err != nil {
		t.Fatal(err)
	}
	if len(source.Cases) != 17 {
		t.Fatal("missing independent original platoon boundaries")
	}
	for _, row := range source.Cases {
		g, err := New(Options{Mute: true})
		if err != nil {
			t.Fatal(err)
		}
		g.cameraY = float64(row.Camera)
		g.playerCount = 1
		g.player = actor{x: float64(row.Player.X), y: float64(row.Player.Y), health: 1, hidden: !row.Player.Active, inTank: row.Tank, tankSlot: 19}
		if !row.Tank {
			switch row.Player.Lower {
			case 0x4a1a8:
				g.player.lower.frame = -2
			case 0x4a248:
				g.player.lower.frame = -3
			default:
				g.player.lower.frame = infantryFrameForOffset(row.Player.Lower-0x46288, overlayBobBytes)
			}
		}
		if row.Last.Slot == 31 {
			g.enemyVehicles[1] = enemyVehicle{x: float64(row.Last.X), y: float64(row.Last.Y), alive: [3]bool{false, false, true}}
		} else if row.Last.Slot != 23 && row.Last.Slot != 25 {
			t.Fatalf("source boundary%d needs another actual wide-object family", row.Frame)
		}
		g.rememberPlatoonMiddlePath()
		if int(g.platoonMiddlePath) != row.ExpectedPaintSelector {
			t.Fatalf("source boundary%d paint selector%d, original%d", row.Frame, g.platoonMiddlePath, row.ExpectedPaintSelector)
		}
		g.applyLevelEvent(levelEvent{kind: row.Kind, progress: levelHeight - row.Camera, x: 100})
		found := false
		for _, enemy := range g.enemies {
			if enemy.slot == 7 {
				found = true
				if enemy.marchX != row.ExpectedSpawnPath {
					t.Fatalf("source boundary%d spawned path%d, original%d", row.Frame, enemy.marchX, row.ExpectedSpawnPath)
				}
			}
		}
		if !found {
			t.Fatal("original middle platoon slot was not created")
		}
	}
}

func TestPlatoonSelectorUsesSimulationPaintOrdering(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.cameraY, g.playerCount = 1000, 1
	g.player = actor{x: 100, y: 1100, health: 1, lower: walkAnimation{frame: 16}}
	g.rememberPlatoonMiddlePath()
	before := g.platoonMiddlePath
	if before == 0 {
		t.Fatal("visible legs did not select their original path")
	}
	g.player.hidden = true
	g.rememberPlatoonMiddlePath()
	if g.platoonMiddlePath != before {
		t.Fatal("hidden player changed the retained last visible selection")
	}
	g.enemyVehicles[1] = enemyVehicle{x: 100, y: 1100, alive: [3]bool{false, false, true}}
	g.rememberPlatoonMiddlePath()
	if g.platoonMiddlePath != 0 {
		t.Fatal("later legless wide slot did not replace the selection")
	}
	g.player.hidden = false
	g.enemyVehicles[1].y = 2000
	g.rememberPlatoonMiddlePath()
	if g.platoonMiddlePath != before {
		t.Fatal("offscreen vehicle replaced visible player legs")
	}
}

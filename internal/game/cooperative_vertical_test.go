package game

import (
	"compress/gzip"
	"encoding/json"
	"os"
	"testing"
)

type originalCooperativePlayer struct {
	X, Y, Progress, RawY, Lives          int
	Active, Tank, Wet, Up, Down, Claimed bool
}
type originalCooperativeState struct {
	Camera, Room int
	Players      [2]originalCooperativePlayer
}

func TestOriginalCooperativeVerticalMovement(t *testing.T) {
	f, err := os.Open("testdata/cooperative-vertical.json.gz")
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
		Name                      string
		Parameters                []any
		Before, After             originalCooperativeState
		BeforeSHA256, AfterSHA256 string
	}
	if err := json.NewDecoder(z).Decode(&rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 544 {
		t.Fatal("incomplete original cooperative movement matrix")
	}
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range rows {
		t.Run(row.Name, func(t *testing.T) {
			g.startGamePlayers(false, 2)
			g.scene = playScene
			g.eventIndex = len(g.resources.events)
			g.cameraY, g.cameraBackY = float64(row.Before.Camera), float64(row.Before.Camera+row.Before.Room)
			g.secondaryTerrain = false
			for r := 505; r < 535; r++ {
				for c := 0; c < mapColumns; c++ {
					g.resources.levelCodes[r][c] = 0
				}
			}
			owner := int(row.Parameters[0].(float64))
			dy := row.Parameters[1].(float64)
			ownCode, peerCode := byte(row.Parameters[6].(float64)), byte(row.Parameters[7].(float64))
			for i, p := range row.Before.Players {
				value := actor{x: float64(p.X), y: float64(p.Y), health: 1, inTank: p.Tank, wet: p.Wet}
				if i == 0 {
					g.player = value
					g.lives = p.Lives
				} else {
					g.player2 = value
					g.lives2 = p.Lives
				}
				g.cooperativeScroll[i] = cooperativeScrollState{up: p.Up, down: p.Down, claimed: p.Claimed}
				code := peerCode
				if i == owner {
					code = ownCode
				}
				for r := 505; r < 535; r++ {
					g.resources.levelCodes[r][p.X/tileSize] = code
				}
			}
			g.moveCooperativeVertically(owner, dy)
			if g.cameraY != float64(row.After.Camera) || g.cameraBackY-g.cameraY != float64(row.After.Room) {
				t.Errorf("camera%.0f room%.0f, original%d/%d", g.cameraY, g.cameraBackY-g.cameraY, row.After.Camera, row.After.Room)
			}
			for i, p := range row.After.Players {
				value := g.player
				if i == 1 {
					value = g.player2
				}
				state := g.cooperativeScroll[i]
				if value.x != float64(p.X) || value.y != float64(p.Y) || value.wet != p.Wet || state.up != p.Up || state.down != p.Down || state.claimed != p.Claimed {
					t.Errorf("player%d %.0f/%.0f wet%v state%+v; original%+v", i, value.x, value.y, value.wet, state, p)
				}
			}
		})
	}
}

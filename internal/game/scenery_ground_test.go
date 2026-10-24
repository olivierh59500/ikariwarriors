package game

import "testing"

type originalGroundCreation struct {
	Paint                     struct{ Row, Column, Rows, Columns, YOffset, ClipTop int }
	Camera, Room              int
	Destroyed                 [][2]int
	BeforeSHA256, AfterSHA256 string
}

func originalGroundCreations(t *testing.T) []originalGroundCreation {
	t.Helper()
	var rows []originalGroundCreation
	readOriginalPlayerAnimation(t, "testdata/scenery-ground-creation.json.gz", &rows)
	if len(rows) != 4 {
		t.Fatal("incomplete original ground creation windows")
	}
	return rows
}

// Representative probes use the observed original bounds/parity. The source
// camera and retreat room independently determine which painted rows survive
// the next map strip; this is a controlled construction, not an input replay.
func TestOriginalSceneryGroundCreation(t *testing.T) {
	for _, source := range originalGroundCreations(t) {
		g, err := New(Options{Mute: true})
		if err != nil {
			t.Fatal(err)
		}
		g.startGame(false)
		g.cameraY = float64(source.Camera)
		g.cameraBackY = float64(source.Camera + source.Room)
		for _, p := range source.Destroyed {
			g.destroyed[p[0]][p[1]] = true
		}
		p := source.Paint
		if !g.destroyTileAt(float64(p.Column*tileSize+8), float64(p.Row*tileSize-8-p.YOffset)) {
			t.Fatal("native could not clear the source object")
		}
		if len(g.sceneryPaint) != 1 {
			t.Fatal("missing destruction paint history")
		}
		got := g.sceneryPaint[0]
		if got.row != p.Row || got.column != p.Column || got.rows != p.Rows || got.columns != p.Columns || got.yOffset != p.YOffset || got.clipTop != p.ClipTop {
			t.Fatalf("native ground%+v, source%+v", got, p)
		}
		g.startGame(false)
		if len(g.sceneryPaint) != 0 {
			t.Fatal("a new game retained a previous game's scenery paint")
		}
	}
}

// Attack forecasts copy Game. Appending a live destruction must preserve a
// previously computed future's different impact parity, including after a
// shared history has spare storage capacity.
func TestSceneryGroundForecastHistoryIsIndependent(t *testing.T) {
	rows := originalGroundCreations(t)
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGame(false)
	for _, p := range rows[0].Destroyed {
		g.destroyed[p[0]][p[1]] = true
	}
	for _, source := range rows[:3] {
		p := source.Paint
		g.cameraY = float64(source.Camera)
		g.cameraBackY = float64(source.Camera + source.Room)
		if !g.destroyTileAt(float64(p.Column*tileSize+8), float64(p.Row*tileSize-7)) {
			t.Fatal("failed to construct the observed history")
		}
	}
	source := rows[3]
	p := source.Paint
	g.cameraY = float64(source.Camera)
	g.cameraBackY = float64(source.Camera + source.Room)
	future := *g
	x, y := float64(p.Column*tileSize+8), float64(p.Row*tileSize-8)
	if !future.destroyTileAt(x, y+1) || !g.destroyTileAt(x, y) {
		t.Fatal("forecast and live clears must both be valid")
	}
	if len(g.sceneryPaint) != 4 || len(future.sceneryPaint) != 4 || g.sceneryPaint[3].yOffset != 0 || future.sceneryPaint[3].yOffset != -1 {
		t.Fatal("the live clear changed an already predicted ground history")
	}
}

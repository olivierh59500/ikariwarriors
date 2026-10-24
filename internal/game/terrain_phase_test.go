package game

import (
	"compress/gzip"
	"encoding/json"
	"os"
	"testing"
)

// Expected passability comes from the actual active table and map bytes in
// the two adjacent original states at the normally played zone transition.
func TestOriginalSceneWideTerrainCollisionStage(t *testing.T) {
	var scenes []struct {
		Name                 string
		CameraY, CameraBackY int
		SecondaryTerrain     bool
		Destroyed            [][2]int
		Queries              []struct {
			X, Y, Code int
			Passable   bool
		}
		SourceSHA256 string
	}
	readTerrainPhaseFixture(t, "testdata/terrain-phase.json.gz", &scenes)
	if len(scenes) != 2 {
		t.Fatal("missing original stage boundary")
	}
	for _, scene := range scenes {
		t.Run(scene.Name, func(t *testing.T) {
			g, err := New(Options{Mute: true})
			if err != nil {
				t.Fatal(err)
			}
			g.startGame(false)
			g.cameraY, g.cameraBackY = float64(scene.CameraY), float64(scene.CameraBackY)
			g.secondaryTerrain = scene.SecondaryTerrain
			for _, cell := range scene.Destroyed {
				g.destroyed[cell[0]][cell[1]] = true
			}
			for _, query := range scene.Queries {
				if code := g.terrainCode(float64(query.X), float64(query.Y)); int(code) != query.Code {
					t.Fatalf("original map differs at%d/%d: got%d expected%d", query.X, query.Y, code, query.Code)
				}
				if solid := g.solidPixelAt(float64(query.X), float64(query.Y)); solid == query.Passable {
					t.Fatalf("active terrain table differs at%d/%d code%d: solid%v expected passable%v", query.X, query.Y, query.Code, solid, query.Passable)
				}
			}
		})
	}
}

func TestSecondaryCollisionStageSurvivesOrdinaryRetreat(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGame(false)
	g.cameraY, g.cameraBackY = 4800, 4832
	if g.collisionTerrainBank() != 0 {
		t.Fatal("second table selected before its original boundary")
	}
	g.demoPath = []navPoint{{20, 4700}}
	g.followPlayer(4894, 4892)
	if g.cameraY != 4798 || g.cameraBackY != 4800 || g.collisionTerrainBank() != 1 || len(g.demoPath) != 0 {
		t.Fatalf("normal scroll failed to change collision stage: camera%g back%g path%d", g.cameraY, g.cameraBackY, len(g.demoPath))
	}
	g.followPlayer(4956, 4958)
	if g.cameraY != 4800 || g.collisionTerrainBank() != 1 {
		t.Fatalf("retreat reverted the scene collision table: camera%g back%g", g.cameraY, g.cameraBackY)
	}
	g.startGame(false)
	if g.collisionTerrainBank() != 0 {
		t.Fatal("a new game retained the second zone collision stage")
	}
}

func readTerrainPhaseFixture(t *testing.T, path string, value any) {
	t.Helper()
	f, e := os.Open(path)
	if e != nil {
		t.Fatal(e)
	}
	defer f.Close()
	z, e := gzip.NewReader(f)
	if e != nil {
		t.Fatal(e)
	}
	defer z.Close()
	if e = json.NewDecoder(z).Decode(value); e != nil {
		t.Fatal(e)
	}
}

package game

import (
	"compress/gzip"
	"encoding/json"
	"os"
	"testing"
)

// Actual source movement drives only the scroll/dispatch handler. Expected
// camera positions and event cursors are independently recorded original data.
func TestRecordedOriginalScrollEventIntervals(t *testing.T) {
	f, err := os.Open("testdata/scroll-events.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	z, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	defer z.Close()
	var cases []struct {
		Name                                                 string
		StartY, StartCamera, StartIndex, EndIndex, EndCamera int
		PlayerY, CameraY, Directions                         []int
		BeforeSHA256, AfterSHA256                            string
	}
	if err := json.NewDecoder(z).Decode(&cases); err != nil {
		t.Fatal(err)
	}
	if len(cases) != 224 {
		t.Fatal("incomplete original event intervals")
	}
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			g.startGame(false)
			g.scene = playScene
			g.cameraY = float64(c.StartCamera)
			g.cameraBackY = levelHeight - float64((int(levelHeight-g.cameraY)-1)&^31)
			g.secondaryTerrain = g.cameraY < secondaryMapRows*tileSize
			g.eventIndex = c.StartIndex
			previous := float64(c.StartY)
			for i, y := range c.PlayerY {
				// Boarding can snap the actor before vertical input; it is not a
				// camera step. Retain only movement requested by the recorded joystick.
				start := previous
				if c.Directions[i]&0x30 == 0 {
					start = float64(y)
				}
				g.followPlayer(start, float64(y))
				previous = float64(y)
				if g.cameraY != float64(c.CameraY[i]) {
					t.Fatalf("scroll%d camera%.0f, original%d", i, g.cameraY, c.CameraY[i])
				}
			}
			if g.eventIndex != c.EndIndex || g.cameraY != float64(c.EndCamera) {
				t.Fatalf("event cursor%d camera%.0f, original%d/%d", g.eventIndex, g.cameraY, c.EndIndex, c.EndCamera)
			}
		})
	}
}

// The original dispatcher is called only by forward scrolling. Holding still
// on an event coordinate must not activate that event during the global loop.
func TestScrollEventWaitsForAnUpwardCameraStep(t *testing.T) {
	for _, direction := range []float64{0, 1, -1} {
		g, err := New(Options{Mute: true})
		if err != nil {
			t.Fatal(err)
		}
		g.startGame(false)
		g.scene = playScene
		g.cameraY, g.cameraBackY = 144, 160
		g.secondaryTerrain = true
		g.player = actor{x: 70, y: 238, health: 1}
		g.ammunition, g.grenades = 94, 96
		for i, e := range g.resources.events {
			if e.kind == 24 {
				g.eventIndex = i
				break
			}
		}
		g.enemies, g.pickups, g.mines, g.projectiles = nil, nil, nil, nil
		g.randomSpawn, g.redSpawn = false, false
		before := g.eventIndex
		g.stepPlay([2]playerControls{{y: direction}})
		want := before
		if direction < 0 {
			want++
		}
		if g.eventIndex != want || g.ending.active != (direction < 0) {
			t.Fatalf("direction%g advanced event%d->%d active%v", direction, before, g.eventIndex, g.ending.active)
		}
	}
}

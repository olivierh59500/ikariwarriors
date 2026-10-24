package game

import (
	"encoding/json"
	"os"
	"strconv"
	"testing"
)

// These poses were sampled after the original event-four handler and normal
// original gameplay ticks. They independently specify all five bob addresses.
func TestOriginalAircraftPartsAndRotor(t *testing.T) {
	data, err := os.ReadFile("testdata/aircraft.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Tick   int
		Actors []struct {
			Slot, X, Y int
			Bob        string
			Active     int
		}
	}
	if err = json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.cameraY = levelHeight - 5700
	g.spawnAircraft(levelEvent{kind: 4, x: 70, progress: 5700})
	tick := 0
	for _, c := range cases {
		for tick < c.Tick {
			g.updateAircraft()
			tick++
		}
		for _, part := range c.Actors {
			if part.Slot > 30 {
				continue
			}
			i := part.Slot - 26
			p := g.aircraft.part(i)
			address64, err := strconv.ParseInt(part.Bob, 0, 32)
			if err != nil {
				t.Fatal(err)
			}
			address := int(address64)
			if int(p.x+48) != part.X || int(p.y-g.cameraY+4) != part.Y || p.bob+0x42b88 != address || !g.aircraft.alive[i] {
				t.Fatalf("original aircraft tick%d slot%d: %+v, expected%+v", tick, part.Slot, p, part)
			}
			if sprite := g.resources.bob(p.bob, vehicleBobHeight); sprite == nil || sprite.Bounds().Dx() != 32 || sprite.Bounds().Dy() != 24 {
				t.Fatal("aircraft part has the wrong original graphic format")
			}
		}
	}
}

func TestOriginalAircraftDamageGroups(t *testing.T) {
	data, err := os.ReadFile("testdata/aircraft-damage.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Part, Flying int
		Active       []int
	}
	if err = json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.cameraY = levelHeight - 5700
	g.spawnAircraft(levelEvent{kind: 4, x: 70, progress: 5700})
	for _, c := range cases {
		g.aircraft.damaged = 1 << uint(c.Part-26)
		g.updateAircraft()
		if g.aircraft.flying != (c.Flying != 0) {
			t.Fatal("aircraft cleanup differs from the original group flag")
		}
		for i := range 5 {
			if g.aircraft.alive[i] != (c.Active[i] != 0) {
				t.Fatalf("original part%d cleanup: %+v", c.Part, g.aircraft.alive)
			}
		}
	}
}

func TestOriginalAircraftSideHitsThroughNormalGameplay(t *testing.T) {
	data, err := os.ReadFile("testdata/aircraft-side-hit.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Hit, Tick, Flying int
		Actors            []struct{ Slot, X, Y, Active int }
	}
	if err = json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.cameraY = levelHeight - 5700
	hit := -1
	for _, original := range cases {
		if hit != original.Hit {
			hit = original.Hit
			g.spawnAircraft(levelEvent{kind: 4, x: 70, progress: 5700})
			g.aircraft.damaged = 1 << uint(hit-26)
		}
		g.updateAircraft()
		if g.aircraft.flying != (original.Flying != 0) {
			t.Fatal("normal original side hit changed the aircraft controller")
		}
		for i, object := range original.Actors {
			part := g.aircraft.part(i)
			if g.aircraft.alive[i] != (object.Active != 0) || int(part.x+48) != object.X || int(part.y-g.cameraY+4) != object.Y {
				t.Fatalf("normal original aircraft hit%d tick%d part%d differs", hit, original.Tick, i)
			}
		}
	}
}

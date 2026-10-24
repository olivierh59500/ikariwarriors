package game

import (
	"encoding/json"
	"os"
	"testing"
)

// Source6606..6621 retain five independent part lifetimes. The anchor is
// painted at rawY238, clipped at240, then stops flight on the following pass.
func TestOriginalAircraftDeparturePartLifetimes(t *testing.T) {
	var source struct {
		Frames []struct {
			Frame, Camera int
			Flying        bool
			Parts         []struct {
				Slot, RawX, RawY, Image int
				Active                  bool
			}
		}
	}
	data, err := os.ReadFile("testdata/original-aircraft-retirement-original.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &source); err != nil {
		t.Fatal(err)
	}
	if len(source.Frames) != 16 {
		t.Fatal("missing independent original aircraft departure frames")
	}
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	first := source.Frames[0]
	g.cameraY = float64(first.Camera)
	g.aircraft = helicopter{x: float64(first.Parts[0].RawX - 48), y: g.cameraY + float64(first.Parts[0].RawY-4) + 30, flying: first.Flying}
	for index, part := range first.Parts {
		g.aircraft.alive[index] = part.Active
	}
	found := false
	for phase := range helicopterArt {
		if helicopterArt[phase][0] == first.Parts[0].Image {
			g.aircraft.phase = phase
			found = true
			break
		}
	}
	if !found {
		t.Fatal("original aircraft image is outside recovered rotor tables")
	}
	for _, expected := range source.Frames[1:] {
		g.cameraY = float64(expected.Camera)
		g.advanceAircraft()
		// Firing belongs here, before the original wide paint retires parts.
		enteredFire := g.aircraft.flying
		g.pruneOffscreenAircraftParts()
		if g.aircraft.flying != expected.Flying {
			t.Fatalf("source frame%d flight lifetime differs", expected.Frame)
		}
		for index, part := range expected.Parts {
			actual := g.aircraft.part(index)
			if g.aircraft.alive[index] != part.Active || int(actual.x+48) != part.RawX || int(actual.y-g.cameraY+4) != part.RawY || actual.bob+0x42b88 != part.Image {
				t.Fatalf("source frame%d part%d native%+v active%t, original%+v", expected.Frame, part.Slot, actual, g.aircraft.alive[index], part)
			}
		}
		if expected.Frame == 6618 && !enteredFire {
			t.Fatal("last departing anchor lost its valid firing phase")
		}
		if expected.Frame == 6619 && enteredFire {
			t.Fatal("inactive anchor retained a later firing phase")
		}
	}
}

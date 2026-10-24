package game

import (
	"encoding/json"
	"os"
	"strconv"
	"testing"
)

func TestOrdinaryMovementPrecedesTheOriginalFragmentProbe(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGame(false)
	g.scene = playScene
	g.player = actor{x: 100, y: 10000, health: 1}
	g.cameraY, g.cameraBackY = 9904, 9936
	g.enemies, g.mines = nil, nil
	g.randomSpawn, g.redSpawn = false, false
	g.eventIndex = len(g.resources.events)
	for row := range g.destroyed {
		for column := range g.destroyed[row] {
			g.destroyed[row][column] = true
		}
	}
	g.burst = radialBurst{active: true, remaining: 1, owner: -1}
	for i := range g.burst.parts {
		g.burst.parts[i] = navPoint{400, 9800}
	}
	// This fragment reaches the old position's right collision edge. The
	// original two-pixel left input moves outside it before the probe.
	g.burst.parts[6] = navPoint{115, 9992}
	g.stepPlay([2]playerControls{{x: -1}})
	if g.player.x != 98 || g.player.health != 1 || g.player.deadTicks != 0 || g.lives != 5 {
		t.Fatal("a fragment was applied before the ordinary escape movement")
	}
}

// Original $11290 and seven subsequent normal original ticks provide every
// shard's independent position, artwork, lifetime and active flag.
func TestOriginalRadialBurst(t *testing.T) {
	data, err := os.ReadFile("testdata/burst.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Tick, Remaining int
		Parts           []struct {
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
	g.createBurst(150-56, g.cameraY+50, -1)
	tick := 0
	for _, c := range cases {
		for tick < c.Tick {
			g.updateBurst()
			tick++
		}
		if g.burst.remaining != c.Remaining {
			t.Fatalf("tick%d remaining%d, original%d", tick, g.burst.remaining, c.Remaining)
		}
		for i, p := range c.Parts {
			address, err := strconv.ParseInt(p.Bob, 0, 32)
			if err != nil {
				t.Fatal(err)
			}
			part := g.burst.parts[i]
			if int(part.x+56) != p.X || int(part.y-g.cameraY) != p.Y || 0x4a2e8+burstFrames[i]*160 != int(address) || g.burst.active != (p.Active != 0) {
				t.Fatalf("original burst tick%d part%d: %+v, expected%+v", tick, i, part, p)
			}
		}
	}
}

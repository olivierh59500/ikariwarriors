package game

import (
	"encoding/json"
	"os"
	"testing"
)

// These three completed original frames bracket the fuel-exhaustion return.
// This isolated test supplies the observed placement-phase entry only; the
// separate cold picture replay proves its integrated effect without imports.
func TestOriginalFuelExhaustionRetainsTurretPlacement(t *testing.T) {
	type part struct {
		X, Y  float64
		Image int
	}
	var frames []struct {
		Frame, Camera, Fuel, Phase, Burn int
		Burning                          bool
		SourceSHA256                     string
		Body, Turret                     part
	}
	blob, err := os.ReadFile("testdata/original-fuel-turret-frames.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(blob, &frames); err != nil {
		t.Fatal(err)
	}
	if len(frames) != 3 {
		t.Fatal("incomplete original fuel transition")
	}
	before, first, next := frames[0], frames[1], frames[2]
	for _, source := range frames {
		if len(source.SourceSHA256) != 64 {
			t.Fatal("missing source RAM identity")
		}
	}
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.cameraY = float64(first.Camera)
	g.player = actor{x: first.Body.X, y: first.Body.Y, health: 1, inTank: true, tankSlot: 20, tankFuel: before.Fuel, tankTurret: navPoint{before.Turret.X, before.Turret.Y}, tankTurretPlaced: true}
	g.tankFuelPhase[0] = uint8(before.Phase)
	compare := func(source int) {
		row := frames[source]
		x, y := g.player.tankTurretPoint()
		if x != row.Turret.X || y != row.Turret.Y || g.player.tankFuel != row.Fuel || int(g.tankFuelPhase[0]) != row.Phase || g.player.tankBurning != row.Burning || g.player.tankBurn != row.Burn || 0x5a6e8+g.player.tankFlame.frame*vehicleBobBytes != row.Turret.Image {
			t.Fatalf("source frame%d turret %.0f/%.0f fuel%d phase%d burn%d differs", row.Frame, x, y, g.player.tankFuel, g.tankFuelPhase[0], g.player.tankBurn)
		}
	}
	g.advanceTankFuel(0)
	g.finishTankTick()
	compare(1)
	g.player.x, g.player.y = next.Body.X, next.Body.Y
	g.advanceTankBurn(0)
	g.advanceTankFuel(0)
	g.finishTankTick()
	compare(2)
}

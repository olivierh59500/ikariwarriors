package game

import "testing"

// The shipped attract mode begins with the landing, which changes the initial
// camera and actor state before ordinary demo controls take over. Exercise the
// actual Update path as well as the direct gameplay tests.
func TestAttractDemoCompletesFromTheArrivalScene(t *testing.T) {
	g, err := New(Options{Demo: true, Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	for tick := 0; tick < 20000 && g.scene != victoryScene && g.scene != titleScene; tick++ {
		if err := g.Update(); err != nil {
			t.Fatal(err)
		}
	}
	if g.scene != victoryScene || !g.ending.active || g.lives < 0 {
		t.Fatalf("attract mode stopped in scene%d after%d gameplay ticks: player%.0f/%.0f lives%d ammo%d grenades%d tank%v fuel%d", g.scene, g.gameplayTicks, g.player.x, g.player.y, g.lives, g.ammunition, g.grenades, g.player.inTank, g.player.tankFuel)
	}
	t.Logf("attract mode reached the ending after%d gameplay ticks, with%d reserve lives and score%d", g.gameplayTicks, g.lives, g.score)
}

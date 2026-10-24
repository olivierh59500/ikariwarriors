package game

import "testing"

// Source994..996 independently show late-clipped parts restored by movement,
// one final firing pass at rawY252 and no firing on the following inactive pass.
func TestEnemyVehicleRetainsItsFinalEnteredFirePass(t *testing.T) {
	g, err := New(Options{Mute: true, RandomSource: func(uint64) uint32 { return 1 }})
	if err != nil {
		t.Fatal(err)
	}
	g.cameraY = 1000
	g.enemyVehicles = [2]enemyVehicle{{
		x: 100, y: 1246, active: true, alive: [3]bool{true, true, true},
		chassis: walkAnimation{frame: 16}, turret: 0x59428, owner: -1,
	}}
	g.projectiles = nil
	v := &g.enemyVehicles[0]
	initial := g.randomSourceIndex
	v.advance(g.cameraY)
	if !v.active {
		t.Fatal("rawY250 ended the original movement controller too early")
	}
	g.fireEnemyVehicles()
	if g.randomSourceIndex != initial+2 {
		t.Fatal("active last-boundary turret missed its target/chance calls")
	}
	g.pruneOffscreenEnemyVehicles()
	if v.alive != [3]bool{} {
		t.Fatal("late wide painter retained clipped vehicle parts")
	}
	g.cameraY -= 2
	v.advance(g.cameraY)
	if v.active || v.alive != [3]bool{true, true, true} {
		t.Fatal("final entered movement did not restore parts before stopping its controller")
	}
	g.fireEnemyVehicles()
	if g.randomSourceIndex != initial+4 {
		t.Fatal("stopped controller suppressed its valid final firing pass")
	}
	g.pruneOffscreenEnemyVehicles()
	v.advance(g.cameraY)
	g.fireEnemyVehicles()
	if g.randomSourceIndex != initial+4 || v.alive != [3]bool{} {
		t.Fatal("inactive clipped turret kept consuming target/chance random words")
	}
}

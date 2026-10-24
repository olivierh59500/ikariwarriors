package game

import "testing"

func controlledFinalGrenadePose(t *testing.T, x float64, direction int) *Game {
	t.Helper()
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGame(false)
	g.player = actor{x: x, y: 414, direction: direction, health: 1}
	g.cameraY, g.cameraBackY = 320, 320
	g.secondaryTerrain = g.cameraY < secondaryMapRows*tileSize
	g.ammunition, g.grenades = 87, 1
	g.enemies, g.projectiles = nil, nil
	g.spawnFinalEncounter(levelEvent{progress: 14640})
	return g
}

func actualControlledGrenade(t *testing.T, g *Game, control playerControls) projectile {
	t.Helper()
	g.stepPlayer(0, control)
	if len(g.projectiles) != 1 || g.grenades != 0 {
		t.Fatalf("ordinary grenade was not launched: player%+v shots%+v", g.player, g.projectiles)
	}
	for tick := 0; tick < 18; tick++ {
		g.advancePlayerProjectiles(0)
	}
	return g.blasts[0]
}

// The naturally retained second cycle spent its last grenade here. Scoring
// the stationary landing135/346 invented a weakpoint hit; the actual turned
// muzzle moves two pixels first and lands137/346, outside the original probe.
func TestDemoGrenadeAimIncludesTheActualTurnedMuzzle(t *testing.T) {
	g := controlledFinalGrenadePose(t, 71, 6)
	if aim, ok := g.demoGrenadeAimForMove(playerControls{x: -1}); ok {
		t.Fatalf("recommended a grenade from the observed miss: direction%d", aim)
	}
	landed := actualControlledGrenade(t, g, playerControls{x: 1, y: -1, action: true})
	if landed.x != 137 || landed.y != 346 || !g.final.active {
		t.Fatalf("actual turned miss changed: landed%+v final%+v", landed, g.final)
	}
}

func TestDemoGrenadeAimIncludesMovementWithTheHeadingLocked(t *testing.T) {
	g := controlledFinalGrenadePose(t, 71, 1)
	if aim, ok := g.demoGrenadeAim(); !ok || aim != 1 {
		t.Fatal("stationary weakpoint-edge shot was not recognized")
	}
	if aim, ok := g.demoGrenadeAimForMove(playerControls{x: 1}); ok {
		t.Fatalf("moving locked shot reused the stationary muzzle: direction%d", aim)
	}
	g.directionLock[0] = true
	landed := actualControlledGrenade(t, g, playerControls{x: 1, action: true})
	if landed.x != 137 || landed.y != 346 || !g.final.active {
		t.Fatalf("actual locked moving miss changed: landed%+v final%+v", landed, g.final)
	}
}

func TestDemoGrenadeAimPreservesTheAdjacentActualWeakpointHit(t *testing.T) {
	g := controlledFinalGrenadePose(t, 69, 6)
	if aim, ok := g.demoGrenadeAimForMove(playerControls{x: -1}); !ok || aim != 1 {
		t.Fatal("adjacent valid turned weakpoint shot was rejected")
	}
	landed := actualControlledGrenade(t, g, playerControls{x: 1, y: -1, action: true})
	if landed.x != 135 || landed.y != 346 || g.final.active || !g.final.defeated {
		t.Fatalf("actual adjacent weakpoint hit changed: landed%+v final%+v", landed, g.final)
	}
}

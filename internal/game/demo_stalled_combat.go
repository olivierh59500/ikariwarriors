package game

import "math"

// A long no-scroll interval can mean a safe firing angle is missing rather
// than a blocked map. Recheck current visible fixed guns and ordinary shots;
// setup never removes any known actor, projectile or mine from the forecast.
func (g *Game) demoStalledGunSetup() (playerControls, bool) {
	if g.idleArtilleryTicks < 120 || g.player.inTank || g.ammunition <= 0 || g.demoFireHeld || !g.bulletAvailable(0) || g.ending.active {
		return playerControls{}, false
	}
	normal := g.demoPlanMoveDetails(navPoint{g.player.x, g.player.y - 16}, nil)
	for _, enemy := range g.enemies {
		if enemy.kind != 2 || enemy.health <= 0 || enemy.deadTicks > 0 || math.Abs(enemy.x-g.player.x) > 100 || math.Abs(enemy.y-g.player.y) > 120 {
			continue
		}
		if control, locked, ok := g.demoKnownGunFiringSetup(enemy.x, enemy.y, normal); ok {
			g.directionLock[0] = locked
			return control, true
		}
	}
	return playerControls{}, false
}

// A rooftop gun is tied to real destructible scenery, even if an open
// walking path technically skirts its building. A stalled expert can select
// that observed building as an ordinary finite-grenade objective.
func (g *Game) demoStalledRoofScenery() (int, int, bool) {
	if g.player.inTank || g.grenades <= 4 || g.ending.active {
		return 0, 0, false
	}
	for _, enemy := range g.enemies {
		if enemy.kind != 6 || enemy.health <= 0 || enemy.deadTicks > 0 || math.Abs(enemy.x-g.player.x) > 128 || enemy.y > g.player.y || g.player.y-enemy.y > 144 {
			continue
		}
		if row, column, _, found := g.sceneryBounds(enemy.x+8, enemy.y-4); found && !g.destroyed[row][column] {
			retained := g.demoPlanScenery && g.demoSceneryRow == row && g.demoSceneryCol == column
			if g.idleArtilleryTicks >= 120 || retained {
				return row, column, true
			}
		}
	}
	return 0, 0, false
}

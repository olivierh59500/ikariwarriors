package game

// When no bullets remain, a stationary soldier is an obstacle to the route,
// even when the surrounding terrain is open. Walking movement and the short
// threat forecast retain their ordinary collision rules.
func (g *Game) demoRouteInfantry(bodies []navPoint) int {
	if g.ammunition > 0 || g.player.inTank || g.ending.active {
		return 0
	}
	count := 0
	for _, enemy := range g.enemies {
		if enemy.slot < 0 || enemy.slot > 18 || enemy.health <= 0 || enemy.deadTicks > 0 || enemy.kind == 4 || enemy.kind == 6 || g.final.active && enemy.slot >= 5 {
			continue
		}
		if (enemy.kind == 0 || enemy.kind == 5) && enemy.marchTime > 0 {
			continue
		}
		if count == len(bodies) {
			break
		}
		bodies[count] = navPoint{enemy.x, enemy.y}
		count++
	}
	return count
}

func demoRouteClearsInfantry(bodies []navPoint, x, y, nx, ny float64) bool {
	for _, body := range bodies {
		dx, dy := nx-body.x, ny-body.y
		if dx < -25 || dx > 25 || dy < -21 || dy > 21 {
			continue
		}
		oldDepth := min(x-body.x+25, body.x+25-x, y-body.y+21, body.y+21-y)
		depth := min(dx+25, 25-dx, dy+21, 21-dy)
		// An actor already inside the safety margin may move out or sideways.
		if oldDepth < 0 || depth > oldDepth {
			return false
		}
	}
	return true
}

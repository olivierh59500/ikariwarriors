package game

// A clear next frame is insufficient: an outward fragment can reach an
// exposed passenger several frames after an apparently safe exit.
func (g *Game) demoTankExitThreat(x, y float64, elapsed int) float64 {
	value := g.player
	value.x, value.y, value.inTank = x, y, false
	threat := 0.0
	for tick := elapsed + 1; tick <= elapsed+demoHorizon; tick++ {
		threat += g.demoThreatForActor(value, tick)
	}
	return threat
}

// A burning tank can still be driven while its passenger is protected. Move
// toward a safer dismount instead of waiting for the fire to consume it when
// no air-strike marker happens to provide an escape direction.
func (g *Game) demoFindSaferTankExit() playerControls {
	directions := [...][2]float64{{0, -1}, {-1, -1}, {1, -1}, {-1, 0}, {1, 0}, {0, 1}, {-1, 1}, {1, 1}}
	best := playerControls{}
	bestCost := g.demoTankExitThreat(g.player.x, g.player.y, 0)
	for _, direction := range directions {
		x, y := g.player.x, g.player.y
		for tick := 1; tick <= demoHorizon && tick < g.player.tankBurn; tick++ {
			previousY := y
			nextX, nextY, ok := g.demoStep(x, y, direction[0], direction[1])
			if !ok {
				break
			}
			value := g.player
			value.x, value.y = nextX, nextY
			if g.demoThreatForMove(value, previousY, tick) > 0 {
				break
			}
			x, y = nextX, nextY
			cost := g.demoTankExitThreat(x, y, tick) + float64(tick)
			if cost < bestCost {
				bestCost = cost
				best = playerControls{x: direction[0], y: direction[1]}
			}
		}
	}
	return best
}

package game

// Infantry can cross the original water codes, but a tank cannot. A parked
// tank facing water would only return the passenger to the same dead end.
func (g *Game) demoTankWaterAhead(x, y float64) bool {
	return tankWaterCode(g.terrainCode(x, y-8))
}

// Reaching a one-point route at a shoreline means the tank has exhausted its
// reachable forward ground. Leave through the ordinary context action once
// the existing exposure forecast says that the passenger can do so safely.
func (g *Game) demoTankAtWaterDeadEnd() bool {
	return g.player.inTank && !g.player.tankBurning && !g.demoPlanScenery &&
		len(g.demoPath) == 1 && g.demoPath[0] == (navPoint{g.player.x, g.player.y}) &&
		g.demoTankWaterAhead(g.player.x, g.player.y)
}

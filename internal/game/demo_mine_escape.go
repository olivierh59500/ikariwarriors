package game

// Keep room for an unexpected water-mine jump. This is a preference, rather
// than a fictitious collision or knowledge of the next random movement gate.
func (g *Game) demoMineEscapeCost(x, y float64, tick int) float64 {
	if g.ending.active || g.player.invulnerable > tick {
		return 0
	}
	value := g.player
	value.x, value.y = x, y
	cost := 0.0
	for _, mine := range g.mines {
		if mine.kind != 19 {
			continue
		}
		if mine.x-x < -48 || mine.x-x > 48 || mine.y-y < -48 || mine.y-y > 48 {
			continue
		}
		next := g.homingMineStep(mine, value)
		if next.x == mine.x && next.y == mine.y {
			continue
		}
		depth := min(x-next.x+23, next.x-x+28, y-next.y+31, next.y-y+16)
		if depth > 0 {
			cost += depth * depth / float64(max(1, tick))
		}
	}
	return cost
}

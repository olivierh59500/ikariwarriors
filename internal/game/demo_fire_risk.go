package game

import "math"

// A visible firing pose warns against waiting just outside body contact.
// Prefer a flank or retreat while comparing ordinary attacks; carrying
// ammunition alone does not guarantee that a shot can reach the target.
// This uses no random state or future shot result, and remains a soft cost.
func (g *Game) demoPossibleShotCost(x, y float64, tick int) float64 {
	if g.ending.active || g.player.inTank || g.player.invulnerable > tick {
		return 0
	}
	cost := 0.0
	for i, enemy := range g.enemies {
		// A verified owned hit precedes that future enemy firing pass.
		// Existing hostile bullets retain their separate flight forecast.
		if i < len(g.demoEnemyDefeat) && g.demoEnemyDefeat[i] > 0 && tick >= g.demoEnemyDefeat[i] {
			continue
		}
		if enemy.health <= 0 || enemy.deadTicks > 0 {
			continue
		}
		marching := enemy.kind == 0 || enemy.kind == 5
		if marching {
			if enemy.shots == 0 || enemy.marchTime > tick {
				continue
			}
		} else if enemy.kind != 2 && enemy.kind != 6 {
			continue
		}
		ex, ey := g.forecastEnemyPosition(i, enemy, tick)
		ey += 12
		if !marching {
			ex -= 4
			if enemy.kind == 6 {
				ey -= 16
			}
		}
		height := y - ey
		if height < -8 || height > 32 {
			continue
		}
		for _, slope := range [...]float64{-1, 0, 1} {
			gap := math.Abs(x - (ex + slope*max(0, height)))
			if gap < 24 {
				cost += (24 - gap) * (24 - gap)
			}
		}
	}
	return cost
}

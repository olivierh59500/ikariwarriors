package game

// Bunkers use the same target chooser after existing enemy shots have moved.
// Their vertical component is always downward, regardless of target height.
func (g *Game) updateBunkers() {
	for slot := 32; slot <= 33; slot++ {
		for index := range g.enemies {
			enemy := &g.enemies[index]
			if enemy.slot != slot || enemy.kind != 6 || enemy.health <= 0 || !g.enemyBulletAvailable() {
				continue
			}
			target, ok := g.chooseEnemyTarget()
			if !ok {
				continue
			}
			dx := 0.0
			if target.x < enemy.x-16 {
				dx = -1
			} else if target.x > enemy.x+16 {
				dx = 1
			}
			enemy.direction = directionForVector(dx, 1)
			enemy.bob = 0x5b048 - 0x42b88
			if dx < 0 {
				enemy.bob = 0x5ae68 - 0x42b88
			} else if dx > 0 {
				enemy.bob = 0x5b228 - 0x42b88
			}
			if g.randomValue()&31 != 0 {
				continue
			}
			shot := g.projectileFor(enemy, true, -1, false, 4)
			shot.y -= 16
			g.addEnemyProjectile(shot)
			if g.audio != nil {
				g.audio.Effect(0, 2)
			}
		}
	}
}

package game

// Hold additional fire when a real owned bullet already has a lethal clear
// trajectory. Other actors can intercept it, and already-fired hostile shots
// remain dangerous after their shooter dies. No future random draw is used.
func (g *Game) demoPendingBulletTargets() uint64 {
	var pending uint64
	for _, shot := range g.projectiles {
		if shot.enemy || shot.owner != 0 || shot.grenade || g.ownedShotUsesCannon(shot) {
			continue
		}
		if index, lethal := g.demoBulletTarget(shot); lethal && index < 64 {
			pending |= uint64(1) << index
		}
	}
	return pending
}

func (g *Game) demoBulletTarget(shot projectile) (int, bool) {
	index, _, lethal := g.demoBulletImpact(shot)
	return index, lethal
}

func (g *Game) demoBulletImpact(shot projectile) (int, int, bool) {
	cleanup := g.ownedSlotCleanupDelay(shot)
	limit := shot.life + 3
	if shot.impact > 0 {
		limit = shot.impact
	}
	for tick := 1; tick <= limit; tick++ {
		moved, visible := g.advancePlayerBullet(shot)
		for i, enemy := range g.enemies {
			if enemy.health <= 0 || enemy.deadTicks > 0 || enemy.kind == 4 || enemy.kind == 6 {
				continue
			}
			enemy.x, enemy.y = g.forecastEnemyPosition(i, enemy, max(0, tick-1))
			if shotHitsActor(moved, enemy) {
				return i, tick, enemy.health <= max(1, moved.power)
			}
		}
		if !visible {
			break
		}
		if moved.impact > 0 {
			// A fresh foot impact restarts its own animation rather than
			// retaining an earlier detached first-slot cleanup.
			cleanup = 0
		} else if cleanup > 0 {
			cleanup--
			if cleanup == 0 {
				break
			}
			moved.frame = g.projectileFrame(moved) + 1
		}
		shot = moved
	}
	return -1, 0, false
}

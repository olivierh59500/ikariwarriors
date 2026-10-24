package game

// $b522/$b6cc probe existing ordinary enemy bullets after horizontal movement,
// before vertical movement and before $a2f4 advances the shots. Tanks neither
// take this hit nor consume the shot, which can reach the other player.
// Player Two also consumes flying grenades; Player One filters their type.
// Each player consumes at most one shot, even while protected.
func (g *Game) hitPlayerWithEnemyBullets(player int) {
	value := g.player
	if player == 1 {
		value = g.player2
	}
	if value.inTank {
		return
	}
	for i, shot := range g.projectiles {
		if shot.enemy && (player == 1 || !shot.grenade && !shot.cannon) && shot.impact == 0 && shotOverlapsActor(shot, value) {
			g.hitPlayer(player)
			g.projectiles = append(g.projectiles[:i], g.projectiles[i+1:]...)
			break
		}
	}
}

// The original bullet keeps moving during its three impact images and can
// still hit an actor. An explicit phase counter preserves that behavior
// without the original remaining-range word underflow.
func (g *Game) advancePlayerBullet(shot projectile) (projectile, bool) {
	wasImpact := shot.impact > 0
	shot.x += shot.vx
	shot.y += shot.vy
	if wasImpact {
		shot.impact--
	} else {
		shot.life--
		if shot.life <= 0 || !(g.ownedBulletUpgrade(shot) || g.projectileFrame(shot) >= 2) && g.solidAt(shot.x+8, shot.y-4+shot.terrainBias) {
			shot.impact = 3
		}
	}
	return shot, !(wasImpact && shot.impact == 0)
}

// Tank shells use the same range counter, but stop immediately in a quad
// explosion. The grenade upgrade ignores actors; the bullet upgrade pierces
// both terrain and actors. These are independent original power-up effects.
func (g *Game) advanceCannon(shot projectile) (projectile, bool) {
	shot.x += shot.vx
	shot.y += shot.vy
	// Boarding can leave a foot bullet's short impact animation in this slot.
	// Its original range underflows instead of expiring; that finite animation
	// still clears it within three updates, without a new range-expiry quad.
	if shot.impact == 0 {
		shot.life--
	}
	return shot, shot.impact == 0 && shot.life <= 0 || !g.ownedBulletUpgrade(shot) && g.solidAt(shot.x+8, shot.y-4+shot.terrainBias)
}

func (g *Game) damageBulletTarget(shot projectile) bool {
	// Ending bodies occupy the first twelve original target slots.
	if g.ending.active && g.shotHitsFinalScene(shot) {
		g.addProjectileScore(shot, 100)
		return true
	}

	for i := range g.enemies {
		enemy := &g.enemies[i]
		if enemy.kind == 4 || enemy.kind == 6 || !shotHitsActor(shot, *enemy) {
			continue
		}
		enemy.health -= max(1, shot.power)
		if enemy.health <= 0 {
			g.audio.Effect(4, 3)
			points := 100
			if enemy.kind == 1 {
				points = 500
			}
			g.addProjectileScore(shot, points)
		}
		return true
	}
	if g.shotHitsFinalScene(shot) {
		g.addProjectileScore(shot, 100)
		return true
	}
	return false
}

func (g *Game) projectileFrame(shot projectile) int {
	if shot.impact > 0 {
		return 2 + 4 - shot.impact
	}
	frame := 1
	if shot.grenade {
		frame = grenadeFlightFrame(shot.life)
		if !shot.enemy && g.grenadePower[shot.owner] > 1 {
			frame += 6
		}
	}
	if shot.cannon {
		frame = 7
	}
	if shot.frame > 0 {
		frame = shot.frame
	}
	return frame
}

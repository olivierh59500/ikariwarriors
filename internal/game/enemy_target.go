package game

// enemyTargetIndex follows the original choice bit and visible-body fallback.
// Both unavailable is outside the original returning domain: the original
// branches between its two unavailable checks. Keep the native guard finite.
func enemyTargetIndex(choice uint32, first, second bool) (int, bool) {
	if choice&1 != 0 && second {
		return 1, true
	}
	if first {
		return 0, true
	}
	if second {
		return 1, true
	}
	return -1, false
}

// Call after live input has restored the player's current body visibility.
// The original always consumes the choice, including single-player play.
func (g *Game) chooseEnemyTarget() (*actor, bool) {
	choice := g.randomValue()
	index, ok := enemyTargetIndex(choice, g.player.health > 0 || g.player.deadTicks > 0 && !g.player.hidden, g.playerCount > 1 && (g.player2.health > 0 || g.player2.deadTicks > 0 && !g.player2.hidden))
	if !ok {
		return nil, false
	}
	if index == 1 {
		return &g.player2, true
	}
	return &g.player, true
}

// The shared shot allocator precedes aiming and both firing gates. Its
// inactive preparation does not consume randomness or activate a projectile.
func (g *Game) updateFixedGun(enemy *actor) {
	if !g.enemyBulletAvailable() {
		return
	}
	target, ok := g.chooseEnemyTarget()
	if !ok {
		return
	}
	g.aimEnemy(enemy, target)
	if g.randomValue()&15 != 0 {
		return
	}
	if enemy.kind == 3 && g.randomValue()&3 != 0 {
		return
	}
	g.fireProjectile(enemy, true, -1, enemy.kind == 3, 4)
	if g.audio != nil {
		g.audio.Effect(0, 2)
	}
}

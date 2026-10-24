package game

// The original counts gameplay frames since scrolling or a damage probe.
// After 600 it requests a diagonal strike at a random point on the screen.
func (g *Game) updateIdleArtillery() {
	if int32(g.idleArtilleryTicks) <= 600 {
		return
	}
	x := float64(64+(g.randomValue()&127)) - 56
	y := g.cameraY + float64(64+(g.randomValue()&127))
	g.fireAirStrike(x, y, 0)
}

// Two marker launchers keep separate locks; idle artillery shares the first.
// The lock is released by its own quad expiry, not merely by removing a shot.
func (g *Game) fireAirStrike(x, y float64, launcher int) bool {
	if g.airStrikeBusy[launcher] || !g.enemyBulletAvailable() {
		return false
	}
	screenY := int(y - g.cameraY)
	rangeY := int(uint8(screenY) & 252)
	life := rangeY / 4
	if life == 0 {
		life = 64
	}
	shot := projectile{y: g.cameraY, vy: 4, life: life, enemy: true, owner: -1,
		grenade: true, fragments: true, airStrike: uint8(launcher + 1)}
	if x+56 < 160 {
		shot.x, shot.vx, shot.frame = x+float64(screenY), -4, 17
	} else {
		shot.x, shot.vx, shot.frame = x-float64(screenY), 4, 18
	}
	g.addEnemyProjectile(shot)
	g.airStrikeBusy[launcher] = true
	return true
}

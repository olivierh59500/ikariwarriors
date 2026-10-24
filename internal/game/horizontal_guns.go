package game

// The four late-level horizontal guns use progress bands and fixed muzzles
// from the original E554 tables. A shot remains at its muzzle for this frame.
var horizontalGuns = [...]struct {
	progress int
	x, y, vx float64
}{
	{11658, 216 - 56, levelHeight - 11678, -4},
	{11734, 104 - 56, levelHeight - 11758, 4},
	{11816, 104 - 56, levelHeight - 11838, 4},
	{11896, 216 - 56, levelHeight - 11920, -4},
}

func (g *Game) updateHorizontalGuns() {
	// E554 scans both logical progress values, including an unentered partner.
	for player := 0; player < 2; player++ {
		value := &g.player
		if player == 1 {
			value = &g.player2
		}
		progress := levelHeight - 16 - int(value.y+g.cooperativeScroll[player].terrainOffset)
		for _, gun := range horizontalGuns {
			if progress <= gun.progress || progress > gun.progress+30 {
				continue
			}
			// E57E consumes the chance before AE14 checks the shared pool.
			// Visibility, protection and falling do not bypass this band scan.
			if g.randomValue()&7 == 0 && g.enemyBulletAvailable() {
				g.addEnemyProjectile(projectile{x: gun.x, y: gun.y, vx: gun.vx,
					enemy: true, owner: -1, life: 28, frame: 1})
			}
			break
		}
	}
}

package game

// Hostile grenades change their saved image at the original remaining-range
// thresholds. Shells and ordinary bullets keep their assigned static image.
func advanceEnemyProjectileImage(shot *projectile) {
	if !shot.grenade || shot.frame <= 28 {
		return
	}
	switch shot.life * 4 {
	case 52, 40, 36, 32, 28:
		shot.frame++
	case 24:
		shot.frame -= 3
	case 20, 12:
		shot.frame--
	}
}

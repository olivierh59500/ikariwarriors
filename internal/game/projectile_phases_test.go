package game

// updateProjectiles isolates the projectile phases in stepPlay's order for
// source fixtures that do not exercise player input or enemy movement. Owned
// quads expire before the shared fragment pass; enemy quads expire after it.
func (g *Game) updateProjectiles() {
	g.advancePlayerProjectiles(0)
	g.advancePlayerProjectiles(1)
	g.advanceBlast(0)
	g.advanceBlast(1)
	g.updateBurst()
	g.advanceEnemyProjectiles()
	g.pruneOffscreenProjectiles()
}

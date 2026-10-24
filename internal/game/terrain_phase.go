package game

// The original switches the collision table for the whole scene when the
// second zone begins. The stage remains selected through retreat and respawn;
// only starting another game restores the first zone's collision table.
func (g *Game) collisionTerrainBank() int {
	if g.secondaryTerrain {
		return 1
	}
	return 0
}

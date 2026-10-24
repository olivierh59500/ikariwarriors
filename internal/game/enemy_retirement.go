package game

// The original wide painter retires live BOBs only after all gameplay passes.
// A body newly crossing the lifetime boundary still finishes this frame's AI.
func (g *Game) pruneOffscreenEnemies() {
	kept := g.enemies[:0]
	for _, enemy := range g.enemies {
		// A death flag can outlive the original BOB's visibility. Preserve
		// existing death carriers here; their hidden/free-slot behavior is a
		// separate qualification from the live-boundary case.
		if g.enemySharedWithEncounter(enemy.slot) || enemy.health <= 0 || g.wideObjectInBounds(enemy.x, enemy.y) {
			kept = append(kept, enemy)
		} else {
			g.rememberInfantryAnimation(enemy)
		}
	}
	g.enemies = kept
}

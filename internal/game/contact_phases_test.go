package game

// These helpers isolate source contact fixtures without player movement. The
// production loop performs each contact inside its player's entered input.
func (g *Game) updateMines() {
	for player := 0; player < g.playerCount; player++ {
		value := g.player
		lives := g.lives
		if player == 1 {
			value = g.player2
			lives = g.lives2
		}
		if value.health > 0 && value.deadTicks == 0 && lives >= 0 {
			g.contactPlayerMine(player)
		}
	}
	g.pruneMines()
}

func (g *Game) updatePickups() {
	for player := 0; player < g.playerCount; player++ {
		value := g.player
		lives := g.lives
		if player == 1 {
			value = g.player2
			lives = g.lives2
		}
		if value.health > 0 && value.deadTicks == 0 && lives >= 0 {
			g.collectPlayerPickup(player)
		}
	}
	g.prunePickups()
}

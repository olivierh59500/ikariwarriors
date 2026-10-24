package game

// Red soldiers drop one of the eight object types selected at $a570. Each
// type owns one active slot, so an existing object of that type is retained.
func (g *Game) dropPickup(enemy actor) {
	value := byte(g.randomValue())
	kind := 0
	if value <= 150 {
		kind = int(g.randomValue() & 3)
	} else if value <= 200 {
		kind = 4 + int(g.randomValue()&1)
	} else {
		kind = 7 - int(g.randomValue()&1)
	}
	g.addPickup(pickup{x: enemy.x - 8, y: enemy.y + 4, kind: kind})
}

package game

// Scenery completion selects only the four basic items. The random choice
// occurs even when an existing item already owns that type's single slot.
func (g *Game) dropSceneryPickup(position navPoint) {
	kind := int(g.randomValue() & 3)
	g.addPickup(pickup{x: position.x, y: position.y, kind: kind})
}

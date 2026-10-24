package game

// Each original pickup type has a separate retained animation phase. Its
// constructor restores the icon without resetting the phase or delay; only
// active pickups advance. The fuel type alternates its GAS label with its icon.
var pickupIconFrames = [...]int{20, 21, 23, 22, 25, 27, 26, 28}

var pickupAnimationFrames = [8][]int{
	{19, 20}, {19, 21}, {24, 23}, {19, 22},
	{19, 25}, {19, 27}, {19, 26}, {19, 28},
}

func (g *Game) addPickup(item pickup) {
	for _, existing := range g.pickups {
		if existing.kind == item.kind {
			return
		}
	}
	if item.bob == 0 && item.kind >= 0 && item.kind < len(pickupIconFrames) {
		g.pickupAnimations[item.kind].frame = pickupIconFrames[item.kind]
	}
	g.pickups = append(g.pickups, item)
}

// Pickup animation follows the shared hostile quad and precedes mines. A
// clipped pickup still animates before the late painter retires its slot.
func (g *Game) animatePickups() {
	var active [8]bool
	for _, item := range g.pickups {
		if item.bob == 0 && item.kind >= 0 && item.kind < len(active) {
			active[item.kind] = true
		}
	}
	for kind, present := range active {
		if present {
			g.pickupAnimations[kind].advance(pickupAnimationFrames[kind])
		}
	}
}

func (g *Game) pickupImage(kind int) int {
	if frame := g.pickupAnimations[kind].frame; frame != 0 {
		return frame
	}
	// Isolated callers may supply an unanimated icon directly.
	return pickupIconFrames[kind]
}

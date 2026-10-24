package game

import "github.com/hajimehoshi/ebiten/v2"

// The final six small-object slots are proximity hazards, not collectible
// bonuses. Types 18/19 explode on contact; type 20 marks an air-strike target.
type landmine struct {
	x, y float64
	kind int
	slot int
}

func mineTouchesActor(mine landmine, value actor) bool {
	return smallObjectTouchesActor(mine.x, mine.y, value)
}

func (g *Game) pruneMines() {
	kept := g.mines[:0]
	for _, mine := range g.mines {
		if !g.smallObjectInBounds(mine.x, mine.y) {
			continue
		}
		kept = append(kept, mine)
	}
	g.mines = kept
}

// $bef6/$bf1c scan one entered player's input between horizontal and vertical
// movement. They stop at the first contacting slot, including a busy marker.
// A hit earlier in this same input does not clear the actor's activity flag.
func (g *Game) contactPlayerMine(player int) {
	value := g.player
	if player == 1 {
		value = g.player2
	}
	for slot := 81; slot <= 86; slot++ {
		for i, mine := range g.mines {
			if mine.slot != slot || !smallObjectOverlapsActor(mine.x, mine.y, value) {
				continue
			}
			if mine.kind == 20 {
				launcher := 0
				if mine.slot == 86 {
					launcher = 1
				}
				g.fireAirStrike(mine.x, mine.y, launcher)
			} else {
				g.mines = append(g.mines[:i], g.mines[i+1:]...)
				g.createBurst(mine.x, mine.y, -1)
			}
			return
		}
	}
}

func (g *Game) drawMines() {
	for slot := 81; slot <= 86; slot++ {
		for _, mine := range g.mines {
			if mine.slot != slot {
				continue
			}
			sprite := g.resources.mine
			if frame := g.mineAnimations[slot-81].frame; frame >= 0 && frame < len(g.resources.effects) {
				sprite = g.resources.effects[frame]
			}
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(mine.x-8, mine.y-g.cameraY-8)
			g.playfield.DrawImage(sprite, op)
		}
	}
}

// Small objects use a twelve-by-twelve rectangle against the player's
// twenty-three-by-nineteen rectangle, with distinct graphic origins.
func smallObjectTouchesActor(x, y float64, value actor) bool {
	return value.health > 0 && value.deadTicks == 0 && smallObjectOverlapsActor(x, y, value)
}

func smallObjectOverlapsActor(x, y float64, value actor) bool {
	dx, dy := x-value.x, y-value.y
	return dx >= -20 && dx <= 15 && dy >= -8 && dy <= 23
}

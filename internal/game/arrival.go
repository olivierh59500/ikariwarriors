package game

import "github.com/hajimehoshi/ebiten/v2"

// arrivalSequence follows the original forced walk from the landing scene.
// Screen positions remain fixed once scrolling takes over. The two-pixel
// handover at the end exposes the first ordinary gameplay frame.
type arrivalSequence struct {
	tick, scroll int
	paintScroll  int
	players      int
	y            int
	secondY      int
	cooperative  [2]cooperativeScrollState
	upper, lower [2]walkAnimation
}

func newArrival(players int) arrivalSequence {
	a := arrivalSequence{players: players, y: 200, secondY: 200}
	for i := range players {
		// The title's completed wave leaves its phase in the walking counters.
		a.upper[i] = walkAnimation{step: 7, delay: 1, frame: 8}
		a.lower[i] = walkAnimation{frame: 24}
	}
	return a
}

func (a *arrivalSequence) advance() bool {
	if a.tick == 116 {
		return true
	}
	a.tick++
	if a.players == 1 {
		a.y = max(98, 200-a.tick*2)
		a.scroll = max(0, (a.tick-51)*2)
	} else {
		positions := [2]*int{&a.y, &a.secondY}
		for player := 0; player < 2; player++ {
			state := &a.cooperative[player]
			state.up = true
			state.claimed = false
			if a.cooperative[1-player].claimed {
				continue
			}
			if *positions[player] >= 100 {
				*positions[player] -= 2
			} else {
				state.claimed = true
				a.scroll += 2
			}
		}
	}
	// The final handover scroll happens after the original landing painter.
	// Keep that completed picture until the first ordinary gameplay update.
	a.paintScroll = a.scroll
	if a.tick == 116 {
		a.scroll = 132
	}
	for i := range a.players {
		advancePlayerWalk(&a.upper[i], &a.lower[i], i, 0, false, false)
	}
	return a.tick == 116
}

func (g *Game) drawArrival(screen *ebiten.Image) {
	a := &g.arrival
	g.drawBackdrop(screen, float64(levelHeight-a.paintScroll))
	for player := range a.players {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(120+60*player-playfieldX), float64(a.playerY(player)-playfieldY))
		value := actor{upper: a.upper[player], lower: a.lower[player]}
		upper, lower := value.playerArtOffsets(player)
		if sprite := g.resources.bob(upper, actorBobHeight); sprite != nil {
			g.playfield.DrawImage(sprite, op)
		}
		op.GeoM.Translate(0, actorBobHeight)
		if sprite := g.resources.bob(lower, overlayBobHeight); sprite != nil {
			g.playfield.DrawImage(sprite, op)
		}
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(playfieldX, playfieldY)
	screen.DrawImage(g.playfield, op)
	g.drawHUD(screen)
}

func (a *arrivalSequence) playerY(player int) int {
	if player == 1 {
		return a.secondY
	}
	return a.y
}

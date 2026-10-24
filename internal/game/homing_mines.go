package game

// Water mines use the same two-image cadence as the original small objects.
// Their animation phases survive inactivity; spawning replaces only the image.
var mineFrameSequences = [3][]int{{10, 11}, {12, 13}, {14, 15}}

func (g *Game) animateMines() {
	nearAny := false
	for slot := 81; slot <= 86; slot++ {
		for _, mine := range g.mines {
			if mine.slot != slot {
				continue
			}
			near := g.mineNearPlayer(mine, g.player, g.lives >= 0)
			if g.playerCount > 1 {
				near = near || g.mineNearPlayer(mine, g.player2, g.lives2 >= 0)
			}
			nearAny = nearAny || near
			phase := &g.mineAnimations[slot-81]
			if mine.kind == 19 || near {
				phase.advance(mineFrameSequences[(slot-81)/2])
			} else {
				phase.frame = -1
			}
		}
	}
	if nearAny {
		g.mineAlarmTicks--
		if g.mineAlarmTicks == 0 {
			g.mineAlarmTicks = 4
			if g.audio != nil {
				g.audio.Effect(17, 3)
			}
		}
	}
}

func (g *Game) mineNearPlayer(mine landmine, player actor, active bool) bool {
	if !active {
		return false
	}
	dx, dy := mine.x-player.x, mine.y-player.y
	return dx >= -45 && dx <= 29 && dy >= -33 && dy <= 41
}

// Both original mine slots consume their target choice after a shared random
// movement gate, including an inactive slot. Inactive positions are replaced
// when that slot spawns again, so only active mines need stored coordinates.
func (g *Game) moveHomingMines(random func() uint32) {
	if random()&7 != 0 {
		return
	}
	for slot := 83; slot <= 84; slot++ {
		choice := random()
		target := g.player
		if (choice&1 != 0 || g.lives < 0) && g.playerCount > 1 && g.lives2 >= 0 {
			target = g.player2
		}
		for index := range g.mines {
			mine := &g.mines[index]
			if mine.slot != slot {
				continue
			}
			*mine = g.homingMineStep(*mine, target)
		}
	}
}

func (g *Game) homingMineStep(mine landmine, target actor) landmine {
	dx, dy := 0.0, 0.0
	if mine.x < target.x-24 {
		dx = 4
	} else if mine.x > target.x+8 {
		dx = -4
	}
	if mine.y < target.y-12 {
		dy = 4
	} else if mine.y > target.y+20 {
		dy = -4
	}
	if dx == 0 && dy == 0 {
		dy = 4
	}
	x, y := mine.x+dx, mine.y+dy
	code := g.terrainCode(x+8, y-4)
	if code == 157 || code == 158 {
		mine.x, mine.y = x, y
	}
	return mine
}

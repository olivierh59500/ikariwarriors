package game

import "math"

// Each original input routine moves horizontally, handles its weapon and
// existing contacts, then moves vertically. A hit starts a fall but does not
// cancel the vertical command already accepted for this same frame.
func (g *Game) stepPlayer(player int, control playerControls) {
	value, lives, ammunition, grenades := &g.player, &g.lives, &g.ammunition, &g.grenades
	if player == 1 {
		value, lives, ammunition, grenades = &g.player2, &g.lives2, &g.ammunition2, &g.grenades2
	}
	if value.deadTicks > 0 || value.health <= 0 || *lives < 0 {
		return
	}
	// A living input restores the body before later enemy target selection.
	value.hidden = false
	dx, dy := joystickAxes(control.x, control.y)
	g.beginCooperativeDirection(player, dx, dy)
	if g.playerCount > 1 {
		state := &g.cooperativeScroll[player]
		state.up, state.down = dy < 0, dy > 0
		if dy >= 0 {
			state.claimed = false
		}
	}
	if (dx != 0 || dy != 0) && !g.directionLock[player] {
		// Input delays follow the shared latch, including after reappearance.
		value.direction = directionForVector(dx, dy)
	}
	previousX := value.x
	g.moveActorStep(value, dx*2, 0)
	horizontalAnimation := previousX != value.x
	if horizontalAnimation {
		value.animateCooperativeMovement(player, value.x-previousX, dy*2, g.directionLock[player])
	}
	action := control.action
	if action && g.useTank(player) {
		action = false
	}
	if control.fire && *ammunition > 0 && g.bulletAvailable(player) {
		g.fireProjectile(value, false, player, false, 6)
		g.audio.Effect(0, 3)
		*ammunition--
	}
	if action && *grenades > 0 && g.grenadeAvailable(player) {
		g.fireProjectile(value, false, player, true, 4)
		g.audio.Effect(7, 3)
		*grenades--
	}
	g.hitPlayerWithEnemyBullets(player)
	g.hitPlayerWithInfantry(player)
	g.collectPlayerPickup(player)
	g.contactPlayerMine(player)
	if g.playerCount > 1 {
		if dy != 0 && g.moveCooperativeVertically(player, dy) && !horizontalAnimation {
			value.animateCooperativeMovement(player, 0, dy*2, g.directionLock[player])
		}
	} else {
		previousY := value.y
		g.moveActorStep(value, 0, dy*2)
		if previousY != value.y && !horizontalAnimation {
			value.animateCooperativeMovement(player, 0, value.y-previousY, g.directionLock[player])
		}
		g.followPlayer(previousY, value.y)
	}
	if player == 1 {
		value.y = min(value.y, g.cameraY+playfieldHeight-10)
	}
}

// The player sprite remains active until its input routine returns, even if
// an earlier bullet started the fall. Every overlapping living soldier is
// consumed in slot order; protection only prevents another player hit.
func (g *Game) hitPlayerWithInfantry(player int) {
	value := &g.player
	if player == 1 {
		value = &g.player2
	}
	for slot := 0; slot <= 18; slot++ {
		if g.ending.active && slot < 12 {
			part := g.endingPart(slot)
			if g.endingPartCanBeHit(slot) && math.Abs(value.x-part.x) <= 23 && math.Abs(value.y-part.y) <= 19 {
				g.ending.hitParts |= 1 << slot
				if player == 0 {
					g.score += 100
				} else {
					g.score2 += 100
				}
				if !value.inTank {
					g.hitPlayer(player)
				}
			}
			continue
		}
		for i := range g.enemies {
			enemy := &g.enemies[i]
			if enemy.slot != slot || enemy.kind == 4 || enemy.kind == 6 || enemy.health <= 0 || enemy.deadTicks > 0 || g.final.active && slot >= 5 {
				continue
			}
			if math.Abs(value.x-enemy.x) > 23 || math.Abs(value.y-enemy.y) > 19 {
				continue
			}
			enemy.health = 0
			if player == 0 {
				g.score += 100
			} else {
				g.score2 += 100
			}
			if !value.inTank {
				g.hitPlayer(player)
			}
		}
	}
}

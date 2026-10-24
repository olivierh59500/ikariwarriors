package game

// Each player's upward scroll claim is retained until their next input. The
// second player therefore sees the first player's current input, while the
// first player sees the second player's previous input.
type cooperativeScrollState struct {
	up, down, claimed bool
	// A departed partner keeps a screen position after its terrain progress
	// returns to the level entrance. This offset preserves both coordinates.
	terrainOffset float64
}

func (g *Game) cooperativeProbe(owner int, value *actor, dy float64) bool {
	y := value.y - 8
	if dy > 0 {
		y = value.y + 4
	}
	if value == &g.player {
		y += g.cooperativeScroll[0].terrainOffset
	} else if value == &g.player2 {
		y += g.cooperativeScroll[1].terrainOffset
	}
	code := g.terrainCode(value.x, y)
	value.wet = code == 157 || code == 158
	if value.inTank && tankWaterCode(code) {
		return false
	}
	if g.damageTerrain(value.x, y, owner) {
		g.hitPlayer(owner)
		return false
	}
	return g.resources.passable[g.collisionTerrainBank()][code]
}

// World coordinates express the original shared scrolling directly; no
// display registers or machine instructions are part of this movement model.
func (g *Game) moveCooperativeVertically(player int, dy float64) bool {
	own, peer := &g.player, &g.player2
	peerLives := g.lives2
	if player == 1 {
		own, peer = peer, own
		peerLives = g.lives
	}
	state, other := &g.cooperativeScroll[player], &g.cooperativeScroll[1-player]
	raw, peerRaw := own.y-g.cameraY+4, peer.y-g.cameraY+4
	peerActive := peerLives >= 0 && peer.health > 0 && peer.deadTicks == 0
	if dy < 0 {
		state.claimed = false
	}
	if !g.cooperativeProbe(player, own, dy) {
		if dy > 0 {
			state.down = false
		}
		return false
	}
	speed := 2.0
	if own.wet {
		speed = 1
	}
	if dy < 0 {
		if other.claimed {
			return true
		}
		if raw >= 100 {
			own.y -= speed
			return true
		}
		state.claimed = true
		drag := 0.0
		if other.up && g.cooperativeProbe(player, peer, -1) {
			drag = 2
			if peer.wet {
				if peerRaw >= 180 {
					state.claimed = false
					return false
				}
				drag = 1
			}
		} else if peerActive && peerRaw >= 175 {
			return false
		}
		previous := own.y
		own.y -= speed
		peer.y -= drag
		g.followPlayer(previous, own.y)
		return true
	}
	if raw <= 160 {
		state.down = false
		own.y += speed
		return true
	}
	room := g.cameraY < g.cameraBackY
	scroll := room
	drag := 0.0
	if !other.down {
		if peerActive && peerRaw <= 100 {
			scroll = false
		}
		if scroll && player == 1 && peer.wet {
			drag = -1
		}
	} else {
		drag = 2
		if peer.wet {
			drag = 1
		}
	}
	if !scroll {
		if raw >= 175 {
			// The second player's original lower-boundary branch releases the
			// first player's downward request, retaining its own request.
			if player == 1 {
				other.down = false
			} else {
				state.down = false
			}
			return false
		}
		own.y += speed
	} else {
		previous := own.y
		own.y += speed
		peer.y += drag
		g.followPlayer(previous, own.y)
	}
	if !own.inTank || !own.tankBurning {
		state.down = false
	}
	return true
}

// The original direction latch also resets delays in a one-player game. Both
// input routines update that first latch. The second routine compares with
// its separate zero-initialized latch, so a moving
// second player refreshes the image delays on each entered input.
func (g *Game) beginCooperativeDirection(player int, dx, dy float64) {
	var mask uint8
	if dy < 0 {
		mask |= 1
	} else if dy > 0 {
		mask |= 2
	}
	if dx < 0 {
		mask |= 4
	} else if dx > 0 {
		mask |= 8
	}
	if mask == 0 {
		return
	}
	value := &g.player
	if player == 1 {
		value = &g.player2
	}
	if player == 1 || g.cooperativeDirection != mask {
		value.upper.delay, value.lower.delay = 0, 0
	}
	g.cooperativeDirection = mask
}

func (a *actor) animateCooperativeMovement(player int, dx, dy float64, locked bool) {
	// Delay resets were decided by the shared input latch, rather than by
	// an independent remembered direction in each rendered actor.
	a.moveDirection = directionForVector(dx, dy)
	a.animatePlayerMovement(player, dx, dy, locked)
}

package game

import "math"

// Each late launcher guards the bottom of an explosive composite. A player
// who knows this field can throw from below the fan's complete bullet range,
// wait for the real explosion, and approach the next composite in order.
type demoFanObjective struct {
	index, row, column int
	anchor             navPoint
}

func (g *Game) demoLateFanObjective() (demoFanObjective, bool) {
	if g.player.y > g.fanTraps[1].y+96 {
		return demoFanObjective{}, false
	}
	for index := 1; index < len(g.fanTraps); index++ {
		trap := g.fanTraps[index]
		if !trap.active || g.player.y < trap.y-40 {
			continue
		}
		row, column, _, intact := g.sceneryBounds(trap.x, trap.y)
		if !intact {
			continue
		}
		// A fan starts at y+4 and advances at most fourteen times before
		// its last player-contact pass: y+60. Turning up from this line
		// still leaves twelve pixels beyond the projectile's lower reach.
		return demoFanObjective{index, row, column, navPoint{trap.x, trap.y + 76}}, true
	}
	return demoFanObjective{}, false
}

func (g *Game) demoLateFanExplosionHold() (navPoint, bool) {
	if !g.sceneryBurst.active && !g.burst.active && len(g.burstQueue) == 0 {
		return navPoint{}, false
	}
	for index := 1; index < len(g.fanTraps); index++ {
		trap := g.fanTraps[index]
		anchor := navPoint{trap.x, trap.y + 76}
		if trap.active || math.Abs(g.player.x-anchor.x) > 24 || math.Abs(g.player.y-anchor.y) > 24 {
			continue
		}
		near := func(x, y float64) bool {
			return math.Abs(x-trap.x) <= 48 && math.Abs(y-trap.y) <= 48
		}
		if g.sceneryBurst.active && near(g.sceneryBurst.center.x, g.sceneryBurst.center.y) {
			return anchor, true
		}
		if g.burst.active {
			var center navPoint
			for _, part := range g.burst.parts {
				center.x += part.x / 8
				center.y += part.y / 8
			}
			if near(center.x, center.y) {
				return anchor, true
			}
		}
		for _, request := range g.burstQueue {
			if near(request.x, request.y) {
				return anchor, true
			}
		}
	}
	return navPoint{}, false
}

// demoLateFanControls uses ordinary finite-resource inputs. It is intended
// after the shared visible-state forecast and normal boundary recovery. The
// opening isolated launcher keeps its existing route; this handles the four
// consecutive late composites using their recovered map and launch pattern.
func (g *Game) demoLateFanControls() (playerControls, bool) {
	if g.player.inTank || g.player.deadTicks > 0 || g.player.health <= 0 || g.ending.active {
		return playerControls{}, false
	}
	hold, holding := g.demoLateFanExplosionHold()
	objective, found := g.demoLateFanObjective()
	if !holding && (!found || g.grenades == 0) {
		return playerControls{}, false
	}
	prepared := g.demoForecastActive
	if !prepared {
		g.prepareDemoForecast()
		defer func() { g.demoForecastActive = false }()
	}
	g.directionLock[0] = false
	fireHeld := g.demoFireHeld
	g.demoFireHeld = false
	// A previous lateral dodge can leave the ordinary path behind the
	// player. Resume normal routing from the real final position afterward.
	g.demoPath = g.demoPath[:0]
	g.demoPlanScenery = false
	target := objective.anchor
	if holding {
		target = hold
	}
	// Scrolling back naturally resets the original idle-artillery timer.
	// It avoids waiting on a firing line after an already prolonged stall.
	if g.idleArtilleryTicks >= 580 && g.cameraY < g.cameraBackY {
		target.y = max(target.y, g.cameraY+159)
	}
	move, score := g.demoChooseMove(target)
	if !holding && math.Abs(g.player.x-objective.anchor.x) <= 2 && math.Abs(g.player.y-objective.anchor.y) <= 2 && g.grenadeAvailable(0) {
		attack := playerControls{action: true}
		if g.player.direction != 0 {
			attack.y = -1
		}
		source := g.player
		source.direction = 0
		shot := g.projectileFor(&source, false, 0, true, 4)
		landing, _, lands := g.demoShotLanding(shot)
		row, column, _, intact := g.sceneryBounds(landing.x+8, landing.y-4)
		if lands && intact && row == objective.row && column == objective.column {
			_, quality := g.demoPlanAttack(objective.anchor, &attack, 0)
			if quality > demoBlockedScore && quality >= score-50 {
				return attack, true
			}
		}
	}
	// Ordinary infantry can still be fought while approaching or holding
	// the safe line. A deep retreat first gets out of the firing lanes.
	if !fireHeld && g.ammunition > 0 && g.bulletAvailable(0) && g.player.y >= target.y-4 {
		if aim, canHit := g.demoAimForMove(move); canHit {
			attack := playerControls{fire: true}
			locked := g.player.direction == aim
			if locked {
				attack = move
				attack.fire = true
			} else {
				attack.x, attack.y = directionVector(aim)
			}
			_, quality := g.demoPlanAttack(target, &attack, aim)
			if quality > demoBlockedScore && quality >= score-50 {
				g.directionLock[0] = locked
				g.demoFireHeld = true
				return attack, true
			}
		}
	}
	return move, true
}

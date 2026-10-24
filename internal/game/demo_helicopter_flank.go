package game

import "math"

// A memorized left approach keeps the moving helicopter's muzzle on the far
// side of the player. The actual throw and its complete danger forecast still
// decide whether an ordinary finite grenade can be used safely.
func (g *Game) demoFlyingAircraftFlank() (playerControls, bool) {
	h := g.aircraft
	if g.player.inTank || g.player.health <= 0 || g.player.deadTicks > 0 || g.ending.active || g.player.y > 9300 || g.player.y < 8150 {
		return playerControls{}, false
	}
	if !h.flying || h.x != 152 && h.x != 102 || !h.alive[0] || !h.alive[1] || h.damaged&1 != 0 {
		// These are fixed original scroll events. Reach the left approach
		// before the plane appears, instead of crossing its firing cone late.
		for _, known := range [...][3]float64{{6000, 80, 9300}, {6710, 30, 8590}} {
			triggerCamera := levelHeight - known[0]
			if g.cameraY < triggerCamera || g.player.y > known[2] || g.player.y < triggerCamera+94 {
				continue
			}
			if !g.demoAircraftEventPending(int(known[0])) {
				continue
			}
			// Clear the nearer corridor gun before preparing for the later
			// aircraft. Its existing guide requires the dry approach above it.
			if known[0] == 6000 {
				for _, enemy := range g.enemies {
					if enemy.kind == 2 && enemy.health > 0 && enemy.deadTicks == 0 && enemy.x == 112 && enemy.y == 9190 {
						return playerControls{}, false
					}
				}
			}
			prepared := g.demoForecastActive
			if !prepared {
				g.prepareDemoForecast()
				defer func() { g.demoForecastActive = false }()
			}
			target := navPoint{known[1], g.player.y - 16}
			if known[0] == 6710 {
				// The left barrier ends at this gap; the next barrier closes
				// its middle. Cross left before committing to the upper lane.
				if g.player.y > 8470 {
					target = navPoint{104, 8464}
				} else if g.player.x > 32 && g.player.y > 8400 {
					target = navPoint{30, 8464}
				}
			}
			move, _ := g.demoApproachMove(target)
			return move, true
		}
		return playerControls{}, false
	}
	separation := g.player.y - h.y
	if separation < -16 || separation > 224 {
		return playerControls{}, false
	}
	prepared := g.demoForecastActive
	if !prepared {
		g.prepareDemoForecast()
		defer func() { g.demoForecastActive = false }()
	}
	// Keep eight extra pixels beyond the fragment contact edge. A required
	// northeast turn moves horizontally before its grenade is constructed.
	flank := navPoint{max(14, h.x-72), g.player.y}
	move, normal := g.demoApproachMove(flank)
	if math.Abs(g.player.x-flank.x) > 2 || g.grenades == 0 || !g.grenadeAvailable(0) {
		return move, true
	}
	attack := playerControls{action: true}
	const aim = 1
	if g.player.direction != aim {
		attack.x, attack.y = directionVector(aim)
	}
	source := g.player
	if x, _, ok := g.demoStep(source.x, source.y, attack.x, 0); ok {
		source.x = x
	}
	source.direction = aim
	landing, _, lands := g.demoShotLanding(g.projectileFor(&source, false, 0, true, 4))
	head := h.part(0)
	head.y += 34
	if lands && grenadeHitsActor(landing, head) {
		_, quality := g.demoPlanAttack(flank, &attack, aim)
		if quality > demoBlockedScore && quality >= normal-50 {
			g.directionLock[0], g.demoFireHeld = false, false
			return attack, true
		}
	}
	return move, true
}

// Backscroll and respawns do not replay consumed original scroll events.
func (g *Game) demoAircraftEventPending(progress int) bool {
	for index, event := range g.resources.events {
		if event.kind == 4 && event.progress == progress {
			return index >= g.eventIndex
		}
	}
	return false
}

// Staging does not suspend ordinary combat. Use the same current-target aim,
// finite owned-shot slots and safe movement comparison as the general player.
func (g *Game) demoApproachMove(target navPoint) (playerControls, float64) {
	// The remembered detour can pass many nodes of the general route. Replan
	// from the real post-flank position when ordinary navigation resumes.
	g.demoPath = g.demoPath[:0]
	g.demoPlanScenery = false
	move, normal := g.demoChooseMove(target)
	g.directionLock[0] = false
	if !g.demoFireHeld && g.ammunition > 0 && g.bulletAvailable(0) {
		if aim, ok := g.demoAimForMove(move); ok {
			dx, dy := directionVector(aim)
			attack := playerControls{x: dx, y: dy, fire: true}
			locked := g.player.direction == aim
			if locked {
				attack = move
				attack.fire = true
			}
			_, quality := g.demoPlanAttack(target, &attack, aim)
			if quality > demoBlockedScore && quality >= normal-50 {
				g.directionLock[0] = locked
				move = attack
			}
		}
	}
	g.demoFireHeld = move.fire
	return move, normal
}

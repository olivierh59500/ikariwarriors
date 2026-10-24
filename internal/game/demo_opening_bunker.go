package game

import "math"

// An experienced player destroys the opening mounted gun's bunker from dry
// ground below its bullet reach, then waits below the scenery fragments.
func (g *Game) demoOpeningBunker() (playerControls, bool) {
	if g.player.inTank || g.player.health <= 0 || g.player.deadTicks > 0 || g.player.y > 15000 || g.player.y < 14770 || g.ending.active {
		return playerControls{}, false
	}
	alive := false
	for _, enemy := range g.enemies {
		if enemy.slot == 32 && enemy.kind == 6 && enemy.health > 0 && enemy.deadTicks == 0 && enemy.x == 94 && enemy.y == 14846 {
			alive = true
			break
		}
	}
	pending := g.demoOpeningBunkerPending()
	if !pending && (!alive || g.grenades == 0) {
		return playerControls{}, false
	}
	g.directionLock[0] = false
	g.demoPath = g.demoPath[:0]
	g.demoPlanScenery = false
	stage, retreat := navPoint{104, 14912}, navPoint{104, 14928}
	// An earlier wet move can change the walking lattice's parity.
	stage.x += math.Mod(g.player.x-stage.x+2, 2)
	stage.y += math.Mod(g.player.y-stage.y+2, 2)
	target := stage
	if pending {
		target = retreat
	}
	move, score := g.demoApproachMove(target)
	if !pending && math.Abs(g.player.x-stage.x) <= 2 && math.Abs(g.player.y-stage.y) <= 2 && g.grenadeAvailable(0) {
		attack := playerControls{action: true}
		if g.player.direction != 0 {
			attack.y = -1
		}
		source := g.player
		source.direction = 0
		shot := g.projectileFor(&source, false, 0, true, 4)
		landing, _, lands := g.demoShotLanding(shot)
		if lands {
			row, column, _, hit := g.sceneryBounds(landing.x+8, landing.y-4)
			if hit && row == 926 && column == 3 {
				_, quality := g.demoPlanAttack(retreat, &attack, 0)
				if quality > demoBlockedScore && quality >= demoHorizon*10000-1000 && quality >= score-1000 {
					// Expose the actual selected scenery throw to offline adapters.
					// The approach itself must retain the safe staging destination.
					g.directionLock[0], g.demoFireHeld = false, false
					g.demoPlanScenery = true
					g.demoSceneryRow, g.demoSceneryCol = row, column
					return attack, true
				}
			}
		}
	}
	return move, true
}

func (g *Game) demoOpeningBunkerPending() bool {
	near := func(x, y float64) bool { return math.Abs(x-84) <= 24 && math.Abs(y-14840) <= 24 }
	for _, shot := range g.demoShots() {
		if shot.enemy || !shot.grenade {
			continue
		}
		if shot.impact > 0 && near(shot.x, shot.y) {
			return true
		}
		if landing, _, ok := g.demoShotLanding(shot); ok && near(landing.x, landing.y) {
			return true
		}
	}
	if g.sceneryBurst.active && near(g.sceneryBurst.center.x, g.sceneryBurst.center.y) {
		return true
	}
	if g.burst.active {
		var center navPoint
		for _, part := range g.burst.parts {
			center.x += part.x / 8
			center.y += part.y / 8
		}
		if near(center.x, center.y) {
			return true
		}
	}
	for _, burst := range g.burstQueue {
		if near(burst.x, burst.y) {
			return true
		}
	}
	return false
}

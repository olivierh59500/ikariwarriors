package game

import "math"

// The factory entrance has two fixed guns facing its narrow approach. A
// player who remembers them can clear each from dry ground before advancing.
// These positions describe the recovered map, never a future random result.
func (g *Game) demoFactoryGuns() (playerControls, bool) {
	if g.player.inTank || g.player.health <= 0 || g.player.deadTicks > 0 || g.player.y > 5160 || g.player.y < 4930 || g.ending.active {
		return playerControls{}, false
	}
	gun := actor{}
	stage, retreat := navPoint{}, navPoint{}
	aim := 0
	found := false
	for _, known := range [...]struct {
		x, anchor float64
		aim       int
	}{{192, 136, 1}, {22, 104, 7}} {
		for _, enemy := range g.enemies {
			if enemy.kind == 2 && enemy.health > 0 && enemy.deadTicks == 0 && enemy.x == known.x && enemy.y == 4990 {
				gun, stage, aim, found = enemy, navPoint{known.anchor, 5060}, known.aim, true
				break
			}
		}
		if found {
			break
		}
	}
	holding := g.demoFactoryExplosionPending()
	if !holding && (!found || g.grenades == 0) {
		return playerControls{}, false
	}
	prepared := g.demoForecastActive
	if !prepared {
		g.prepareDemoForecast()
		defer func() { g.demoForecastActive = false }()
	}
	g.directionLock[0], g.demoFireHeld = false, false
	g.demoPath = g.demoPath[:0]
	g.demoPlanScenery = false
	retreat = navPoint{g.player.x, 5080}
	target := stage
	if holding {
		target = retreat
	}
	move, score := g.demoChooseMove(target)
	if !holding && math.Abs(g.player.x-stage.x) <= 2 && math.Abs(g.player.y-stage.y) <= 2 && g.grenadeAvailable(0) {
		attack := playerControls{action: true}
		if g.player.direction != aim {
			attack.x, attack.y = directionVector(aim)
		}
		// Horizontal movement precedes the ordinary throw; vertical movement
		// follows it. Check that exact muzzle, including a necessary turn.
		source := g.player
		if x, _, ok := g.demoStep(source.x, source.y, attack.x, 0); ok {
			source.x = x
		}
		source.direction = aim
		shot := g.projectileFor(&source, false, 0, true, 4)
		landing, _, lands := g.demoShotLanding(shot)
		if lands && grenadeHitsActor(landing, gun) {
			_, quality := g.demoPlanAttack(retreat, &attack, aim)
			// Allow the short retreat while requiring a complete safe plan.
			if quality >= demoHorizon*10000-1000 && quality >= score-1000 {
				return attack, true
			}
		}
	}
	return move, true
}

// An upgraded grenade emits hostile fragments after its landing quad. Retire
// below their actual reach until the visible flight, quad and burst are over.
func (g *Game) demoFactoryExplosionPending() bool {
	near := func(x, y float64) bool { return x >= -16 && x <= 224 && math.Abs(y-4992) <= 24 }
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
	for _, request := range g.burstQueue {
		if near(request.x, request.y) {
			return true
		}
	}
	return false
}

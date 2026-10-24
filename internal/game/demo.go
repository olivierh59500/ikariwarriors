package game

import "math"

const demoHorizon = 24
const demoBeamWidth = 64
const demoBlockedScore = -1e9

type demoBulletPrediction struct {
	x, y   float64
	active bool
}

type demoBurstPrediction struct {
	x, y      float64
	firstMove int
}

type demoSceneryPrediction struct {
	burst   sceneryBurst
	landing int
}

// Movement candidates share one future scene. Compute it once per decision,
// rather than simulating the same soldier and bullet for every search node.
func (g *Game) prepareDemoForecast() {
	g.demoEnemyDefeat = [64]int{}
	for i, enemy := range g.enemies {
		if i >= len(g.demoEnemyForecast[0]) {
			break
		}
		if enemy.health <= 0 || enemy.deadTicks > 0 {
			continue
		}
		for tick := range demoHorizon + 1 {
			x, y := g.demoEnemyPosition(enemy, tick)
			g.demoEnemyForecast[tick][i] = navPoint{x, y}
		}
	}
	for i, shot := range g.demoShots() {
		if i >= len(g.demoBulletForecast[0]) {
			break
		}
		if !shot.enemy || shot.grenade || shot.impact > 0 {
			continue
		}
		for tick := range demoHorizon + 1 {
			future, active := g.demoBulletAt(shot, tick)
			g.demoBulletForecast[tick][i] = demoBulletPrediction{future.x, future.y, active}
		}
	}
	g.demoForecastActive = true
	g.prepareTargetBursts()
}

func (g *Game) addTargetBurst(x, y float64, firstMove int) {
	if g.demoBurstCount < len(g.demoTargetBursts) {
		g.demoTargetBursts[g.demoBurstCount] = demoBurstPrediction{x, y, firstMove}
		g.demoBurstCount++
	}
}

// Destroying a vehicle or aircraft can be more dangerous than the grenade
// itself. Forecast that ordinary consequence while the shot is still flying.
func (g *Game) prepareTargetBursts() {
	g.demoBurstCount = 0
	g.demoSceneryCount = 0
	for _, enemy := range g.enemies {
		if enemy.kind != 4 {
			continue
		}
		if enemy.health <= 0 {
			g.addTargetBurst(enemy.x, enemy.y+7, 2)
		} else if enemy.tankBurning {
			g.addTargetBurst(enemy.x-8, enemy.y+4, enemy.tankBurn+2)
		}
	}
	for _, i := range [...]int{0, 3} {
		if g.aircraft.alive[i] && g.aircraft.damaged&(1<<i) != 0 {
			part := g.aircraft.part(i)
			g.addTargetBurst(part.x, part.y+7, 2)
		}
		if !g.boss.active && g.boss.alive[i] && g.boss.damaged&(1<<i) != 0 {
			part := g.boss.part(i)
			g.addTargetBurst(part.x, part.y+7, 2)
		}
	}
	for group, vehicle := range g.enemyVehicles {
		if vehicle.alive[0] && vehicle.damaged&1 != 0 {
			part := vehicle.part(group, 0)
			g.addTargetBurst(part.x, part.y+7, 2)
		}
	}
	g.prepareVehicleFragmentBursts()
	delay := g.burst.remaining + 2
	for i := len(g.burstQueue) - 1; i >= 0; i-- {
		request := g.burstQueue[i]
		if request.owner != 0 {
			g.addTargetBurst(request.x, request.y, delay)
		}
		delay += 7
	}
	for _, shot := range g.demoShots() {
		if shot.enemy || shot.life <= 0 || !(shot.grenade || g.ownedShotUsesCannon(shot)) {
			continue
		}
		landing, ticks, ok := g.demoShotLanding(shot)
		if !ok {
			continue
		}
		// Owned impacts precede the same frame's moving enemy updates.
		targetMoves := max(0, ticks-1)
		// Scenery destruction first holds two contact quads, then emits
		// the hostile radial pool. Anticipate both before the weapon lands.
		if !g.burst.active || ticks > g.burst.remaining+1 {
			if burst, found := g.sceneryBurstAt(landing.x+8, landing.y-4); found && g.demoSceneryCount < len(g.demoSceneryForecast) {
				g.demoSceneryForecast[g.demoSceneryCount] = demoSceneryPrediction{burst, ticks}
				g.demoSceneryCount++
				g.addTargetBurst(burst.center.x, burst.center.y, ticks+9)
			}
		}
		for _, enemy := range g.enemies {
			if enemy.kind == 4 && enemy.health > 0 && (!enemy.tankBurning || enemy.tankBurn >= ticks) && grenadeHitsActor(landing, enemy) {
				g.addTargetBurst(enemy.x, enemy.y+7, ticks+1)
			}
		}
		for _, i := range [...]int{0, 3} {
			if g.aircraft.alive[i] {
				part := g.aircraft.part(i)
				if g.aircraft.flying {
					part.y += float64(targetMoves * 2)
				}
				if grenadeHitsActor(landing, part) {
					g.addTargetBurst(part.x, part.y+7, ticks+1)
				}
			}
			if g.boss.active && g.boss.health == 1 && g.boss.alive[i] {
				future := g.boss
				for range targetMoves {
					future.advance(g.cameraY)
				}
				part := future.part(i)
				if grenadeHitsActor(landing, part) {
					// The boss advances before its final hit round is applied.
					future.advance(g.cameraY)
					part = future.part(i)
					g.addTargetBurst(part.x, part.y+7, ticks+2)
				}
			}
		}
		for group, vehicle := range g.enemyVehicles {
			if !vehicle.alive[0] {
				continue
			}
			for range targetMoves {
				vehicle.advance(g.cameraY)
			}
			part := vehicle.part(group, 0)
			if grenadeHitsActor(landing, part) {
				g.addTargetBurst(part.x, part.y+7, ticks+1)
			}
		}
	}
	g.demoAircraftRisk = g.demoAircraftBurstRisk()
}

func (g *Game) demoShotLanding(shot projectile) (projectile, int, bool) {
	limit := shot.life
	if shot.impact > 0 && !g.isOwnedQuad(shot) {
		limit = shot.impact
	}
	if cleanup := g.ownedSlotCleanupDelay(shot); cleanup > 0 {
		limit = min(limit, cleanup)
	}
	for tick := 1; tick <= limit; tick++ {
		future := shot
		future.x += shot.vx * float64(tick)
		future.y += shot.vy * float64(tick)
		if shot.impact == 0 && tick == shot.life {
			return future, tick, true
		}
		if !g.smallObjectInBounds(future.x, future.y) {
			return future, tick, false
		}
		if shot.grenade {
			continue
		}
		if !g.ownedShotPiercesTerrain(shot) && g.solidAt(future.x+8, future.y-4+shot.terrainBias) {
			return future, tick, true
		}
		if g.ownedShotUsesCannon(shot) && (g.ownedCannonIgnoresActors(shot) || g.ownedBulletUpgrade(shot)) {
			continue
		}
		for i, enemy := range g.enemies {
			if enemy.health <= 0 || enemy.deadTicks > 0 || enemy.kind == 4 || enemy.kind == 6 {
				continue
			}
			enemy.x, enemy.y = g.forecastEnemyPosition(i, enemy, max(0, tick-1))
			if shotHitsActor(future, enemy) {
				return future, tick, true
			}
		}
		if g.shotHitsFinalScene(future) {
			return future, tick, true
		}
	}
	return shot, 0, false
}

func (g *Game) forecastEnemyPosition(index int, enemy actor, tick int) (float64, float64) {
	if g.demoForecastActive && tick >= 0 && tick <= demoHorizon && index < len(g.demoEnemyForecast[0]) {
		point := g.demoEnemyForecast[tick][index]
		return point.x, point.y
	}
	return g.demoEnemyPosition(enemy, tick)
}

func (g *Game) forecastBulletPosition(index int, shot projectile, tick int) (float64, float64, bool) {
	if g.demoForecastActive && tick >= 0 && tick <= demoHorizon && index < len(g.demoBulletForecast[0]) {
		point := g.demoBulletForecast[tick][index]
		return point.x, point.y, point.active
	}
	future, active := g.demoBulletAt(shot, tick)
	return future.x, future.y, active
}

// demoControls chooses ordinary movement, fire and context actions. The
// demonstration has the same reserve lives, ammunition, collisions and weapon
// slots as a human player; it never rewrites the player's position or health.
func (g *Game) demoControls() playerControls {
	if g.player.deadTicks > 0 || g.lives < 0 {
		return playerControls{}
	}
	g.prepareDemoForecast()
	defer func() { g.demoForecastActive = false }()
	if g.player.inTank && g.player.tankBurning {
		if g.demoTankExitSafe() || g.player.tankBurn < 2 {
			return playerControls{action: true}
		}
		g.demoFireHeld = false
		move := g.demoLeaveStrikeMarker()
		if move.x != 0 || move.y != 0 {
			return move
		}
		return g.demoFindSaferTankExit()
	}
	if g.player.x < 13 || g.player.x > 195 {
		// Boarding or an ordinary respawn can place an actor beyond the
		// normal walking range. Return directly when possible; a solid
		// boundary row needs an ordinary forward detour to the next gap. The original
		// current-position guards also permit odd walking positions 13/195;
		// those remain valid route nodes rather than forced-return states.
		target := navPoint{max(14, min(194, g.player.x)), g.player.y}
		direction := 1.0
		if g.player.x > 195 {
			direction = -1
		}
		if _, _, clear := g.demoStep(g.player.x, g.player.y, direction, 0); !clear {
			target.y -= 16
		}
		move, _ := g.demoChooseMove(target)
		return move
	}
	if controls, handled := g.demoOpeningBunker(); handled {
		return controls
	}
	if controls, handled := g.demoLateFanControls(); handled {
		return controls
	}
	if g.ammunition == 0 && !g.ending.active {
		if g.player.inTank && g.demoTankExitSafe() {
			return playerControls{action: true}
		}
	}
	if !g.player.inTank && g.ammunition > 0 {
		if i := g.boardingTankAt(0, g.player.x, g.player.y); i >= 0 && !g.demoTankWaterAhead(g.enemies[i].x, g.enemies[i].y) {
			return playerControls{action: true}
		}
	}
	if control, active := g.demoFlyingAircraftFlank(); active {
		return control
	}
	if control, active := g.demoKnownRiverGunners(); active {
		g.demoFireHeld = control.fire
		return control
	}
	if control, active := g.demoFactoryGuns(); active {
		return control
	}
	if control, active := g.demoStalledGunSetup(); active {
		g.demoFireHeld = control.fire
		return control
	}
	if math.Abs(g.player.x-g.demoLastX)+math.Abs(g.player.y-g.demoLastY) < 0.05 {
		g.demoStall++
	} else {
		g.demoStall = 0
	}
	g.demoLastX, g.demoLastY = g.player.x, g.player.y
	goal, targeted := g.demoDestination()
	goalChanged := targeted && goal != g.demoTarget || !targeted && g.demoTargetKind != 0
	gateCleared := g.demoPlanScenery && g.destroyed[g.demoSceneryRow][g.demoSceneryCol]
	_, _, stalledRoof := g.demoStalledRoofScenery()
	if len(g.demoPath) == 0 || g.demoStall >= 6 || g.demoPlanMines != len(g.mines) || g.demoPlanVehicle != g.player.inTank || goalChanged || gateCleared || stalledRoof && !g.demoPlanScenery {
		g.buildDemoPath()
		g.demoStall = 0
	}
	closest, distance := g.demoPathIndex, math.MaxFloat64
	for i := g.demoPathIndex; i < min(len(g.demoPath), g.demoPathIndex+48/navigationStep); i++ {
		d := math.Abs(g.demoPath[i].x-g.player.x) + math.Abs(g.demoPath[i].y-g.player.y)
		if d <= distance {
			closest, distance = i, d
		}
	}
	if distance <= navigationStep {
		g.demoPathIndex = closest
	}
	for g.demoPathIndex < len(g.demoPath)-1 && math.Abs(g.demoPath[g.demoPathIndex].x-g.player.x) <= 1 && math.Abs(g.demoPath[g.demoPathIndex].y-g.player.y) <= 1 {
		g.demoPathIndex++
	}
	// A firing tile must actually be reached. Replanning one movement before
	// it can keep the player alternating beside a valid grenade launch point.
	if g.demoPathIndex == len(g.demoPath)-1 && g.demoPath[g.demoPathIndex] == (navPoint{g.player.x, g.player.y}) {
		g.buildDemoPath()
	}
	if g.demoTankAtWaterDeadEnd() && g.demoTankExitSafe() {
		return playerControls{action: true}
	}
	target := navPoint{g.player.x, g.player.y - 16}
	if g.demoPathIndex < len(g.demoPath) {
		target = g.demoPath[min(g.demoPathIndex+16/navigationStep, len(g.demoPath)-1)]
	}
	normalPlan := g.demoPlanMoveDetails(target, nil)
	best, score := normalPlan.control, normalPlan.score
	if !g.demoFireHeld && g.ammunition > 0 && g.bulletAvailable(0) {
		aim, ok := g.demoAimForMove(best)
		if ok {
			dx, dy := directionVector(aim)
			attack := playerControls{x: dx, y: dy, fire: true}
			locked := g.player.direction == aim
			if locked {
				attack = best
				attack.fire = true
			}
			attackPlan := g.demoPlanAttackDetails(target, &attack, aim)
			// During normal progress retain the route comparison. A verified
			// clearing shot can resolve a sustained no-scroll combat barrier.
			if preferDemoAttack(normalPlan, attackPlan) && (attackPlan.score >= score-50 || g.idleArtilleryTicks >= 60) {
				g.directionLock[0] = locked
				best = attack
			} else {
				g.directionLock[0] = false
			}
		} else {
			g.directionLock[0] = false
		}
	} else {
		g.directionLock[0] = false
	}
	finalAttack := false
	if g.final.active && !g.player.inTank && g.grenades > 0 && g.grenadeAvailable(0) {
		if aim, ok := g.demoFinalAimFrom(g.player.x, g.player.y); ok {
			attack := playerControls{action: true}
			if g.player.direction != aim {
				attack.x, attack.y = directionVector(aim)
			}
			_, quality := g.demoPlanAttack(target, &attack, aim)
			if quality > demoBlockedScore && quality >= score-50 {
				g.directionLock[0] = false
				best, finalAttack = attack, true
			}
		}
	}
	if !finalAttack && !g.player.inTank && g.grenades > 0 && g.grenadeAvailable(0) {
		if aim, ok := g.demoGrenadeAimForMove(best); ok {
			dx, dy := directionVector(aim)
			attack := playerControls{x: dx, y: dy, action: true}
			if g.player.direction == aim {
				attack = best
				attack.fire, attack.action = false, true
			}
			_, quality := g.demoPlanAttack(target, &attack, aim)
			if quality > demoBlockedScore && quality >= score-50 {
				g.directionLock[0] = g.player.direction == aim
				best = attack
			}
		}
	}
	if g.player.inTank && g.ammunition > 0 && g.bulletAvailable(0) || !g.player.inTank && g.grenades > 0 && g.grenadeAvailable(0) {
		if aim, ok := g.demoSceneryAim(); ok {
			dx, dy := directionVector(aim)
			attack := playerControls{x: dx, y: dy}
			if g.player.direction == aim {
				attack = playerControls{}
			}
			attack.fire, attack.action = g.player.inTank, !g.player.inTank
			_, quality := g.demoPlanAttack(target, &attack, aim)
			if quality > demoBlockedScore && quality >= score-50 {
				g.directionLock[0] = g.player.direction == aim
				best = attack
			}
		}
	}
	g.demoFireHeld = best.fire
	return best
}

// Aim at the composite gate actually blocking the planned route. Planning
// never changes terrain; the real weapon must travel and clear the object.
func (g *Game) demoSceneryAim() (int, bool) {
	if !g.demoPlanScenery {
		return 0, false
	}
	return g.demoSceneryAimFrom(g.player.x, g.player.y)
}

func (g *Game) demoRouteScenery() (int, int, bool) {
	row, column, found := 0, 0, false
	for i := g.demoPathIndex; i < min(len(g.demoPath), g.demoPathIndex+128/navigationStep) && !found; i++ {
		point := g.demoPath[i]
		for _, probe := range [...]navPoint{{point.x, point.y - 8}, {point.x + 8, point.y}, {point.x - 4, point.y}, {point.x, point.y + 4}} {
			if r, c, _, ok := g.sceneryBounds(probe.x, probe.y); ok {
				row, column, found = r, c, true
				break
			}
		}
	}
	return row, column, found
}

func (g *Game) demoSceneryAimFrom(x, y float64) (int, bool) {
	for direction := range 8 {
		dx, dy := directionVector(direction)
		originX, originY := x, y
		if g.player.direction != direction {
			if _, _, ok := g.demoStep(x, y, dx, dy); !ok {
				continue
			}
			var ok bool
			originX, originY, ok = g.demoStep(originX, originY, dx, 0)
			if !ok {
				continue
			}
		}
		x, y := originX-8+dx*72, originY+4+dy*72
		if g.player.inTank {
			source := g.player
			source.x, source.y, source.direction = originX, originY, direction
			shot := g.projectileFor(&source, false, 0, false, 6)
			landing, _, ok := g.demoShotLanding(shot)
			if !ok {
				continue
			}
			x, y = landing.x, landing.y
		}
		r, c, _, ok := g.sceneryBounds(x+8, y-4)
		if ok && r == g.demoSceneryRow && c == g.demoSceneryCol {
			return direction, true
		}
	}
	return 0, false
}

func (g *Game) demoGrenadeAim() (int, bool) {
	return g.demoGrenadeAimForMove(playerControls{})
}

func (g *Game) demoGrenadeAimForMove(move playerControls) (int, bool) {
	// The grenade lands before the eighteenth enemy movement.
	best, quality := -1, 0
	for direction := range 8 {
		dx, dy := directionVector(direction)
		movement := playerControls{x: dx, y: dy}
		if g.player.direction == direction {
			movement = move
		}
		source := g.player
		// Horizontal movement precedes the weapon phase. A blocked horizontal
		// step still permits firing from the unchanged position.
		if x, _, ok := g.demoStep(source.x, source.y, movement.x, 0); ok {
			source.x = x
		}
		source.direction = direction
		shot, _, lands := g.demoShotLanding(g.projectileFor(&source, false, 0, true, 4))
		if !lands {
			continue
		}
		value := 0
		for i, enemy := range g.enemies {
			if enemy.health <= 0 || enemy.deadTicks > 0 || enemy.kind == 6 {
				continue
			}
			if enemy.kind == 4 {
				if !enemy.tankBurning && enemy.tankFuel > 1 && grenadeHitsActor(shot, enemy) {
					value -= 600
				}
				continue
			}
			// Keep a small explosive reserve for armored targets and the
			// final encounter. Ordinary soldiers can still be shot or avoided.
			if g.grenades <= 4 {
				continue
			}
			enemy.x, enemy.y = g.forecastEnemyPosition(i, enemy, 17)
			if grenadeHitsActor(shot, enemy) {
				value += 100
				if enemy.kind == 3 {
					value += 200
				}
			}
		}
		if g.aircraft.flying {
			for i, alive := range g.aircraft.alive {
				if alive && (i == 0 || i == 3) {
					part := g.aircraft.part(i)
					part.y += 34
					if grenadeHitsActor(shot, part) {
						value += 400
					}
				}
			}
		}
		for group, vehicle := range g.enemyVehicles {
			for range 17 {
				vehicle.advance(g.cameraY)
			}
			if vehicle.alive[0] && grenadeHitsActor(shot, vehicle.part(group, 0)) {
				value += 300
			}
		}
		if g.boss.active {
			boss := g.boss
			for range 17 {
				boss.advance(g.cameraY)
			}
			for i, alive := range boss.alive {
				if alive && grenadeHitsActor(shot, boss.part(i)) {
					value += 150
				}
			}
		}
		if g.final.active {
			for _, i := range [...]int{1, 4, 7} {
				if grenadeHitsActor(shot, g.final.part(i)) {
					value += 500
				}
			}
		}
		if value > quality {
			best, quality = direction, value
		}
	}
	minimum := 200
	if g.ammunition == 0 && g.grenades > 8 {
		minimum = 100
	}
	return best, quality >= minimum
}

// Include horizontal movement before the shot; vertical movement follows it.
// The final portrait is protected until the actual blast reaches its center.
func (g *Game) demoFinalAimFrom(x, y float64) (int, bool) {
	for direction := range 8 {
		dx, dy := directionVector(direction)
		ox, oy := x, y
		moveX, moveY := x, y
		if g.player.direction != direction {
			var ok bool
			moveX, moveY, ok = g.demoStep(x, y, dx, dy)
			if !ok {
				continue
			}
			ox, oy, ok = g.demoStep(x, y, dx, 0)
			if !ok {
				continue
			}
		}
		probe := g.player
		probe.x, probe.y = moveX, moveY
		if g.demoThreatForMove(probe, y, 1) > 0 {
			continue
		}
		shot := projectile{x: ox - 8, y: oy + 4, vx: dx * 4, vy: dy * 4, life: 18, grenade: true}
		landing, _, ok := g.demoShotLanding(shot)
		if !ok {
			continue
		}
		for _, i := range [...]int{1, 4, 7} {
			if grenadeHitsActor(landing, g.final.part(i)) {
				return direction, true
			}
		}
	}
	return 0, false
}

// A burning tank can still drive. Repeated marker hits reset its burn timer,
// so waiting on a marker can last forever; use ordinary movement to leave it
// before looking for a safe infantry exit.
func (g *Game) demoLeaveStrikeMarker() playerControls {
	directions := [...][2]float64{{0, 1}, {-1, 0}, {1, 0}, {0, -1}}
	bestDistance := math.MaxFloat64
	target := navPoint{g.player.x, g.player.y}
	found := false
	for _, d := range directions {
		if _, _, ok := g.demoStep(g.player.x, g.player.y, d[0], d[1]); !ok {
			continue
		}
		distance := 0.0
		overlaps := false
		for _, mine := range g.mines {
			if mine.kind != 20 || !mineTouchesActor(mine, g.player) {
				continue
			}
			overlaps = true
			edge := mine.y + 12 - g.player.y
			if d[0] < 0 {
				edge = g.player.x - mine.x + 19
			} else if d[0] > 0 {
				edge = mine.x + 24 - g.player.x
			} else if d[1] < 0 {
				edge = g.player.y - mine.y + 27
			}
			distance = max(distance, edge+2)
		}
		if overlaps && distance < bestDistance {
			bestDistance = distance
			target = navPoint{g.player.x + d[0]*(distance+8), g.player.y + d[1]*(distance+8)}
			found = true
		}
	}
	if !found {
		return playerControls{}
	}
	move, _ := g.demoChooseMove(target)
	return move
}

// A burning tank still shields its passenger during the escape interval.
// Wait out an active blast before dismounting into ordinary infantry danger.
func (g *Game) demoTankExitSafe() bool {
	return g.demoTankExitThreat(g.player.x, g.player.y, 0) == 0
}

func (g *Game) demoMoveScore(c playerControls, target navPoint) float64 {
	x, y, ok := g.demoStep(g.player.x, g.player.y, c.x, c.y)
	if !ok {
		return demoBlockedScore
	}
	return 8*(math.Abs(g.player.x-target.x)+math.Abs(g.player.y-target.y)-math.Abs(x-target.x)-math.Abs(y-target.y)) - g.demoThreatAt(x, y, 1)
}

type demoNode struct {
	x, y, cost, score, baseCost float64
	first                       playerControls
}

// A short beam search can plan an actual turn around a shot. Assuming that
// every candidate direction is held for several ticks caused safe detours to
// look dangerous and left the old player waiting indefinitely.
func (g *Game) demoChooseMove(target navPoint) (playerControls, float64) {
	return g.demoPlanMove(target, nil)
}

const demoPositionSide = 4*demoHorizon + 1

// Every ordinary walking step changes an axis by at most two pixels. Index
// exact integral displacements directly; fractional actor positions
// retain the original equality scan rather than rounding distinct coordinates.
func (g *Game) demoPositionCell(x, y float64) int {
	dx, dy := x-g.player.x, y-g.player.y
	ix, iy := int(dx), int(dy)
	const radius = 2 * demoHorizon
	if float64(ix) != dx || float64(iy) != dy || uint(ix+radius) >= demoPositionSide || uint(iy+radius) >= demoPositionSide {
		return -1
	}
	return (iy+radius)*demoPositionSide + ix + radius
}

// Hard forecast survival and soft route costs are separate decisions. A clear
// ordinary kill may be worth a short firing detour, but never an earlier hit.
type demoMovePlan struct {
	control     playerControls
	score       float64
	survived    int
	clearsActor bool
}

func preferDemoAttack(normal, attack demoMovePlan) bool {
	if attack.score <= demoBlockedScore || attack.survived < normal.survived {
		return false
	}
	return attack.score >= normal.score-50 || attack.clearsActor && attack.survived == demoHorizon
}

func (g *Game) demoPlanMove(target navPoint, first *playerControls) (playerControls, float64) {
	plan := g.demoPlanMoveDetails(target, first)
	return plan.control, plan.score
}

func (g *Game) demoPlanMoveDetails(target navPoint, first *playerControls) demoMovePlan {
	var current, next [demoBeamWidth]demoNode
	integral := g.player.x == float64(int(g.player.x)) && g.player.y == float64(int(g.player.y))
	var costs [demoPositionSide * demoPositionSide]float64
	var occupied [demoPositionSide * demoPositionSide]bool
	current[0] = demoNode{x: g.player.x, y: g.player.y}
	count := 1
	directions := [...][2]float64{{0, -1}, {-1, -1}, {1, -1}, {-1, 0}, {1, 0}, {0, 1}, {-1, 1}, {1, 1}, {0, 0}}
	startDistance := math.Abs(g.player.x-target.x) + math.Abs(g.player.y-target.y)
	for depth := 1; depth <= demoHorizon; depth++ {
		clear(occupied[:])
		written := 0
		for _, node := range current[:count] {
			for _, direction := range directions {
				if depth == 1 && first != nil && (direction[0] != first.x || direction[1] != first.y) {
					continue
				}
				x, y, ok := g.demoStep(node.x, node.y, direction[0], direction[1])
				if !ok {
					continue
				}
				// Paths reaching the same position at the same depth receive
				// identical soft costs. An existing valid path with a lower
				// incoming cost already dominates this candidate.
				candidate := demoNode{x: x, y: y, cost: node.cost, first: node.first}
				if depth == 1 {
					candidate.first = playerControls{x: direction[0], y: direction[1]}
				}
				if direction[0] == 0 && direction[1] == 0 {
					candidate.cost += 0.05
				}
				candidate.baseCost = candidate.cost
				duplicate := -1
				cell := -1
				if integral {
					cell = g.demoPositionCell(x, y)
				}
				if cell < 0 || occupied[cell] {
					if cell >= 0 && candidate.baseCost >= costs[cell] {
						continue
					}
					for i := 0; i < written; i++ {
						if next[i].x == x && next[i].y == y {
							duplicate = i
							break
						}
					}
				}
				if duplicate >= 0 && candidate.baseCost >= next[duplicate].baseCost {
					continue
				}
				candidate.cost += 0.15 * (math.Abs(x-target.x) + math.Abs(y-target.y))
				// The remaining soft costs are nonnegative. If even this upper
				// score cannot enter a full beam, avoid the expensive hazard probes.
				optimisticScore := 8*(startDistance-math.Abs(x-target.x)-math.Abs(y-target.y)) - candidate.cost
				if written == len(next) && next[written-1].score >= optimisticScore {
					continue
				}
				probe := g.player
				probe.x, probe.y = x, y
				if g.demoThreatForMove(probe, node.y, depth) > 0 {
					continue
				}
				candidate.cost += g.demoGrenadeEscapeCost(x, y, depth)
				candidate.cost += g.demoMineEscapeCost(x, y, depth)
				candidate.cost += g.demoPossibleShotCost(x, y, depth)
				candidate.score = 8*(startDistance-math.Abs(x-target.x)-math.Abs(y-target.y)) - candidate.cost
				if duplicate >= 0 {
					if next[duplicate].score >= candidate.score {
						continue
					}
					copy(next[duplicate:written-1], next[duplicate+1:written])
					written--
				}
				i := written
				if i == len(next) {
					i--
					if next[i].score >= candidate.score {
						continue
					}
					if old := g.demoPositionCell(next[i].x, next[i].y); old >= 0 {
						occupied[old] = false
					}
				} else {
					written++
				}
				for i > 0 && next[i-1].score < candidate.score {
					next[i] = next[i-1]
					i--
				}
				next[i] = candidate
				if cell >= 0 {
					costs[cell], occupied[cell] = candidate.baseCost, true
				}
			}
		}
		if written == 0 {
			if depth == 1 {
				return demoMovePlan{score: demoBlockedScore}
			}
			return demoMovePlan{control: current[0].first, score: float64(depth-1)*10000 + current[0].score, survived: depth - 1}
		}
		current, next = next, current
		count = written
	}
	return demoMovePlan{control: current[0].first, score: demoHorizon*10000 + current[0].score, survived: demoHorizon}
}

// A grenade must be anticipated before its blast starts. Remaining near the
// center until the last few ticks leaves too little time to cross the original
// 52-by-48 collision area at the player's ordinary two-pixel movement speed.
func (g *Game) demoGrenadeEscapeCost(x, y float64, tick int) float64 {
	if g.ending.active || g.player.invulnerable > tick || g.player.inTank && g.player.tankBurning && g.player.tankBurn > tick {
		return 0
	}
	cost := 0.0
	if !g.demoForecastActive {
		g.prepareTargetBursts()
	}
	for _, burst := range g.demoTargetBursts[:g.demoBurstCount] {
		if tick > burst.firstMove+5 {
			continue
		}
		depth := min(x-burst.x+64, burst.x+69-x, y-burst.y+72, burst.y+57-y)
		if depth > 0 {
			cost += 32 * depth * depth / float64(max(1, burst.firstMove-tick+1))
		}
	}
	for _, shot := range g.demoShots() {
		endX := shot.x + shot.vx*float64(max(0, shot.life))
		endY := shot.y + shot.vy*float64(max(0, shot.life))
		if first, fragments := g.demoFragmentStart(shot); fragments && tick <= first+5 {
			depth := min(x-endX+64, endX+69-x, y-endY+72, endY+57-y)
			if depth > 0 {
				cost += 32 * depth * depth / float64(max(1, first-tick+1))
			}
		}
		if !shot.enemy || !shot.grenade || shot.impact > 0 || shot.life <= tick || g.player.invulnerable > shot.life+3 {
			continue
		}
		depth := min(x-endX+24, endX+29-x, y-endY+32, endY+17-y)
		if depth > 0 {
			cost += 4 * depth * depth / float64(shot.life-tick+1)
		}
	}
	if g.burst.active && g.burst.owner != 0 && tick <= g.burst.remaining {
		center := g.burst.parts[1]
		center.y += float64(6-g.burst.remaining) * 8
		depth := min(x-center.x+64, center.x+69-x, y-center.y+72, center.y+57-y)
		if depth > 0 {
			cost += 32 * depth * depth
		}
	}
	cost += g.demoAircraftStandoffCost(x, y, tick)
	return cost
}

func (g *Game) demoThreatAt(x, y float64, tick int) float64 {
	value := g.player
	value.x, value.y = x, y
	return g.demoThreatForActor(value, tick)
}

func (g *Game) demoThreatForActor(value actor, tick int) float64 {
	return g.demoThreatForMove(value, value.y, tick)
}

func (g *Game) demoThreatForMove(value actor, previousY float64, tick int) float64 {
	if g.ending.active || value.invulnerable > tick || value.inTank && value.tankBurning && value.tankBurn > tick {
		return 0
	}
	x, y := value.x, value.y
	penalty := 0.0
	// Mine contacts see the new X before vertical movement. A diagonal can
	// leave the contact rectangle afterward and still have triggered the mine.
	contact := value
	contact.y = previousY
	for _, mine := range g.mines {
		if mine.kind != 20 && smallObjectOverlapsActor(mine.x, mine.y, contact) {
			penalty += 5000
		}
	}
	// A water mine can move after this input and threaten the next one. Allow
	// for an early jump in each eight-update interval without reading future RNG.
	if tick > 1 {
		for _, mine := range g.mines {
			if mine.kind != 19 {
				continue
			}
			for range (tick + 6) / 8 {
				mine = g.homingMineStep(mine, contact)
			}
			if smallObjectOverlapsActor(mine.x, mine.y, contact) {
				penalty += 5000
			}
		}
	}
	if !g.demoForecastActive {
		g.prepareTargetBursts()
	}
	penalty += g.sceneryBurstThreat(value, tick)
	for _, predicted := range g.demoSceneryForecast[:g.demoSceneryCount] {
		if tick < predicted.landing || tick >= predicted.landing+8 {
			continue
		}
		for _, part := range predicted.burst.parts {
			if smallObjectTouchesActor(part.x, part.y, value) {
				penalty += 5000
			}
		}
	}
	for _, burst := range g.demoTargetBursts[:g.demoBurstCount] {
		if g.demoPendingBurstQuadHit(burst, value, tick) {
			penalty += 5000
		}
		age := tick - burst.firstMove + 1
		if age < 1 || age > 6 {
			continue
		}
		for _, motion := range burstMotions {
			if smallObjectTouchesActor(burst.x+motion.x*float64(age), burst.y+motion.y*float64(age), value) {
				penalty += 5000
			}
		}
	}
	if g.burst.active && g.burst.owner != 0 && tick <= g.burst.remaining {
		for i, part := range g.burst.parts {
			motion := burstMotions[i]
			if smallObjectTouchesActor(part.x+motion.x*float64(tick), part.y+motion.y*float64(tick), value) {
				penalty += 5000
			}
		}
	}
	for i, shot := range g.demoShots() {
		if g.demoFutureFragmentsHit(shot, value, tick) {
			penalty += 5000
		}
		if !shot.enemy {
			continue
		}
		if shot.grenade {
			if shot.life > tick || shot.life > 0 && tick >= shot.life+3 || shot.impact > 0 && tick >= shot.impact {
				continue
			}
			future := shot
			future.x += shot.vx * float64(max(0, shot.life))
			future.y += shot.vy * float64(max(0, shot.life))
			if grenadeHitsActor(future, value) {
				penalty += 5000
			}
		} else if !value.inTank && value.invulnerable <= tick && shot.impact == 0 && shot.life >= tick {
			x, y, active := g.forecastBulletPosition(i, shot, max(0, tick-1))
			// The original bullet pass sees the new X and the preceding Y.
			probe := value
			probe.y = previousY
			if active && shotHitsActor(projectile{x: x, y: y}, probe) {
				penalty += 5000
			}
		}
	}
	if !value.inTank && value.invulnerable <= tick {
		if g.final.active {
			for i := range 9 {
				part := g.final.part(i)
				if math.Abs(part.x-x) <= 25 && math.Abs(part.y-y) <= 21 {
					penalty += 3000
				}
			}
		}
		for i, enemy := range g.enemies {
			if i < len(g.demoEnemyDefeat) && g.demoEnemyDefeat[i] > 0 && tick > g.demoEnemyDefeat[i] {
				continue
			}
			if enemy.health <= 0 || enemy.deadTicks > 0 || enemy.kind == 4 || enemy.kind == 6 {
				continue
			}
			ex, ey := g.forecastEnemyPosition(i, enemy, max(0, tick-1))
			if math.Abs(ex-x) <= 25 && math.Abs(ey-previousY) <= 21 {
				penalty += 3000
			}
		}
	}
	return penalty
}

// Upgraded player grenades produce the original hostile fragment pool too.
// Anticipate it before creation; seeing the first moving fragments is too late
// when the player has followed the grenade toward its landing point.
func (g *Game) demoFragmentStart(shot projectile) (int, bool) {
	if !shot.enemy && !shot.grenade && shot.impact > 0 && !g.isOwnedQuad(shot) {
		return 0, false
	}
	if cleanup := g.ownedSlotCleanupDelay(shot); cleanup > 0 && shot.life > cleanup {
		return 0, false
	}
	fragments := shot.fragments
	if !shot.enemy && (shot.grenade || g.ownedShotUsesCannon(shot) || g.isOwnedQuad(shot)) && shot.owner >= 0 && shot.owner < len(g.grenadePower) && g.grenadePower[shot.owner] > 1 {
		fragments = true
	}
	if !fragments || shot.life <= 0 && shot.impact <= 0 {
		return 0, false
	}
	firstMove := shot.impact
	if shot.life > 0 {
		firstMove = shot.life + 3
	}
	if shot.enemy {
		firstMove++
	}
	return firstMove, true
}

func (g *Game) demoFutureFragmentsHit(shot projectile, value actor, tick int) bool {
	firstMove, fragments := g.demoFragmentStart(shot)
	if !fragments {
		return false
	}
	age := tick - firstMove + 1
	if age < 1 || age > 6 {
		return false
	}
	x, y := shot.x+shot.vx*float64(max(0, shot.life)), shot.y+shot.vy*float64(max(0, shot.life))
	for _, motion := range burstMotions {
		if smallObjectTouchesActor(x+motion.x*float64(age), y+motion.y*float64(age), value) {
			return true
		}
	}
	return false
}

func (g *Game) demoEnemyPosition(enemy actor, ticks int) (float64, float64) {
	if enemy.kind != 0 && enemy.kind != 5 {
		return enemy.x, enemy.y
	}
	for range ticks {
		if _, _, moving := g.advanceInfantryMarch(&enemy); !moving {
			break
		}
	}
	return enemy.x, enemy.y
}

func (g *Game) demoAim() (int, bool) {
	return g.demoAimForMove(playerControls{})
}

func (g *Game) demoAimForMove(move playerControls) (int, bool) {
	best, quality := -1, math.Inf(-1)
	pending := g.demoPendingBulletTargets()
	for direction := 0; direction < 8; direction++ {
		source := g.player
		// Turning uses its directional input; a locked heading keeps the
		// selected movement. The actual shot follows horizontal movement.
		dx, _ := directionVector(direction)
		if source.direction == direction {
			dx = move.x
		}
		if x, _, ok := g.demoStep(source.x, source.y, dx, 0); ok {
			source.x = x
		} else {
			continue
		}
		source.direction = direction
		shot := g.projectileFor(&source, false, 0, false, 6)
		limit := shot.life + 3
		if source.inTank {
			limit = shot.life
			if shot.frame == 8 {
				continue
			}
		}
		for tick := 1; tick <= limit; tick++ {
			visible := true
			if source.inTank {
				var landed bool
				shot, landed = g.advanceCannon(shot)
				visible = !landed
			} else {
				shot, visible = g.advancePlayerBullet(shot)
			}
			for i, enemy := range g.enemies {
				if i < 64 && pending&(uint64(1)<<i) != 0 {
					continue
				}
				if enemy.health <= 0 || enemy.deadTicks > 0 || enemy.kind == 4 || enemy.kind == 6 {
					continue
				}
				// Fixed bullet gunners cannot harm an intact tank. Marching
				// infantry can also throw grenades, so keep those valid targets.
				if source.inTank && enemy.kind == 2 {
					continue
				}
				// Owned shots advance before the enemy movement pass.
				enemy.x, enemy.y = g.forecastEnemyPosition(i, enemy, max(0, tick-1))
				if shotHitsActor(shot, enemy) {
					value := 100 - float64(tick)*4
					if enemy.kind == 2 || enemy.kind == 3 {
						value += 10
					}
					if value > quality {
						best, quality = direction, value
					}
				}
			}
			if !visible {
				break
			}
		}
	}
	return best, best >= 0
}

// Ordinary enemy bullets ignore terrain in $a2f4. Only range and the original
// sprite bounds end their flight; the forecast must preserve the same threat.
func (g *Game) demoBulletAt(shot projectile, ticks int) (projectile, bool) {
	for range ticks {
		shot.x += shot.vx
		shot.y += shot.vy
		shot.life--
		if shot.life <= 0 || !g.smallObjectInBounds(shot.x, shot.y) {
			return shot, false
		}
		if !shot.enemy && !g.ownedShotPiercesTerrain(shot) && g.solidAt(shot.x+8, shot.y-4+shot.terrainBias) {
			return shot, false
		}
	}
	return shot, true
}

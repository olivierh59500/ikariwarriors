package game

import "math"

// Use the ordinary two-pixel movement lattice. A four-pixel graph misses
// the walkable X14 boundary beside late-level terrain and air-strike markers.
const navigationStep = 2

// buildDemoPath plans on the actor's own pixel lattice. Each edge uses the
// same asymmetric terrain probes as ordinary movement; it cannot cut a wall
// corner by skipping ahead to a distant waypoint.
func (g *Game) buildDemoPath() {
	roofRow, roofColumn, roofGoal := g.demoStalledRoofScenery()
	var bodies [19]navPoint
	countBodies := g.demoRouteInfantry(bodies[:])
	columns, rows := playfieldWidth/navigationStep, levelHeight/navigationStep
	count := columns * rows
	fresh := false
	if cap(g.navParents) < count {
		g.navParents = make([]int, count)
		fresh = true
	} else {
		g.navParents = g.navParents[:count]
	}
	xOffset := int(g.player.x) % navigationStep
	yOffset := int(g.player.y) % navigationStep
	startColumn := max(0, min(columns-1, int(g.player.x)/navigationStep))
	startRow := max(0, min(rows-1, int(g.player.y)/navigationStep))
	start := startRow*columns + startColumn
	if cap(g.navQueue) < count {
		g.navQueue = make([]int, 0, count)
		fresh = true
	}
	if fresh {
		for i := range g.navParents {
			g.navParents[i] = -2
		}
		g.navQueue = g.navQueue[:0]
	}
	goal, targeted := g.demoDestination()
	pickupGoal := false
	if targeted {
		for _, item := range g.pickups {
			if item.x == goal.x && item.y == goal.y {
				pickupGoal = true
				break
			}
		}
	}
	g.demoTarget, g.demoTargetKind = goal, 0
	if targeted {
		g.demoTargetKind = 1
	}
	best := start
	directions := [...][2]int{{0, -1}, {-1, 0}, {1, 0}, {0, 1}}
	allowScenery := false
	for {
		// Every marked parent belongs to the previous queue. Clear those
		// visited cells rather than the entire 780,000-cell level graph.
		for _, previous := range g.navQueue {
			g.navParents[previous] = -2
		}
		g.navParents[start] = -1
		g.navQueue = g.navQueue[:1]
		g.navQueue[0] = start
		bestDistance := math.MaxFloat64
		goalReached := false
		best = start
		for cursor := 0; cursor < len(g.navQueue); cursor++ {
			current := g.navQueue[cursor]
			row, column := current/columns, current%columns
			x, y := float64(column*navigationStep+xOffset), float64(row*navigationStep+yOffset)
			distance := y
			if targeted {
				distance = math.Abs(x-goal.x) + math.Abs(y-goal.y)
			}
			if distance < bestDistance {
				best, bestDistance = current, distance
			}
			reached := math.Abs(x-goal.x) <= 23 && math.Abs(y-goal.y) <= 19
			if pickupGoal {
				reached = smallObjectTouchesActor(goal.x, goal.y, actor{x: x, y: y, health: 1})
			} else if targeted {
				i := g.boardingTankAt(0, x, y)
				reached = i >= 0 && g.enemies[i].x == goal.x && g.enemies[i].y == goal.y
			}
			if targeted && reached {
				best = current
				goalReached = true
				break
			}
			for _, direction := range directions {
				c, r := column+direction[0], row+direction[1]
				if c < 0 || c >= columns || r < 0 || r >= rows {
					continue
				}
				next := r*columns + c
				if g.navParents[next] != -2 {
					continue
				}
				px, py := x, y
				blocked := false
				for step := 0; step < navigationStep; step++ {
					if math.Max(math.Abs(px-x), math.Abs(py-y)) >= navigationStep {
						break
					}
					nx, ny, ok := g.demoTerrainStep(px, py, float64(direction[0]), float64(direction[1]), allowScenery)
					if !ok || !demoRouteClearsInfantry(bodies[:countBodies], px, py, nx, ny) {
						blocked = true
						break
					}
					px, py = nx, ny
				}
				if blocked {
					continue
				}
				g.navParents[next] = current
				g.navQueue = append(g.navQueue, next)
			}
		}
		// Prefer open ground. A sealed route may require an ordinary grenade
		// before the demonstration can follow its next waypoint.
		if allowScenery || targeted && goalReached || !targeted && bestDistance < g.player.y-32 || g.grenades == 0 && (!g.player.inTank || g.ammunition == 0) {
			if targeted && !goalReached {
				// An optional pickup can be sealed into another compartment.
				// Once both searches fail, continue along the open route instead
				// of waiting forever at its closest unreachable point. Keep the
				// requested goal cached until the scene or its usefulness changes.
				goal, targeted, pickupGoal = navPoint{}, false, false
				allowScenery = false
				continue
			}
			break
		}
		allowScenery = true
	}
	g.setDemoPath(best, columns, xOffset, yOffset)
	g.demoPlanMines = len(g.mines)
	g.demoPlanVehicle = g.player.inTank
	g.demoPlanScenery = false
	if g.final.active && !g.player.inTank && g.grenades > 0 {
		g.planFiringApproach(start, columns, rows, xOffset, yOffset, true)
		return
	}
	if roofGoal {
		g.demoSceneryRow, g.demoSceneryCol, g.demoPlanScenery = roofRow, roofColumn, true
		// A recovery goal searches current reachable firing poses first;
		// a remembered far anchor can repeat the very combat barrier.
		g.planFiringApproachSearch(start, columns, rows, xOffset, yOffset, false, true)
		return
	}
	if allowScenery {
		if row, column, ok := g.demoRouteScenery(); ok {
			g.demoSceneryRow, g.demoSceneryCol, g.demoPlanScenery = row, column, true
			g.planFiringApproach(start, columns, rows, xOffset, yOffset, false)
		}
	}
}

func (g *Game) setDemoPath(best, columns, xOffset, yOffset int) {
	length := 0
	for current := best; current >= 0; current = g.navParents[current] {
		length++
	}
	if cap(g.demoPath) < length {
		g.demoPath = make([]navPoint, length)
	} else {
		g.demoPath = g.demoPath[:length]
	}
	current := best
	for i := length - 1; i >= 0; i-- {
		g.demoPath[i] = navPoint{float64(current%columns*navigationStep + xOffset), float64(current/columns*navigationStep + yOffset)}
		current = g.navParents[current]
	}
	g.demoPathIndex = min(1, len(g.demoPath)-1)
}

// A blocked gate needs a reachable firing position, including enough space
// to turn before throwing. Search the current open compartment with the same
// real movement probes; a planning decision never opens the gate itself.
func (g *Game) planFiringApproach(start, columns, rows, xOffset, yOffset int, finalEncounter bool) {
	g.planFiringApproachSearch(start, columns, rows, xOffset, yOffset, finalEncounter, false)
}

func (g *Game) planFiringApproachSearch(start, columns, rows, xOffset, yOffset int, finalEncounter, currentFirst bool) {
	// Spread repeated alternative forecasts for remembered firing goals over updates.
	// Each later search checks the current world again; only the candidate
	// cursor survives, never a promised future hit or an altered game state.
	skip, seen, checked := g.demoFiringSearchOffset, 0, 0
	var bodies [19]navPoint
	countBodies := g.demoRouteInfantry(bodies[:])
	// Try remembered map positions first, then search the whole compartment
	// when current terrain, weapon geometry or threats reject those positions.
	useHints := !currentFirst && g.hasDemoExpertAnchors(finalEncounter)
	useFallbackHints := !currentFirst && !useHints && g.hasDemoFallbackAnchors(finalEncounter)
search:
	for _, previous := range g.navQueue {
		g.navParents[previous] = -2
	}
	g.navParents[start] = -1
	g.navQueue = g.navQueue[:1]
	g.navQueue[0] = start
	for cursor := 0; cursor < len(g.navQueue); cursor++ {
		current := g.navQueue[cursor]
		row, column := current/columns, current%columns
		x, y := float64(column*navigationStep+xOffset), float64(row*navigationStep+yOffset)
		aim, ready := 0, false
		if !useHints && !useFallbackHints || useHints && g.atDemoExpertAnchor(x, y) || useFallbackHints && g.atDemoFallbackAnchor(x, y) {
			if finalEncounter {
				aim, ready = g.demoFinalAimFrom(x, y)
			} else {
				aim, ready = g.demoSceneryAimFrom(x, y)
			}
		}
		if ready && !finalEncounter && !currentFirst {
			seen++
			if seen <= skip {
				ready = false
			} else {
				if checked == 8 {
					g.demoFiringSearchOffset = seen - 1
					return
				}
				checked++
				ready = g.demoAnchorReady(x, y, aim)
			}
		}
		if ready && !finalEncounter && currentFirst {
			ready = g.demoAnchorReady(x, y, aim)
		}

		if ready {
			g.demoFiringSearchOffset = 0
			g.setDemoPath(current, columns, xOffset, yOffset)
			return
		}
		if math.Abs(x-g.player.x)+math.Abs(y-g.player.y) > 160 {
			continue
		}
		for _, direction := range [...][2]int{{0, -1}, {-1, 0}, {1, 0}, {0, 1}} {
			c, r := column+direction[0], row+direction[1]
			if c < 0 || c >= columns || r < 0 || r >= rows || g.navParents[r*columns+c] != -2 {
				continue
			}
			px, py, blocked := x, y, false
			for step := 0; step < navigationStep; step++ {
				if math.Max(math.Abs(px-x), math.Abs(py-y)) >= navigationStep {
					break
				}
				nx, ny, ok := g.demoStep(px, py, float64(direction[0]), float64(direction[1]))
				if !ok || !demoRouteClearsInfantry(bodies[:countBodies], px, py, nx, ny) {
					blocked = true
					break
				}
				if finalEncounter && g.demoThreatAt(nx, ny, 1) > 0 {
					blocked = true
					break
				}
				px, py = nx, ny
			}
			if !blocked {
				g.navParents[r*columns+c] = current
				g.navQueue = append(g.navQueue, r*columns+c)
			}
		}
	}
	if useHints {
		useHints = false
		useFallbackHints = g.hasDemoFallbackAnchors(finalEncounter)
		goto search
	}
	if useFallbackHints {
		useFallbackHints = false
		goto search
	}
	g.demoFiringSearchOffset = 0
}

func (g *Game) demoDestination() (navPoint, bool) {
	if g.ending.active {
		return navPoint{}, false
	}
	best := 200.0
	target := navPoint{}
	if !g.player.inTank && g.ammunition > 0 && !g.hasBurningTank(0) {
		for _, enemy := range g.enemies {
			if enemy.kind != 4 || enemy.health <= 0 || enemy.tankBurning || enemy.y > g.player.y+24 || g.player.inTank && enemy.tankFuel <= g.player.tankFuel+8 || g.demoTankWaterAhead(enemy.x, enemy.y) {
				continue
			}
			distance := math.Abs(enemy.x-g.player.x) + math.Abs(enemy.y-g.player.y)
			if g.demoKeepsSceneryGoal(navPoint{enemy.x, enemy.y}, distance) {
				return g.demoTarget, true
			}
			if distance < best {
				best = distance
				target = navPoint{enemy.x, enemy.y}
			}
		}
	}
	if best < 200 {
		return target, true
	}
	for _, item := range g.pickups {
		useful := item.kind == 1 && g.ammunition < 35 || item.kind == 0 && g.grenades < 15 || item.kind == 2 && g.player.inTank && g.player.tankFuel < 20 || item.kind == 6 && g.bulletPower[0] < 2 || item.kind == 7 && g.grenadePower[0] < 2
		if !useful {
			continue
		}
		distance := math.Abs(item.x-g.player.x) + math.Abs(item.y-g.player.y)
		if g.demoKeepsSceneryGoal(navPoint{item.x, item.y}, distance) {
			return g.demoTarget, true
		}
		if distance < best {
			best = distance
			target = navPoint{item.x, item.y}
		}
	}
	return target, best < 200
}

// A valid selected goal stays selected during its firing detour. Dropping it
// at the acquisition boundary can reverse the approach every two-pixel step.
// Its original 200-pixel range plus the 160-pixel firing search bounds retention.
func (g *Game) demoKeepsSceneryGoal(point navPoint, distance float64) bool {
	return g.demoPlanScenery && g.demoTargetKind == 1 && point == g.demoTarget && distance < 360
}

func (g *Game) demoStep(x, y, dx, dy float64) (float64, float64, bool) {
	return g.demoTerrainStep(x, y, dx, dy, false)
}

func (g *Game) demoTerrainStep(x, y, dx, dy float64, allowScenery bool) (float64, float64, bool) {
	nx, ny := x, y
	if dy > 0 && y-g.cameraY >= 171 && g.cameraY >= g.cameraBackY {
		dy = 0
		if dx == 0 {
			return x, y, false
		}
	}
	if dx != 0 {
		probe := x - 4
		if dx > 0 {
			probe = x + 8
		}
		code := g.terrainCode(probe, y)
		if g.player.inTank && tankWaterCode(code) || (!g.resources.passable[g.collisionTerrainBank()][code] || code > 0xec) && !(allowScenery && code > 0xec) {
			return x, y, false
		}
		speed := 2.0
		if code == 157 || code == 158 {
			speed = 1
		}
		if dx < 0 && x > 14 || dx > 0 && x < 194 {
			nx = x + dx*speed
		}
	}
	if dy != 0 {
		probe := y - 8
		if dy > 0 {
			probe = y + 4
		}
		code := g.terrainCode(nx, probe)
		if g.player.inTank && tankWaterCode(code) || (!g.resources.passable[g.collisionTerrainBank()][code] || code > 0xec) && !(allowScenery && code > 0xec) {
			return x, y, false
		}
		speed := 2.0
		if code == 157 || code == 158 {
			speed = 1
		}
		ny = y + dy*speed
	}
	if ny < 8 {
		return x, y, false
	}
	if nx == x && ny == y && (dx != 0 || dy != 0) {
		return x, y, false
	}
	for _, mine := range g.mines {
		dx, dy := mine.x-nx, mine.y-ny
		if dx >= -24 && dx <= 19 && dy >= -12 && dy <= 27 {
			oldX, oldY := mine.x-x, mine.y-y
			oldDepth := min(19-oldX, oldX+24, 27-oldY, oldY+12)
			newDepth := min(19-dx, dx+24, 27-dy, dy+12)
			// Once inside a hazard margin, allow a sidestep at equal depth
			// to get around a wall; entering or moving deeper remains unsafe.
			if oldDepth < 0 || newDepth > oldDepth {
				return x, y, false
			}
		}
	}
	return nx, ny, true
}

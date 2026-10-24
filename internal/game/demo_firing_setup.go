package game

// A remembered fixed gun can be cleared from a nearby firing pose. This
// searches only four ordinary setup inputs, checks the actual weapon path,
// and keeps existing projectiles/mines/body contacts in the hard forecast.
// The real controller must still reach that pose and fire on a later frame.
func (g *Game) demoKnownGunFiringSetup(x, y float64, normal demoMovePlan) (playerControls, bool, bool) {
	target := -1
	for i, enemy := range g.enemies {
		if enemy.kind == 2 && enemy.health > 0 && enemy.deadTicks == 0 && enemy.x == x && enemy.y == y {
			target = i
			break
		}
	}
	if target < 0 || target >= len(g.demoEnemyDefeat) {
		return playerControls{}, false, false
	}
	if target < 64 && g.demoPendingBulletTargets()&(uint64(1)<<target) != 0 {
		return playerControls{}, false, false
	}
	standing := playerControls{fire: true}
	source := g.player
	index, _, lethal := g.demoBulletImpact(g.projectileFor(&source, false, 0, false, 6))
	if index == target && lethal {
		plan := g.demoPlanAttackDetails(navPoint{g.player.x, g.player.y}, &standing, g.player.direction)
		if plan.survived == demoHorizon && plan.survived >= normal.survived {
			return standing, true, true
		}
	}
	type node struct {
		x, y    float64
		heading int
		first   playerControls
	}
	var current, next [256]node
	current[0] = node{x: g.player.x, y: g.player.y, heading: g.player.direction}
	count := 1
	directions := [...][2]float64{{0, -1}, {-1, -1}, {1, -1}, {-1, 0}, {1, 0}, {0, 1}, {-1, 1}, {1, 1}}
	for depth := 1; depth <= 4; depth++ {
		written := 0
		for _, previous := range current[:count] {
			for _, d := range directions {
				nx, ny, ok := g.demoStep(previous.x, previous.y, d[0], d[1])
				if !ok || nx == previous.x && ny == previous.y {
					continue
				}
				actor := g.player
				actor.x, actor.y = nx, ny
				actor.direction = directionForVector(d[0], d[1])
				if g.demoThreatForMove(actor, previous.y, depth) > 0 {
					continue
				}
				first := previous.first
				if depth == 1 {
					first = playerControls{x: d[0], y: d[1]}
				}
				duplicate := false
				for _, old := range next[:written] {
					if old.x == nx && old.y == ny && old.heading == actor.direction {
						duplicate = true
						break
					}
				}
				if duplicate {
					continue
				}
				if written < len(next) {
					next[written] = node{nx, ny, actor.direction, first}
					written++
				}
				shot := g.projectileFor(&actor, false, 0, false, 6)
				index, hit, lethal := g.demoBulletImpact(shot)
				if index != target || !lethal || depth+hit > demoHorizon {
					continue
				}
				// Setup never promises a future kill. Keep every known actor and
				// shot in the forecast, then recheck and fire on an actual later input.
				plan := g.demoPlanMoveDetails(navPoint{nx, ny}, &first)
				if plan.survived == demoHorizon {
					return first, false, true
				}
			}
		}
		if written == 0 {
			break
		}
		current, next = next, current
		count = written
	}
	return playerControls{}, false, false
}

package game

import "math"

// An experienced player clears these fixed guns from the dry approach,
// before entering the narrow approach and the slow river mine passage.
func (g *Game) demoKnownRiverGunners() (playerControls, bool) {
	if g.player.inTank || g.ammunition == 0 || g.player.y > 9360 || g.player.y < 8860 {
		return playerControls{}, false
	}
	for _, known := range [...][4]float64{{112, 9190, 104, 9280}, {170, 8920, 164, 9010}, {92, 8890, 100, 8986}} {
		if g.player.y > known[1]+180 || g.player.y < known[1]-30 {
			continue
		}
		alive := false
		for _, enemy := range g.enemies {
			if enemy.kind == 2 && enemy.health > 0 && enemy.deadTicks == 0 && enemy.x == known[0] && enemy.y == known[1] {
				alive = true
				break
			}
		}
		if !alive {
			continue
		}
		stage := navPoint{known[2], known[3]}
		// Retain the current walking lattice when a prior wet step changed parity.
		stage.x += math.Mod(g.player.x-stage.x+2, 2)
		stage.y += math.Mod(g.player.y-stage.y+2, 2)
		// Once the mine flank has gained ground, do not route back down
		// into the following pair merely to revisit the original dry point.
		if known[0] != 112 && g.player.y < stage.y {
			stage.y = g.player.y
		}
		// After clearing the right gun, keep a lateral standoff from the
		// following mines, move above their whole short jump envelope, then
		// cross toward the left gun. Returning straight to its old stage can
		// drag the player back into the pair after an earlier successful shot.
		if known[0] == 92 {
			nearest := -1
			for i, mine := range g.mines {
				if mine.kind != 19 || mine.x >= g.player.x || g.player.x-mine.x > 64 || g.player.y <= mine.y-40 || g.player.y >= mine.y+48 {
					continue
				}
				if nearest < 0 || mine.x > g.mines[nearest].x {
					nearest = i
				}
			}
			if nearest >= 0 {
				mine := g.mines[nearest]
				stage.x = max(g.player.x, min(194, mine.x+24))
				if g.player.x-mine.x <= 20 {
					stage.y = g.player.y
				} else {
					stage.x = g.player.x
					stage.y = min(stage.y, mine.y-40)
				}
			}
		}
		// A respawn can leave a moving mine on the normal approach. Open a
		// lateral gap, then advance around it without chasing it into the bank.
		if known[0] == 112 {
			for _, mine := range g.mines {
				if mine.kind != 19 || mine.slot != 84 || mine.y < 9240 || mine.y > 9360 || g.player.y < mine.y-32 {
					continue
				}
				stage = navPoint{184, known[3]}
				if g.player.x < mine.x+28 && g.player.x < 176 {
					stage.y = min(9350, max(g.player.y+20, mine.y+32))
				}
				break
			}
		}
		normalPlan := g.demoPlanMoveDetails(stage, nil)
		move := normalPlan.control
		if !g.demoFireHeld && g.bulletAvailable(0) {
			if aim, ok := g.demoAimForMove(move); ok {
				dx, dy := directionVector(aim)
				attack := playerControls{x: dx, y: dy, fire: true}
				locked := g.player.direction == aim
				if locked {
					attack = move
					attack.fire = true
				}
				attackPlan := g.demoPlanAttackDetails(stage, &attack, aim)
				if preferDemoAttack(normalPlan, attackPlan) {
					g.directionLock[0] = locked
					return attack, true
				}
			}
		}

		// A projected firing lane can leave the player just outside a real
		// shot angle. Look for a short ordinary setup and recheck next frame,
		// rather than waiting for an uncommitted future shot to disappear.
		if !g.demoFireHeld && g.bulletAvailable(0) {
			if setup, locked, found := g.demoKnownGunFiringSetup(known[0], known[1], normalPlan); found {
				g.directionLock[0] = locked
				return setup, true
			}
		}

		if math.Abs(g.player.x-stage.x) > 2 || math.Abs(g.player.y-stage.y) > 2 {
			g.directionLock[0] = false
			return move, true
		}
		attack := playerControls{fire: true}
		if g.player.direction != 0 {
			attack.y = -1
		}
		source := g.player
		source.direction = 0
		index, _ := g.demoBulletTarget(g.projectileFor(&source, false, 0, false, 6))
		pending := g.demoPendingBulletTargets()
		covered := false
		for i, enemy := range g.enemies {
			if i < 64 && enemy.kind == 2 && enemy.x == known[0] && enemy.y == known[1] && pending&(uint64(1)<<i) != 0 {
				covered = true
			}
		}
		if index >= 0 && !covered && !g.demoFireHeld && g.bulletAvailable(0) {
			attackPlan := g.demoPlanAttackDetails(stage, &attack, 0)
			if preferDemoAttack(normalPlan, attackPlan) {
				g.directionLock[0] = g.player.direction == 0
				return attack, true
			}
		}
		g.directionLock[0] = false
		return move, true
	}
	return playerControls{}, false
}

package game

// Plan an attack with its visible consequences already in the forecast.
// Ordinary bullets can clear an infantry contact; explosive attacks may create
// hostile fragments that require an escape before the weapon reaches its target.
// This copies ordinary game data; it does not fire a weapon or change resources.
func (g *Game) demoPlanAttack(target navPoint, attack *playerControls, aim int) (playerControls, float64) {
	plan := g.demoPlanAttackDetails(target, attack, aim)
	return plan.control, plan.score
}

func (g *Game) demoPlanAttackDetails(target navPoint, attack *playerControls, aim int) demoMovePlan {
	if !attack.action && !g.player.inTank {
		if !attack.fire {
			return g.demoPlanMoveDetails(target, attack)
		}
		source := g.player
		if x, _, ok := g.demoStep(source.x, source.y, attack.x, 0); ok {
			source.x = x
		}
		source.direction = aim
		shot := g.projectileFor(&source, false, 0, false, 6)
		index, hit, lethal := g.demoBulletImpact(shot)
		if !lethal || hit > demoHorizon || index < 0 || index >= len(g.demoEnemyDefeat) {
			return g.demoPlanMoveDetails(target, attack)
		}
		future := *g
		if !future.demoForecastActive {
			future.prepareDemoForecast()
		}
		// Input contact precedes this shot. Only subsequent inputs may
		// rely on the observed target's independently predicted lethal hit.
		// Already-fired hostile projectiles remain unchanged in this copy.
		future.demoEnemyDefeat[index] = hit
		plan := future.demoPlanMoveDetails(target, attack)
		plan.clearsActor = true
		return plan
	}
	source := g.player
	if x, _, ok := g.demoStep(source.x, source.y, attack.x, 0); ok {
		source.x = x
	}
	source.direction = aim
	if attack.action && !source.inTank && g.boardingTankAt(0, source.x, source.y) >= 0 {
		return demoMovePlan{score: demoBlockedScore}
	}
	speed := 6.0
	if attack.action {
		speed = 4
	}
	shot := g.projectileFor(&source, false, 0, attack.action, speed)
	var shots [64]projectile
	if len(g.projectiles) >= len(shots) {
		return demoMovePlan{score: demoBlockedScore}
	}
	count := copy(shots[:], g.projectiles)
	shots[count] = shot
	future := *g
	future.projectiles = shots[:count+1]
	future.prepareTargetBursts()
	return future.demoPlanMoveDetails(target, attack)
}

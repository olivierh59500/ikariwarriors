package game

// A damaged vehicle can request fragments while the shared pool is still
// occupied. Its constructor then creates an enemy blast immediately, before
// the enemy-quad damage pass in that same update. Forecast those three contact
// phases as well as the later queued fragments, using only visible damage.
func (g *Game) demoPendingBurstQuadHit(burst demoBurstPrediction, value actor, tick int) bool {
	request := burst.firstMove - 1
	if request < 1 || !g.burst.active || request > g.burst.remaining || tick < request || tick >= request+3 {
		return false
	}
	return grenadeHitsActor(projectile{x: burst.x, y: burst.y, enemy: true, owner: -1, grenade: true}, value)
}

// Existing fragments move before the vehicle cleanup pass. Predict a head
// hit while there is still time to stand clear: the transient damage bit is
// consumed by cleanup before the next complete gameplay frame is observed.
func (g *Game) prepareVehicleFragmentBursts() {
	if !g.burst.active {
		return
	}
	for group, vehicle := range g.enemyVehicles {
		if !vehicle.alive[0] || vehicle.damaged&1 != 0 {
			continue
		}
		for tick := 1; tick <= min(g.burst.remaining, demoHorizon); tick++ {
			head := vehicle.part(group, 0)
			hit := false
			for index, part := range g.burst.parts {
				motion := burstMotions[index]
				if smallObjectTouchesActor(part.x+motion.x*float64(tick), part.y+motion.y*float64(tick), head) {
					hit = true
					break
				}
			}
			if hit {
				g.addTargetBurst(head.x, head.y+7, tick+1)
				break
			}
			vehicle.advance(g.cameraY)
		}
	}
}

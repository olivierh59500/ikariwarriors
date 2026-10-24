package game

// The original advances past phase zero before presenting a complete frame.
// Three remaining phases correspond to the last three four-bob groups.
func grenadeImpactFrame(remaining, quadrant int) int {
	return 49 + (4-remaining)*4 + quadrant
}

func (g *Game) addProjectileScore(shot projectile, points int) {
	if shot.enemy {
		return
	}
	if shot.owner == 1 {
		g.score2 += points
	} else {
		g.score += points
	}
}

// grenadeFlightFrame follows the remaining-range cases in $fef4-$ffac.
// The grenade travels four pixels per axis for eighteen gameplay ticks.
func grenadeFlightFrame(life int) int {
	switch {
	case life >= 16:
		return 29
	case life >= 13:
		return 30
	case life >= 11:
		return 31
	case life >= 9:
		return 32
	case life >= 7:
		return 33
	case life >= 5:
		return 34
	case life >= 3:
		return 30
	default:
		return 29
	}
}

// The four explosion quadrants each use the original 12-by-12 collision
// probe against a 23-by-19 actor. The thrower is exempt from its own blast.
func grenadeHitsActor(shot projectile, value actor) bool {
	if value.health <= 0 || value.deadTicks > 0 {
		return false
	}
	for quadrant := 0; quadrant < 4; quadrant++ {
		dx := shot.x - 16 + float64(quadrant%2*16) - (value.x - 16)
		dy := shot.y - 16 + float64(quadrant/2*16) - (value.y - 4)
		if dx >= -12 && dx <= 23 && dy >= -12 && dy <= 19 {
			return true
		}
	}
	return false
}

func (g *Game) damageGrenadeArea(shot projectile) {
	g.damageFinalEncounter(shot, false)
	g.damageMiddleBoss(shot, false)
	g.damageEnemyVehicles(shot, false)
	g.damageAircraft(shot, false)
	for i := range g.enemies {
		enemy := &g.enemies[i]
		if enemy.kind == 6 {
			continue
		}
		if !grenadeHitsActor(shot, *enemy) {
			continue
		}
		enemy.health = 0
		if !shot.enemy {
			if shot.owner == 0 {
				g.score += 100
			} else {
				g.score2 += 100
			}
		}
	}
	for player := 0; player < g.playerCount; player++ {
		if !shot.enemy && shot.owner == player {
			continue
		}
		value := &g.player
		if player == 1 {
			value = &g.player2
		}
		if grenadeHitsActor(shot, *value) {
			g.hitPlayer(player)
		}
	}
	if !shot.enemy {
		// $10480/$10adc use the original shot's single terrain probe. The
		// four visible blast quadrants only define damage against actors.
		g.damageTerrain(shot.x+8, shot.y-4, shot.owner)
	}
}

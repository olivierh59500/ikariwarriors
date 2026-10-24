package game

// The staggered platoon's middle path retains the previous wide actor's leg
// selection ($12038/$ec20/$ea9e). Keep this original game compatibility rule
// in the simulation paint phase, independent of display refresh or Draw calls.
func (g *Game) rememberPlatoonMiddlePath() {
	lastSlot := -1
	selectLegs := func(slot int, x, y float64, lower int) {
		if slot < lastSlot || slot < 0 || slot >= 34 || !g.wideObjectInBounds(x, y) {
			return
		}
		lastSlot = slot
		g.platoonMiddlePath = 0
		if lower != 0 {
			g.platoonMiddlePath = byte(lower + 128)
		}
	}
	// Families use the painter's precedence when they share an original slot.
	if g.ending.active {
		for slot := 0; slot < 12; slot++ {
			part := g.endingPart(slot)
			selectLegs(slot, part.x, part.y, 0)
		}
	} else if g.final.active {
		for index := 0; index < 9; index++ {
			part := g.final.part(index)
			selectLegs(5+index, part.x, part.y, 0)
		}
	}
	for index, alive := range g.boss.alive {
		if alive {
			part := g.boss.part(index)
			selectLegs(26+index, part.x, part.y, 0)
		}
	}
	for group := range g.enemyVehicles {
		for index, alive := range g.enemyVehicles[group].alive {
			if alive {
				part := g.enemyVehicles[group].part(group, index)
				selectLegs(26+group*3+index, part.x, part.y, 0)
			}
		}
	}
	for index, alive := range g.aircraft.alive {
		if alive {
			part := g.aircraft.part(index)
			selectLegs(26+index, part.x, part.y, 0)
		}
	}
	for i := range g.enemies {
		enemy := &g.enemies[i]
		lower := 0
		if enemy.bob == 0 && (enemy.kind == 0 || enemy.kind == 5 || enemy.kind == 2 || enemy.kind == 3) {
			_, lower = enemy.infantryArtAddresses()
		}
		selectLegs(enemy.slot, enemy.x, enemy.y, lower)
		if enemy.kind == 4 && enemy.tankBurning {
			x, y := enemy.tankTurretPoint()
			selectLegs(25-enemy.tankDriver, x, y, 0)
		}
	}
	for player := 0; player < g.playerCount; player++ {
		value, bodySlot, turretSlot := &g.player, 23, 25
		if player == 1 {
			value, bodySlot, turretSlot = &g.player2, 22, 24
		}
		if value.inTank && !value.tankBurning {
			selectLegs(value.tankSlot, value.x+4, value.y+8, 0)
		}
		if !value.hidden && (value.health != 0 || value.deadTicks != 0) {
			lower := 0
			if value.deadTicks > 0 || !value.inTank {
				_, offset := value.playerArtOffsets(player)
				lower = 0x42b88 + offset
			}
			selectLegs(bodySlot, value.x, value.y, lower)
		}
		if value.inTank {
			x, y := value.tankTurretPoint()
			selectLegs(turretSlot, x, y, 0)
		}
	}
}

package game

// levelEvent is a six-byte record from the original scroll event table.
// Progress is measured in pixels, X is the original screen coordinate, and
// spacing separates actors that share one event.
type levelEvent struct {
	progress int
	x        int
	count    int
	kind     int
	spacing  int
}

func (g *Game) processLevelEvents() {
	progress := levelHeight - int(g.cameraY)
	for g.eventIndex < len(g.resources.events) {
		event := g.resources.events[g.eventIndex]
		if event.progress > progress {
			break
		}
		g.eventIndex++
		g.applyLevelEvent(event)
	}
}

// The original consumes one matching record when an upward camera step begins.
// Starting a game separately catches up events traversed by the landing scene.
func (g *Game) processScrollEvent() {
	if g.eventIndex >= len(g.resources.events) {
		return
	}
	event := g.resources.events[g.eventIndex]
	if event.progress != levelHeight-int(g.cameraY) {
		return
	}
	g.eventIndex++
	g.applyLevelEvent(event)
}

func (g *Game) slotActive(slot int) bool {
	if g.ending.active && slot >= 0 && slot < 12 || g.final.active && slot >= 5 && slot <= 13 {
		return true
	}
	if slot >= 26 && slot <= 30 && g.boss.alive[slot-26] {
		return true
	}
	if slot >= 26 && slot <= 31 && g.enemyVehicles[(slot-26)/3].alive[(slot-26)%3] {
		return true
	}
	if slot >= 26 && slot <= 30 && g.aircraft.alive[slot-26] {
		return true
	}
	if g.player.inTank && g.player.tankSlot == slot || g.playerCount > 1 && g.player2.inTank && g.player2.tankSlot == slot {
		return true
	}
	for _, enemy := range g.enemies {
		if enemy.slot == slot {
			return true
		}
	}
	return false
}

func (g *Game) placeEnemy(event levelEvent, slot, kind, x, topY int) {
	if g.slotActive(slot) {
		return
	}
	g.enemies = append(g.enemies, actor{
		x: float64(x - 48), y: float64(levelHeight - event.progress + topY - 4),
		kind: kind, slot: slot, health: 1, direction: 4,
		upper: walkAnimation{frame: 24}, lower: walkAnimation{frame: 32},
	})
	if kind == 0 || kind == 5 {
		value := &g.enemies[len(g.enemies)-1]
		value.upper, value.lower = g.infantryPoses[slot].upper, g.infantryPoses[slot].lower
		value.upper.frame = 0
	}
}

func (g *Game) applyLevelEvent(event levelEvent) {
	switch event.kind {
	case 24:
		g.spawnEnding(event)
	case 25:
		g.spawnFinalEncounter(event)
	case 30:
		g.spawnMiddleBoss(event)
	case 2, 21, 22, 23:
		g.spawnEnemyVehicles(event)
	case 4:
		g.spawnAircraft(event)
	case 1, 3, 5:
		first, last, kind := 3, 4, 2
		if event.kind == 1 {
			first, last, kind = 19, 21, 4
		} else if event.kind == 3 {
			first, last, kind = 0, 2, 3
		}
		remaining, x := event.count, event.x
		for slot := first; slot <= last && remaining > 0; slot++ {
			if g.slotActive(slot) {
				continue
			}
			g.placeEnemy(event, slot, kind, x, -14)
			enemy := &g.enemies[len(g.enemies)-1]
			if kind == 3 {
				enemy.upper.frame, enemy.lower.frame = 32, 40
			} else if kind == 4 {
				enemy.bob = 0x5a328 - 0x42b88
				enemy.tankFuel = 64
			}
			x += event.spacing
			remaining--
		}
	case 6, 7, 8, 9, 10, 11, 12, 14, 15, 16:
		// $eb20-$ec32 select each platoon's vertical and horizontal paths.
		patterns := map[int][7]int{
			6: {0, 0, 0, 0, 0, 0, 0}, 7: {1, 1, 1, 0, 0, 2, 2},
			8: {1, 0, 0, 0, 0, 0, 0}, 9: {0, 1, 1, 0, 0, 2, 2},
			10: {0, 1, 1, 1, 1, 1, 1}, 11: {0, 2, 2, 2, 2, 2, 2},
			12: {0, 0, 0, 0, 0, 0, 0}, 14: {0, 2, 2, 2, 1, 1, 1},
			15: {1, 2, 2, 2, 1, 1, 1}, 16: {0, 0, 0, 0, 0, 0, 0},
		}
		pattern := patterns[event.kind]
		for column := 0; column < 6; column++ {
			slot := 5 + column
			if g.slotActive(slot) {
				continue
			}
			x, topY := event.x+column*20, -5
			if event.kind == 12 && column >= 3 {
				x += 64
			}
			if event.kind == 16 {
				// $ed3a-$ee36 assign a staggered formation in slot order.
				x = [...]int{145, 165, 151, 171, 147, 167}[column]
				topY = -10 + column/2*5
			}
			g.placeEnemy(event, slot, 0, x, topY)
			enemy := &g.enemies[len(g.enemies)-1]
			enemy.marchY, enemy.marchX = pattern[0], pattern[column+1]
			if event.kind == 16 && column == 2 {
				enemy.marchX = int(g.platoonMiddlePath)
			}
			enemy.marchTime = 30
			if column == 2 || column == 3 {
				enemy.marchTime = 40
			}
			enemy.shots = int(g.randomValue() & 3)
			if event.kind == 12 {
				enemy.shots = 255
			}
			enemy.upper.frame = 0
		}
	case 17:
		for slot := 32; slot <= 33; slot++ {
			if g.slotActive(slot) {
				continue
			}
			g.placeEnemy(event, slot, 6, event.x, -14)
			g.enemies[len(g.enemies)-1].bob = 0x5b048 - 0x42b88
			break
		}
	case 252:
		g.randomSpawn = false
	case 18, 19, 20:
		first := 81 + (event.kind-18)*2
		for slot := first; slot < first+2; slot++ {
			busy := false
			for _, mine := range g.mines {
				if mine.slot == slot {
					busy = true
					break
				}
			}
			if !busy {
				g.mines = append(g.mines, landmine{x: float64(event.x - 56), y: float64(levelHeight - event.progress - 14), kind: event.kind, slot: slot})
				g.mineAnimations[slot-81].frame = -1
				if event.kind == 19 {
					g.mineAnimations[slot-81].frame = 12
				}
				break
			}
		}
	case 253:
		g.randomSpawn = true
	case 254:
		g.redSpawn = false
	case 255:
		g.redSpawn = true
	}
}

func (g *Game) aimEnemy(enemy *actor, target *actor) {
	dx, dy := 0.0, 0.0
	if target.x < enemy.x-16 {
		dx = -1
	} else if target.x > enemy.x+16 {
		dx = 1
	}
	if target.y < enemy.y-16 {
		dy = -1
	} else if target.y > enemy.y+16 {
		dy = 1
	}
	if dx == 0 && dy == 0 {
		dy = 1
	}
	enemy.face(directionForVector(dx, dy))
	if enemy.kind == 3 && dx != 0 && dy != 0 {
		enemy.face(directionForVector(0, dy))
	}
	poses := [...][2]int{{27, 35}, {31, 39}, {26, 34}, {29, 37}, {24, 32}, {28, 36}, {25, 33}, {30, 38}}
	pose := poses[enemy.direction]
	if enemy.kind == 3 {
		grenadier := [...][2]int{{33, 41}, {33, 41}, {35, 43}, {32, 40}, {32, 40}, {32, 40}, {34, 42}, {33, 41}}
		pose = grenadier[enemy.direction]
	}
	enemy.upper.frame, enemy.lower.frame = pose[0], pose[1]
}

func (g *Game) marchEnemy(enemy *actor) {
	g.marchEnemyWithRandom(enemy, g.randomValue)
}

func (g *Game) marchEnemyWithRandom(enemy *actor, random func() uint32) {
	direction, wet, moving := g.advanceInfantryMarch(enemy)
	if moving {
		// Soldier turns preserve each independent animation delay. The original
		// assigns the legs first, including the shared wet overlay, then the torso.
		enemy.direction = direction
		g.animateInfantryWalk(enemy, direction, wet)
		return
	}
	g.fireMarchingInfantry(enemy, random)
}

// $b25a creates reinforcements only when the map scrolls, independently of
// the scripted platoons. The red soldiers use a full-byte comparison to $3f.
func (g *Game) spawnScrollEnemies() {
	g.spawnScrollEnemiesAt(levelHeight - int(g.cameraY))
}

// Landing needs the two real scroll positions even though the playable
// player's world pose has already been prepared for the final camera.
func (g *Game) spawnScrollEnemiesAt(progress int) {
	g.lastSpawnScroll = progress
	if g.randomSpawn && g.randomValue()&15 == 1 {
		for slot := 11; slot <= 13; slot++ {
			g.spawnReinforcementAt(progress, slot, 0, 140+(slot-11)*10, -10)
		}
	}
	if g.redSpawn {
		for slot := 15; slot <= 18; slot++ {
			if byte(g.randomValue()) == 0x3f {
				g.spawnReinforcementAt(progress, slot, 5, 150+(slot-15)*10, 0)
			}
		}
	}
}

func (g *Game) spawnReinforcement(slot, kind, x, topY int) {
	g.spawnReinforcementAt(levelHeight-int(g.cameraY), slot, kind, x, topY)
}

func (g *Game) spawnReinforcementAt(progress, slot, kind, x, topY int) {
	if g.slotActive(slot) {
		return
	}
	event := levelEvent{progress: progress}
	worldX := float64(x - 48)
	worldY := float64(levelHeight - progress + topY - 4)
	if g.solidAt(worldX, worldY) {
		return
	}
	g.placeEnemy(event, slot, kind, x, topY)
	enemy := &g.enemies[len(g.enemies)-1]
	for {
		if kind == 5 {
			enemy.marchY = 1
		} else {
			enemy.marchY = int(g.randomValue() & 1)
		}
		value := int(g.randomValue() & 3)
		if value != 0 {
			enemy.marchX = value - 1
			break
		}
		// The normal initializer retries its vertical choice as well. The
		// red initializer has no vertical random draw to discard or repeat.
	}
	enemy.marchTime = 30
	enemy.shots = int(g.randomValue() & 3)
	enemy.upper.frame = 0
}

func (g *Game) enemyBulletAvailable() bool {
	count := 0
	for _, shot := range g.projectiles {
		if shot.enemy {
			count++
		}
	}
	return count < 12
}

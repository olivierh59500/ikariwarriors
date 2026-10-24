package game

// $adf2 prepares the first inactive shared enemy shot. Its second random
// draw exists only when the first draw selected an explosive projectile.
// Position, velocity and activation belong to the caller, not this allocator.
func (g *Game) allocateInfantryShot(random func() uint32) (projectile, bool) {
	var occupied [12]bool
	for _, shot := range g.projectiles {
		if shot.enemy {
			occupied[shot.slot] = true
		}
	}
	slot := -1
	for candidate, busy := range occupied {
		if !busy {
			slot = candidate
			break
		}
	}
	if slot < 0 {
		return projectile{}, false
	}
	shot := projectile{enemy: true, owner: -1, slot: slot, life: 15, power: 1, frame: 1}
	if random()&15 == 0 {
		shot.grenade, shot.frame = true, 29
		if random()&7 == 0 {
			shot.fragments, shot.frame = true, 35
		}
	}
	return shot, true
}

// $a9bc-$adee pauses a platoon's movement while it fires. The original
// selects its actual firing direction independently of its marching path.
// A successful wet shot retries immediately, preserving the previous legs.
func (g *Game) fireMarchingInfantry(enemy *actor, random func() uint32) {
	directions := [...]int{4, 5, 3}
	poses := [...][2]int{{24, 32}, {28, 36}, {29, 37}}
	if enemy.kind == 5 {
		poses = [...][2]int{{7, 8}, {13, 16}, {16, 21}}
	}
	index := 0
	if enemy.marchX == 1 {
		index = 1
	} else if enemy.marchX == 2 {
		index = 2
	}
	enemy.direction = directions[index]
	enemy.upper.frame = poses[index][0]
	code := g.terrainCode(enemy.x, enemy.y)
	wet := code == 157 || code == 158
	if !wet {
		enemy.lower.frame = poses[index][1]
	}
	for {
		index = 0
		if random()&3 == 1 {
			index = 1
		} else if random()&3 == 2 {
			index = 2
		}
		if random()&31 != 0 {
			return
		}
		shot, ok := g.allocateInfantryShot(random)
		if !ok {
			return
		}
		if enemy.kind == 5 {
			// Red soldiers consume the same allocation draws, then overwrite
			// an explosive result with the original ordinary bullet image.
			shot.grenade, shot.fragments, shot.frame = false, false, 1
		}
		// Marching soldiers use rawX+8, unlike the fixed gunners' rawX+4.
		// The native small-bob coordinate therefore equals actor.x here.
		shot.x, shot.y = enemy.x, enemy.y+12
		shot.vx, shot.vy = [...]float64{0, -4, 4}[index], 4
		g.projectiles = append(g.projectiles, shot)
		enemy.shots = int(uint8(enemy.shots - 1))
		enemy.direction = directions[index]
		enemy.upper.frame = poses[index][0]
		if !wet {
			enemy.lower.frame = poses[index][1]
			if enemy.kind == 5 && index == 1 {
				enemy.lower.frame = 17
			}
		}
		if g.audio != nil {
			g.audio.Effect(0, 2)
		}
		if !wet {
			return
		}
		// At most twelve free enemy slots can be consumed before the
		// allocator ends this retry, including a byte-count underflow.
	}
}

package game

// These five unrendered large contact records are separate from the six
// small landmines. The original initializes their positions at $82f0 and
// preserves each cooldown byte when a later game resets their activity.
type fanTrap struct {
	x, y     float64
	cooldown int
	active   bool
}

var fanTrapPositions = [...]navPoint{
	{124 - 48, levelHeight - 4 - 1172},
	{154 - 48, levelHeight - 4 - 11220},
	{90 - 48, levelHeight - 4 - 11316},
	{234 - 48, levelHeight - 4 - 11316},
	{154 - 48, levelHeight - 4 - 11412},
}

func (g *Game) resetFanTraps() {
	for i, position := range fanTrapPositions {
		g.fanTraps[i].x, g.fanTraps[i].y = position.x, position.y
		g.fanTraps[i].active = true
	}
}

// $9330 checks the second player before the first, and stops at the first
// contacting trap even when its countdown has not reached zero. Protection
// and tank occupancy do not prevent this contact; the player's active/dead
// state is the predicate used by the original $c768 rectangle helper.
func (g *Game) updateFanTraps() {
	players := [2]*actor{&g.player2, &g.player}
	for i := range g.fanTraps {
		trap := &g.fanTraps[i]
		if !trap.active {
			continue
		}
		for player, value := range players {
			if player == 0 && g.playerCount < 2 || value.health <= 0 && value.deadTicks == 0 {
				continue
			}
			dx, dy := trap.x-value.x, trap.y-value.y
			if dx < -48 || dx > 37 || dy < -80 || dy > 37 {
				continue
			}
			trap.cooldown = int(uint8(trap.cooldown - 1))
			if trap.cooldown != 0 {
				return
			}
			trap.cooldown = 15
			if g.audio != nil {
				g.audio.Effect(13, 3)
			}
			// $ae14 supplies an ordinary bullet without drawing randomness.
			// Each allocation takes the first free shared enemy slot. A full
			// pool aborts the remaining directions but keeps the reset above.
			for _, vx := range [...]float64{0, 4, -4} {
				count := len(g.projectiles)
				g.addEnemyProjectile(projectile{
					x: trap.x - 8, y: trap.y + 4, vx: vx, vy: 4,
					enemy: true, owner: -1, life: 15, power: 1, frame: 1,
				})
				if len(g.projectiles) == count {
					return
				}
			}
			return
		}
	}
}

// $c768 uses a 23-by-19 wide contact against the constructor's 80-by-80
// temporary probe. Native wide and small origins differ by eight X pixels
// and four Y pixels, producing these asymmetric inclusive rectangle edges.
func sceneryConstructorTouches(origin navPoint, x, y float64) bool {
	dx, dy := x-origin.x, y-origin.y
	return dx >= -15 && dx <= 88 && dy >= -23 && dy <= 76
}

// $114f6-$11598 also runs while radial fragments own the shared effect pool.
// Clearing an actor flag removes the gun immediately; it does not award a
// kill, drop an item or start the infantry death sequence.
func (g *Game) clearSceneryConstructorContacts(origin navPoint, blocked bool) {
	if !blocked {
		for i, enemy := range g.enemies {
			if enemy.slot == 32 && sceneryConstructorTouches(origin, enemy.x, enemy.y) {
				g.enemies = append(g.enemies[:i], g.enemies[i+1:]...)
				return
			}
		}
	}
	// This source flag clear is unconditional when the first check failed.
	for i, enemy := range g.enemies {
		if enemy.slot == 33 {
			g.enemies = append(g.enemies[:i], g.enemies[i+1:]...)
			break
		}
	}
	if blocked {
		return
	}
	for i := range g.fanTraps {
		trap := &g.fanTraps[i]
		if trap.active && sceneryConstructorTouches(origin, trap.x, trap.y) {
			trap.active = false
			return
		}
	}
}

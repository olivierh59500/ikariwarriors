package game

import "github.com/hajimehoshi/ebiten/v2"

// Vehicle and rotor owners share these retained animation clocks.
type vehicleChassisPhase struct{ step, delay int }

// Each enemy tank owns a chassis, its displaced partner and a separate turret.
// The two groups reuse the same six wide-object slots as the helicopter.
type enemyVehicle struct {
	x, y    float64
	mode    int
	turn    byte
	reverse byte
	active  bool
	alive   [3]bool
	chassis walkAnimation
	turret  int
	damaged byte
	owner   int
}

func (v *enemyVehicle) part(group, index int) actor {
	x, y := v.x, v.y
	address := 0x60e08 + v.chassis.frame*vehicleBobBytes
	if index == 1 {
		x, y, address = x+4, y+8, address+vehicleBobBytes
	} else if index == 2 {
		address = v.turret
	}
	return actor{x: x, y: y, health: 1, slot: 26 + group*3 + index, bob: address - 0x42b88}
}

func (g *Game) spawnEnemyVehicles(event levelEvent) {
	mode := 0
	if event.kind >= 21 {
		mode = event.kind - 20
	}
	remaining, x := min(2, event.count), event.x
	for group := range g.enemyVehicles {
		if remaining == 0 {
			break
		}
		if g.enemyVehicles[group].active || g.slotActive(26+group*3) {
			continue
		}
		reverse := byte(1)
		if mode == 3 {
			reverse = 255
		}
		phase := g.vehicleChassisPhases[group]
		g.enemyVehicles[group] = enemyVehicle{
			x: float64(x - 48), y: float64(levelHeight - event.progress - 18),
			mode: mode, reverse: reverse, active: true, alive: [3]bool{true, true, true},
			chassis: walkAnimation{frame: 16, step: phase.step, delay: phase.delay}, turret: 0x59ba8, owner: -1,
		}
		x += event.spacing
		remaining--
	}
}

// $99f8 selects a two-pixel vector, reverses it every forty updates and
// advances the chassis every other update. It does not use infantry terrain
// probes. The original tests the sign for movement and zero for animation.
func (v *enemyVehicle) advance(cameraY float64) {
	if !v.active {
		return
	}
	// $9a5c restores all three parts during an entered movement pass.
	// Even its final raw-Y>250 pass can fire before late painting retires them.
	v.alive = [3]bool{true, true, true}
	v.turn++
	if v.turn >= 40 {
		v.turn = 0
		v.reverse = ^v.reverse
	}
	motion := [...]navPoint{{2, 0}, {0, 2}, {2, 2}, {-2, 2}}[v.mode]
	if int8(v.reverse) < 0 {
		motion.x, motion.y = -motion.x, -motion.y
	}
	v.x, v.y = v.x+motion.x, v.y+motion.y
	sequences := [...][2][]int{
		{{16, 18, 20, 22}, {22, 20, 18, 16}},
		{{0, 2, 4, 6}, {6, 4, 2, 0}},
		{{12, 14}, {14, 12}},
		{{8, 10}, {10, 8}},
	}
	index := 0
	if v.reverse != 0 {
		index = 1
	}
	v.chassis.advance(sequences[v.mode][index])
	if v.y-cameraY+4 > 250 {
		v.active = false
	}
}

func (g *Game) updateEnemyVehicles() {
	for group := range g.enemyVehicles {
		v := &g.enemyVehicles[group]
		if v.damaged&1 != 0 {
			g.createBurst(v.x, v.y+7, -1)
			v.active, v.alive, v.damaged = false, [3]bool{}, 0
		}
		entered := v.active
		v.advance(g.cameraY)
		if entered {
			g.vehicleChassisPhases[group] = vehicleChassisPhase{v.chassis.step, v.chassis.delay}
		}
	}
}

func (g *Game) damageEnemyVehicles(shot projectile, fragments bool) {
	for group := range g.enemyVehicles {
		v := &g.enemyVehicles[group]
		for index, alive := range v.alive {
			if !alive {
				continue
			}
			part := v.part(group, index)
			hit := grenadeHitsActor(shot, part)
			if fragments {
				hit = smallObjectTouchesActor(shot.x, shot.y, part)
			}
			if hit {
				if v.damaged&(1<<index) == 0 {
					g.addProjectileScore(shot, 100)
				}
				v.damaged |= 1 << index
				v.owner = shot.owner
				if shot.enemy {
					v.owner = -1
				}
			}
		}
	}
}

func (g *Game) fireEnemyVehicles() {
	if g.boss.active || g.aircraft.flying || !g.enemyBulletAvailable() {
		return
	}
	for _, shot := range g.projectiles {
		if shot.enemy && shot.cannon && shot.impact == 0 {
			return
		}
	}
	for group := range g.enemyVehicles {
		v := &g.enemyVehicles[group]
		if !v.alive[2] {
			continue
		}
		dx, dy := g.enemyFiringVector(v.x, v.y)
		order := [...]int{0, 7, 6, 5, 4, 3, 2, 1}
		v.turret = 0x59428 + order[directionForVector(dx, dy)]*vehicleBobBytes
		if g.randomValue()&31 != 0 {
			continue
		}
		v.reverse = ^v.reverse
		v.turn = 40 - v.turn
		g.addEnemyProjectile(projectile{
			x: v.x - 4, y: v.y + 12, vx: dx, vy: dy,
			enemy: true, owner: -1, grenade: true, cannon: true, life: 15, frame: 7,
		})
		return
	}
}

func (g *Game) enemyFiringVector(x, y float64) (float64, float64) {
	target := &g.player
	if g.randomValue()&1 != 0 && g.playerCount > 1 && g.player2.health > 0 {
		target = &g.player2
	}
	dx, dy := 0.0, 0.0
	if target.x < x-16 {
		dx = -4
	} else if target.x > x+16 {
		dx = 4
	}
	if target.y < y-16 {
		dy = -4
	} else if target.y > y+16 {
		dy = 4
	}
	if dx == 0 && dy == 0 {
		dy = 4
	}
	return dx, dy
}

func (g *Game) drawEnemyVehiclePart(slot int) {
	if slot < 26 || slot > 31 {
		return
	}
	group, index := (slot-26)/3, (slot-26)%3
	v := &g.enemyVehicles[group]
	if !v.alive[index] {
		return
	}
	part := v.part(group, index)
	if sprite := g.resources.bob(part.bob, vehicleBobHeight); sprite != nil {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(part.x-actorBobWidth/2, part.y-g.cameraY-4)
		g.playfield.DrawImage(sprite, op)
	}
}

// The wide painter retires each part after gameplay. A moving group can
// restore it on the following tick; a stopped group leaves it inactive.
func (g *Game) pruneOffscreenEnemyVehicles() {
	for group := range g.enemyVehicles {
		v := &g.enemyVehicles[group]
		for index, alive := range v.alive {
			if !alive {
				continue
			}
			part := v.part(group, index)
			if !g.wideObjectInBounds(part.x, part.y) {
				v.alive[index] = false
			}
		}
	}
}

// $e6f2 clears the first five shared part marks after enemy explosions and
// firing. The second turret lies outside that pass and retains its hit mark.
func (g *Game) finishEnemyVehicleDamage() {
	g.enemyVehicles[0].damaged = 0
	g.enemyVehicles[1].damaged &= 4
}

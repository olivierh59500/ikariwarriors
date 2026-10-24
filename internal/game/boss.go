package game

import "github.com/hajimehoshi/ebiten/v2"

// The middle encounter is a four-part vehicle with a separately positioned
// turret. It has six hit rounds and a fifty-update vertical oscillator.
type middleBoss struct {
	x, y         float64
	active       bool
	alive        [5]bool
	turn         int
	direction    int8
	health       int
	turretOffset navPoint
	turretBob    int
	damaged      byte
	owner        int
}

var bossTurrets = [8]struct {
	x, y, bob int
}{
	{15, 8, 0x687e8}, {13, 12, 0x68428}, {16, 16, 0x689c8}, {13, 15, 0x68068},
	{15, 16, 0x68608}, {11, 15, 0x67e88}, {6, 16, 0x68ba8}, {11, 13, 0x68248},
}

func (g *Game) spawnMiddleBoss(event levelEvent) {
	g.aircraft, g.enemyVehicles = helicopter{}, [2]enemyVehicle{}
	g.boss = middleBoss{
		x: 80, y: float64(levelHeight - event.progress - 52),
		active: true, alive: [5]bool{true, true, true, true, true},
		direction: 1, health: 6, turretOffset: navPoint{16, 12}, turretBob: 0x68608, owner: -1,
	}
}

func (b *middleBoss) part(index int) actor {
	x, y := b.x+float64(index%2*32), b.y+float64(index/2*24)
	address := 0x68d88 + index*vehicleBobBytes
	if index == 4 {
		x, y, address = b.x+b.turretOffset.x, b.y+b.turretOffset.y, b.turretBob
	}
	return actor{x: x, y: y, health: 1, slot: 26 + index, bob: address - 0x42b88}
}

func (b *middleBoss) advance(cameraY float64) {
	if !b.active {
		return
	}
	if b.y-cameraY+4 >= 200 {
		b.active, b.alive = false, [5]bool{}
		return
	}
	b.y += float64(b.direction) * 2
	b.turn++
	if b.turn == 50 {
		b.turn, b.direction = 0, -b.direction
	}
}

// $e6f2 consumes one health point for a round containing any part hits.
// Several parts touched by the same round still consume only one point.
func (b *middleBoss) applyDamage() {
	if !b.active || b.damaged == 0 {
		return
	}
	b.health--
	if b.health == 0 {
		b.active = false
		return
	}
	b.damaged = 0
}

func (g *Game) updateMiddleBoss() {
	b := &g.boss
	if !b.active {
		// Once the boss controller stops, the original shared group cleanup
		// handles its surviving bobs and the final round's retained flags.
		if b.damaged&1 != 0 {
			part := b.part(0)
			g.createBurst(part.x, part.y+7, -1)
			b.alive[0], b.alive[1], b.alive[2] = false, false, false
		}
		if b.damaged&8 != 0 {
			part := b.part(3)
			g.createBurst(part.x, part.y+7, -1)
			b.alive[3], b.alive[4] = false, false
		}
		b.damaged = 0
		return
	}
	b.advance(g.cameraY)
	if !b.active || !g.enemyBulletAvailable() {
		return
	}
	turret := b.part(4)
	dx, dy := g.enemyFiringVector(turret.x, turret.y)
	pose := bossTurrets[directionForVector(dx, dy)]
	b.turretOffset, b.turretBob = navPoint{float64(pose.x), float64(pose.y)}, pose.bob
	if g.randomValue()&63 == 0 {
		b.direction, b.turn = -b.direction, 50-b.turn
		turret = b.part(4)
		g.addEnemyProjectile(projectile{
			x: turret.x - 4, y: turret.y + 12, vx: dx, vy: dy,
			enemy: true, owner: -1, grenade: true, life: 18, frame: 8, fragments: true,
		})
	}
}

func (g *Game) damageMiddleBoss(shot projectile, fragments bool) {
	b := &g.boss
	for index, alive := range b.alive {
		if !alive {
			continue
		}
		part := b.part(index)
		hit := grenadeHitsActor(shot, part)
		if fragments {
			hit = smallObjectTouchesActor(shot.x, shot.y, part)
		}
		if hit {
			if b.damaged&(1<<index) == 0 {
				g.addProjectileScore(shot, 100)
			}
			b.damaged |= 1 << index
			b.owner = shot.owner
			if shot.enemy {
				b.owner = -1
			}
		}
	}
}

func (g *Game) drawMiddleBossPart(slot int) {
	i := slot - 26
	if i < 0 || i >= 5 || !g.boss.alive[i] {
		return
	}
	part := g.boss.part(i)
	if sprite := g.resources.bob(part.bob, vehicleBobHeight); sprite != nil {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(part.x-actorBobWidth/2, part.y-g.cameraY-4)
		g.playfield.DrawImage(sprite, op)
	}
}

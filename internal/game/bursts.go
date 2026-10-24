package game

import "github.com/hajimehoshi/ebiten/v2"

type burstRequest struct {
	x, y  float64
	owner int
}

type radialBurst struct {
	parts     [8]navPoint
	remaining int
	owner     int
	active    bool
}

var burstMotions = [8]navPoint{{-8, -8}, {0, -8}, {8, -8}, {-8, 0}, {8, 0}, {-8, 8}, {0, 8}, {8, 8}}
var burstFrames = [8]int{45, 41, 46, 43, 44, 47, 42, 48}

// $11290 owns one eight-bob burst pool. Further requests wait on a stack.
func (g *Game) createBurst(x, y float64, owner int) {
	if g.burst.active {
		g.burstQueue = append(g.burstQueue, burstRequest{x, y, owner})
		g.startBlast(projectile{x: x, y: y, owner: -1, enemy: true, grenade: true})
		return
	}
	g.burst = radialBurst{remaining: 6, owner: owner, active: true}
	if g.audio != nil {
		g.audio.Effect(9, 3)
	}
	for i := range g.burst.parts {
		g.burst.parts[i] = navPoint{x, y}
	}
}

func (g *Game) updateBurst() {
	b := &g.burst
	if !b.active {
		return
	}
	if b.remaining == 0 {
		b.active = false
		if n := len(g.burstQueue); n > 0 {
			next := g.burstQueue[n-1]
			g.burstQueue = g.burstQueue[:n-1]
			g.createBurst(next.x, next.y, next.owner)
		}
		return
	}
	for i, motion := range burstMotions {
		part := &b.parts[i]
		if i == 0 || i == 7 {
			// These two original directions add a packed coordinate long;
			// retain its word carry when the screen Y crosses zero.
			packed := uint32(uint16(int16(part.x+56)))<<16 | uint32(uint16(int16(part.y-g.cameraY)))
			if i == 0 {
				packed -= 0x80008
			} else {
				packed += 0x80008
			}
			part.x, part.y = float64(int16(packed>>16))-56, float64(int16(packed))+g.cameraY
		} else {
			part.x += motion.x
			part.y += motion.y
		}
		g.damageBurstPart(*part, b.owner)
	}
	b.remaining--
}

func (g *Game) damageBurstPart(part navPoint, owner int) {
	for i := range g.enemies {
		enemy := &g.enemies[i]
		if enemy.slot < 32 && smallObjectTouchesActor(part.x, part.y, *enemy) {
			enemy.health = 0
		}
	}
	for player := 0; player < g.playerCount; player++ {
		if player == owner {
			continue
		}
		value := &g.player
		if player == 1 {
			value = &g.player2
		}
		if smallObjectTouchesActor(part.x, part.y, *value) {
			g.hitPlayer(player)
		}
	}
	g.damageAircraft(projectile{x: part.x, y: part.y, owner: owner, enemy: owner < 0}, true)
	g.damageFinalEncounter(projectile{x: part.x, y: part.y, owner: owner, enemy: owner < 0}, true)
	g.damageMiddleBoss(projectile{x: part.x, y: part.y, owner: owner, enemy: owner < 0}, true)
	g.damageEnemyVehicles(projectile{x: part.x, y: part.y, owner: owner, enemy: owner < 0}, true)
}

func (g *Game) drawBurst() {
	if !g.burst.active {
		return
	}
	for i, part := range g.burst.parts {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(part.x-8, part.y-g.cameraY-8)
		g.playfield.DrawImage(g.resources.effects[burstFrames[i]], op)
	}
}

package game

import "github.com/hajimehoshi/ebiten/v2"

var tankFlameSequence = []int{0, 1, 2, 3}

// These offsets are the original $12eca-$12f18 chassis animation tables,
// expressed as 480-byte, 32-by-24 graphic indices. The turret is a separate bob.
var tankChassisSequences = [8][]int{
	{0, 2, 4, 6}, {8, 10}, {22, 20, 18, 16}, {14, 12},
	{6, 4, 2, 0}, {10, 8}, {16, 18, 20, 22}, {12, 14},
}

func (a *actor) animateMovement(dx, dy float64, locked bool) {
	a.animatePlayerMovement(0, dx, dy, locked)
}

func (a *actor) animatePlayerMovement(player int, dx, dy float64, locked bool) {
	direction := directionForVector(dx, dy)
	if direction != a.moveDirection {
		a.upper.delay, a.lower.delay = 0, 0
	}
	a.moveDirection = direction
	if a.inTank {
		if !a.tankBurning {
			a.upper.advance(tankChassisSequences[direction])
		}
		return
	}
	advancePlayerWalk(&a.upper, &a.lower, player, direction, a.wet, locked)
}

// The original scroll routines move two pixels without snapping the actor.
// Retreat is limited to the current thirty-two-pixel strip retained by the
// game, which also preserves the original lower-screen behavior ($893e/$8a52).
func (g *Game) followPlayer(previousY, currentY float64) {
	previousCamera := g.cameraY
	if currentY < previousY && previousY-g.cameraY < 96 && g.cameraY > 0 {
		g.idleArtilleryTicks = 0
		g.processScrollEvent()
		g.cameraY = max(0, g.cameraY-2)
		back := levelHeight - float64((int(levelHeight-g.cameraY)-1)&^31)
		g.cameraBackY = min(g.cameraBackY, back)
	} else if currentY > previousY && previousY-g.cameraY >= 157 && g.cameraY < g.cameraBackY {
		g.idleArtilleryTicks = 0
		g.cameraY = min(g.cameraBackY, g.cameraY+2)
	}
	// The original's last map-strip request enters its victory routine before
	// any later gameplay phase or a replacement playfield is drawn.
	if g.ending.active && previousCamera >= 32 && g.cameraY < 32 {
		g.enterVictory()
		return
	}
	if !g.secondaryTerrain && g.cameraY < secondaryMapRows*tileSize {
		g.secondaryTerrain = true
		// Reachable routes were computed with the previous terrain rules.
		g.demoPath = g.demoPath[:0]
	}
	// Reinforcements belong to each accepted upward scroll, before this
	// player's owned shots and the next player's input. Revisits count too.
	if g.cameraY < previousCamera {
		g.spawnScrollEnemies()
	}
}

// A held fire button enters an overlapping empty tank, or leaves the current
// tank, before the grenade action is considered ($c7f8-$ca20).
func (g *Game) useTank(player int) bool {
	value := &g.player
	if player == 1 {
		value = &g.player2
	}
	if value.deadTicks > 0 || value.health <= 0 {
		return false
	}
	if value.inTank {
		g.enemies = append(g.enemies, actor{
			x: value.x, y: value.y, kind: 4, slot: value.tankSlot,
			tankTurret: value.tankTurret, tankTurretPlaced: value.tankTurretPlaced,
			health: 1, bob: 0x5a328 - 0x42b88, tankFuel: value.tankFuel, tankBurn: value.tankBurn, tankBurning: value.tankBurning, tankFlame: value.tankFlame, tankDriver: player,
		})
		value.inTank, value.tankBurning, value.tankBurn = false, false, 0
		value.upper.frame, value.lower.frame = 8, 12
		return true
	}
	if i := g.boardingTankAt(player, value.x, value.y); i >= 0 {
		tank := g.enemies[i]
		value.x, value.y = tank.x, tank.y
		value.invulnerable = 0
		value.inTank, value.tankSlot, value.tankFuel = true, tank.slot, tank.tankFuel
		// Boarding places the turret before fuel processing ($c8bc/$c8c2).
		value.tankTurret, value.tankTurretPlaced = navPoint{value.x, value.y}, true
		value.fuelGaugeFull = false
		value.direction, value.moveDirection = 0, 0
		value.upper.frame = 0
		g.enemies = append(g.enemies[:i], g.enemies[i+1:]...)
		return true
	}
	return false
}

func (g *Game) hasBurningTank(player int) bool {
	for _, enemy := range g.enemies {
		if enemy.kind == 4 && enemy.tankBurning && enemy.tankDriver == player {
			return true
		}
	}
	return false
}

// The original context action chooses the first overlapping slot19..21, then
// checks its ownership. It does not skip an unavailable tank to take another
// overlapping one. A driver's burning tank also blocks new boarding.
func (g *Game) boardingTankAt(player int, x, y float64) int {
	if g.hasBurningTank(player) {
		return -1
	}
	overlaps := func(tx, ty float64) bool {
		dx, dy := x-tx, y-ty
		return dx >= -23 && dx <= 23 && dy >= -19 && dy <= 19
	}
	for slot := 19; slot <= 21; slot++ {
		for other := 0; other < g.playerCount; other++ {
			value := &g.player
			if other == 1 {
				value = &g.player2
			}
			if other != player && value.inTank && !value.tankBurning && value.tankSlot == slot && overlaps(value.x+4, value.y+8) {
				return -1
			}
		}
		for i, enemy := range g.enemies {
			if enemy.slot != slot || enemy.kind != 4 || enemy.health <= 0 || !overlaps(enemy.x, enemy.y) {
				continue
			}
			if enemy.tankBurning {
				return -1
			}
			return i
		}
	}
	return -1
}

// Original wide actors are composited in slot order before the small objects.
// A boarded tank occupies the reserved empty-tank slot, the player body slot
// and a separate turret slot. The adjacent chassis graphic is a displaced
// partner, not an unused animation frame ($8b2c-$8b68).
func (g *Game) drawWideActors() {
	for slot := 0; slot < 34; slot++ {
		g.drawFinalWideSlot(slot)
		g.drawMiddleBossPart(slot)
		g.drawEnemyVehiclePart(slot)
		g.drawAircraftPart(slot)
		for i := range g.enemies {
			enemy := &g.enemies[i]
			if enemy.slot == slot {
				g.drawActor(g.playfield, enemy, -1)
			}
			if enemy.kind == 4 && enemy.tankBurning && slot == 25-enemy.tankDriver {
				g.drawTankPart(enemy, enemy.tankDriver, true)
			}
		}
		for player := 0; player < g.playerCount; player++ {
			value, bodySlot, turretSlot := &g.player, 23, 25
			if player == 1 {
				value, bodySlot, turretSlot = &g.player2, 22, 24
			}
			if value.inTank && !value.tankBurning && slot == value.tankSlot {
				g.drawTankPart(value, player, false)
			}
			if slot == bodySlot {
				g.drawActor(g.playfield, value, player)
			}
			if value.inTank && slot == turretSlot {
				g.drawTankPart(value, player, true)
			}
		}
	}
}

func (g *Game) drawTankPart(value *actor, player int, turret bool) {
	bank := 0x5b408
	if player == 1 {
		bank = 0x5e108
	}
	offset := bank - 0x42b88 + (value.upper.frame+1)*vehicleBobBytes
	x, y := value.x-actorBobWidth/2+4, value.y-g.cameraY-4+8
	if turret {
		bank = 0x57628
		if player == 1 {
			bank = 0x58528
		}
		order := [...]int{0, 7, 6, 5, 4, 3, 2, 1}
		offset = bank - 0x42b88 + order[value.direction&7]*vehicleBobBytes
		turretX, turretY := value.tankTurretPoint()
		x, y = turretX-actorBobWidth/2, turretY-g.cameraY-4
	}
	if turret && value.tankBurning {
		offset = 0x5a6e8 - 0x42b88 + value.tankFlame.frame*vehicleBobBytes
	}
	if sprite := g.resources.bob(offset, vehicleBobHeight); sprite != nil {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(x, y)
		g.playfield.DrawImage(sprite, op)
	}
}

// Burning vehicles and fuel follow the controls and moving fragment pass.
// Each player keeps a separate consumption phase when boarding another tank.
func (g *Game) advanceTanks() {
	for player := 0; player < 2; player++ {
		g.advanceTankBurn(player)
	}
	for player := 0; player < 2; player++ {
		g.advanceTankFuel(player)
	}
}

func (g *Game) advanceTankFuel(player int) {
	value := &g.player
	if player == 1 {
		value = &g.player2
	}
	if !value.inTank {
		return
	}
	phase := &g.tankFuelPhase[player]
	if !value.tankBurning && *phase == 0 {
		*phase = 15
		value.tankFuel = int(uint8(value.tankFuel) - 1)
		if value.tankFuel == 1 {
			g.hitPlayer(player)
			return
		}
		value.fuelGaugeFull = false
	}
	*phase--
	// $8b32/$8bd2 place the turret after ordinary fuel processing. A new
	// fuel-exhaustion hit returns before this block and retains its last pose.
	value.tankTurret, value.tankTurretPlaced = navPoint{value.x, value.y}, true
}

func (g *Game) advanceTankBurn(player int) {
	value := &g.player
	if player == 1 {
		value = &g.player2
	}
	if value.inTank && value.tankBurning {
		if value.tankBurn == 0 {
			value.inTank, value.tankBurning = false, false
			g.createTankBurst(value.x, value.y)
			g.hitPlayer(player)
		} else {
			value.tankFlame.advance(tankFlameSequence)
		}
	}
	kept := g.enemies[:0]
	for i := range g.enemies {
		tank := g.enemies[i]
		if tank.kind == 4 && tank.tankBurning && tank.tankDriver == player {
			if tank.tankBurn == 0 {
				g.createTankBurst(tank.x, tank.y)
				continue
			}
			if tank.health <= 0 {
				tank.tankBurning = false
			} else {
				tank.tankFlame.advance(tankFlameSequence)
				value.tankFlame = tank.tankFlame
			}
		}
		kept = append(kept, tank)
	}
	g.enemies = kept
}

// The original countdown runs after the collision passes. A new hit therefore
// ends its first complete frame with 69 ticks available for dismounting.
func (g *Game) finishTankTick() {
	for _, value := range []*actor{&g.player, &g.player2} {
		if value.tankBurning && value.tankBurn > 0 {
			value.tankBurn--
		}
	}
	for i := range g.enemies {
		tank := &g.enemies[i]
		if tank.tankBurning && tank.tankBurn > 0 {
			tank.tankBurn--
		}
	}
}

func (g *Game) createTankBurst(x, y float64) {
	g.createBurst(x-8, y+4, -1)
}

// A turret has its own position once the normal tank-placement pass has run.
func (value *actor) tankTurretPoint() (float64, float64) {
	if value.tankTurretPlaced {
		return value.tankTurret.x, value.tankTurret.y
	}
	return value.x, value.y
}

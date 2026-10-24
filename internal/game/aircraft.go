package game

import "github.com/hajimehoshi/ebiten/v2"

type helicopter struct {
	x, y    float64
	phase   int
	flying  bool
	alive   [5]bool
	damaged byte
	owner   int
}

// Event four reuses wide slots26..30. Each phase contains five independent
// masked32x24 bobs; the rotor sequence runs backwards through its three phases.
var helicopterArt = [3][5]int{
	{0x55dc8, 0x55fa8, 0x56548, 0x55a08, 0x55be8},
	{0x56728, 0x56188, 0x56ea8, 0x56368, 0x56cc8},
	{0x56908, 0x56ae8, 0x57088, 0x57268, 0x57448},
}

var helicopterOffsets = [5]navPoint{{0, -30}, {32, -30}, {16, 18}, {0, -6}, {32, -6}}

func (h *helicopter) part(index int) actor {
	p := helicopterOffsets[index]
	return actor{x: h.x + p.x, y: h.y + p.y, health: 1, slot: 26 + index, bob: helicopterArt[h.phase][index] - 0x42b88}
}

func (g *Game) spawnAircraft(event levelEvent) {
	g.enemyVehicles = [2]enemyVehicle{}
	// Aircraft initialization explicitly restarts the shared rotor clocks.
	g.vehicleChassisPhases = [2]vehicleChassisPhase{}
	kept := g.enemies[:0]
	for _, enemy := range g.enemies {
		if enemy.slot < 26 || enemy.slot > 31 {
			kept = append(kept, enemy)
		}
	}
	g.enemies = kept
	g.aircraft = helicopter{x: float64(event.x - 48), y: float64(levelHeight-event.progress) - 4, flying: true, alive: [5]bool{true, true, true, true, true}, owner: -1}
}

// updateAircraft retains a complete isolated aircraft operation for diagnostics.
func (g *Game) updateAircraft() {
	g.resolveAircraftDamage()
	g.advanceAircraft()
	g.fireAircraft()
}

// The original resolves helicopter damage before living infantry and guns.
func (g *Game) resolveAircraftDamage() {
	h := &g.aircraft
	// The original cleanup has two independent damage groups.
	if h.damaged&1 != 0 {
		part := h.part(0)
		g.createBurst(part.x, part.y+7, -1)
		h.alive[0], h.alive[1], h.alive[2] = false, false, false
	}
	if h.damaged&8 != 0 {
		part := h.part(3)
		g.createBurst(part.x, part.y+7, -1)
		h.alive[3], h.alive[4] = false, false
	}
	h.damaged = 0
}

// $9ce6 moves and animates after existing enemy shots and bunker firing.
func (g *Game) advanceAircraft() {
	h := &g.aircraft
	if !h.flying {
		return
	}
	h.y += 2
	h.phase = (h.phase + 2) % 3
	// Rotor images run in reverse order; their source animation step is
	// the inverse selection. Both chassis slots retain this last rotor step.
	phase := vehicleChassisPhase{step: (3 - h.phase) % 3, delay: 1}
	g.vehicleChassisPhases = [2]vehicleChassisPhase{phase, phase}
	if !h.alive[1] {
		h.flying, h.alive[2], h.alive[4] = false, false, false
		return
	}
}

// $9e56 leaves a fresh shot at the current muzzle until the next update.
func (g *Game) fireAircraft() {
	h := &g.aircraft
	if !h.flying {
		return
	}
	if g.randomValue()&3 == 0 && g.enemyBulletAvailable() {
		vectors := [8]navPoint{{-4, -4}, {0, -4}, {4, -4}, {-4, 0}, {4, 0}, {-4, 4}, {0, 4}, {4, 4}}
		motion := vectors[g.randomValue()&7]
		if g.audio != nil {
			g.audio.Effect(0, 2)
		}
		// $9e56 launches from slot27's top coordinate plus sixteen X pixels.
		g.addEnemyProjectile(projectile{x: h.x + 40, y: h.y - 26, vx: motion.x, vy: motion.y, enemy: true, owner: -1, life: 15})
	}
}

func (g *Game) damageAircraft(shot projectile, fragments bool) {
	h := &g.aircraft
	for i, active := range h.alive {
		if !active {
			continue
		}
		part := h.part(i)
		hit := grenadeHitsActor(shot, part)
		if fragments {
			hit = smallObjectTouchesActor(shot.x, shot.y, part)
		}
		if hit {
			if h.damaged&(1<<i) == 0 {
				g.addProjectileScore(shot, 100)
			}
			h.damaged |= 1 << i
			h.owner = shot.owner
			if shot.enemy {
				h.owner = -1
			}
		}
	}
}

func (g *Game) drawAircraftPart(slot int) {
	i := slot - 26
	if i < 0 || i >= len(g.aircraft.alive) || !g.aircraft.alive[i] {
		return
	}
	part := g.aircraft.part(i)
	if sprite := g.resources.bob(part.bob, vehicleBobHeight); sprite != nil {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(part.x-actorBobWidth/2, part.y-g.cameraY-4)
		g.playfield.DrawImage(sprite, op)
	}
}

// Each aircraft part keeps its own wide-object lifetime. The last entered
// flight update can fire before painting clips its departing anchor part.
func (g *Game) pruneOffscreenAircraftParts() {
	for index, alive := range g.aircraft.alive {
		if !alive {
			continue
		}
		part := g.aircraft.part(index)
		if !g.wideObjectInBounds(part.x, part.y) {
			g.aircraft.alive[index] = false
		}
	}
}

package game

import "github.com/hajimehoshi/ebiten/v2"

// sceneryBurst is the original $113a6 two-quad effect. It shares the eight
// small-bob slots with moving fragments, keeps its animation phase between
// destructions, and requests a radial burst after eight contact passes.
type sceneryBurst struct {
	parts [8]navPoint
	// Completion rewards retain their construction screen positions.
	pickupAnchors [2]navPoint
	origin        navPoint
	center        navPoint
	remaining     int
	phase         int
	frame         int
	owner         int
	active        bool
	enabled       uint8
	shared        bool
}

// sceneryBurstAt predicts only the real effect geometry. It neither clears
// terrain nor allocates a weapon or changes the player's resources.
func (g *Game) sceneryBurstAt(x, y float64) (sceneryBurst, bool) {
	row, column, size, ok := g.sceneryBounds(x, y)
	if !ok {
		return sceneryBurst{}, false
	}
	// The source searches left in four-pixel steps and upward in two-pixel
	// steps, retaining the impact's parity instead of snapping to tile edges.
	origin := navPoint{float64(column*tileSize + 4 + int(x)%4), float64(row*tileSize + 8 - int(y)%2)}
	value := sceneryBurst{origin: origin, center: navPoint{origin.x + float64((size.columns-1)*8), origin.y + float64((size.rows-1)*8)}, remaining: 8, phase: g.sceneryBurst.phase, frame: 49, owner: -1, active: true, enabled: 0xff}
	for i := range value.parts {
		group, quadrant := i/4, i%4
		value.parts[i] = navPoint{origin.x + float64(group*(size.columns-2)*16+quadrant%2*16), origin.y + float64(group*(size.rows-2)*16+quadrant/2*16)}
	}
	return value, true
}

func (g *Game) startSceneryBurst(value sceneryBurst) {
	g.startSceneryBurstWithContactGuard(value, false)
}

func (g *Game) startSceneryBurstWithContactGuard(value sceneryBurst, blocked bool) {
	anchors := [2]navPoint{value.parts[3], value.parts[4]}
	if g.burst.active {
		// The original shared pool is copied before its later moving pass.
		anchors = [2]navPoint{g.burst.parts[3], g.burst.parts[4]}
	}
	for index := range anchors {
		anchors[index].y -= g.cameraY
	}
	value.pickupAnchors = anchors
	if g.burst.active {
		// $113a6 still replaces the shared scenery bounds while fragments
		// own the bobs; it does not start or replace the quad countdown.
		g.sceneryBurst.origin, g.sceneryBurst.center = value.origin, value.center
		g.sceneryBurst.pickupAnchors = anchors
	} else {
		g.sceneryBurst = value
		if g.audio != nil {
			g.audio.Effect(21, 3)
		}
	}
	g.clearSceneryConstructorContacts(value.origin, blocked)
}

// updateSceneryBurst is $885a-$895c, after moving fragments and before the
// tank lifecycle. The final zero-count frame stays visible until the next
// update replaces it with the unmoved radial pool.
func (g *Game) updateSceneryBurst() {
	value := &g.sceneryBurst
	if !value.active {
		return
	}
	if g.burst.active || value.shared {
		// The original effects occupy the same eight records. When moving
		// fragments finish before this countdown, their final positions
		// become the positions of the resumed scenery animation.
		value.parts = g.burst.parts
		if g.burst.active {
			value.enabled = 0xff
		} else {
			// The original radial cleanup clears every shared bob flag.
			value.enabled = 0
		}
		value.shared = g.burst.active
	}
	if value.remaining != 0 {
		if !g.burst.active {
			value.phase = (value.phase + 1) & 3
			value.frame = 49 + value.phase*4
			for i, part := range value.parts {
				if value.enabled&(1<<i) != 0 {
					g.damageBurstPart(part, value.owner)
				}
			}
		}
		value.remaining--
		return
	}
	for _, anchor := range value.pickupAnchors {
		g.dropSceneryPickup(navPoint{anchor.x, g.cameraY + anchor.y})
	}
	value.active = false
	g.createBurst(value.center.x, value.center.y, value.owner)
}

func (g *Game) drawSceneryBurst() {
	value := &g.sceneryBurst
	if !value.active || g.burst.active {
		return
	}
	for i, part := range value.parts {
		if value.enabled&(1<<i) == 0 {
			continue
		}
		frame := value.frame + i%4
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(part.x-8, part.y-g.cameraY-8)
		g.playfield.DrawImage(g.resources.effects[frame], op)
	}
}

// sceneryBurstThreat predicts danger for the demonstration's first player,
// preserving its owner exemption and the current observed effect's delayed
// radial consequence. Future weapon impacts can use sceneryBurstAt with their
// own landing time before this state exists in the ordinary game.
func (g *Game) sceneryBurstThreat(value actor, tick int) float64 {
	burst := g.sceneryBurst
	if !burst.active || burst.owner == 0 || value.invulnerable > tick || value.inTank && value.tankBurning && value.tankBurn > tick {
		return 0
	}
	if !g.burst.active && tick <= burst.remaining {
		for i, part := range burst.parts {
			if burst.enabled&(1<<i) != 0 && smallObjectTouchesActor(part.x, part.y, value) {
				return 5000
			}
		}
	}
	firstMove := burst.remaining + 2
	if g.burst.active {
		firstMove = max(burst.remaining, g.burst.remaining) + 2
		// A queued request also creates the ordinary fallback enemy quad
		// while the older radial fragments still own the shared slots.
		if burst.remaining < g.burst.remaining && tick >= burst.remaining+1 && tick <= burst.remaining+3 {
			if grenadeHitsActor(projectile{x: burst.center.x, y: burst.center.y, enemy: true, owner: -1, grenade: true}, value) {
				return 5000
			}
		}
	}
	age := tick - firstMove + 1
	if age >= 1 && age <= 6 {
		for _, motion := range burstMotions {
			if smallObjectTouchesActor(burst.center.x+motion.x*float64(age), burst.center.y+motion.y*float64(age), value) {
				return 5000
			}
		}
	}
	return 0
}

package game

import (
	"image"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

type finalEncounter struct {
	ruinsClipTop     int
	x, y             float64
	active, defeated bool
}

type endingSequence struct {
	hitParts        uint16
	x, y            float64
	active          bool
	ticks           int
	displayAmmo     [2]int
	displayGrenades [2]int
	displayScores   [2]int
}

func (g *Game) spawnFinalEncounter(event levelEvent) {
	g.final = finalEncounter{x: 80, y: float64(levelHeight - event.progress - 76), active: true}
	kept := g.enemies[:0]
	for _, enemy := range g.enemies {
		if enemy.slot < 5 || enemy.slot > 13 {
			kept = append(kept, enemy)
		}
	}
	g.enemies = kept
	g.clearInfantryLegImages(5, 13)
}

func (f *finalEncounter) part(index int) actor {
	bank := 0x65ea8
	if f.defeated {
		bank = 0x66f88
	}
	return actor{
		x: f.x + float64(index%3*32), y: f.y + float64(index/3*24),
		slot: 5 + index, health: 1, bob: bank - 0x42b88 + index*vehicleBobBytes,
	}
}

func (g *Game) updateFinalEncounter() {
	if !g.final.active {
		return
	}
	for player := 0; player < g.playerCount; player++ {
		value := &g.player
		if player == 1 {
			value = &g.player2
		}
		if value.inTank {
			continue
		}
		for i := range 9 {
			part := g.final.part(i)
			if math.Abs(value.x-part.x) <= 23 && math.Abs(value.y-part.y) <= 19 {
				g.hitPlayer(player)
				break
			}
		}
	}
	base := g.final.part(0)
	if g.ending.active {
		base = g.endingPart(5)
	}
	for _, offset := range [...]navPoint{{-64, 32}, {-32, 32}, {80, 32}, {112, 32}} {
		x, y := base.x+offset.x, base.y+offset.y
		if g.randomValue()&15 != 0 || g.terrainCode(x, y) == 13 || !g.enemyBulletAvailable() {
			continue
		}
		dx, dy := g.enemyFiringVector(x, y)
		g.addEnemyProjectile(projectile{
			x: x - 4, y: y + 12, vx: dx, vy: dy,
			enemy: true, owner: -1, grenade: true, life: 15, frame: 7,
		})
	}
}

// $a6c6 allows a player-owned blast to defeat the final encounter through
// its middle column. $e46e then bakes eight replacement bobs into the scenery;
// the original actor flags become zero, rather than running infantry AI.
func (g *Game) damageFinalEncounter(shot projectile, fragments bool) {
	if g.ending.active {
		count := 4
		if fragments {
			count = 1
		}
		for quadrant := 0; quadrant < count; quadrant++ {
			x, y := shot.x, shot.y
			if !fragments {
				x, y = x-8+float64(quadrant%2*16), y-8+float64(quadrant/2*16)
			}
			for index := 0; index < 12; index++ {
				part := g.endingPart(index)
				if !g.endingPartCanBeHit(index) || !smallObjectTouchesActor(x, y, part) {
					continue
				}
				g.ending.hitParts |= 1 << index
				if !fragments && !shot.enemy && shot.owner >= 0 {
					g.addProjectileScore(shot, 100)
				}
			}
		}
		return
	}

	if !g.final.active || shot.enemy || shot.owner < 0 {
		return
	}
	count := 4
	if fragments {
		count = 1
	}
	for quadrant := 0; quadrant < count; quadrant++ {
		x, y := shot.x, shot.y
		if !fragments {
			x, y = x-8+float64(quadrant%2*16), y-8+float64(quadrant/2*16)
		}
		middle := false
		// A6C6 finishes the entire quadrant's active-part scan before E46E
		// defeats the middle column. Later quadrants then see inactive parts.
		for index := 0; index < 9; index++ {
			part := g.final.part(index)
			if !g.wideObjectInBounds(part.x, part.y) || !smallObjectTouchesActor(x, y, part) {
				continue
			}
			if !fragments {
				g.addProjectileScore(shot, 100)
			}
			middle = middle || index%3 == 1
		}
		if middle {
			g.final.active, g.final.defeated = false, true
			g.final.ruinsClipTop = int(g.cameraBackY) - 32
			return
		}
	}
}

func (g *Game) drawFinalPart(index int) {
	part := g.final.part(index)
	g.drawFixedWidePart(part)
}

func (g *Game) shotHitsFinalScene(shot projectile) bool {
	if g.ending.active {
		for i := range 12 {
			if g.endingPartCanBeHit(i) && shotHitsActor(shot, g.endingPart(i)) {
				g.ending.hitParts |= 1 << i
				return true
			}
		}
	}
	if g.final.active {
		for i := range 9 {
			if shotHitsActor(shot, g.final.part(i)) {
				return true
			}
		}
	}
	return false
}

func (g *Game) drawFinalRuins() {
	if !g.final.defeated {
		return
	}
	for index := 0; index < 8; index++ {
		part := g.final.part(index)
		top := int(part.y) - 4
		cut := max(0, g.final.ruinsClipTop-top)
		if cut >= vehicleBobHeight {
			continue
		}
		sprite := g.resources.bob(part.bob, vehicleBobHeight)
		if sprite == nil {
			continue
		}
		// E46E bakes into the retained terrain image. Rows above its loaded
		// origin are later supplied by the map strip, not by these ruins.
		if cut != 0 {
			sprite = sprite.SubImage(image.Rect(0, cut, actorBobWidth, vehicleBobHeight)).(*ebiten.Image)
		}
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(part.x-actorBobWidth/2, float64(top+cut)-g.cameraY)
		g.playfield.DrawImage(sprite, op)
	}
}

func (g *Game) spawnEnding(event levelEvent) {
	// The ending disables live weapons without repainting their HUD counters.
	g.ending = endingSequence{
		x: 80, y: float64(levelHeight - event.progress - 104), active: true,
		displayAmmo:     [2]int{g.ammunition, g.ammunition2},
		displayGrenades: [2]int{g.grenades, g.grenades2},
	}
	kept := g.enemies[:0]
	for _, enemy := range g.enemies {
		if enemy.slot > 11 {
			kept = append(kept, enemy)
		} else if enemy.slot >= 0 && enemy.health <= 0 {
			// The ending initializer replaces images without clearing old hits.
			g.ending.hitParts |= 1 << enemy.slot
		}
	}
	g.enemies = kept
	g.clearInfantryLegImages(0, 11)
}

func (g *Game) endingPart(index int) actor {
	phase := g.ending.ticks / 2 & 1
	return actor{
		x: g.ending.x + float64(index%3*32), y: g.ending.y + float64(index/3*24),
		slot: index, health: 1,
		bob: 0x63b08 - 0x42b88 + int(g.resources.endingFrames[phase][index])*vehicleBobBytes,
	}
}

func (g *Game) drawFinalWideSlot(slot int) {
	if g.ending.active && slot < 12 {
		g.drawFixedWidePart(g.endingPart(slot))
	} else if g.final.active && slot >= 5 && slot <= 13 {
		g.drawFinalPart(slot - 5)
	}
}

func (g *Game) drawFixedWidePart(part actor) {
	if sprite := g.resources.bob(part.bob, vehicleBobHeight); sprite != nil {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(part.x-actorBobWidth/2, part.y-g.cameraY-4)
		g.playfield.DrawImage(sprite, op)
	}
}

func (g *Game) endingPartCanBeHit(index int) bool {
	if !g.ending.active || g.ending.hitParts&(1<<index) != 0 {
		return false
	}
	part := g.endingPart(index)
	return g.wideObjectInBounds(part.x, part.y)
}

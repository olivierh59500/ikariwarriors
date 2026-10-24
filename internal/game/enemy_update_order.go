package game

import (
	"cmp"
	"slices"
)

func enemyUpdatePhase(slot int) int {
	if slot >= 5 && slot <= 18 {
		return 0
	}
	if slot >= 3 && slot <= 4 {
		return 1
	}
	if slot >= 0 && slot <= 2 {
		return 2
	}
	return 3
}

func (g *Game) enemySharedWithEncounter(slot int) bool {
	return slot <= 18 && (g.ending.active || g.final.active && slot >= 5)
}

// The original retires/falls slots0..21 before any living infantry or gun.
// In particular, red pickup draws must precede the live marching/firing draws.
func (g *Game) advanceEnemyDeaths() {
	slices.SortStableFunc(g.enemies, func(a, b actor) int { return cmp.Compare(a.slot, b.slot) })
	kept := g.enemies[:0]
	for index := range g.enemies {
		enemy := &g.enemies[index]
		// Only starting a normal fall pauses while encounter art owns its
		// slots. Existing falls and the separate red routine keep advancing.
		paused := enemy.deadTicks == 0 && enemy.slot >= 0 && enemy.slot <= 14 && (g.ending.active || g.final.active)
		if paused || enemy.health > 0 {
			kept = append(kept, *enemy)
			continue
		}
		if enemy.kind == 4 {
			g.createBurst(enemy.x, enemy.y+7, -1)
			continue
		}
		if enemy.deadTicks == 0 {
			enemy.deadTicks = 16
			enemy.upper, enemy.lower = walkAnimation{}, walkAnimation{}
		}
		enemy.deadTicks--
		if enemy.deadTicks > 0 {
			if enemy.kind == 5 {
				enemy.upper.advance(redEnemyDeathSequence)
			} else {
				enemy.upper.advance(enemyDeathSequence)
			}
			if enemy.kind != 5 {
				enemy.y--
			}
			code := g.terrainCode(enemy.x, enemy.y)
			g.animateInfantryLegs(enemy, enemyDeathLegSequence, code == 157 || code == 158)
			kept = append(kept, *enemy)
		} else if enemy.kind == 5 {
			g.dropPickup(*enemy)
		}
		g.rememberInfantryAnimation(*enemy)
		continue
	}
	g.enemies = kept
}

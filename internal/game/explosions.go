package game

import "iter"

// The original has one quad for each player (slots41..48), plus one shared
// enemy quad (77..80). Starting another blast replaces its pool immediately.
func (g *Game) startBlast(shot projectile) {
	pool := shot.owner
	if shot.enemy {
		pool = 2
	}
	shot.life, shot.impact, shot.vx, shot.vy = 0, 4, 0, 0
	g.blasts[pool] = shot
}

func (g *Game) advanceBlast(pool int) {
	shot := g.blasts[pool]
	if shot.impact == 0 {
		return
	}
	shot.impact--
	g.blasts[pool] = shot
	if shot.impact > 0 {
		if shot.enemy {
			g.damageGrenadeArea(shot)
		}
		return
	}
	if shot.enemy && shot.airStrike != 0 {
		g.airStrikeBusy[shot.airStrike-1] = false
	}
	if !shot.fragments && (shot.enemy || g.grenadePower[shot.owner] <= 1) {
		return
	}
	// $11290's fragments are hostile even to the thrower. Owned expiry
	// precedes the shared movement pass; enemy expiry follows it. The
	// constructor itself leaves every fragment at the blast origin.
	g.createBurst(shot.x, shot.y, -1)
}

// $ae14 allocates the first of twelve inactive enemy shot slots. Blast
// quads do not reserve a flying-shot slot after landing.
func (g *Game) addEnemyProjectile(shot projectile) {
	var occupied [12]bool
	for _, current := range g.projectiles {
		if current.enemy {
			occupied[current.slot] = true
		}
	}
	for slot, busy := range occupied {
		if !busy {
			shot.slot = slot
			g.projectiles = append(g.projectiles, shot)
			return
		}
	}
}

// The demonstration sees exactly the active shots and shared blast pools.
// Iteration avoids allocating a combined scene for every movement candidate.
func (g *Game) demoShots() iter.Seq2[int, projectile] {
	return func(yield func(int, projectile) bool) {
		for i, shot := range g.projectiles {
			if !yield(i, shot) {
				return
			}
		}
		for i, shot := range g.blasts {
			if shot.impact > 0 && !yield(len(g.projectiles)+i, shot) {
				return
			}
		}
	}
}

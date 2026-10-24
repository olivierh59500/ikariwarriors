package game

// The original reads the owner's current tank flag for the first shot slot.
// Birth artwork and movement remain attached to the shot after boarding or
// dismounting; they do not determine its current collision behavior.
func (g *Game) ownedShotUsesCannon(shot projectile) bool {
	if shot.enemy || shot.grenade || shot.slot != 0 {
		return false
	}
	if shot.owner == 0 {
		return g.player.inTank
	}
	return shot.owner == 1 && g.player2.inTank
}

func (g *Game) ownedBulletUpgrade(shot projectile) bool {
	return shot.owner >= 0 && shot.owner < len(g.bulletPower) && g.bulletPower[shot.owner] > 1
}

// The live bullet upgrade takes precedence over the live grenade upgrade:
// the former hits and pierces actors, whereas grenade-only tank fire ignores
// them. Saved shell artwork does not replace either current flag.
func (g *Game) ownedCannonIgnoresActors(shot projectile) bool {
	return !g.ownedBulletUpgrade(shot) && shot.owner >= 0 && shot.owner < len(g.grenadePower) && g.grenadePower[shot.owner] > 1
}

// Foot bullets retain terrain immunity from their saved upgraded artwork,
// and ordinary artwork can gain it from a newly collected live upgrade.
// Mounted first-slot terrain tests use only the current bullet upgrade.
func (g *Game) ownedShotPiercesTerrain(shot projectile) bool {
	return g.ownedBulletUpgrade(shot) || !g.ownedShotUsesCannon(shot) && g.projectileFrame(shot) >= 2
}

func (g *Game) isOwnedQuad(shot projectile) bool {
	return !shot.enemy && shot.owner >= 0 && shot.owner < 2 && g.blasts[shot.owner].impact > 0 && g.blasts[shot.owner] == shot
}

// Only the demonstration forecast reads this helper. Each real flying record
// has its own retained cleanup, including the two secondary foot-shot slots.
func (g *Game) ownedSlotCleanupDelay(shot projectile) int {
	if shot.enemy || shot.owner < 0 || shot.owner >= len(g.pendingFootImpact) || shot.slot < 0 || shot.slot >= 3 {
		return 0
	}
	return *g.pendingOwnedFootImpact(shot.owner, shot.slot)
}

// A boarded foot impact can outlive its projectile when the shell creates a
// quad. Its remaining short animation still advances any replacement in the
// first slot and clears that slot at its normal end. Keep a finite timer;
// reproducing the original range-word underflow is unnecessary.

func (g *Game) pendingOwnedFootImpact(player, slot int) *int {
	if slot == 0 {
		return &g.pendingFootImpact[player]
	}
	return &g.pendingOtherFootImpacts[player][slot-1]
}

// Keep the original first-slot helper for its separately qualified cannon path.
func (g *Game) finishPendingFootImpact(player, previous int) {
	g.finishPendingOwnedFootImpact(player, 0, previous)
}

func (g *Game) finishPendingOwnedFootImpact(player, slot, previous int) {
	pending := g.pendingOwnedFootImpact(player, slot)
	if previous == 0 || *pending != previous {
		return
	}
	*pending--
	for i := range g.projectiles {
		shot := &g.projectiles[i]
		if shot.enemy || shot.owner != player || shot.slot != slot {
			continue
		}
		if *pending == 0 {
			g.projectiles = append(g.projectiles[:i], g.projectiles[i+1:]...)
		} else {
			shot.frame = g.projectileFrame(*shot) + 1
		}
		return
	}
}

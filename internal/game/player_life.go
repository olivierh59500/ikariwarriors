package game

var playerDeathSequences = [2][]int{
	{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 10},
	{12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 22},
}

var deathLegSequence = []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 10, 10}

var enemyDeathSequence = []int{36, 37, 38, 39, 40, 41, 42, 43}
var redEnemyDeathSequence = []int{24, 25, 26, 27, 28, 29, 30, 31}
var enemyDeathLegSequence = []int{11, 17, 3, 29, 15, 24, 6, 21}

// $c768 compares a 6-by-6 projectile with a 23-by-19 actor. The projectile
// and actor coordinates have different graphic origins, hence the offsets.
func shotHitsActor(shot projectile, value actor) bool {
	return value.health > 0 && value.deadTicks == 0 && shotOverlapsActor(shot, value)
}

func shotOverlapsActor(shot projectile, value actor) bool {
	dx, dy := shot.x-value.x, shot.y-value.y
	return dx >= -14 && dx <= 15 && dy >= -2 && dy <= 23
}

// advancePlayerLife retains the complete isolated life operation used by
// diagnostics. Normal gameplay schedules its fall and protection separately.
func (g *Game) advancePlayerLife(player int) {
	g.advancePlayerFall(player)
	g.advancePlayerProtection(player)
}

// $8ed8/$8f28 resolve falling and reappearance before any living NPC.
func (g *Game) advancePlayerFall(player int) {
	value, lives, ammunition, grenades := &g.player, &g.lives, &g.ammunition, &g.grenades
	if player == 1 {
		value, lives, ammunition, grenades = &g.player2, &g.lives2, &g.ammunition2, &g.grenades2
	}
	if value.deadTicks == 0 {
		return
	}
	value.deadTicks--
	if value.deadTicks > 0 {
		// A same-frame vertical walk can leave either delay nonzero. The
		// original retains that walking image until its fall animation
		// actually advances, independently for the torso and the legs.
		if value.upper.delay == 0 {
			value.upperFalling = true
		}
		if value.lower.delay == 0 {
			value.lowerFalling = true
		}
		value.upper.advance(playerDeathSequences[player])
		code := g.terrainCode(value.x, value.y)
		g.animateInfantryLegs(value, deathLegSequence, code == 157 || code == 158)
		return
	}
	// The original prepares the standing pose and supplies before deciding
	// whether another reserve exists. Walking delays survive that preparation.
	value.x, value.y = 112, g.cameraY+176
	for value.x < playfieldWidth-14 && g.solidAt(value.x, value.y) {
		value.x += 16
	}
	value.upper = walkAnimation{frame: 8, delay: value.upper.delay}
	value.lower = walkAnimation{frame: 12, delay: value.lower.delay}
	value.upperFalling, value.lowerFalling = false, false
	*ammunition, *grenades = 99, 50
	g.bulletPower[player], g.grenadePower[player], g.rangeBonus[player] = 1, 1, 0
	*lives--
	if *lives < 0 {
		state := &g.cooperativeScroll[player]
		state.claimed = false
		state.terrainOffset = levelHeight - 16 - value.y
		if g.lives < 0 && (g.playerCount == 1 || g.lives2 < 0) {
			g.scene, g.sceneTick, g.idle, g.demo = titleScene, 0, 0, false
			g.audio.Music("title")
		}
		return
	}
	value.health, value.inTank = 1, false
	g.directionLock[player] = false
}

// $9108/$9134 reduce protection and step blink only after the collision passes.
func (g *Game) advancePlayerProtection(player int) {
	value := &g.player
	if player == 1 {
		value = &g.player2
	}
	protected := value.invulnerable > 0
	if protected {
		value.invulnerable--
	}
	value.advancePlayerBlink(protected)
}

// $9108-$915e decrements protection before stepping its independent visual
// cycle. Falling pauses the cycle; reappearance and later games retain its
// phase. Living bobs become active before input, then every second eligible
// protected frame hides them for the final painter pass.
func (value *actor) advancePlayerBlink(protected bool) {
	value.hidden = value.health <= 0 && value.deadTicks == 0
	if !protected || value.deadTicks > 0 {
		return
	}
	value.blinkPhase--
	if value.blinkPhase == 0 {
		value.blinkPhase = 2
		value.hidden = true
	}
}

// Parts keep their own art bank when a new fall overlaps an accepted walk.
// Offsets index the recovered graphic blob; no machine state is required.
func (value actor) playerArtOffsets(player int) (upper, lower int) {
	upper = player*22*actorBobBytes + infantryImageOffset(value.upper.frame, actorBobBytes)
	lower = 0x46288 - 0x42b88 + infantryImageOffset(value.lower.frame, overlayBobBytes)
	if value.lower.frame == -2 {
		lower = 0x4a1a8 - 0x42b88
	} else if value.lower.frame == -3 {
		lower = 0x4a248 - 0x42b88
	}
	if value.inTank {
		return 0x5b408 - 0x42b88 + player*24*vehicleBobBytes + value.upper.frame*vehicleBobBytes, -1
	}
	if value.upperFalling {
		upper = 0x47cc8 - 0x42b88 + value.upper.frame*actorBobBytes
	}
	if value.lowerFalling && value.lower.frame >= 0 {
		lower = 0x49ac8 - 0x42b88 + value.lower.frame*overlayBobBytes
	}
	return upper, lower
}

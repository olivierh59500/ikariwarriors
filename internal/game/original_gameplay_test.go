package game

import "testing"

func TestOriginalOpeningMovementAndCollision(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGame(false)
	// A fresh FS-UAE one-player run stops at $848c with actor 23 at
	// (120,98), longitudinal progress 22. Twenty-five up ticks reach 72;
	// thirty left ticks stop at X=82 against raw terrain code $35.
	for range 25 {
		g.moveActor(&g.player, 0, -2)
		g.player.animateWalk(true)
	}
	if g.player.x != 72 || g.player.y != levelHeight-88 {
		t.Fatalf("up: got (%g,%g), want (72,%d)", g.player.x, g.player.y, levelHeight-88)
	}
	if g.player.upper.frame != 9 || g.player.lower.frame != 17 {
		t.Fatalf("up pose: torso=%d legs=%d, want 9/17", g.player.upper.frame, g.player.lower.frame)
	}
	g.player.face(6)
	for range 30 {
		g.moveActor(&g.player, -2, 0)
	}
	if g.player.x != 34 || g.player.y != levelHeight-88 {
		t.Fatalf("wall: got (%g,%g), want (34,%d)", g.player.x, g.player.y, levelHeight-88)
	}
	for range 10 {
		g.moveActor(&g.player, -2, 0)
	}
	if g.player.x != 34 {
		t.Fatal("continued input crossed the original collision boundary")
	}
}

func TestOriginalIndependentWalkAnimation(t *testing.T) {
	a := actor{upper: walkAnimation{step: 1, frame: 8}, lower: walkAnimation{step: 4, frame: 16}}
	for range 25 {
		a.animateWalk(true)
	}
	a.face(2)
	for range 16 {
		a.animateWalk(true)
	}
	// Original snapshots: upper pointer $42f48, lower pointer $466e8.
	if a.upper.frame != 3 || a.lower.frame != 7 {
		t.Fatalf("right pose: torso=%d legs=%d, want 3/7", a.upper.frame, a.lower.frame)
	}
}

func TestOriginalHeldFireGrenadeTiming(t *testing.T) {
	var launches []int
	for duration := 1; duration <= 32; duration++ {
		if heldGrenade(duration) {
			launches = append(launches, duration)
		}
	}
	if len(launches) != 3 || launches[0] != 11 || launches[1] != 21 || launches[2] != 31 {
		t.Fatalf("grenade ticks: %v, want [11 21 31]", launches)
	}
}

func TestOriginalHitAndReappearance(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGame(false)
	g.ammunition, g.grenades = 12, 3
	g.hitPlayer(0)
	if g.player.health != 0 || g.player.deadTicks != 24 || g.lives != 5 {
		t.Fatal("a hit did not begin the original falling sequence")
	}
	for range 23 {
		g.advancePlayerLife(0)
	}
	if g.player.deadTicks != 1 || g.lives != 5 {
		t.Fatal("a reserve life was consumed before the fall ended")
	}
	g.advancePlayerLife(0)
	if g.player.deadTicks != 0 || g.player.invulnerable != 76 || g.lives != 4 || g.ammunition != 99 || g.grenades != 50 || g.player.y != g.cameraY+176 {
		t.Fatalf("reappearance: life=%d protection=%d ammo=%d grenades=%d Y=%g", g.lives, g.player.invulnerable, g.ammunition, g.grenades, g.player.y)
	}
	g.hitPlayer(0)
	if g.player.deadTicks != 0 {
		t.Fatal("the respawn protection allowed an immediate second death")
	}
}

func TestOriginalOpeningTapFire(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGame(false)
	for tick := 0; tick < 25; tick++ {
		g.moveActor(&g.player, 0, -2)
		g.cameraY = g.player.y - 94
		if (tick == 0 || tick > 1 && (tick-1)%4 == 0) && g.bulletAvailable(0) {
			g.fireProjectile(&g.player, false, 0, false, 6)
			g.ammunition--
		}
		g.updateProjectiles()
	}
	// The original run uses five rounds: the three bullet slots remain busy
	// for the four impact frames after a bullet reaches its range limit.
	if g.ammunition != 94 {
		t.Fatalf("opening ammunition=%d, want 94", g.ammunition)
	}
}

func TestOriginalHeldGrenadesAndSharedSlot(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGame(false)
	g.enemies = nil
	for duration := 1; duration <= 60; duration++ {
		if duration == 1 {
			g.fireProjectile(&g.player, false, 0, false, 6)
		}
		if heldGrenade(duration) && g.grenadeAvailable(0) {
			g.fireProjectile(&g.player, false, 0, true, 4)
			g.grenades--
		}
		g.updateProjectiles()
	}
	if g.grenades != 47 || g.player.health != 1 {
		t.Fatalf("held grenades: remaining=%d health=%d, want 47/1", g.grenades, g.player.health)
	}
	g.projectiles = nil
	for range 2 {
		g.fireProjectile(&g.player, false, 0, false, 6)
	}
	if g.grenadeAvailable(0) {
		t.Fatal("a grenade occupied the second live bullet's original slot")
	}
}

func TestOriginalBushBoundaryAndDetour(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGame(false)
	g.player.x, g.player.y = 116-48, levelHeight-16-632
	g.cameraY = g.player.y - 94
	for range 32 {
		g.moveActor(&g.player, 0, -2)
	}
	if g.player.y != levelHeight-16-632 {
		t.Fatal("walking crossed the bush seen in the played reference")
	}
	if g.destroyTileAt(g.player.x, g.player.y-8) {
		t.Fatal("ordinary fire removed the indestructible bush")
	}
	for range 18 {
		g.moveActor(&g.player, -2, -2)
	}
	// The original detour reaches screen X=80 and progress 648.
	if g.player.x != 80-48 || g.player.y != levelHeight-16-648 {
		t.Fatalf("detour: X=%g progress=%g, want 32/648", g.player.x, levelHeight-16-g.player.y)
	}
}

func TestOriginalScriptedGunAndMineAreHazards(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGame(false)
	g.applyLevelEvent(levelEvent{progress: 144, x: 142, kind: 17})
	if len(g.pickups) != 0 || !g.slotActive(32) {
		t.Fatal("the scripted gun was interpreted as a collectible")
	}
	g.applyLevelEvent(levelEvent{progress: 1250, x: 100, kind: 18})
	if len(g.mines) != 1 || g.mines[0].slot != 81 {
		t.Fatal("the original proximity-mine event was lost")
	}
	mine := g.mines[0]
	g.player.x, g.player.y = mine.x+8, mine.y-4
	g.cameraY = g.player.y - 94
	g.updateMines()
	g.updateProjectiles()
	if len(g.mines) != 0 || g.player.deadTicks != 24 || g.ammunition != 99 || g.grenades != 50 {
		t.Fatal("contact with the original mine did not trigger the hostile explosion")
	}
}

func TestTouchAndAnalogCardinalSectors(t *testing.T) {
	tests := []struct{ x, y, wantX, wantY float64 }{
		{1, 0.12, 1, 0}, {-1, -0.12, -1, 0},
		{0.12, 1, 0, 1}, {-0.12, -1, 0, -1},
		{0.8, -0.8, 1, -1}, {0.1, 0.1, 0, 0},
	}
	for _, test := range tests {
		x, y := joystickAxes(test.x, test.y)
		if x != test.wantX || y != test.wantY {
			t.Fatalf("stick (%g,%g) produced (%g,%g), want (%g,%g)", test.x, test.y, x, y, test.wantX, test.wantY)
		}
	}
}

func TestOriginalBunkerDestructionFootprint(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGame(false)
	if !g.damageTerrain(88, 14832, 0) {
		t.Fatal("the shot failed to destroy the bunker")
	}
	changed := 0
	for row := range mapRows {
		for column := range mapColumns {
			if !g.destroyed[row][column] {
				continue
			}
			changed++
			// FS-UAE played-tank-road: exactly these fifteen raw map bytes
			// changed from $f4/$f5/$f6 to zero, with no neighboring changes.
			if row < 926 || row > 928 || column < 3 || column > 7 {
				t.Fatalf("unexpected changed terrain byte %d,%d", row, column)
			}
		}
	}
	if changed != 15 || g.score != 1000 {
		t.Fatalf("changed=%d score=%d, want 15/1000", changed, g.score)
	}
}

func TestOriginalTankBoardingAndChassisPhase(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGame(false)
	g.player.x, g.player.y = 52, levelHeight-16-402
	g.cameraY = g.player.y - 94
	// The played pre-boarding frame retains these torso and leg counters.
	g.player.upper = walkAnimation{step: 3, delay: 1, frame: 8}
	g.player.lower = walkAnimation{step: 5, delay: 1, frame: 17}
	g.player.invulnerable = 20
	g.enemies = []actor{{x: 52, y: g.player.y - 16, kind: 4, slot: 19, health: 1, tankFuel: 64}}
	if !g.useTank(0) || g.player.y-g.cameraY+4 != 82 || g.player.x != 52 || g.player.upper.step != 3 || g.player.upper.delay != 1 {
		t.Fatalf("boarding differs from played original: %+v", g.player)
	}
	for range 24 {
		previous := g.player.y
		g.moveActor(&g.player, 0, -2)
		g.player.animateMovement(0, -2, false)
		g.followPlayer(previous, g.player.y)
	}
	if levelHeight-16-g.player.y != 466 || g.player.y-g.cameraY+4 != 82 || g.player.upper.frame != 6 {
		t.Fatalf("driven original north pose differs: %+v", g.player)
	}
	g.player.face(2)
	for range 18 {
		g.moveActor(&g.player, 2, 0)
		g.player.animateMovement(2, 0, false)
	}
	if g.player.x != 88 || g.player.upper.frame != 22 {
		t.Fatalf("driven original east pose differs: X%g frame%d", g.player.x, g.player.upper.frame)
	}
	g.fireProjectile(&g.player, false, 0, false, 6)
	if len(g.projectiles) != 1 || !g.projectiles[0].cannon || g.projectiles[0].slot != 0 || g.projectiles[0].x != 91 || g.projectiles[0].y != g.player.y+12 || g.bulletAvailable(0) {
		t.Fatal("a tank cannon did not use its single original projectile slot/muzzle")
	}
}

func TestOriginalBurningTankLastTicks(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGame(false)
	g.player.inTank, g.player.tankBurning, g.player.tankBurn = true, true, 38
	g.player.tankFlame = walkAnimation{frame: 0, step: 0, delay: 1}
	for range 37 {
		g.advanceTanks()
		g.finishTankTick()
	}
	if g.player.tankBurn != 1 || !g.player.inTank || g.player.deadTicks != 0 || g.player.tankFlame.frame != 2 {
		t.Fatal("the observed thirty-seven burning ticks differ")
	}
	g.advanceTanks()
	g.finishTankTick()
	if g.player.tankBurn != 0 || !g.player.inTank || g.player.deadTicks != 0 || g.player.tankFlame.frame != 3 {
		t.Fatal("the original last escape tick was removed")
	}
	g.advanceTanks()
	g.finishTankTick()
	g.advancePlayerLife(0)
	if g.player.inTank || g.player.tankBurning || g.player.deadTicks != 23 {
		t.Fatal("remaining in the burning tank did not start the original fall")
	}
}

func TestOriginalRespawnUpwardMovement(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGame(false)
	g.enemies = nil
	g.player.x, g.player.y = 112, g.cameraY+176
	g.moveActor(&g.player, 0, -2)
	if g.player.y != g.cameraY+174 {
		t.Fatal("the lower down-input limit changed the first upward respawn step")
	}
}

func TestOriginalExplosiveSceneryContact(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGame(false)
	g.enemies = nil
	g.projectiles = []projectile{{x: 80, y: 14836, vy: -6, life: 10, owner: 0, power: 1}}
	g.updateProjectiles()
	if g.destroyed[927][5] || g.score != 0 {
		t.Fatal("an ordinary infantry bullet destroyed explosive scenery")
	}
	g.player.x, g.player.y = 88, 14840
	g.moveActor(&g.player, 0, -2)
	if !g.destroyed[927][5] || g.player.deadTicks != 24 || g.score != 1000 {
		t.Fatal("touching explosive scenery did not detonate it and hit the player")
	}
}

func TestOriginalLimitedBackwardScroll(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGame(false)
	g.enemies = nil
	// Played obstacle-detonate checkpoint: the bunker was cleared through
	// contact during normal respawn protection, without any fired projectiles.
	g.player.x, g.player.y = 112, levelHeight-16-146
	g.cameraY, g.cameraBackY = 14752, 14784
	g.destroyTileAt(88, 14832)
	move := func(n int) {
		for range n {
			old := g.player.y
			g.moveActor(&g.player, 0, 2)
			g.followPlayer(old, g.player.y)
		}
	}
	move(40)
	if levelHeight-16-g.player.y != 66 || g.player.y-g.cameraY+4 != 162 || g.cameraY != 14768 {
		t.Fatalf("first retreat differs: L%g screenY%g camera%g", levelHeight-16-g.player.y, g.player.y-g.cameraY+4, g.cameraY)
	}
	move(40)
	if levelHeight-16-g.player.y != 36 || g.player.y-g.cameraY+4 != 176 || g.cameraY != 14784 {
		t.Fatalf("retreat limit differs: L%g screenY%g camera%g", levelHeight-16-g.player.y, g.player.y-g.cameraY+4, g.cameraY)
	}
}

package game

import "testing"

func TestDemoDecisionsOnlyReturnOrdinaryControls(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGame(true)
	player, lives, ammo, grenades := g.player, g.lives, g.ammunition, g.grenades
	g.demoControls()
	if g.player != player || g.lives != lives || g.ammunition != ammo || g.grenades != grenades {
		t.Fatal("a demonstration decision changed player state instead of returning controls")
	}
}

func TestDemoNavigatesAroundTheOriginalExplosiveBunker(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGame(true)
	g.enemies = nil
	// This controlled navigation case excludes combat and future actor births.
	g.eventIndex = len(g.resources.events)
	g.randomSpawn, g.redSpawn = false, false
	for tick := 0; tick < 350 && levelHeight-16-g.player.y < 350; tick++ {
		control := g.demoControls()
		previous := g.player.y
		g.moveActor(&g.player, control.x*2, control.y*2)
		g.followPlayer(previous, g.player.y)
	}
	if levelHeight-16-g.player.y < 350 || g.player.health != 1 || g.destroyed[927][5] {
		t.Fatalf("the terrain-only route hit or stalled at the bunker: progress%g health%d", levelHeight-16-g.player.y, g.player.health)
	}
}

func TestDemoAimsAlongAnActualProjectileTrajectory(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGame(false)
	g.enemies = []actor{{x: g.player.x, y: g.player.y - 48, kind: 2, health: 1, slot: 3}}
	direction, ok := g.demoAim()
	if !ok {
		t.Fatal("a clear stationary enemy was not targeted")
	}
	g.player.direction = direction
	g.fireProjectile(&g.player, false, 0, false, 6)
	for range 10 {
		g.updateProjectiles()
	}
	if g.enemies[0].health != 0 || g.score != 100 {
		t.Fatal("the selected direction did not actually hit its target")
	}
}

func TestDemoCanLeaveTheOriginalRespawnStrip(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGame(false)
	g.cameraY, g.cameraBackY = levelHeight-1024, levelHeight-1024
	g.secondaryTerrain = g.cameraY < secondaryMapRows*tileSize
	for x := 16.0; x < playfieldWidth-16; x += 2 {
		y := g.cameraY + 176
		if g.solidAt(x, y-8) || g.terrainCode(x, y-8) > 0xec {
			continue
		}
		value := actor{x: x, y: y, health: 1}
		g.moveActor(&value, 0, -2)
		nx, ny, ok := g.demoStep(x, y, 0, -1)
		if !ok || nx != value.x || ny != value.y || ny != y-2 {
			t.Fatalf("planner blocked an ordinary upward respawn move: %g,%g,%v versus %+v", nx, ny, ok, value)
		}
		return
	}
	t.Fatal("no clear original respawn probe found")
}

func TestDemoCanMoveOutOfAHazardMargin(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGame(false)
	x, y := g.player.x, g.player.y
	g.mines = []landmine{{x: x + 16, y: y, kind: 20}}
	if mineTouchesActor(g.mines[0], g.player) {
		t.Fatal("the fixture must begin outside the actual trigger rectangle")
	}
	nx, _, ok := g.demoStep(x, y, -1, 0)
	if !ok || nx != x-2 {
		t.Fatal("the conservative margin prevented movement away from a hazard")
	}
}

func TestDemoWaitsForABlastBeforeLeavingATank(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGame(true)
	g.eventIndex = len(g.resources.events)
	g.enemies = nil
	g.player.inTank, g.player.tankBurning, g.player.tankBurn = true, true, 70
	// The complete original-frame oracle has three visible phases. After
	// phase one, two damaging updates remain; the last phase is then removed
	// before damage, so an ordinary dismount is safe on that third update.
	g.blasts[2] = projectile{x: g.player.x - 8, y: g.player.y + 4, enemy: true, grenade: true, impact: 3}
	for range 2 {
		g.stepPlay([2]playerControls{})
		if !g.player.inTank || g.player.health != 1 || g.lives != 5 {
			t.Fatal("the demonstration stepped out into an active blast")
		}
	}
	g.stepPlay([2]playerControls{})
	if g.player.inTank || g.player.health != 1 || g.lives != 5 {
		t.Fatal("the demonstration failed to use its ordinary safe exit")
	}
}

func TestOccupiedTankRetainsItsOriginalObjectSlot(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGame(false)
	g.enemies = []actor{{x: g.player.x, y: g.player.y, kind: 4, slot: 19, health: 1, tankFuel: 64}}
	if !g.useTank(0) {
		t.Fatal("fixture could not enter its tank")
	}
	g.applyLevelEvent(levelEvent{kind: 1, count: 2, x: 100, spacing: 30, progress: 1290})
	if len(g.enemies) != 2 || g.enemies[0].slot != 20 || g.enemies[1].slot != 21 {
		t.Fatal("a new vehicle reused the occupied passenger vehicle's slot")
	}
}

func TestDemoGrenadeAimUsesTheRealFlightAndBlast(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGame(false)
	g.enemies = []actor{
		{x: g.player.x - 8, y: g.player.y - 68, kind: 2, slot: 3, health: 1},
		{x: g.player.x, y: g.player.y - 68, kind: 2, slot: 4, health: 1},
		{x: g.player.x - 16, y: g.player.y - 68, kind: 3, slot: 0, health: 1},
	}
	aim, ok := g.demoGrenadeAim()
	if !ok {
		t.Fatal("a reachable group was not targeted")
	}
	g.player.direction = aim
	g.fireProjectile(&g.player, false, 0, true, 4)
	for range 18 {
		g.updateProjectiles()
	}
	for _, enemy := range g.enemies {
		if enemy.health != 0 {
			t.Fatal("the selected grenade trajectory did not hit its targets")
		}
	}
}

func TestDemoReplansForANewAmmunitionPickup(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGame(true)
	g.enemies = nil
	g.ammunition = 10
	g.pickups = []pickup{{x: g.player.x + 16, y: g.player.y - 48, kind: 1}}
	g.demoControls()
	if g.demoTargetKind != 1 || g.demoTarget != (navPoint{g.pickups[0].x, g.pickups[0].y}) || g.ammunition != 10 {
		t.Fatal("the new pickup did not trigger ordinary route planning")
	}
}

func TestDemoOpensTheOriginalBlockedBridgeWithARealGrenade(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGame(true)
	g.scene = playScene
	g.player.x, g.player.y = 78, 12026
	g.cameraY, g.cameraBackY = 11930, 11936
	g.secondaryTerrain = g.cameraY < secondaryMapRows*tileSize
	g.enemies, g.mines, g.projectiles = nil, nil, nil
	g.randomSpawn, g.redSpawn = false, false
	g.eventIndex = len(g.resources.events)
	g.ammunition, g.grenades = 0, 2
	g.buildDemoPath()
	if g.destroyed[748][5] || g.grenades != 2 {
		t.Fatal("route planning changed the gate or spent ammunition")
	}
	for tick := 0; tick < 200 && g.player.y > 11940; tick++ {
		g.stepPlay([2]playerControls{})
	}
	if !g.destroyed[748][5] || g.grenades >= 2 || g.player.y > 11940 || g.lives != 5 || g.player.health != 1 {
		t.Fatalf("ordinary grenade route did not cross the gate: y%.0f grenades%d lives%d health%d cleared%v", g.player.y, g.grenades, g.lives, g.player.health, g.destroyed[748][5])
	}
}

func TestDemoPredictsTheActualMarchPauseAndBlockedTurn(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	for row := range g.destroyed {
		for column := range g.destroyed[row] {
			g.destroyed[row][column] = true
		}
	}
	for _, start := range []actor{
		{x: 100, y: 10000, kind: 0, health: 1, marchX: 1, marchY: 1, marchTime: 3, shots: 255},
		{x: 100, y: 10000, kind: 0, health: 1, marchY: 1, marchTime: 1},
	} {
		if start.marchTime == 1 {
			row, column := int(start.y+17)/tileSize, int(start.x)/tileSize
			g.destroyed[row][column] = false
			g.resources.levelCodes[row][column] = 200
			if !g.solidAt(start.x, start.y+2) {
				t.Fatal("the blocked-turn fixture has no wall")
			}
		}
		x, y := g.demoEnemyPosition(start, 12)
		actual := start
		for range 12 {
			g.marchEnemy(&actual)
		}
		if x != actual.x || y != actual.y {
			t.Fatalf("forecast%.0f,%.0f differs from ordinary march%.0f,%.0f", x, y, actual.x, actual.y)
		}
	}
}

func TestDemoRejectsThePassableExplosiveAtTheOriginalTankApproach(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGame(false)
	g.player.x, g.player.y = 94, 12392
	g.player.inTank, g.player.tankFuel = true, 50
	// Raw255 is listed as passable, but the player's explosive-scenery
	// branch runs before that lookup. This exact approach burned the demo.
	if g.terrainCode(g.player.x, g.player.y-8) != 255 || g.solidAt(g.player.x, g.player.y-8) {
		t.Fatal("the original passable-explosive fixture changed")
	}
	if _, _, safe := g.demoStep(g.player.x, g.player.y, 0, -1); safe || g.player.tankBurning {
		t.Fatal("route prediction accepted or triggered the explosive object")
	}
	g.moveActor(&g.player, 0, -2)
	if !g.player.tankBurning || g.player.tankBurn != 70 {
		t.Fatal("the ordinary player movement did not reproduce the tank hazard")
	}
}

func TestDemoForecastCachePreservesThreatDecisions(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGame(false)
	g.enemies = []actor{
		{x: g.player.x - 20, y: g.player.y - 40, kind: 0, health: 1, marchTime: 3, marchY: 1, shots: 3},
		{x: g.player.x + 20, y: g.player.y - 10, kind: 3, health: 1, slot: 2},
	}
	g.projectiles = []projectile{
		{x: g.player.x - 30, y: g.player.y - 20, vx: 4, vy: 4, life: 15, enemy: true, owner: -1},
		{x: g.player.x + 30, y: g.player.y - 60, vy: 4, life: 15, enemy: true, owner: -1, grenade: true},
	}
	var expected []float64
	nonzero := false
	for depth := 1; depth <= demoHorizon; depth++ {
		for _, dx := range []float64{-20, 0, 20} {
			for _, dy := range []float64{-20, 0, 20} {
				value := g.demoThreatAt(g.player.x+dx, g.player.y+dy, depth)
				expected = append(expected, value)
				nonzero = nonzero || value != 0
			}
		}
	}
	if !nonzero {
		t.Fatal("the forecast fixture contains no real threats")
	}
	g.prepareDemoForecast()
	index := 0
	for depth := 1; depth <= demoHorizon; depth++ {
		for _, dx := range []float64{-20, 0, 20} {
			for _, dy := range []float64{-20, 0, 20} {
				if got := g.demoThreatAt(g.player.x+dx, g.player.y+dy, depth); got != expected[index] {
					t.Fatalf("cached threat at depth%d changed from%g to%g", depth, expected[index], got)
				}
				index++
			}
		}
	}
	if g.player.health != 1 || g.ammunition != 99 || g.grenades != 50 {
		t.Fatal("forecast preparation changed ordinary player resources")
	}
}

func TestDemoEscapesAnApproachingOriginalGrenade(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGame(true)
	g.scene = playScene
	g.player = actor{x: 104, y: 10000, health: 1}
	g.cameraY, g.cameraBackY = 9904, 9936
	g.secondaryTerrain = g.cameraY < secondaryMapRows*tileSize
	g.enemies, g.mines = nil, nil
	g.randomSpawn, g.redSpawn = false, false
	g.eventIndex = len(g.resources.events)
	g.ammunition, g.grenades = 0, 0
	for row := range g.destroyed {
		for column := range g.destroyed[row] {
			g.destroyed[row][column] = true
		}
	}
	g.projectiles = []projectile{{x: 96, y: 9940, vy: 4, life: 15, enemy: true, owner: -1, grenade: true, frame: 9}}
	g.buildDemoPath()
	for range 20 {
		g.stepPlay([2]playerControls{})
	}
	if g.player.health != 1 || g.player.deadTicks != 0 || g.lives != 5 || g.ammunition != 0 || g.grenades != 0 {
		t.Fatal("the demonstration failed to escape using only ordinary movement")
	}
}

func TestDemoIgnoresAircraftPartsThatCannotDestroyTheirGroup(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGame(true)
	g.player.x, g.player.y = 140, 10000
	g.enemies = nil
	g.aircraft = helicopter{x: 20, y: 9998, flying: true, alive: [5]bool{true, true, true, true, true}}
	shot := projectile{x: 60, y: 10004}
	side := g.aircraft.part(1)
	side.y += 36
	if !grenadeHitsActor(shot, side) {
		t.Fatal("the original side-part targeting fixture changed")
	}
	if _, ok := g.demoGrenadeAim(); ok {
		t.Fatal("the demonstration wasted a grenade on an ineffective side-part hit")
	}
}

func TestDemoReachesTheActualPickupRectangle(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGame(true)
	g.scene = playScene
	g.player = actor{x: 79, y: 9788, health: 1}
	g.cameraY, g.cameraBackY = 9692, 9696
	g.secondaryTerrain = g.cameraY < secondaryMapRows*tileSize
	g.enemies, g.mines = nil, nil
	g.randomSpawn, g.redSpawn = false, false
	g.eventIndex = len(g.resources.events)
	g.ammunition = 10
	g.pickups = []pickup{{x: 68, y: 9776, kind: 1}}
	if smallObjectTouchesActor(68, 9776, g.player) {
		t.Fatal("the twelve-pixel-above pickup fixture changed")
	}
	g.buildDemoPath()
	for tick := 0; tick < 80 && g.ammunition < 99; tick++ {
		g.stepPlay([2]playerControls{})
	}
	if g.ammunition != 99 || len(g.pickups) != 0 || g.player.health != 1 || g.lives != 5 {
		t.Fatal("the demonstration stopped within its old tolerance before collecting the real pickup")
	}
}

func TestDemoPreservesGrenadesForTargetsThatNeedThem(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGame(true)
	g.enemies = []actor{{x: g.player.x, y: g.player.y - 68, health: 1, kind: 2, slot: 3}}
	if _, ok := g.demoGrenadeAim(); ok {
		t.Fatal("the demonstration spent a grenade on a lone ordinary bullet target")
	}
	g.ammunition = 0
	if _, ok := g.demoGrenadeAim(); !ok {
		t.Fatal("the demonstration ignored its remaining ordinary grenade action")
	}
}

func TestDemoSavesTankAmmunitionForThreatsAndSupplies(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGame(true)
	g.player.inTank = true
	g.enemies = []actor{{x: g.player.x, y: g.player.y - 40, health: 1, kind: 2, slot: 3}}
	if _, ok := g.demoAim(); ok {
		t.Fatal("the demonstration spent tank ammunition on a harmless fixed bullet gunner")
	}
	g.enemies[0].kind, g.enemies[0].slot = 3, 0
	if _, ok := g.demoAim(); !ok {
		t.Fatal("the demonstration ignored a dangerous grenadier in cannon range")
	}
	g.enemies[0].kind, g.enemies[0].slot, g.enemies[0].shots = 0, 5, 3
	if _, ok := g.demoAim(); !ok {
		t.Fatal("the demonstration ignored marching infantry that can throw grenades")
	}
}

func TestDemoContinuesAtTheOriginalOddLeftBoundary(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGame(true)
	g.scene = playScene
	g.player = actor{x: 13, y: 2606, health: 1}
	g.cameraY, g.cameraBackY = 2510, 2528
	g.secondaryTerrain = g.cameraY < secondaryMapRows*tileSize
	g.enemies, g.mines = nil, nil
	g.eventIndex = len(g.resources.events)
	g.randomSpawn, g.redSpawn = false, false
	g.buildDemoPath()
	for range 80 {
		g.stepPlay([2]playerControls{})
	}
	if g.player.y >= 2580 || g.player.health != 1 {
		t.Fatalf("the demonstration alternated at the original legal edge: player%.0f/%.0f", g.player.x, g.player.y)
	}
}

func TestDemoPreservesAnAvailableTankBesideTheTargetGroup(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGame(true)
	g.enemies = []actor{
		{x: g.player.x - 8, y: g.player.y - 68, health: 1, kind: 2, slot: 3},
		{x: g.player.x, y: g.player.y - 68, health: 1, kind: 2, slot: 4},
		{x: g.player.x + 8, y: g.player.y - 68, health: 1, kind: 0, slot: 5},
		{x: g.player.x, y: g.player.y - 68, health: 1, kind: 4, slot: 19, tankFuel: 64},
	}
	if _, ok := g.demoGrenadeAim(); ok {
		t.Fatal("the demonstration destroyed its available transport to hit ordinary infantry")
	}
}

func TestDemoKeepsTankProtectionUntilTheOutwardBurstPasses(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGame(true)
	g.scene = playScene
	g.player = actor{x: 102, y: 10000, health: 1, inTank: true, tankSlot: 19, tankFuel: 46, tankBurning: true, tankBurn: 60}
	g.cameraY, g.cameraBackY = 9904, 9936
	g.secondaryTerrain = g.cameraY < secondaryMapRows*tileSize
	g.enemies, g.mines = nil, nil
	g.randomSpawn, g.redSpawn = false, false
	g.eventIndex = len(g.resources.events)
	g.ammunition, g.grenades = 0, 0
	for row := range g.destroyed {
		for column := range g.destroyed[row] {
			g.destroyed[row][column] = true
		}
	}
	g.burst = radialBurst{active: true, remaining: 6, owner: -1}
	for i := range g.burst.parts {
		g.burst.parts[i] = navPoint{52, 10000}
	}
	g.buildDemoPath()
	g.stepPlay([2]playerControls{})
	if !g.player.inTank {
		t.Fatal("the demonstration left its protection before the outward fragments arrived")
	}
	for tick := 0; tick < 20 && g.player.inTank; tick++ {
		g.stepPlay([2]playerControls{})
	}
	if g.player.inTank || g.player.health != 1 || g.player.deadTicks != 0 || g.lives != 5 {
		t.Fatal("the demonstration failed to move and dismount safely using the original burning interval")
	}
}

func TestDemoKeepsExplosivesForTargetsItsBulletsCannotDefeat(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGame(false)
	g.player = actor{x: 104, y: 10000, health: 1}
	g.cameraY = 9904
	g.secondaryTerrain = g.cameraY < secondaryMapRows*tileSize
	g.grenades, g.ammunition = 4, 99
	g.enemies = []actor{
		{x: 88, y: 9932, health: 1, kind: 2},
		{x: 96, y: 9932, health: 1, kind: 2},
		{x: 104, y: 9932, health: 1, kind: 2},
	}
	if _, ok := g.demoGrenadeAim(); ok {
		t.Fatal("the demonstration spent its last explosives on ordinary infantry")
	}
	g.final = finalEncounter{x: 80, y: 9928, active: true}
	if _, ok := g.demoFinalAimFrom(g.player.x, g.player.y); !ok {
		t.Fatal("the saved explosives could not be aimed at the original final target")
	}
}

func TestDemoAvoidsItsUpgradedGrenadesOriginalHostileFragments(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGame(true)
	g.scene = playScene
	g.player = actor{x: 104, y: 10000, health: 1}
	g.cameraY, g.cameraBackY = 9904, 9936
	g.secondaryTerrain = g.cameraY < secondaryMapRows*tileSize
	g.enemies, g.mines = nil, nil
	g.randomSpawn, g.redSpawn = false, false
	g.eventIndex = len(g.resources.events)
	g.ammunition, g.grenades, g.grenadePower[0] = 0, 0, 2
	for row := range g.destroyed {
		for column := range g.destroyed[row] {
			g.destroyed[row][column] = true
		}
	}
	g.projectiles = []projectile{{x: 96, y: 10004, vy: -4, life: 18, owner: 0, grenade: true}}
	g.buildDemoPath()
	for range 28 {
		g.stepPlay([2]playerControls{})
	}
	if g.player.health != 1 || g.player.deadTicks != 0 || g.lives != 5 || g.ammunition != 0 || g.grenades != 0 {
		t.Fatal("the demonstration failed to anticipate its original hostile fragments")
	}
}

func TestDemoAnticipatesTheTankExplosionCausedByItsOwnGrenade(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGame(true)
	g.scene = playScene
	g.player = actor{x: 104, y: 10000, health: 1}
	g.cameraY, g.cameraBackY = 9904, 9936
	g.secondaryTerrain = g.cameraY < secondaryMapRows*tileSize
	g.enemies = []actor{{x: 96, y: 9932, health: 1, kind: 4, slot: 19, tankFuel: 64}}
	g.mines = nil
	g.randomSpawn, g.redSpawn = false, false
	g.eventIndex = len(g.resources.events)
	g.ammunition, g.grenades = 0, 0
	for row := range g.destroyed {
		for column := range g.destroyed[row] {
			g.destroyed[row][column] = true
		}
	}
	g.projectiles = []projectile{{x: 96, y: 10004, vy: -4, life: 18, owner: 0, grenade: true}}
	g.buildDemoPath()
	for range 30 {
		g.stepPlay([2]playerControls{})
	}
	if g.player.health != 1 || g.player.deadTicks != 0 || g.lives != 5 || g.ammunition != 0 || g.grenades != 0 {
		t.Fatal("the demonstration walked into the real vehicle explosion after its grenade landed")
	}
}

func TestDemoFindsARealLaunchPositionForTheOriginalFinalEncounter(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGame(true)
	g.scene = playScene
	g.player = actor{x: 72, y: 362, health: 1}
	g.cameraY, g.cameraBackY = 266, 288
	g.secondaryTerrain = g.cameraY < secondaryMapRows*tileSize
	g.enemies, g.mines = nil, nil
	g.randomSpawn, g.redSpawn = false, false
	g.eventIndex = len(g.resources.events)
	g.ammunition, g.grenades = 79, 8
	g.applyLevelEvent(levelEvent{kind: 25, progress: 14640})
	g.buildDemoPath()
	for tick := 0; tick < 160 && !g.final.defeated; tick++ {
		g.stepPlay([2]playerControls{})
	}
	if !g.final.defeated || g.grenades >= 8 || g.player.health != 1 || g.lives != 5 {
		t.Fatalf("the ordinary final attack failed: player%.0f/%.0f lives%d health%d grenades%d defeated%v", g.player.x, g.player.y, g.lives, g.player.health, g.grenades, g.final.defeated)
	}
}

// Measure the complete native demonstration tick, including combat decisions,
// scrolling, events and mortal restarts. Rendering is measured separately.
func BenchmarkDemonstrationTick(b *testing.B) {
	g, err := New(Options{Mute: true})
	if err != nil {
		b.Fatal(err)
	}
	g.startGame(true)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if g.scene == titleScene || g.scene == victoryScene {
			g.startGame(true)
		}
		g.tick++
		g.updatePlay()
	}
}

// At this original late-level marker the open boundary passes through X14.
// Ordinary two-pixel input can reach it from X28, but a four-pixel route graph
// silently discarded that reachable column and waited forever below the wall.
func TestDemoUsesTheNarrowOriginalBoundaryBypass(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGame(true)
	g.scene = playScene
	g.player = actor{x: 28, y: 2608, health: 1}
	g.cameraY, g.cameraBackY = 2496, 2528
	g.secondaryTerrain = g.cameraY < secondaryMapRows*tileSize
	g.enemies = nil
	g.randomSpawn, g.redSpawn = false, false
	g.eventIndex = len(g.resources.events)
	g.mines = []landmine{{x: 34, y: 2594, kind: 20, slot: 85}}
	g.buildDemoPath()
	for tick := 0; tick < 160 && g.player.y > 2544; tick++ {
		g.stepPlay([2]playerControls{})
	}
	if g.player.y > 2544 || g.player.health != 1 || g.lives != 5 {
		t.Fatalf("the ordinary boundary bypass failed: position%.0f/%.0f health%d lives%d", g.player.x, g.player.y, g.player.health, g.lives)
	}
}

// The real late-level pair of strike markers can continually re-hit a tank.
// Waiting for their bursts to end without driving away caused an endless demo.
func TestDemoDrivesBurningTankOutOfRepeatedStrikeMarker(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGame(true)
	g.scene = playScene
	g.player = actor{x: 102, y: 6190, health: 1, inTank: true, tankSlot: 20, tankFuel: 63, tankBurn: 66, tankBurning: true}
	g.cameraY, g.cameraBackY = 6114, 6144
	g.secondaryTerrain = g.cameraY < secondaryMapRows*tileSize
	g.enemies = nil
	g.randomSpawn, g.redSpawn = false, false
	g.eventIndex = len(g.resources.events)
	g.ammunition, g.grenades = 0, 51
	g.mines = []landmine{{x: 64, y: 6214, kind: 20, slot: 85}, {x: 104, y: 6190, kind: 20, slot: 86}, {x: 64, y: 6188, kind: 18, slot: 81}, {x: 129, y: 6164, kind: 18, slot: 82}}
	g.burst = radialBurst{active: true, remaining: 6, owner: -1}
	for i := range g.burst.parts {
		g.burst.parts[i] = navPoint{104, 6190}
	}
	for tick := 0; tick < 80 && g.player.inTank; tick++ {
		g.stepPlay([2]playerControls{})
	}
	if g.player.inTank || g.player.health != 1 || g.player.deadTicks != 0 || g.lives != 5 || g.player.x == 102 && g.player.y == 6190 {
		t.Fatalf("ordinary driving and dismount failed: player%+v lives%d", g.player, g.lives)
	}
}

// This exercises one uninterrupted game through the shared normal-control
// loop. It catches route traps that isolated movement decisions cannot reveal.
func TestDemonstrationReachesOriginalEnding(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGame(true)
	if g.lives != 5 || g.ammunition != 99 || g.grenades != 50 {
		t.Fatal("unexpected initial resources")
	}
	for tick := 0; tick < 18000 && g.scene != victoryScene && g.scene != titleScene; tick++ {
		g.tick++
		g.updatePlay()
	}
	if g.scene != victoryScene || !g.ending.active || g.lives < 0 {
		t.Fatalf("demonstration did not reach the original ending: scene%d position%.0f/%.0f lives%d", g.scene, g.player.x, g.player.y, g.lives)
	}
	t.Logf("Reached the original ending after %d gameplay ticks, with %d reserve lives and score %d", g.gameplayTicks, g.lives, g.score)
}

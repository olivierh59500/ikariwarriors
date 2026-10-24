// Package game implements the native Ikari Warriors conversion.
package game

import (
	"cmp"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/olivierh59500/ikariwarriors/internal/sound"
)

const (
	TicksPerSecond  = 25
	ScreenWidth     = 320
	ScreenHeight    = 200
	TouchWidth      = 480
	TouchGameX      = 80
	TouchRightX     = TouchGameX + ScreenWidth
	playfieldX      = 64
	playfieldY      = 8
	playfieldWidth  = mapColumns * tileSize
	playfieldHeight = ScreenHeight - playfieldY
	levelHeight     = mapRows * tileSize
)

type scene uint8

const (
	titleScene scene = iota
	introScene
	playScene
	victoryScene
)

type actor struct {
	x                float64
	y                float64
	direction        int
	health           int
	upper            walkAnimation
	lower            walkAnimation
	upperFalling     bool
	lowerFalling     bool
	kind             int
	slot             int
	marchX           int
	marchY           int
	marchTime        int
	shots            int
	bob              int
	deadTicks        int
	invulnerable     int
	blinkPhase       uint8
	hidden           bool
	wet              bool
	moveDirection    int
	tankSlot         int
	inTank           bool
	tankFuel         int
	tankBurn         int
	tankBurning      bool
	tankFlame        walkAnimation
	tankDriver       int
	tankTurret       navPoint
	tankTurretPlaced bool
	fuelGaugeFull    bool
}

type projectile struct {
	x           float64
	y           float64
	vx          float64
	vy          float64
	enemy       bool
	owner       int
	grenade     bool
	cannon      bool
	power       int
	life        int
	impact      int
	slot        int
	frame       int
	fragments   bool
	airStrike   uint8
	terrainBias float64
}

// playerControls contains only actions available through the ordinary controls.
type playerControls struct {
	x, y         float64
	fire, action bool
	toggleLock   bool
}

type navPoint struct {
	x float64
	y float64
}

type pickup struct {
	x    float64
	y    float64
	kind int
	bob  int
}

// Options selects desktop, Android and verification behavior.
type Options struct {
	Touch   bool
	Demo    bool
	Mute    bool
	Smoke   int
	Capture string
	// RandomSource supplies deterministic randomness when non-nil. It must
	// return a pure, stateless value for each index so copied forecast games
	// advance their own sequence without consuming the live game's inputs.
	RandomSource func(index uint64) uint32 `json:"-"`
}

// Game owns one shared desktop and Android simulation.
type Game struct {
	options                 Options
	resources               *resources
	audio                   *sound.Audio
	scene                   scene
	tick                    int
	gameplayTicks           int
	sceneTick               int
	idle                    int
	paused                  bool
	demo                    bool
	player                  actor
	player2                 actor
	playerCount             int
	arrival                 arrivalSequence
	cooperativeDirection    uint8
	cooperativeScroll       [2]cooperativeScrollState
	directionLock           [2]bool
	enemies                 []actor
	infantryPoses           [19]infantryPose
	platoonMiddlePath       byte
	sceneryBurst            sceneryBurst
	fanTraps                [5]fanTrap
	projectiles             []projectile
	blasts                  [3]projectile
	cannonReservation       [2]int
	pendingFootImpact       [2]int
	pendingOtherFootImpacts [2][2]int
	pickups                 []pickup
	pickupAnimations        [8]walkAnimation
	mines                   []landmine
	mineAnimations          [6]walkAnimation
	mineAlarmTicks          uint8
	aircraft                helicopter
	enemyVehicles           [2]enemyVehicle
	vehicleChassisPhases    [2]vehicleChassisPhase
	boss                    middleBoss
	final                   finalEncounter
	ending                  endingSequence
	tankFuelPhase           [2]uint8
	idleArtilleryTicks      uint32
	airStrikeBusy           [2]bool
	burst                   radialBurst
	burstQueue              []burstRequest
	cameraY                 float64
	cameraBackY             float64
	secondaryTerrain        bool
	eventIndex              int
	randomSpawn             bool
	redSpawn                bool
	lastSpawnScroll         int
	lives                   int
	lives2                  int
	ammunition              int
	ammunition2             int
	grenades                int
	grenades2               int
	score                   int
	score2                  int
	bulletPower             [2]int
	grenadePower            [2]int
	rangeBonus              [2]int
	fireDelay               [2]int
	random                  uint32
	randomSourceIndex       uint64
	destroyed               [mapRows][mapColumns]bool
	sceneryPaint            []sceneryGroundPatch
	demoPath                []navPoint
	demoPathIndex           int
	demoLastX               float64
	demoLastY               float64
	demoStall               int
	demoPlanMines           int
	demoPlanVehicle         bool
	demoPlanScenery         bool
	demoSceneryRow          int
	demoSceneryCol          int
	demoTarget              navPoint
	demoTargetKind          int
	demoFireHeld            bool
	demoForecastActive      bool
	demoEnemyDefeat         [64]int
	demoEnemyForecast       [demoHorizon + 1][34]navPoint
	demoBulletForecast      [demoHorizon + 1][32]demoBulletPrediction
	demoTargetBursts        [64]demoBurstPrediction
	demoBurstCount          int
	demoSceneryForecast     [6]demoSceneryPrediction
	demoSceneryCount        int
	demoAircraftRisk        bool
	demoFiringSearchOffset  int
	navParents              []int
	navQueue                []int
	gamepadIDs              []ebiten.GamepadID
	victoryBackdrop         *ebiten.Image
	playfield               *ebiten.Image
	touchGame               *ebiten.Image
	capture                 bool
	done                    bool
	err                     error
}

// New loads the exact disk graphics, map, font and custom music driver.
func New(options Options) (*Game, error) {
	original, err := loadResources()
	if err != nil {
		return nil, err
	}
	audioEngine, err := sound.New(options.Mute)
	if err != nil {
		return nil, err
	}
	game := &Game{
		cooperativeScroll: [2]cooperativeScrollState{{claimed: true}, {claimed: true}},
		options:           options, resources: original, audio: audioEngine,
		playfield:      ebiten.NewImage(playfieldWidth, playfieldHeight),
		touchGame:      ebiten.NewImage(ScreenWidth, ScreenHeight),
		random:         0x1987cafe,
		mineAlarmTicks: 4,
		tankFuelPhase:  [2]uint8{20, 20},
	}
	for slot := 5; slot <= 18; slot++ {
		game.infantryPoses[slot].lower.frame = -1
	}
	for i := range game.fanTraps {
		game.fanTraps[i].cooldown = 1
	}
	// The official title's forced walk and wave finish at these phases.
	game.player.upper = walkAnimation{step: 7, delay: 1}
	game.player2.upper = game.player.upper
	game.player.blinkPhase, game.player2.blinkPhase = 1, 1
	game.audio.Music("title")
	if options.Demo {
		game.startGame(true)
	}
	return game, nil
}

// Update advances one gameplay tick, separated by two PAL vertical blanks.
func (g *Game) Update() error {
	if g.done {
		if g.err != nil {
			return g.err
		}
		return ebiten.Termination
	}
	if g.capture {
		return nil
	}
	playing := g.scene == playScene || g.scene == victoryScene
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) || playing && inpututil.IsKeyJustPressed(ebiten.KeyF1) {
		g.Back()
		return nil
	}
	gamepadPause := false
	if id, ok := g.standardGamepad(0); ok {
		gamepadPause = inpututil.IsStandardGamepadButtonJustPressed(id, ebiten.StandardGamepadButtonCenterRight)
	}
	if playing && (inpututil.IsKeyJustPressed(ebiten.KeyP) || inpututil.IsKeyJustPressed(ebiten.KeyF2) || gamepadPause) {
		g.paused = !g.paused
	}
	if playing && g.options.Touch && g.touchPausePressed() {
		g.paused = !g.paused
	}
	if g.paused {
		g.audio.SetPaused(true)
		return nil
	}
	g.audio.SetPaused(false)
	g.tick++
	g.sceneTick++
	switch g.scene {
	case titleScene:
		g.updateTitle()
	case introScene:
		g.updateIntro()
	case playScene:
		g.updatePlay()
	case victoryScene:
		g.updateVictory()
	}
	if g.options.Smoke > 0 && g.tick >= g.options.Smoke {
		g.capture = true
	}
	return nil
}

func (g *Game) updateTitle() {
	if inpututil.IsKeyJustPressed(ebiten.KeyDigit2) {
		g.startGamePlayers(false, 2)
		return
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDigit1) {
		g.startGamePlayers(false, 1)
		return
	}
	if g.confirmPressed() {
		g.startGamePlayers(false, 1)
		return
	}
	g.idle++
	if g.idle >= 15*TicksPerSecond {
		g.startGame(true)
	}
}

func (g *Game) updateIntro() {
	if g.confirmPressed() {
		for !g.arrival.advance() {
		}
	}
	if g.arrival.advance() {
		g.player.upper, g.player.lower = g.arrival.upper[0], g.arrival.lower[0]
		// The absent player's walking counters also survive a one-player game.
		g.player2.upper, g.player2.lower = g.arrival.upper[1], g.arrival.lower[1]
		if g.playerCount > 1 {
			g.cooperativeScroll = g.arrival.cooperative
		} else {
			g.cooperativeScroll[0].up, g.cooperativeScroll[0].claimed = true, true
		}
		// The last forced scroll loads the first playable strip. Both that
		// step and the extra handover scroll can create reinforcements.
		for _, progress := range [2]int{g.arrival.paintScroll, g.arrival.scroll} {
			if progress >= 130 {
				g.spawnScrollEnemiesAt(progress)
			}
		}
		g.scene = playScene
		g.sceneTick = 0
		g.audio.Music("game")
	}
}

func (g *Game) updateVictory() {
	// The original keeps the message over its last frame, with gameplay stopped.
	if g.confirmPressed() || g.sceneTick >= victoryMessageTicks {
		g.Back()
	}
}

func (g *Game) startGame(demo bool) {
	g.startGamePlayers(demo, 1)
}

func (g *Game) startGamePlayers(demo bool, players int) {
	g.scene, g.sceneTick, g.idle, g.paused, g.demo = introScene, 0, 0, false, demo
	g.playerCount = max(1, min(2, players))
	if g.playerCount == 1 {
		g.cooperativeScroll[0].claimed, g.cooperativeScroll[1].claimed = false, false
	}
	g.cooperativeScroll[0].terrainOffset, g.cooperativeScroll[1].terrainOffset = 0, 0
	g.arrival = newArrival(g.playerCount)
	for player, value := range [2]actor{g.player, g.player2} {
		g.arrival.upper[player].step, g.arrival.upper[player].delay = value.upper.step, value.upper.delay
		g.arrival.lower[player].step, g.arrival.lower[player].delay = value.lower.step, value.lower.delay
	}
	g.arrival.cooperative = g.cooperativeScroll
	g.directionLock = [2]bool{}
	firstBlink, secondBlink := g.player.blinkPhase, g.player2.blinkPhase
	g.player = actor{x: 72, y: levelHeight - 38, direction: 0, health: 1, blinkPhase: firstBlink}
	g.player2 = actor{x: 132, y: levelHeight - 38, direction: 0, health: 1, blinkPhase: secondBlink}
	if g.playerCount == 1 {
		// The unentered partner keeps original terrain progress-210 separately
		// from its unused landing pose. Both progress values feed late guns.
		g.cooperativeScroll[1].terrainOffset = levelHeight - 16 + 210 - g.player2.y
		g.arrival.cooperative[1].terrainOffset = g.cooperativeScroll[1].terrainOffset
	}

	g.player.upper = walkAnimation{step: 1, frame: 8}
	g.player.lower = walkAnimation{step: 4, frame: 16}
	g.player2.upper = g.player.upper
	g.player2.lower = g.player.lower
	g.cameraY = g.player.y - 94
	g.cameraBackY = levelHeight - float64((int(levelHeight-g.cameraY)-1)&^31)
	g.secondaryTerrain = false
	for _, enemy := range g.enemies {
		g.rememberInfantryAnimation(enemy)
	}
	g.enemies = g.enemies[:0]
	g.projectiles = g.projectiles[:0]
	g.blasts = [3]projectile{}
	g.cannonReservation = [2]int{}
	g.pendingFootImpact = [2]int{}
	g.pendingOtherFootImpacts = [2][2]int{}
	g.pickups = g.pickups[:0]
	g.mines = g.mines[:0]
	g.resetFanTraps()
	g.aircraft, g.burst = helicopter{}, radialBurst{}
	g.sceneryBurst = sceneryBurst{phase: g.sceneryBurst.phase}
	g.enemyVehicles = [2]enemyVehicle{}
	g.boss = middleBoss{}
	g.final, g.ending = finalEncounter{}, endingSequence{}
	g.victoryBackdrop = nil
	g.idleArtilleryTicks, g.airStrikeBusy = 0, [2]bool{}
	g.demoFiringSearchOffset = 0
	g.burstQueue = g.burstQueue[:0]
	g.destroyed = [mapRows][mapColumns]bool{}
	g.sceneryPaint = nil
	g.eventIndex, g.randomSpawn, g.redSpawn = 0, true, true
	g.gameplayTicks = 0
	g.lastSpawnScroll = levelHeight - int(g.cameraY)
	g.lives, g.ammunition, g.grenades, g.score = 5, 99, 50, 0
	g.lives2, g.ammunition2, g.grenades2, g.score2 = 5, 99, 50, 0
	g.bulletPower, g.grenadePower = [2]int{1, 1}, [2]int{1, 1}
	g.rangeBonus, g.fireDelay = [2]int{}, [2]int{7, 7}
	g.audio.Music("game")
	g.processLevelEvents()
	if demo {
		g.buildDemoPath()
		g.demoLastX, g.demoLastY, g.demoStall = g.player.x, g.player.y, 0
		g.demoFireHeld = false
	}
}

func (g *Game) updatePlay() {
	var controls [2]playerControls
	for player := 0; player < g.playerCount; player++ {
		controls[player].x, controls[player].y = g.playerInput(player)
		controls[player].fire, controls[player].action = g.fireInput(player)
	}
	if !g.demo {
		controls[0].toggleLock = inpututil.IsKeyJustPressed(ebiten.KeySpace)
		controls[1].toggleLock = g.playerCount > 1 && inpututil.IsKeyJustPressed(ebiten.KeyY)
	}
	g.stepPlay(controls)
}

// stepPlay shares the same movement, weapons and life rules between physical
// controls, the demonstration player and deterministic gameplay regression tests.
func (g *Game) stepPlay(controls [2]playerControls) {
	g.gameplayTicks++
	g.idleArtilleryTicks++
	if g.demo {
		controls[0] = g.demoControls()
	}
	// The original jumps to its victory message during the next upward scroll.
	// Preserve the previous completed image before that input changes the world.
	if g.ending.active && g.cameraY >= 30 && g.cameraY <= 32 && (controls[0].y < 0 || g.playerCount > 1 && controls[1].y < 0) {
		if g.victoryBackdrop == nil {
			g.victoryBackdrop = ebiten.NewImage(ScreenWidth, ScreenHeight)
		}
		g.drawPlay(g.victoryBackdrop)
	}
	for player := 0; player < g.playerCount; player++ {
		g.stepPlayer(player, controls[player])
		if g.scene == victoryScene {
			return
		}
		g.advancePlayerProjectiles(player)
	}
	// Each owned shot pool runs immediately after its player's input. Both
	// blast pools then advance before the shared moving fragments and enemies.
	g.advanceBlast(0)
	g.advanceBlast(1)
	// The original player controls run before the moving fragment probes.
	// Applying a burst first prevents a valid last-moment sidestep.
	g.updateBurst()
	g.updateSceneryBurst()
	g.advanceTanks()
	g.updateEnemyVehicles()
	g.advanceEnemyDeaths()
	// The original resolves P2 and P1 falls before living enemies choose targets.
	previousScene := g.scene
	for player := g.playerCount - 1; player >= 0; player-- {
		g.advancePlayerFall(player)
	}
	if g.scene != previousScene {
		return
	}
	g.resolveAircraftDamage()
	g.updateLivingEnemies()
	g.advanceEnemyProjectiles()
	g.updateBunkers()
	g.advanceAircraft()
	g.fireAircraft()
	g.animatePickups()
	g.animateMines()
	g.moveHomingMines(g.randomValue)
	g.updateMiddleBoss()
	g.boss.applyDamage()
	// $9330 follows enemy-shot movement and precedes the vehicle firing pass.
	// Its new fan bullets remain at their muzzle until the following frame.
	g.updateFanTraps()
	g.fireEnemyVehicles()
	g.finishEnemyVehicleDamage()
	g.updateHorizontalGuns()
	g.updateFinalEncounter()
	g.updateIdleArtillery()
	// Protection and blink follow collisions; falling/reappearance ran earlier.
	g.finishTankTick()
	for player := 0; player < g.playerCount; player++ {
		g.advancePlayerProtection(player)
	}
	if g.ending.active {
		g.ending.ticks++
		g.ammunition, g.ammunition2, g.grenades, g.grenades2 = 0, 0, 0, 0
	}
	// $916c/$1224a retire small objects after both inputs and their scrolling.
	g.pruneOffscreenProjectiles()
	g.prunePickups()
	g.pruneMines()
	g.pruneOffscreenEnemies()
	g.pruneOffscreenEnemyVehicles()
	g.pruneOffscreenAircraftParts()
	g.rememberPlatoonMiddlePath()
	// The original accepts keyboard lock actions after physics and painting
	// ($119ca/$11ab6), so they affect the next movement and firing pass.
	for player := 0; player < g.playerCount; player++ {
		if controls[player].toggleLock {
			g.directionLock[player] = !g.directionLock[player]
		}
	}
}

func (g *Game) playerInput(player int) (float64, float64) {
	dx, dy := 0.0, 0.0
	if player == 1 {
		if ebiten.IsKeyPressed(ebiten.KeyJ) {
			dx--
		}
		if ebiten.IsKeyPressed(ebiten.KeyL) {
			dx++
		}
		if ebiten.IsKeyPressed(ebiten.KeyI) {
			dy--
		}
		if ebiten.IsKeyPressed(ebiten.KeyK) {
			dy++
		}
	} else {
		if ebiten.IsKeyPressed(ebiten.KeyArrowLeft) || ebiten.IsKeyPressed(ebiten.KeyA) {
			dx--
		}
		if ebiten.IsKeyPressed(ebiten.KeyArrowRight) || ebiten.IsKeyPressed(ebiten.KeyD) {
			dx++
		}
		if ebiten.IsKeyPressed(ebiten.KeyArrowUp) || ebiten.IsKeyPressed(ebiten.KeyW) {
			dy--
		}
		if ebiten.IsKeyPressed(ebiten.KeyArrowDown) || ebiten.IsKeyPressed(ebiten.KeyS) {
			dy++
		}
	}
	if id, ok := g.standardGamepad(player); ok {
		horizontal := ebiten.StandardGamepadAxisValue(id, ebiten.StandardGamepadAxisLeftStickHorizontal)
		vertical := ebiten.StandardGamepadAxisValue(id, ebiten.StandardGamepadAxisLeftStickVertical)
		if math.Abs(horizontal) > 0.18 {
			dx = horizontal
		}
		if math.Abs(vertical) > 0.18 {
			dy = vertical
		}
		if ebiten.IsStandardGamepadButtonPressed(id, ebiten.StandardGamepadButtonLeftLeft) {
			dx = -1
		}
		if ebiten.IsStandardGamepadButtonPressed(id, ebiten.StandardGamepadButtonLeftRight) {
			dx = 1
		}
		if ebiten.IsStandardGamepadButtonPressed(id, ebiten.StandardGamepadButtonLeftTop) {
			dy = -1
		}
		if ebiten.IsStandardGamepadButtonPressed(id, ebiten.StandardGamepadButtonLeftBottom) {
			dy = 1
		}
	}
	if player == 0 && g.options.Touch {
		for _, id := range ebiten.AppendTouchIDs(nil) {
			x, y := ebiten.TouchPosition(id)
			if x >= TouchGameX || y < 42 {
				continue
			}
			dx = float64(x-TouchGameX/2) / 30
			dy = float64(y-124) / 30
			magnitude := math.Hypot(dx, dy)
			if magnitude < 0.16 {
				dx, dy = 0, 0
			} else if magnitude > 1 {
				dx, dy = dx/magnitude, dy/magnitude
			}
		}
	}
	return dx, dy
}

func (g *Game) fireInput(player int) (bool, bool) {
	gamepadFire, gamepadGrenade := false, false
	if id, ok := g.standardGamepad(player); ok {
		duration := inpututil.StandardGamepadButtonPressDuration(id, ebiten.StandardGamepadButtonRightBottom)
		gamepadFire = duration == 1
		gamepadGrenade = heldGrenade(duration) || grenadeButton(inpututil.StandardGamepadButtonPressDuration(id, ebiten.StandardGamepadButtonRightRight))
	}
	if player == 1 {
		duration := inpututil.KeyPressDuration(ebiten.KeyO)
		return duration == 1 || gamepadFire, grenadeButton(inpututil.KeyPressDuration(ebiten.KeyU)) || heldGrenade(duration) || gamepadGrenade
	}
	duration := inpututil.KeyPressDuration(ebiten.KeyZ)
	fire := duration == 1 || gamepadFire
	grenade := grenadeButton(inpututil.KeyPressDuration(ebiten.KeyX)) || heldGrenade(duration) || gamepadGrenade
	if g.options.Touch {
		for _, id := range ebiten.AppendTouchIDs(nil) {
			x, y := ebiten.TouchPosition(id)
			if x < TouchRightX+8 || x >= TouchRightX+72 {
				continue
			}
			duration := inpututil.TouchPressDuration(id)
			if y >= 22 && y < 90 {
				fire = fire || duration == 1
				grenade = grenade || heldGrenade(duration)
			} else if y >= 110 && y < 178 {
				grenade = grenade || grenadeButton(duration)
			}
		}
	}
	return fire, grenade
}

func (g *Game) moveActor(value *actor, dx, dy float64) {
	if value.deadTicks > 0 || value.health <= 0 {
		return
	}
	g.moveActorStep(value, dx, dy)
}

func (g *Game) moveActorStep(value *actor, dx, dy float64) {
	nextX := value.x + dx
	nextY := max(8, value.y+dy)
	// The lower limit only rejects downward input. A respawn begins below
	// it at original Y=180; upward movement must still advance by two pixels.
	if dy > 0 && value.y-g.cameraY >= 171 && g.cameraY >= g.cameraBackY {
		dy = 0
	}
	probeX, probeY := value.x, value.y
	if dx < 0 {
		probeX -= 4
	} else if dx > 0 {
		probeX += 8
	}
	if dx != 0 && g.playerTerrainPassable(value, probeX, value.y) {
		code := g.terrainCode(probeX, value.y)
		value.wet = code == 157 || code == 158
		if value.wet {
			nextX = value.x + digitalAxis(dx)
		}
		rightLimit := 194.0
		if value == &g.player2 {
			rightLimit = 192
		}
		// The original checks the current coordinate before adding its
		// step. It preserves an odd-pixel overshoot and lets a boarded
		// vehicle outside the walking range return without teleporting.
		if dx < 0 && value.x > 14 || dx > 0 && value.x < rightLimit {
			value.x = nextX
		}
	}
	if dy < 0 {
		probeY -= 8
	} else if dy > 0 {
		probeY += 4
	}
	if dy != 0 && g.playerTerrainPassable(value, value.x, probeY) {
		code := g.terrainCode(value.x, probeY)
		value.wet = code == 157 || code == 158
		if value.wet {
			nextY = value.y + digitalAxis(dy)
		}
		value.y = nextY
	}
}

func (g *Game) solidAt(x, y float64) bool {
	return g.solidPixelAt(x, y)
}

func (g *Game) solidPixelAt(x, y float64) bool {
	// $10c3e looks up the original map byte; $10ec8 searches the passable
	// code table. Outside its X range it returns code zero, not a wall.
	return !g.resources.passable[g.collisionTerrainBank()][g.terrainCode(x, y)]
}

func (g *Game) terrainCode(x, y float64) byte {
	// Original raw X must lie in48..250; actor coordinates subtract48.
	if x < 0 || x > 202 || y < 0 || y >= levelHeight {
		return 0
	}
	row := int(y+15) / tileSize
	if row >= mapRows {
		return 0
	}
	column := int(x) / tileSize
	if g.destroyed[row][column] {
		return 0
	}
	return g.resources.levelCodes[row][column]
}

func (g *Game) updateEnemies() {
	g.advanceEnemyDeaths()
	g.updateLivingEnemies()
}

func (g *Game) updateLivingEnemies() {
	// The backing slice stays in slot order for subsequent contact passes.
	// Only iteration follows the original infantry, normal, grenadier groups.
	for phase := 0; phase < 4; phase++ {
		for index := range g.enemies {
			enemy := &g.enemies[index]
			if enemyUpdatePhase(enemy.slot) != phase || g.enemySharedWithEncounter(enemy.slot) || enemy.health <= 0 {
				continue
			}
			switch enemy.kind {
			case 4:
				if !enemy.tankBurning && g.gameplayTicks%2 == 0 {
					if enemy.bob == 0x5a508-0x42b88 {
						enemy.bob = 0x5a328 - 0x42b88
					} else {
						enemy.bob = 0x5a508 - 0x42b88
					}
				}
			case 6:
				// Bunker aiming and firing follow existing enemy-shot motion.
			case 0, 5:
				g.marchEnemy(enemy)
			case 2, 3:
				g.updateFixedGun(enemy)
			}
			g.rememberInfantryAnimation(*enemy)
		}
	}
}

// $fec4/$10520 run between the two input routines, before enemies move.
// Keep the original slot order even when a lower free slot was reused later.
func (g *Game) advancePlayerProjectiles(player int) {
	pendingImpacts := [3]int{g.pendingFootImpact[player], g.pendingOtherFootImpacts[player][0], g.pendingOtherFootImpacts[player][1]}
	if g.cannonReservation[player] > 0 {
		g.cannonReservation[player]--
	}
	slices.SortStableFunc(g.projectiles, func(a, b projectile) int {
		return cmp.Compare(projectileOrder(a), projectileOrder(b))
	})
	kept := g.projectiles[:0]
	for _, shot := range g.projectiles {
		if shot.enemy || shot.owner != player {
			kept = append(kept, shot)
			continue
		}
		moved, active := g.advanceOwnedProjectile(shot)
		if !active && moved.impact > 0 && !shot.grenade && !g.ownedShotUsesCannon(shot) {
			// A newly detached timer must not advance again this same pass,
			// even if it happens to equal the previous generation's value.
			pendingImpacts[shot.slot] = 0
		}
		if active {
			kept = append(kept, moved)
		}
	}
	g.projectiles = kept
	for slot, previous := range pendingImpacts {
		g.finishPendingOwnedFootImpact(player, slot, previous)
	}
}

// $a2f4 advances enemy slots only after the earlier player collision passes.
func (g *Game) advanceEnemyProjectiles() {
	slices.SortStableFunc(g.projectiles, func(a, b projectile) int {
		return cmp.Compare(projectileOrder(a), projectileOrder(b))
	})
	kept := g.projectiles[:0]
	for _, shot := range g.projectiles {
		if !shot.enemy {
			kept = append(kept, shot)
			continue
		}
		shot.x += shot.vx
		shot.y += shot.vy
		shot.life--
		advanceEnemyProjectileImage(&shot)
		if shot.life <= 0 {
			if shot.grenade || shot.cannon {
				g.startBlast(shot)
			}
		} else {
			kept = append(kept, shot)
		}
	}
	g.projectiles = kept
	g.advanceBlast(2)
}

func projectileOrder(shot projectile) int {
	if shot.enemy {
		return 6 + shot.slot
	}
	return shot.owner*3 + shot.slot
}

func (g *Game) advanceOwnedProjectile(shot projectile) (projectile, bool) {
	if !shot.grenade && !g.ownedShotUsesCannon(shot) {
		moved, visible := g.advancePlayerBullet(shot)
		pending := g.pendingOwnedFootImpact(shot.owner, shot.slot)
		if moved.impact > 0 {
			// While the projectile is present it owns its animation itself.
			*pending = 0
		}
		hit := g.damageBulletTarget(moved)
		// $104be/$10b1a reactivate only the first slot when a new foot
		// impact begins after actor contact. Secondary slots stay cleared.
		renewed := shot.slot == 0 && shot.impact == 0 && moved.impact > 0
		if hit && moved.impact > 0 && !renewed {
			// Actor contact clears the BOB and gun flag, but the original
			// independent animation can still clear a later replacement.
			*pending = moved.impact
		}
		return moved, visible && (!hit || renewed)
	}
	landed := false
	if !shot.grenade {
		shot, landed = g.advanceCannon(shot)
		if !g.ownedCannonIgnoresActors(shot) && g.damageBulletTarget(shot) && !g.ownedBulletUpgrade(shot) {
			landed = true
		}
	} else {
		shot.x += shot.vx
		shot.y += shot.vy
		shot.life--
		landed = shot.life <= 0
	}
	if landed {
		if shot.impact > 0 {
			// The first-slot foot animation continues after this new quad.
			g.pendingFootImpact[shot.owner] = shot.impact - 1
		}
		g.startBlast(shot)
		g.damageGrenadeArea(shot)
		return shot, false
	}
	if shot.impact > 0 {
		shot.impact--
		return shot, shot.impact > 0
	}
	return shot, true
}

// These are the original sprite lifetime bounds. A projectile or departing
// soldier may cross the map edge before its original screen coordinate leaves
// the display; the terrain boundary is not a lifetime boundary.
func (g *Game) smallObjectInBounds(x, y float64) bool {
	return x >= -56 && x <= 264 && y >= g.cameraY-40 && y < g.cameraY+240
}

// The original small-object painter also clears newly allocated off-screen
// shots after the late idle-artillery pass, in the same complete game frame.
func (g *Game) pruneOffscreenProjectiles() {
	g.projectiles = slices.DeleteFunc(g.projectiles, func(shot projectile) bool {
		return !g.smallObjectInBounds(shot.x, shot.y)
	})
}

func (g *Game) wideObjectInBounds(x, y float64) bool {
	return x >= -48 && x <= 272 && y >= g.cameraY-44 && y < g.cameraY+236
}

func (g *Game) prunePickups() {
	kept := g.pickups[:0]
	for _, item := range g.pickups {
		if g.smallObjectInBounds(item.x, item.y) {
			kept = append(kept, item)
		}
	}
	g.pickups = kept
}

// $c0f6/$c282 collect one slot during the entered input routine, after body
// contacts and before mines and vertical movement. Activity survives a new hit
// until this input returns; prior falls are excluded by stepPlayer's entry gate.
func (g *Game) collectPlayerPickup(player int) {
	value := g.player
	if player == 1 {
		value = g.player2
	}
	for kind := 0; kind < 8; kind++ {
		for i, item := range g.pickups {
			if item.kind != kind || !smallObjectOverlapsActor(item.x, item.y, value) {
				continue
			}
			if g.audio != nil {
				g.audio.Effect(4, 3)
			}
			g.collectPickup(player, item.kind)
			g.pickups = append(g.pickups[:i], g.pickups[i+1:]...)
			return
		}
	}
}

func (g *Game) collectPickup(player, kind int) {
	ammunition, grenades, score, value := &g.ammunition, &g.grenades, &g.score, &g.player
	if player == 1 {
		ammunition, grenades, score, value = &g.ammunition2, &g.grenades2, &g.score2, &g.player2
	}
	switch kind {
	case 0:
		*grenades = 99
	case 1:
		*ammunition = 99
	case 2:
		value.tankFuel = 64
		if value.inTank {
			value.fuelGaugeFull = true
		}
	case 3:
		g.rangeBonus[player] = 0
	case 4:
		g.rangeBonus[player] = 20
	case 5:
		for i := range g.enemies {
			enemy := &g.enemies[i]
			// $c238-$c264 scores every active infantry BOB, including
			// soldiers whose fall animation has already started.
			if enemy.slot >= 0 && enemy.slot <= 18 {
				enemy.health = 0
				*score += 100
			}
		}
		return
	case 6:
		g.bulletPower[player] = 2
	case 7:
		g.grenadePower[player] = 2
	default:
		return
	}
	*score += 100
}

func (g *Game) hitPlayer(player int) {
	g.idleArtilleryTicks = 0
	if g.ending.active {
		return
	}
	value := &g.player
	if player == 1 {
		value = &g.player2
	}
	if value.deadTicks > 0 || value.invulnerable > 0 {
		return
	}
	if value.inTank {
		value.tankBurning = true
		// The original replaces the flame image but retains its animation
		// counter and delay, including after an earlier abandoned vehicle.
		value.tankFlame.frame = 0
		// Every original explosive hit renews the escape interval.
		value.tankBurn = 70
		return
	}
	g.audio.Effect(19, 3)
	value.health, value.deadTicks, value.invulnerable = 0, 24, 100
	value.upper = walkAnimation{frame: value.upper.frame}
	value.lower = walkAnimation{frame: value.lower.frame}
	value.upperFalling, value.lowerFalling = false, false
}

func (g *Game) fireProjectile(source *actor, enemy bool, owner int, grenade bool, speed float64) {
	// A tank shot also reserves the second bullet slot without drawing it.
	// Its ordinary range and three ending phases continue after a shell hits.
	if !enemy && source.inTank && !grenade {
		g.cannonReservation[owner] = 13 + g.rangeBonus[owner]/4
	}
	shot := g.projectileFor(source, enemy, owner, grenade, speed)
	if enemy {
		g.addEnemyProjectile(shot)
	} else {
		g.projectiles = append(g.projectiles, shot)
	}
}

func (g *Game) projectileFor(source *actor, enemy bool, owner int, grenade bool, speed float64) projectile {
	dx, dy := directionVector(source.direction)
	power, life := 1, 10
	muzzle := [...]struct{ x, y float64 }{{6, 4}, {4, 10}, {6, 10}, {-5, 8}, {-5, 14}, {-9, 8}, {-10, 10}, {-5, 6}}
	x, y := source.x+muzzle[source.direction&7].x, source.y+muzzle[source.direction&7].y
	if enemy {
		// The original range byte starts at60 and decreases by4 per tick.
		life = 15
		x, y = source.x-4, source.y+12
	}
	if !enemy && owner >= 0 && owner < len(g.bulletPower) {
		power = g.bulletPower[owner]
		if grenade {
			power = 1 + g.grenadePower[owner]
		}
		life += g.rangeBonus[owner] / 4
	}
	if grenade {
		life, speed = 18, 4
		x, y = source.x-8, source.y+4
		if enemy {
			// Enemy grenadiers share the sixty-unit shot allocator. Their
			// horizontal throw has a hundred-unit range and a lower muzzle.
			life, x, y = 15, source.x-4, source.y+12
			if dy == 0 {
				life, y = 25, source.y+5
			} else if dy < 0 {
				x += 12
			}
		}
	}
	if source.inTank && !grenade {
		x, y = source.x+3, source.y+12
		if !enemy && g.grenadePower[owner] > 1 {
			life = 15
		}
	}
	slot := 0
	if !enemy && !grenade && !source.inTank {
		for _, candidate := range [...]int{1, 0, 2} {
			if g.projectileSlotAvailable(owner, candidate) {
				slot = candidate
				break
			}
		}
	}
	shot := projectile{
		x: x, y: y,
		vx: dx * speed, vy: dy * speed,
		enemy: enemy, owner: owner, grenade: grenade, cannon: source.inTank && !grenade, power: power, life: life, slot: slot,
	}
	if !enemy && grenade && owner == 1 {
		// $cd40 reduces Player Two's horizontal throw by one, yielding
		// five pixels per tick; its vertical component remains four.
		shot.vx = dx * 5
	}
	if enemy && grenade {
		// The enemy grenade uses its own static bob, not the player's arc.
		shot.frame = 9
	}
	if !enemy && shot.cannon {
		if g.bulletPower[owner] > 1 {
			shot.frame = 9
		} else if g.grenadePower[owner] > 1 {
			shot.frame = 8
		}
	}
	if !enemy && !grenade && !source.inTank {
		if power > 1 {
			shot.frame = 2
		}
		if slot != 0 {
			shot.terrainBias = 4 - muzzle[source.direction&7].y
		}
	}
	return shot
}

// $bbdc allocates three independently active bullets, without a firing delay.
func (g *Game) bulletAvailable(player int) bool {
	value := &g.player
	if player == 1 {
		value = &g.player2
	}
	if value.inTank {
		return !value.tankBurning && g.projectileSlotAvailable(player, 1) && g.projectileSlotAvailable(player, 0)
	}
	for _, slot := range [...]int{1, 0, 2} {
		if g.projectileSlotAvailable(player, slot) {
			return true
		}
	}
	return false
}

func (g *Game) grenadeAvailable(player int) bool {
	return g.projectileSlotAvailable(player, 0)
}

func (g *Game) projectileSlotAvailable(player, slot int) bool {
	return g.ownedShotMask(player)&(1<<slot) == 0
}

func (g *Game) ownedShotMask(player int) uint8 {
	mask := uint8(0)
	if g.cannonReservation[player] > 0 {
		mask = 2
	}
	for _, shot := range g.projectiles {
		if !shot.enemy && shot.owner == player {
			mask |= 1 << shot.slot
		}
	}
	return mask
}

func (g *Game) randomValue() uint32 {
	if source := g.options.RandomSource; source != nil {
		value := source(g.randomSourceIndex)
		g.randomSourceIndex++
		g.random = value
		return value
	}
	g.random ^= g.random << 13
	g.random ^= g.random >> 17
	g.random ^= g.random << 5
	return g.random
}

func directionForVector(dx, dy float64) int {
	angle := math.Atan2(dy, dx)
	return (int(math.Round((angle+math.Pi/2)/(math.Pi/4))) + 8) & 7
}

func directionVector(direction int) (float64, float64) {
	vectors := [...][2]float64{{0, -1}, {1, -1}, {1, 0}, {1, 1}, {0, 1}, {-1, 1}, {-1, 0}, {-1, -1}}
	value := vectors[direction&7]
	return value[0], value[1]
}

func (g *Game) confirmPressed() bool {
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeyZ) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		return true
	}
	if id, ok := g.standardGamepad(0); ok && inpututil.IsStandardGamepadButtonJustPressed(id, ebiten.StandardGamepadButtonRightBottom) {
		return true
	}
	for _, id := range inpututil.AppendJustPressedTouchIDs(nil) {
		x, _ := ebiten.TouchPosition(id)
		if !g.options.Touch || x >= TouchRightX {
			return true
		}
	}
	return false
}

func (g *Game) standardGamepad(index int) (ebiten.GamepadID, bool) {
	g.gamepadIDs = ebiten.AppendGamepadIDs(g.gamepadIDs[:0])
	current := 0
	for _, id := range g.gamepadIDs {
		if !ebiten.IsStandardGamepadLayoutAvailable(id) {
			continue
		}
		if current == index {
			return id, true
		}
		current++
	}
	return 0, false
}

func (g *Game) touchPausePressed() bool {
	for _, id := range inpututil.AppendJustPressedTouchIDs(nil) {
		x, y := ebiten.TouchPosition(id)
		if x < TouchGameX && y < 42 {
			return true
		}
	}
	return false
}

// Draw renders the native 320 by 200 game and optional side controls.
func (g *Game) Draw(screen *ebiten.Image) {
	target := screen
	if g.options.Touch {
		target = g.touchGame
	}
	switch g.scene {
	case titleScene:
		g.drawTitle(target)
	case introScene:
		g.drawArrival(target)
	case playScene:
		if g.gameplayTicks == 0 && g.arrival.tick == 116 {
			g.drawArrival(target)
		} else {
			g.drawPlay(target)
		}
	case victoryScene:
		g.drawVictory(target)
	}
	if g.paused && (g.scene == playScene || g.scene == victoryScene) {
		vector.DrawFilledRect(target, 96, 88, 128, 24, color.Black, false)
		g.drawText(target, "PAUSED", 136, 96)
	}
	if g.options.Touch {
		screen.Fill(color.Black)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(TouchGameX, 0)
		screen.DrawImage(target, op)
		g.drawTouch(screen)
	}
	if g.capture {
		g.capture = false
		if g.options.Capture != "" {
			pixels := image.NewRGBA(screen.Bounds())
			screen.ReadPixels(pixels.Pix)
			file, err := os.Create(g.options.Capture)
			if err == nil {
				err = png.Encode(file, pixels)
				if closeErr := file.Close(); err == nil {
					err = closeErr
				}
			}
			g.err = err
		}
		g.done = true
	}
}

func (g *Game) drawVictory(screen *ebiten.Image) {
	if g.victoryBackdrop != nil {
		screen.DrawImage(g.victoryBackdrop, nil)
	} else {
		g.drawPlay(screen)
	}
	g.drawVictoryMessage(screen)
}

func (g *Game) drawTitle(screen *ebiten.Image) {
	screen.DrawImage(g.resources.title, nil)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(112, 116)
	screen.DrawImage(g.resources.titleActors, op)
}

func (g *Game) drawBackdrop(screen *ebiten.Image, cameraY float64) {
	g.playfield.Clear()
	screen.DrawImage(g.resources.intro, nil)
	// The landing scene remains beneath the first map rows as they scroll in.
	landing := g.resources.intro.SubImage(image.Rect(playfieldX, playfieldY, playfieldX+playfieldWidth, ScreenHeight)).(*ebiten.Image)
	landingOptions := &ebiten.DrawImageOptions{}
	landingOptions.GeoM.Translate(0, float64(levelHeight)-cameraY)
	g.playfield.DrawImage(landing, landingOptions)
	startChunk := int(cameraY) / levelChunkHeight
	endChunk := min(len(g.resources.levelChunks)-1, int(cameraY+playfieldHeight)/levelChunkHeight)
	for chunk := startChunk; chunk <= endChunk; chunk++ {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(0, float64(chunk*levelChunkHeight)-cameraY)
		g.playfield.DrawImage(g.resources.levelChunks[chunk], op)
	}
	g.drawDestroyedTerrain(cameraY)
}

func (g *Game) drawPlay(screen *ebiten.Image) {
	g.drawBackdrop(screen, g.cameraY)
	g.drawFinalRuins()
	g.drawSmallObjects()
	// The original draws its small-object pass before the wide actors.
	// Bullets and blast fragments therefore pass behind their opaque pixels.
	g.drawWideActors()
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(playfieldX, playfieldY)
	screen.DrawImage(g.playfield, op)
	g.drawHUD(screen)
}

func (g *Game) drawSmallObjects() {
	// $1224a paints the small bobs in ascending slot order.
	for player := range 2 {
		g.drawProjectiles(player, false)
	}
	g.drawBlast(0)
	g.drawBlast(1)
	g.drawBurst()
	g.drawSceneryBurst()
	g.drawPickups()
	g.drawProjectiles(-1, true)
	g.drawBlast(2)
	g.drawMines()
}

func (g *Game) drawPickups() {
	for kind := 0; kind <= 8; kind++ {
		for _, item := range g.pickups {
			if item.kind != kind {
				continue
			}
			if item.bob != 0 {
				offset := item.bob
				if item.kind == 5 {
					if g.player.x < item.x-16 {
						offset = 0x5ae68 - 0x42b88
					} else if g.player.x > item.x+16 {
						offset = 0x5b228 - 0x42b88
					}
				}
				if sprite := g.resources.bob(offset, vehicleBobHeight); sprite != nil {
					op := &ebiten.DrawImageOptions{}
					op.GeoM.Translate(item.x-actorBobWidth/2, item.y-g.cameraY-4)
					g.playfield.DrawImage(sprite, op)
				}
				continue
			}
			if item.kind == 8 {
				if len(g.resources.vehicles[0]) == 0 {
					continue
				}
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Translate(item.x-actorBobWidth/2, item.y-g.cameraY-vehicleBobHeight/2)
				g.playfield.DrawImage(g.resources.vehicles[0][0], op)
				continue
			}
			if item.kind < 0 || item.kind >= len(pickupIconFrames) {
				continue
			}
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(item.x-bobSize/2, item.y-g.cameraY-bobSize/2)
			g.playfield.DrawImage(g.resources.effects[g.pickupImage(item.kind)], op)
		}
	}
}

func (g *Game) drawProjectiles(owner int, enemy bool) {
	var ordered [12]*projectile
	for i := range g.projectiles {
		shot := &g.projectiles[i]
		if shot.enemy == enemy && (enemy || shot.owner == owner) {
			ordered[shot.slot] = shot
		}
	}
	for _, shot := range ordered {
		if shot == nil {
			continue
		}
		frame := g.projectileFrame(*shot)
		if frame >= len(g.resources.effects) {
			continue
		}
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(shot.x-bobSize/2, shot.y-g.cameraY-bobSize/2)
		g.playfield.DrawImage(g.resources.effects[frame], op)
	}
}

func (g *Game) drawBlast(pool int) {
	shot := g.blasts[pool]
	if shot.impact == 0 {
		return
	}
	for quadrant := range 4 {
		frame := grenadeImpactFrame(shot.impact, quadrant)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(shot.x-16+float64(quadrant%2*16), shot.y-g.cameraY-16+float64(quadrant/2*16))
		g.playfield.DrawImage(g.resources.effects[frame], op)
	}
}

var enemyActorSequences = [...][]int{
	{9, 10, 11, 10},
	{18, 19, 20, 19},
	{3, 4, 5, 4},
	{15, 16, 17, 16},
	{6, 7, 8, 7},
	{12, 13, 14, 13},
	{0, 1, 2, 1},
	{21, 22, 23, 22},
}

var enemyPartSequences = [...][]int{
	{12, 13, 14, 15},
	{24, 25, 26, 27},
	{4, 5, 6, 7},
	{20, 21, 22, 23},
	{8, 9, 10, 11},
	{16, 17, 18, 19},
	{0, 1, 2, 3},
	{28, 29, 30, 31},
}

func (g *Game) drawActor(target *ebiten.Image, value *actor, player int) {
	if player >= 0 && (value.hidden || value.health == 0 && value.deadTicks == 0) {
		return
	}
	if player >= 0 && (value.deadTicks > 0 || !value.inTank) {
		upperOffset, lowerOffset := value.playerArtOffsets(player)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(value.x-actorBobWidth/2, value.y-g.cameraY-4)
		if sprite := g.resources.bob(upperOffset, actorBobHeight); sprite != nil {
			target.DrawImage(sprite, op)
		}
		op.GeoM.Translate(0, actorBobHeight)
		if sprite := g.resources.bob(lowerOffset, overlayBobHeight); sprite != nil {
			target.DrawImage(sprite, op)
		}
		return
	}
	if player < 0 && value.bob == 0 && (value.kind == 0 || value.kind == 5 || value.deadTicks > 0 && (value.kind == 2 || value.kind == 3)) {
		upperAddress, lowerAddress := value.infantryArtAddresses()
		bodyHeight := actorBobHeight
		if lowerAddress == 0 {
			bodyHeight = vehicleBobHeight
		}
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(value.x-actorBobWidth/2, value.y-g.cameraY-4)
		if sprite := g.resources.bob(upperAddress-0x42b88, bodyHeight); sprite != nil {
			target.DrawImage(sprite, op)
		}
		op.GeoM.Translate(0, actorBobHeight)
		if lowerAddress != 0 {
			if sprite := g.resources.bob(lowerAddress-0x42b88, overlayBobHeight); sprite != nil {
				target.DrawImage(sprite, op)
			}
		}
		return
	}
	if player >= 0 && value.inTank && value.tankBurning {
		if sprite := g.resources.bob(0x5a328-0x42b88, vehicleBobHeight); sprite != nil {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(value.x-16, value.y-g.cameraY-4)
			target.DrawImage(sprite, op)
		}
		return
	}
	if value.bob != 0 {
		if sprite := g.resources.bob(value.bob, vehicleBobHeight); sprite != nil {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(value.x-actorBobWidth/2, value.y-g.cameraY-4)
			target.DrawImage(sprite, op)
		}
		return
	}
	frames := g.resources.enemies
	parts := g.resources.enemyParts
	direction := value.direction & 7
	vehicle := false
	frame := value.upper.frame
	partFrame := value.lower.frame
	if player < 0 && value.kind == 1 {
		frames = g.resources.vehicles[0]
		parts = nil
		frame = value.upper.frame
		vehicle = true
	}
	if player >= 0 {
		frames = g.resources.players[min(1, player)]
		parts = g.resources.playerParts
		frame = value.upper.frame
		partFrame = value.lower.frame
		if value.inTank {
			vehicleBank := 1
			if player == 1 {
				vehicleBank = 2
			}
			frames = g.resources.vehicles[vehicleBank]
			parts = nil
			frame = value.upper.frame
			vehicle = true
		}
	}
	if frame >= len(frames) {
		return
	}
	image := frames[frame]
	op := &ebiten.DrawImageOptions{}
	// Actor Y is the original terrain probe, four rows below the torso top.
	yOffset := 4
	op.GeoM.Translate(value.x-actorBobWidth/2, value.y-g.cameraY-float64(yOffset))
	target.DrawImage(image, op)
	if vehicle && player < 0 {
		turretBase := 0x59428
		order := [...]int{0, 7, 6, 5, 4, 3, 2, 1}
		if turret := g.resources.bob(turretBase-0x42b88+order[direction]*vehicleBobBytes, vehicleBobHeight); turret != nil {
			target.DrawImage(turret, op)
		}
	}
	if partFrame >= 0 && partFrame < len(parts) {
		// The original Blitter appends the eight leg rows after the torso.
		op.GeoM.Translate(0, actorBobHeight)
		if value.wet {
			offset := 0x4a248 - 0x42b88
			if partFrame*overlayBobBytes&0x40 != 0 {
				offset = 0x4a1a8 - 0x42b88
			}
			if sprite := g.resources.bob(offset, overlayBobHeight); sprite != nil {
				target.DrawImage(sprite, op)
			}
		} else {
			target.DrawImage(parts[partFrame], op)
		}
	}
}

func (g *Game) drawHUD(screen *ebiten.Image) {
	scores := [2]int{g.score, g.score2}
	ammunition := [2]int{g.ammunition, g.ammunition2}
	grenades := [2]int{g.grenades, g.grenades2}
	if g.ending.active {
		ammunition, grenades = g.ending.displayAmmo, g.ending.displayGrenades
		if g.scene == victoryScene {
			scores = g.ending.displayScores
		}
	}
	g.drawFuelGauges(screen)
	vector.DrawFilledRect(screen, 64, 0, 56, 8, g.resources.colors[0], false)
	vector.DrawFilledRect(screen, 224, 0, 56, 8, g.resources.colors[0], false)
	vector.DrawFilledRect(screen, 40, 16, 16, 8, g.resources.colors[0], false)
	vector.DrawFilledRect(screen, 40, 32, 16, 8, g.resources.colors[0], false)
	vector.DrawFilledRect(screen, 304, 16, 16, 8, g.resources.colors[0], false)
	vector.DrawFilledRect(screen, 304, 32, 16, 8, g.resources.colors[0], false)
	g.drawText(screen, fmt.Sprintf("%07d", scores[0]), 64, 0)
	g.drawText(screen, fmt.Sprintf("%07d", scores[1]), 224, 0)
	g.drawText(screen, fmt.Sprintf("%02d", grenades[0]), 40, 16)
	g.drawText(screen, fmt.Sprintf("%02d", ammunition[0]), 40, 32)
	g.drawText(screen, fmt.Sprintf("%02d", grenades[1]), 304, 16)
	g.drawText(screen, fmt.Sprintf("%02d", ammunition[1]), 304, 32)
	vector.DrawFilledRect(screen, 16, 56, 48, 8, g.resources.colors[0], false)
	vector.DrawFilledRect(screen, 272, 56, 48, 8, g.resources.colors[0], false)
	for life := range max(0, min(g.lives+1, 6)) {
		g.drawText(screen, ">", 16+life*8, 56)
	}
	for life := range max(0, min(g.lives2+1, 6)) {
		g.drawText(screen, ">", 272+life*8, 56)
	}
}

func (g *Game) drawText(target *ebiten.Image, text string, x, y int) {
	for _, character := range text {
		if character == ' ' {
			x += 8
			continue
		}
		glyph := int(character) - 0x2f
		if glyph < 0 || glyph >= 46 {
			x += 8
			continue
		}
		source := g.resources.font.SubImage(image.Rect(glyph*8, 0, glyph*8+8, 8)).(*ebiten.Image)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(x), float64(y))
		target.DrawImage(source, op)
		x += 8
	}
}

func (g *Game) drawTouch(screen *ebiten.Image) {
	panel := color.NRGBA{7, 22, 24, 255}
	accent := color.NRGBA{0, 210, 220, 255}
	vector.DrawFilledRect(screen, 0, 0, TouchGameX, ScreenHeight, panel, false)
	vector.DrawFilledRect(screen, TouchRightX, 0, TouchWidth-TouchRightX, ScreenHeight, panel, false)
	if g.scene == titleScene || g.scene == victoryScene {
		vector.DrawFilledRect(screen, TouchRightX+8, 52, 64, 96, color.NRGBA{0, 132, 0, 255}, false)
		g.drawText(screen, "START", TouchRightX+16, 96)
		return
	}
	vector.DrawFilledRect(screen, 8, 8, 64, 28, color.NRGBA{70, 70, 82, 255}, false)
	g.drawText(screen, "PAUSE", 16, 18)
	vector.StrokeCircle(screen, TouchGameX/2, 124, 30, 3, accent, false)
	vector.DrawFilledRect(screen, 7, 121, 66, 6, accent, false)
	vector.DrawFilledRect(screen, 37, 91, 6, 66, accent, false)
	g.drawText(screen, "MOVE", 24, 164)
	vector.DrawFilledRect(screen, TouchRightX+8, 22, 64, 68, color.NRGBA{0, 132, 0, 255}, false)
	vector.DrawFilledRect(screen, TouchRightX+8, 110, 64, 68, color.NRGBA{170, 0, 0, 255}, false)
	g.drawText(screen, "FIRE", TouchRightX+24, 52)
	g.drawText(screen, "BOMB", TouchRightX+24, 140)
}

// Pause freezes a manual game when Android suspends the surface.
func (g *Game) Pause() {
	if g.scene == playScene && !g.demo {
		g.paused = true
	}
}

// Back returns to the official title screen.
func (g *Game) Back() {
	if g.scene != titleScene {
		g.scene, g.sceneTick, g.idle, g.paused, g.demo = titleScene, 0, 0, false, false
		g.audio.Music("title")
	}
}

// Layout preserves native pixels and adds side controls on touch devices.
func (g *Game) Layout(_, _ int) (int, int) {
	if g.options.Touch {
		return TouchWidth, ScreenHeight
	}
	return ScreenWidth, ScreenHeight
}

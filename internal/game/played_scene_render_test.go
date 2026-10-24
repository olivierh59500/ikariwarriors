//go:build rendercheck

package game

import (
	"compress/gzip"
	"encoding/json"
	"os"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

type originalPlayedObject struct{ Slot, X, Y, Upper, Lower int }
type originalPlayedPlayer struct {
	X, Y, Upper, Lower, Turret, Fuel, TankSlot, Fall int
	Active, InTank, Burning, Dead                    bool
}
type originalPlayedEnding struct {
	X, Y, Ticks    int
	Ammo, Grenades [2]int
}

type originalPlayedGround struct{ Row, Column, Rows, Columns, YOffset, ClipTop int }

type originalPlayedScene struct {
	Ground                                                         []originalPlayedGround
	Ending                                                         *originalPlayedEnding
	Name                                                           string
	CameraY                                                        int
	Destroyed                                                      [][2]int
	Player                                                         originalPlayedPlayer
	Wide, Small                                                    []originalPlayedObject
	Score, Score2, Lives, Lives2, Ammo, Ammo2, Grenades, Grenades2 int
	Pixels                                                         []byte
	SourceSHA256                                                   string
}

func readOriginalPaintFixture(t *testing.T, path string, value any) {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	packed, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	defer packed.Close()
	if err = json.NewDecoder(packed).Decode(value); err != nil {
		t.Fatal(err)
	}
}

func observedOriginalPlayer(source originalPlayedPlayer, cameraY float64) actor {
	value := actor{x: float64(source.X - 48), y: cameraY + float64(source.Y-4), health: 1, hidden: !source.Active, inTank: source.InTank, tankBurning: source.Burning, tankFuel: source.Fuel, tankSlot: source.TankSlot}
	if source.InTank {
		if !source.Burning {
			value.upper.frame = (source.Upper - 0x5b408) / vehicleBobBytes
		}
		value.direction = (8 - (source.Turret-0x57628)/vehicleBobBytes) & 7
		if source.Burning {
			value.tankFlame.frame = (source.Turret - 0x5a6e8) / vehicleBobBytes
		}
		return value
	}
	value.upper.frame = (source.Upper - 0x42b88) / actorBobBytes
	value.lower.frame = (source.Lower - 0x46288) / overlayBobBytes
	if source.Dead {
		value.health, value.deadTicks = 0, source.Fall
		if source.Upper >= 0x47cc8 && source.Upper < 0x49ac8 {
			value.upperFalling = true
			value.upper.frame = (source.Upper - 0x47cc8) / actorBobBytes
		}
		if source.Lower >= 0x49ac8 && source.Lower < 0x4a2e8 {
			value.lowerFalling = true
			value.lower.frame = (source.Lower - 0x49ac8) / overlayBobBytes
		}
	} else if source.Lower == 0x4a1a8 || source.Lower == 0x4a248 {
		value.wet = true
		value.lower.frame = -3
		if source.Lower == 0x4a1a8 {
			value.lower.frame = -2
		}
	}
	return value
}

// These thirty-three complete 320-by-200 rasters come from normally played Amiga
// frames through the original ending animation. Source poses initialize the production
// painters; this checks composition, not continuous simulation from input.
// Every pixel is compared, including actors, small objects, gauges and HUD.
func TestNormallyPlayedOriginalFullScenePainting(t *testing.T) {
	var scenes []originalPlayedScene
	readOriginalPaintFixture(t, "testdata/played-scenes.json.gz", &scenes)
	if len(scenes) != 33 {
		t.Fatal("incomplete normally played full-frame observations")
	}
	for _, source := range scenes {
		t.Run(source.Name, func(t *testing.T) {
			if len(source.Pixels) != ScreenWidth*ScreenHeight || len(source.SourceSHA256) != 64 {
				t.Fatal("invalid original raster provenance")
			}
			g := observedPlayedSceneGame(t, source)
			screen := ebiten.NewImage(ScreenWidth, ScreenHeight)
			check := &originalSmallDrawCheck{g: g, target: screen, background: make([]byte, ScreenWidth*ScreenHeight*4), expected: source.Pixels}
			check.draw = func() {
				g.drawBackdrop(screen, g.cameraY)
				// Preserve the source small-slot order while using the shared
				// native projectile painter for each observed sprite.
				for _, part := range source.Small {
					frame := (part.Upper - 0x4a2e8) / overlayBobBytes
					if frame < 0 || frame >= len(g.resources.effects) {
						check.err = os.ErrInvalid
						return
					}
					g.projectiles = []projectile{{x: float64(part.X - 56), y: g.cameraY + float64(part.Y), owner: 0, slot: 0, frame: frame}}
					g.drawProjectiles(0, false)
				}
				g.projectiles = nil
				g.drawWideActors()
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Translate(playfieldX, playfieldY)
				screen.DrawImage(g.playfield, op)
				g.drawHUD(screen)
			}
			smallDrawScenes <- check
			if err := <-smallDrawResults; err != nil {
				t.Fatal(err)
			}
		})
	}
}

// Original framebuffer pixels independently cover 63 naturally observed fuel
// levels, including full tanks and the retained two-unit burnout indication.
func TestOriginalTankFuelGaugePainting(t *testing.T) {
	var cases []struct {
		Name            string
		Fuel            int
		Burning, InTank bool
		Pixels          []byte
		SourceSHA256    string
	}
	readOriginalPaintFixture(t, "testdata/fuel-gauge.json.gz", &cases)
	if len(cases) != 63 {
		t.Fatal("incomplete original fuel-gauge observations")
	}
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, source := range cases {
		t.Run(source.Name, func(t *testing.T) {
			if len(source.Pixels) != 16*64 || len(source.SourceSHA256) != 64 {
				t.Fatal("invalid original gauge provenance")
			}
			g.player = actor{inTank: source.InTank, tankBurning: source.Burning, tankFuel: source.Fuel}
			g.player2 = actor{}
			expected := make([]byte, ScreenWidth*ScreenHeight)
			for y := 0; y < 64; y++ {
				copy(expected[(80+y)*ScreenWidth+32:(80+y)*ScreenWidth+48], source.Pixels[y*16:(y+1)*16])
			}
			screen := ebiten.NewImage(ScreenWidth, ScreenHeight)
			check := &originalSmallDrawCheck{g: g, target: screen, background: make([]byte, len(expected)*4), expected: expected}
			check.draw = func() { g.drawFuelGauges(screen) }
			smallDrawScenes <- check
			if err := <-smallDrawResults; err != nil {
				t.Fatal(err)
			}
		})
	}
}

// Initialize only poses observed in the original; gameplay transition tests
// additionally supply their independently observed animation counters and input.
func observedPlayedSceneGame(t *testing.T, source originalPlayedScene) *Game {
	t.Helper()
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGame(false)
	g.scene = playScene
	g.cameraY = float64(source.CameraY)
	g.player = observedOriginalPlayer(source.Player, g.cameraY)
	g.player2.hidden = true
	g.enemies, g.projectiles, g.pickups, g.mines = nil, nil, nil, nil
	g.aircraft, g.enemyVehicles, g.boss = helicopter{}, [2]enemyVehicle{}, middleBoss{}
	g.final, g.ending, g.burst, g.sceneryBurst = finalEncounter{}, endingSequence{}, radialBurst{}, sceneryBurst{}
	if source.Ending != nil {
		e := source.Ending
		g.ending = endingSequence{x: float64(e.X), y: float64(e.Y), active: true, ticks: e.Ticks, displayAmmo: e.Ammo, displayGrenades: e.Grenades}
	}
	g.score, g.score2, g.lives, g.lives2 = source.Score, source.Score2, source.Lives, source.Lives2
	g.ammunition, g.ammunition2, g.grenades, g.grenades2 = source.Ammo, source.Ammo2, source.Grenades, source.Grenades2
	for _, cell := range source.Destroyed {
		g.destroyed[cell[0]][cell[1]] = true
	}
	for _, p := range source.Ground {
		g.sceneryPaint = append(g.sceneryPaint, sceneryGroundPatch{row: p.Row, column: p.Column, rows: p.Rows, columns: p.Columns, yOffset: p.YOffset, clipTop: p.ClipTop})
	}
	for _, part := range source.Wide {
		if part.Slot == 23 || g.player.inTank && (part.Slot == 25 || !g.player.tankBurning && part.Slot == g.player.tankSlot) {
			continue
		}
		value := actor{x: float64(part.X - 48), y: g.cameraY + float64(part.Y-4), health: 1, kind: 0, slot: part.Slot}
		value.upper.frame = infantryFrameForOffset(part.Upper-0x4cb88, actorBobBytes)
		switch part.Lower {
		case 0:
			value.lower.frame = -1
		case 0x4a1a8:
			value.lower.frame = -2
		case 0x4a248:
			value.lower.frame = -3
		default:
			value.lower.frame = infantryFrameForOffset(part.Lower-0x50288, overlayBobBytes)
		}
		upper, lower := value.infantryArtAddresses()
		if upper != part.Upper || lower != part.Lower {
			t.Fatal("source actor pose was not retained exactly")
		}
		g.enemies = append(g.enemies, value)
	}
	return g
}

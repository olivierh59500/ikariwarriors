package game

import (
	"compress/gzip"
	"encoding/json"
	"os"
	"testing"
)

type departedSourcePlayer struct {
	X, Y, RawY, Progress, Lives, Ammo, Grenades, Fall          int
	Upper, Lower, UpperStep, UpperDelay, LowerStep, LowerDelay int
	BulletPower, GrenadePower, RangeBonus                      int
	AimX, AimY                                                 int
	Active, Dead, Wet, Tank, Up, Down, Claimed, Locked         bool
}

type departedSourceFrame struct {
	Camera, Room   int
	DirectionLatch uint8
	Players        [2]departedSourcePlayer
	SourceSHA256   string
}

// The original controlled dispatcher finishes a fall, including the last
// reserve life, followed
// by ordinary partner input routines. Each case starts from its independently
// recorded initial conditions; no state is restored within its input sequence.
// Single-player movement does not use the native cooperative scroll claims.
// Its complete physical poses, resources and image phases are still checked.
func TestOriginalDepartedPartnerLifecycle(t *testing.T) {
	f, err := os.Open("testdata/departed-peer-inputs.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	z, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	defer z.Close()
	var rows []struct {
		Name                             string
		Owner, Faller, PlayerCount, Code int
		Before, Departure                departedSourceFrame
		Moves                            []struct {
			Direction int
			State     departedSourceFrame
		}
	}
	if err := json.NewDecoder(z).Decode(&rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 66 {
		t.Fatal("incomplete original final-fall observations")
	}
	inputs := 0
	for _, row := range rows {
		inputs += len(row.Moves)
		t.Run(row.Name, func(t *testing.T) {
			g, err := New(Options{Mute: true})
			if err != nil {
				t.Fatal(err)
			}
			g.startGamePlayers(false, row.PlayerCount)
			g.scene = playScene
			g.cameraY, g.cameraBackY = float64(row.Before.Camera), float64(row.Before.Camera+row.Before.Room)
			g.cooperativeDirection = row.Before.DirectionLatch
			g.eventIndex = len(g.resources.events)
			g.randomSpawn, g.redSpawn = false, false
			g.enemies, g.projectiles, g.pickups, g.mines = nil, nil, nil, nil
			for r := 505; r < 535; r++ {
				for c := range mapColumns {
					g.resources.levelCodes[r][c] = byte(row.Code)
				}
			}
			for player, p := range row.Before.Players {
				value := &g.player
				lives, ammunition, grenades := &g.lives, &g.ammunition, &g.grenades
				if player == 1 {
					value, lives, ammunition, grenades = &g.player2, &g.lives2, &g.ammunition2, &g.grenades2
				}
				*value = actor{x: float64(p.X), y: float64(p.Y), health: 1, wet: p.Wet, inTank: p.Tank}
				if p.Dead || !p.Active {
					value.health, value.deadTicks = 0, p.Fall
				}
				value.direction = directionForVector(float64(p.AimX), float64(p.AimY))
				value.moveDirection = value.direction
				value.upper = walkAnimation{step: p.UpperStep, delay: p.UpperDelay, frame: (p.Upper - 0x42b88 - player*22*actorBobBytes) / actorBobBytes}
				value.lower = walkAnimation{step: p.LowerStep, delay: p.LowerDelay, frame: (p.Lower - 0x46288) / overlayBobBytes}
				*lives, *ammunition, *grenades = p.Lives, p.Ammo, p.Grenades
				g.directionLock[player] = p.Locked
				g.bulletPower[player], g.grenadePower[player], g.rangeBonus[player] = p.BulletPower, p.GrenadePower, p.RangeBonus
				g.cooperativeScroll[player] = cooperativeScrollState{up: p.Up, down: p.Down, claimed: p.Claimed}
			}
			compare := func(stage string, frame departedSourceFrame, images bool) {
				t.Helper()
				if g.cameraY != float64(frame.Camera) || g.cameraBackY-g.cameraY != float64(frame.Room) {
					t.Errorf("%s camera/room %.0f/%.0f original%d/%d", stage, g.cameraY, g.cameraBackY-g.cameraY, frame.Camera, frame.Room)
				}
				for player, p := range frame.Players {
					value := g.player
					lives, ammunition, grenades := g.lives, g.ammunition, g.grenades
					if player == 1 {
						value, lives, ammunition, grenades = g.player2, g.lives2, g.ammunition2, g.grenades2
					}
					state := g.cooperativeScroll[player]
					if g.directionLock[player] != p.Locked || (p.Active && value.direction != directionForVector(float64(p.AimX), float64(p.AimY))) || value.x != float64(p.X) || value.y != float64(p.Y) || lives != p.Lives || ammunition != p.Ammo || grenades != p.Grenades || value.wet != p.Wet || value.inTank != p.Tank || value.deadTicks != p.Fall || (value.health > 0 && lives >= 0) != p.Active || (row.PlayerCount > 1 && (state.up != p.Up || state.down != p.Down || state.claimed != p.Claimed)) || g.bulletPower[player] != p.BulletPower || g.grenadePower[player] != p.GrenadePower || g.rangeBonus[player] != p.RangeBonus {
						t.Errorf("%s player%d pose%.0f/%.0f lives%d ammo%d gren%d wet%v gate%+v original%+v", stage, player, value.x, value.y, lives, ammunition, grenades, value.wet, state, p)
					}
					if images {
						upper, lower := value.playerArtOffsets(player)
						if upper+0x42b88 != p.Upper || lower+0x42b88 != p.Lower || value.upper.step != p.UpperStep || value.upper.delay != p.UpperDelay || value.lower.step != p.LowerStep || value.lower.delay != p.LowerDelay {
							t.Errorf("%s player%d phases%+v/%+v images%x/%x original%+v", stage, player, value.upper, value.lower, upper+0x42b88, lower+0x42b88, p)
						}
					}
				}
			}
			g.advancePlayerLife(row.Faller)
			compare("final fall", row.Departure, true)
			for _, move := range row.Moves {
				g.stepPlayer(row.Owner, playerControls{y: float64(move.Direction)})
				compare("partner input", move.State, true)
			}
		})
	}
	if inputs != 768 {
		t.Fatal("incomplete original post-fall input sequences")
	}
}

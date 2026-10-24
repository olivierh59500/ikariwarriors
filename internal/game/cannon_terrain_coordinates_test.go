package game

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

// These observations come from normally played original RAM, independently
// of the native cannon implementation. Source $1006a and $10480 both pass the
// retained shot progress at $13914 to the same map query at $10c3e.
func TestNormallyPlayedOriginalCannonTerrainCoordinates(t *testing.T) {
	data, err := os.ReadFile("testdata/cannon-terrain-coordinates.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Cases []struct {
			Run, Snapshot                                   string
			Frame, PlayerProgress, PlayerRawY, ShotProgress int
			RawX, RawY, VX, VY, Remaining, RangeBonus       int
			BulletUpgrade, GrenadeUpgrade, NextPassable     bool
			NextCode                                        byte
		}
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	if len(fixture.Cases) != 25 {
		t.Fatalf("original cannon coordinate observations %d, expected 25", len(fixture.Cases))
	}
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, source := range fixture.Cases {
		t.Run(fmt.Sprintf("%s_%s", source.Run, source.Snapshot), func(t *testing.T) {
			g.startGame(false)
			g.bulletPower[0], g.grenadePower[0], g.rangeBonus[0] = 1, 1, source.RangeBonus
			if source.BulletUpgrade {
				g.bulletPower[0] = 2
			}
			if source.GrenadeUpgrade {
				g.grenadePower[0] = 2
			}
			// The wide player origin is raw Y minus four. Its absolute
			// progress therefore establishes the camera without a native guess.
			playerY := float64(levelHeight - 16 - source.PlayerProgress)
			g.cameraY = playerY - float64(source.PlayerRawY) + 4
			player := actor{x: 72, y: playerY, health: 1, inTank: true,
				direction: directionForVector(float64(source.VX), float64(source.VY))}
			shot := g.projectileFor(&player, false, 0, false, 6)
			if g.projectileFrame(shot) != source.Frame {
				t.Fatalf("constructor cannon frame %d, original %d", g.projectileFrame(shot), source.Frame)
			}
			// Import the observed in-flight pose, retaining the constructor's
			// collision parameters. Small objects use raw Y without subtraction.
			shot.x, shot.y = float64(source.RawX-56), g.cameraY+float64(source.RawY)
			shot.vx, shot.vy, shot.life = float64(source.VX), float64(source.VY), source.Remaining/4
			actualX, actualY := shot.x+8, shot.y-4+shot.terrainBias
			expectedX := float64(source.RawX - 48)
			expectedY := float64(levelHeight - 16 - source.ShotProgress)
			if actualX != expectedX || actualY != expectedY {
				t.Errorf("native terrain query %g/%g, original progress query %g/%g (bias %g)", actualX, actualY, expectedX, expectedY, shot.terrainBias)
			}
			// Decode the next original map cell rather than assuming the
			// supplied disk still contains scenery cleared during this source run.
			nextX, nextY := expectedX+shot.vx, expectedY+shot.vy
			if nextX >= 0 && nextX <= 202 && nextY >= 0 && nextY < levelHeight {
				row, column := int(nextY+15)/tileSize, int(nextX)/tileSize
				if row < mapRows && source.NextCode == 0 && g.resources.levelCodes[row][column] != 0 {
					g.destroyed[row][column] = true
				}
			}
			if code := g.terrainCode(nextX, nextY); code != source.NextCode {
				t.Fatalf("source map cell %d differs from recovered native cell %d", source.NextCode, code)
			}
			moved, landed := g.advanceCannon(shot)
			expectedLanding := source.Remaining <= 4 || !source.BulletUpgrade && !source.NextPassable
			if moved.x != shot.x+shot.vx || moved.y != shot.y+shot.vy || moved.life != shot.life-1 || landed != expectedLanding {
				t.Errorf("next cannon %+v landed %v; original movement %g/%g remaining %d, terrain/range landing %v", moved, landed, shot.x+shot.vx, shot.y+shot.vy, shot.life-1, expectedLanding)
			}
		})
	}
}

// These separate controlled captures invoke the real original $bbdc allocator.
// They retain the newly constructed shot's $13914 value as well as its visual
// pose, covering normal, grenade, bullet and combined cannon upgrades.
func TestOriginalCannonConstructorProgressForEachUpgrade(t *testing.T) {
	data, err := os.ReadFile("testdata/cannon-terrain-coordinates.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		ControlledConstructors []struct {
			Kind                                               string
			PlayerRawX, PlayerRawY, PlayerProgress             int
			RawX, RawY, ShotProgress, Frame, VX, VY, Remaining int
			NextRawX, NextRawY, NextRemaining                  int
			BulletUpgrade, GrenadeUpgrade, NextActive          bool
		}
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	if len(fixture.ControlledConstructors) != 4 {
		t.Fatal("missing original cannon constructors for all four upgrade combinations")
	}
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, source := range fixture.ControlledConstructors {
		t.Run(source.Kind, func(t *testing.T) {
			g.startGame(false)
			g.bulletPower[0], g.grenadePower[0] = 1, 1
			if source.BulletUpgrade {
				g.bulletPower[0] = 2
			}
			if source.GrenadeUpgrade {
				g.grenadePower[0] = 2
			}
			playerY := float64(levelHeight - 16 - source.PlayerProgress)
			g.cameraY = playerY - float64(source.PlayerRawY) + 4
			player := actor{x: float64(source.PlayerRawX - 48), y: playerY, health: 1, inTank: true,
				direction: directionForVector(float64(source.VX), float64(source.VY))}
			shot := g.projectileFor(&player, false, 0, false, 6)
			if shot.x+56 != float64(source.RawX) || shot.y-g.cameraY != float64(source.RawY) || shot.vx != float64(source.VX) || shot.vy != float64(source.VY) || shot.life*4 != source.Remaining || g.projectileFrame(shot) != source.Frame {
				t.Fatalf("constructed native cannon %+v differs from original %+v", shot, source)
			}
			if probe := shot.y - 4 + shot.terrainBias; probe != float64(levelHeight-16-source.ShotProgress) {
				t.Errorf("constructed cannon terrain Y %g differs from original progress Y %d", probe, levelHeight-16-source.ShotProgress)
			}
			moved, landed := g.advanceCannon(shot)
			// The original clears the flight range when it allocates a quad.
			// The native caller removes that flight object rather than storing
			// advanceCannon's transient remaining count after a collision.
			rangeMatches := moved.life*4 == source.NextRemaining
			if landed {
				rangeMatches = source.NextRemaining == 0
			}
			if moved.x+56 != float64(source.NextRawX) || moved.y-g.cameraY != float64(source.NextRawY) || !rangeMatches || landed == source.NextActive {
				t.Errorf("first native cannon update %+v landed %v differs from observed next pose %+v", moved, landed, source)
			}
		})
	}
}

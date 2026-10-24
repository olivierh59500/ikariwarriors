package game

import (
	"encoding/json"
	"os"
	"strconv"
	"testing"
)

// These frames were recorded during ordinary original play. Unlike the
// stationary shot fixtures, they include vertical and diagonal evasive input.
func TestOriginalPlayedMovingBulletCollisions(t *testing.T) {
	data, err := os.ReadFile("testdata/played-moving-bullet-collisions.json")
	if err != nil {
		t.Fatal(err)
	}
	var rows []struct {
		Tick, X, Y, Progress, Invulnerable, Direction int
		Shots                                         []struct{ X, Y, VX, VY, Remaining int }
		Dead                                          bool
		AfterX, AfterProgress                         int
	}
	if err := json.Unmarshal(data, &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) == 0 {
		t.Fatal("missing independently recorded moving-player frames")
	}
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, original := range rows {
		g.startGame(false)
		g.scene = playScene
		g.player = actor{x: float64(original.X - 48), y: float64(levelHeight - 16 - original.Progress), health: 1, invulnerable: original.Invulnerable}
		g.cameraY = g.player.y - float64(original.Y) + 4
		g.cameraBackY = levelHeight - float64((int(levelHeight-g.cameraY)-1)&^31)
		g.enemies = nil
		g.eventIndex = len(g.resources.events)
		g.randomSpawn, g.redSpawn = false, false
		for _, shot := range original.Shots {
			g.projectiles = append(g.projectiles, projectile{x: float64(shot.X - 56), y: g.cameraY + float64(shot.Y), vx: float64(shot.VX), vy: float64(shot.VY), enemy: true, owner: -1, life: shot.Remaining / 4})
		}
		control := playerControls{}
		if original.Direction&0x10 != 0 {
			control.y = -1
		} else if original.Direction&0x20 != 0 {
			control.y = 1
		}
		if original.Direction&0x40 != 0 {
			control.x = -1
		} else if original.Direction&0x80 != 0 {
			control.x = 1
		}
		g.stepPlay([2]playerControls{control})
		if (g.player.deadTicks > 0) != original.Dead || int(g.player.x+48) != original.AfterX || levelHeight-16-int(g.player.y) != original.AfterProgress {
			t.Fatalf("played tick%d: player%+v differs from the original: rawX%d progress%d dead%v", original.Tick, g.player, original.AfterX, original.AfterProgress, original.Dead)
		}
	}
}

// Unlike an isolated shot update, these complete source frames include the
// player collision pass and the late fall/protection countdowns.
func TestOriginalCompleteEnemyBulletCollisions(t *testing.T) {
	data, err := os.ReadFile("testdata/enemy-bullets-complete.json")
	if err != nil {
		t.Fatal(err)
	}
	var rows []struct {
		Kind                                                        string
		Tick, Range, Death, Invulnerable, PlayerX, PlayerY, CameraY int
		Shot                                                        struct{ X, Y, Active int }
	}
	if err = json.Unmarshal(data, &rows); err != nil {
		t.Fatal(err)
	}
	var g *Game
	kind := ""
	for _, original := range rows {
		if kind != original.Kind {
			kind = original.Kind
			g, err = New(Options{Mute: true})
			if err != nil {
				t.Fatal(err)
			}
			g.startGame(false)
			g.scene = playScene
			g.cameraY = float64(original.CameraY)
			g.player = actor{x: float64(original.PlayerX - 48), y: g.cameraY + float64(original.PlayerY-4), health: 1, inTank: kind == "tank", tankFuel: 64}
			g.enemies = nil
			g.eventIndex = len(g.resources.events)
			g.randomSpawn, g.redSpawn = false, false
			g.projectiles = []projectile{{x: float64(original.Shot.X - 56), y: g.cameraY + float64(original.Shot.Y), vx: -4, enemy: true, owner: -1, power: 1, life: 15}}
		}
		if original.Tick > 0 {
			g.stepPlay([2]playerControls{})
		}
		if (len(g.projectiles) > 0) != (original.Shot.Active != 0) {
			t.Fatalf("%s tick%d ordinary shot lifetime differs", kind, original.Tick)
		}
		if len(g.projectiles) > 0 {
			shot := g.projectiles[0]
			if int(shot.x+56) != original.Shot.X || int(shot.y-g.cameraY) != original.Shot.Y || shot.life*4 != original.Range {
				t.Fatalf("%s tick%d bullet%+v differs from original%+v", kind, original.Tick, shot, original.Shot)
			}
		}
		if g.player.deadTicks != original.Death || g.player.invulnerable != original.Invulnerable || int(g.player.x+48) != original.PlayerX || int(g.player.y-g.cameraY+4) != original.PlayerY {
			t.Fatalf("%s tick%d player%+v differs from original%+v", kind, original.Tick, g.player, original)
		}
	}
}

// This trajectory was sampled from the original allocator and its ordinary
// projectile update: range60 means fifteen four-pixel steps, not sixty ticks.
func TestOriginalEnemyBulletTrajectoryAndLifetime(t *testing.T) {
	data, err := os.ReadFile("testdata/enemy-shot.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Tick    int
		Control []int
		Objects []struct{ X, Y, Active int }
	}
	if err = json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.player.health = 0
	for row := range g.destroyed {
		for column := range g.destroyed[row] {
			g.destroyed[row][column] = true
		}
	}
	source := actor{x: 98, y: 38, direction: 2}
	g.fireProjectile(&source, true, -1, false, 4)
	for _, original := range cases {
		if original.Tick > 0 {
			g.updateProjectiles()
		}
		part := original.Objects[0]
		if part.Active == 0 {
			if len(g.projectiles) != 0 {
				t.Fatalf("original enemy bullet has expired at tick%d", original.Tick)
			}
			continue
		}
		if len(g.projectiles) != 1 {
			t.Fatalf("missing original enemy bullet at tick%d", original.Tick)
		}
		shot := g.projectiles[0]
		if int(shot.x+56) != part.X || int(shot.y) != part.Y || shot.life*4 != original.Control[2] {
			t.Fatalf("enemy bullet tick%d: %+v, original%+v range%d", original.Tick, shot, part, original.Control[2])
		}
	}
}

func TestOriginalEnemyGrenadeFlights(t *testing.T) {
	for _, initial := range []int{60, 100} {
		t.Run(strconv.Itoa(initial), func(t *testing.T) {
			data, err := os.ReadFile("testdata/enemy-grenade-" + strconv.Itoa(initial) + ".json")
			if err != nil {
				t.Fatal(err)
			}
			var cases []struct {
				Tick    int
				Control []int
				Objects []struct {
					X, Y, Active int
					Bob          string
				}
			}
			if err = json.Unmarshal(data, &cases); err != nil {
				t.Fatal(err)
			}
			g, err := New(Options{Mute: true})
			if err != nil {
				t.Fatal(err)
			}
			source := actor{x: 98, y: 38, direction: 4}
			if initial == 100 {
				source.direction, source.y = 2, 45
			}
			g.fireProjectile(&source, true, -1, true, 4)
			for _, original := range cases {
				if original.Tick > 0 {
					g.updateProjectiles()
				}
				part := original.Objects[0]
				var shot projectile
				if part.Active == 0 {
					if len(g.projectiles) != 0 || g.blasts[2].impact != 3 {
						t.Fatal("the original landing did not release its flying slot and start the shared blast")
					}
					shot = g.blasts[2]
				} else {
					if len(g.projectiles) != 1 {
						t.Fatalf("missing original grenade at tick%d", original.Tick)
					}
					shot = g.projectiles[0]
				}

				if int(shot.x+56) != part.X || int(shot.y) != part.Y || shot.life*4 != original.Control[2] {
					t.Fatalf("enemy grenade tick%d: %+v, original%+v range%d", original.Tick, shot, part, original.Control[2])
				}
				if part.Active == 0 {
					if shot.impact != 3 {
						t.Fatal("the original grenade's last flight step did not start its blast")
					}
				} else if shot.frame != 9 || part.Bob != "0x4a888" {
					t.Fatal("the enemy grenade used the player's flight graphics")
				}
			}
		})
	}
}

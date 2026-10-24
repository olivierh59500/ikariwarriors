package game

import (
	"fmt"
	"testing"
)

// The complete original loop calls $fec4/$10520 before the moving enemies.
// These controlled native scenes isolate that established ordering; they are
// not source-capture goldens and do not inspect or serialize future RNG state.
func newGrenadeTimingScene(t *testing.T, kind string, hit bool) *Game {
	t.Helper()
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGame(false)
	g.scene = playScene
	g.player = actor{x: 104, y: 10000, health: 1, direction: 0}
	g.cameraY, g.cameraBackY = 9904, 9936
	g.secondaryTerrain = g.cameraY < secondaryMapRows*tileSize
	g.enemies, g.mines, g.pickups, g.projectiles = nil, nil, nil, nil
	g.eventIndex, g.randomSpawn, g.redSpawn = len(g.resources.events), false, false
	for row := range g.destroyed {
		for column := range g.destroyed[row] {
			g.destroyed[row][column] = true
		}
	}
	// Twelve distant horizontal grenades occupy the real enemy shot slots.
	// Their original 25-update flight outlasts this bounded test, excluding
	// unrelated firing/reversals without changing the random generator or
	// any target movement or controller implementation.
	for slot := range 12 {
		g.projectiles = append(g.projectiles, projectile{x: -48, y: 9940, vx: 4, life: 25, grenade: true, frame: 9, power: 1, enemy: true, owner: -1, slot: slot})
	}
	// An upward throw from (104, 10000) lands at (96, 9932). Its inclusive actor
	// contact spans target Y9901..9948 and target X73..124. Each moving head
	// starts two pixels outside the obsolete eighteenth-update prediction.
	targetY := 9865.0
	if hit {
		targetY = 9914
	}
	switch kind {
	case "infantry":
		g.ammunition = 0
		g.enemies = []actor{{x: 96, y: targetY, health: 1, kind: 0, slot: 5, marchTime: 64}}
	case "aircraft":
		g.aircraft = helicopter{x: 96, y: targetY + 30, flying: true, alive: [5]bool{true, true, true, false, false}, owner: -1}
	case "vehicle":
		x := 37.0
		if hit {
			x = 90
		}
		g.enemyVehicles[0] = enemyVehicle{x: x, y: 9932, mode: 0, reverse: 1, active: true, alive: [3]bool{true, true, true}, owner: -1}
	case "boss":
		g.ammunition = 0
		g.boss = middleBoss{x: 96, y: targetY, active: true, alive: [5]bool{true, true, true, true, true}, direction: 1, health: 1, turretOffset: navPoint{16, 12}, owner: -1}
	default:
		t.Fatalf("unknown target %q", kind)
	}
	return g
}

func assertGrenadeHeadForecast(t *testing.T, g *Game, kind string, hit bool, remaining int) {
	t.Helper()
	g.prepareTargetBursts()
	want := 0
	if hit && kind != "infantry" {
		want = 1
	}
	if g.demoBurstCount != want {
		t.Errorf("remaining%d %s predicted%d head bursts, expected%d: %+v", remaining, kind, g.demoBurstCount, want, g.demoTargetBursts[:g.demoBurstCount])
		return
	}
	if want == 0 {
		return
	}
	x, y, first := 96.0, 9955.0, remaining+1
	if kind == "vehicle" {
		x, y = 124, 9939
	}
	if kind == "tank" {
		y = 9939
	}
	if kind == "boss" {
		y, first = 9957, remaining+2
	}
	b := g.demoTargetBursts[0]
	if b.x != x || b.y != y || b.firstMove != first {
		t.Errorf("remaining%d %s burst=%+v; expected origin%.0f/%.0f firstMove%d", remaining, kind, b, x, y, first)
	}
}

func TestDemoGrenadeForecastUsesTheActualMovingTargetImpactPhase(t *testing.T) {
	for _, kind := range []string{"infantry", "aircraft", "vehicle", "boss"} {
		for _, hit := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s_hit%t", kind, hit), func(t *testing.T) {
				g := newGrenadeTimingScene(t, kind, hit)
				ammunition := g.ammunition
				aim, ok := g.demoGrenadeAim()
				// A last-round boss can still lose health through its other body parts
				// when the front head misses. Only head destruction emits this burst.
				if (kind != "boss" || hit) && (ok != hit || hit && aim != 0) {
					t.Errorf("grenade aim%d/%t disagrees with actual head hit%t", aim, ok, hit)
				}
				// Inspect both the proposed fresh throw and the actual shot with one
				// update remaining. Neither forecast may mutate ordinary scene slices.
				proposed := *g
				proposed.projectiles = append(append([]projectile(nil), g.projectiles...), g.projectileFor(&g.player, false, 0, true, 4))
				assertGrenadeHeadForecast(t, &proposed, kind, hit, 18)
				for tick := 1; tick <= 18; tick++ {
					if tick == 18 {
						assertGrenadeHeadForecast(t, g, kind, hit, 1)
					}
					g.stepPlay([2]playerControls{{action: tick == 1}})
					if tick < 18 && g.score != 0 {
						t.Fatalf("target damaged before real landing tick%d", tick)
					}
				}
				if g.blasts[0].x != 96 || g.blasts[0].y != 9932 || g.blasts[0].impact != 3 {
					t.Fatalf("ordinary grenade failed to land at96/9932 on tick18: %+v", g.blasts[0])
				}
				// Aircraft and ordinary vehicle cleanup is before their movement.
				// Boss damage is applied after one more advance; cleanup is next tick.
				if kind == "boss" {
					if g.burst.active {
						t.Fatal("boss burst appeared before its real delayed cleanup")
					}
					g.stepPlay([2]playerControls{})
				}
				alive := false
				switch kind {
				case "infantry":
					alive = g.enemies[0].health > 0
				case "aircraft":
					alive = g.aircraft.alive[0]
				case "vehicle":
					alive = g.enemyVehicles[0].alive[0]
				case "boss":
					alive = g.boss.alive[0]
				}
				if alive == hit {
					t.Fatalf("actual head alive%t, expected hit%t", alive, hit)
				}
				if kind != "infantry" {
					if g.burst.active != hit {
						t.Fatalf("actual burst active%t, expected%t", g.burst.active, hit)
					}
					if hit {
						x, y := 96.0, 9955.0
						if kind == "vehicle" {
							x, y = 124, 9939
						}
						if kind == "boss" {
							y = 9957
						}
						if g.burst.parts[1] != (navPoint{x, y}) || g.burst.remaining != 6 {
							t.Fatalf("actual birth origin/phase %+v/%d; expected %.0f/%.0f/6", g.burst.parts[1], g.burst.remaining, x, y)
						}
						g.stepPlay([2]playerControls{})
						if g.burst.parts[1] != (navPoint{x, y - 8}) || g.burst.remaining != 5 {
							t.Fatalf("first real radial movement %+v/%d; expected %.0f/%.0f/5", g.burst.parts[1], g.burst.remaining, x, y-8)
						}
					}
				}
				if g.lives != 5 || g.player.health != 1 || g.player.invulnerable != 0 || g.grenades != 49 || g.ammunition != ammunition {
					t.Fatalf("ordinary resources/life changed unexpectedly: lives%d player%+v grenades%d ammo%d", g.lives, g.player, g.grenades, g.ammunition)
				}
			})
		}
	}
}

func TestDemoOwnedGrenadeTankExplosionStartsOnTheNextFrame(t *testing.T) {
	g := newGrenadeTimingScene(t, "infantry", false)
	g.ammunition = 99
	g.enemies = []actor{{x: 96, y: 9932, health: 1, kind: 4, slot: 19, tankFuel: 1}}
	proposed := *g
	proposed.projectiles = append(append([]projectile(nil), g.projectiles...), g.projectileFor(&g.player, false, 0, true, 4))
	assertGrenadeHeadForecast(t, &proposed, "tank", true, 18)
	for tick := 1; tick <= 18; tick++ {
		if tick == 18 {
			assertGrenadeHeadForecast(t, g, "tank", true, 1)
		}
		g.stepPlay([2]playerControls{{action: tick == 1}})
		if tick < 18 && g.burst.active {
			t.Fatalf("tank exploded before the real grenade landing at%d", tick)
		}
	}
	if len(g.enemies) != 0 || !g.burst.active || g.burst.parts[1] != (navPoint{96, 9939}) || g.burst.remaining != 6 {
		t.Fatalf("tank cleanup did not create the real same-frame burst: enemies%d burst%+v", len(g.enemies), g.burst)
	}
	g.stepPlay([2]playerControls{})
	if g.burst.parts[1] != (navPoint{96, 9931}) || g.burst.remaining != 5 {
		t.Fatalf("tank burst did not make its first movement on tick19: %+v", g.burst)
	}
	if g.lives != 5 || g.player.health != 1 || g.player.invulnerable != 0 || g.ammunition != 99 || g.grenades != 49 {
		t.Fatal("the ordinary tank hit altered unrelated life or finite resources")
	}
}

// Already dead and already burning tanks have pending cleanup at the start of
// the decision. Their offsets are different from damage by a future owned shot.
func TestDemoTankPendingCleanupRetainsItsOriginalDelay(t *testing.T) {
	for _, burning := range []bool{false, true} {
		t.Run(fmt.Sprintf("burning%t", burning), func(t *testing.T) {
			g := newGrenadeTimingScene(t, "infantry", false)
			g.ammunition = 99
			tank := actor{x: 96, y: 9932, kind: 4, slot: 19, tankFuel: 1}
			first, x, y := 2, 96.0, 9939.0
			if burning {
				tank.health, tank.tankBurning, tank.tankBurn = 1, true, 2
				first, x, y = 4, 88, 9936
			}
			g.enemies = []actor{tank}
			g.prepareTargetBursts()
			if g.demoBurstCount != 1 || g.demoTargetBursts[0] != (demoBurstPrediction{x, y, first}) {
				t.Fatalf("pending tank forecast %+v; expected %.0f/%.0f first%d", g.demoTargetBursts[:g.demoBurstCount], x, y, first)
			}
			for tick := 1; tick <= first; tick++ {
				g.stepPlay([2]playerControls{})
				if tick == first-1 && (!g.burst.active || g.burst.parts[1] != (navPoint{x, y}) || g.burst.remaining != 6) {
					t.Fatalf("pending tank birth tick%d: %+v", tick, g.burst)
				}
			}
			if g.burst.parts[1] != (navPoint{x, y - 8}) || g.burst.remaining != 5 {
				t.Fatalf("pending tank first movement at%d: %+v", first, g.burst)
			}
		})
	}
}

package game

import (
	"fmt"
	"testing"
)

// These are controlled native scenes using recovered map/events and ordinary
// aircraft fire. They do not restore a complete original run or future RNG.
func prepareHelicopterFlankScene(t *testing.T, power, grenades int, x, y float64) *Game {
	t.Helper()
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGame(false)
	g.scene, g.demo, g.randomSpawn, g.redSpawn = playScene, false, false, false
	g.player = actor{x: x, y: y, health: 1}
	g.cameraY, g.cameraBackY = y-94, y-72
	g.secondaryTerrain = g.cameraY < secondaryMapRows*tileSize
	g.enemies, g.projectiles, g.pickups, g.mines = nil, nil, nil, nil
	g.grenadePower[0], g.grenades = power, grenades
	return g
}

func assertHelicopterFlankDidNotTakeAHit(t *testing.T, g *Game, tick int) {
	t.Helper()
	if g.player.health != 1 || g.player.deadTicks != 0 || g.player.invulnerable != 0 || g.lives != 5 {
		t.Fatalf("flank took a real hit at tick %d: player %+v aircraft %+v", tick, g.player, g.aircraft)
	}
}

func assertHelicopterFlankCompletedItsCascade(t *testing.T, g *Game) {
	t.Helper()
    // Original9D14 stops the anchor and its dependent parts. Independent
    // static wreck parts can remain visible until the painter clips them;
    // they neither fly nor fire and are not an unfinished hostile cascade.
    if g.aircraft.alive[1] || g.aircraft.alive[2] || g.aircraft.alive[4] || g.aircraft.flying || g.aircraft.damaged != 0 || g.burst.active || len(g.burstQueue) != 0 || len(g.projectiles) != 0 {
		t.Fatalf("the complete destruction cascade is still active: aircraft %+v burst %+v queue %+v shots %+v", g.aircraft, g.burst, g.burstQueue, g.projectiles)
	}
	for pool, blast := range g.blasts {
		if blast.impact != 0 {
			t.Fatalf("quad pool %d survived the bounded cascade: %+v", pool, blast)
		}
	}
}

func TestRememberedHelicopterFlankSurvivesItsCompleteDestructionCascade(t *testing.T) {
	for _, location := range [][2]float64{{152, 9014}, {102, 8316}} {
		for _, power := range []int{0, 2} {
			t.Run(fmt.Sprintf("head%.0f/%.0f-power%d", location[0], location[1], power), func(t *testing.T) {
                // Match the entered scroll-event geometry: the hero is96
                // pixels below the anchor. The old128 placed the head above
                // the original wide-object lifetime and clipped it immediately.
				g := prepareHelicopterFlankScene(t, power, 10, location[0]-72, location[1]+96)
				g.eventIndex = len(g.resources.events)
				g.aircraft = helicopter{x: location[0], y: location[1], flying: true, alive: [5]bool{true, true, true, true, true}, owner: -1}
				throws := 0
				for tick := 0; tick < 100; tick++ {
					control, handled := g.demoFlyingAircraftFlank()
					if !handled {
						control = playerControls{}
					}
					prior := g.grenades
					g.stepPlay([2]playerControls{control})
					if prior != g.grenades {
						throws++
					}
					assertHelicopterFlankDidNotTakeAHit(t, g, tick)
				}
				assertHelicopterFlankCompletedItsCascade(t, g)
				if throws != 1 || g.grenades != 9 || g.ammunition != 99 {
					t.Fatalf("the flank did not use exactly one ordinary grenade: throws %d grenades %d ammo %d", throws, g.grenades, g.ammunition)
				}
			})
		}
	}
}

// The aircraft is born only when actual movement reaches the original scroll
// event. Earlier unrelated enemies are excluded from this controlled scene.
func TestRememberedHelicopterFlankReachesTheOriginalScrollEvents(t *testing.T) {
	for _, known := range [][4]float64{{6000, 152, 104, 9270}, {6710, 102, 120, 8550}} {
		for _, power := range []int{0, 2} {
			t.Run(fmt.Sprintf("event%.0f-power%d", known[0], power), func(t *testing.T) {
				g := prepareHelicopterFlankScene(t, power, 10, known[2], known[3])
				found := false
				for i, event := range g.resources.events {
					if event.kind == 4 && event.progress == int(known[0]) {
						g.eventIndex, found = i, true
						break
					}
				}
				if !found {
					t.Fatal("missing original aircraft scroll event")
				}
				spawned, disabledAt, throws := false, -1, 0
				for tick := 0; tick < 380; tick++ {
					control, handled := g.demoFlyingAircraftFlank()
					if !handled {
						control = playerControls{}
					}
					prior := g.grenades
					g.stepPlay([2]playerControls{control})
					if g.aircraft.flying && !spawned {
						spawned = true
						t.Logf("ordinary spawn tick %d hero %.0f/%.0f aircraft %.0f/%.0f", tick, g.player.x, g.player.y, g.aircraft.x, g.aircraft.y)
					}
					if prior != g.grenades {
						throws++
						t.Logf("ordinary throw tick %d hero %.0f/%.0f aircraft %.0f/%.0f", tick, g.player.x, g.player.y, g.aircraft.x, g.aircraft.y)
					}
					assertHelicopterFlankDidNotTakeAHit(t, g, tick)
					if spawned && !g.aircraft.alive[0] && disabledAt < 0 {
						disabledAt = tick
					}
					if disabledAt >= 0 && tick >= disabledAt+40 {
						assertHelicopterFlankCompletedItsCascade(t, g)
						if throws != 1 || g.grenades != 9 || g.ammunition != 99 {
							t.Fatalf("finite approach used unexpected resources: throws %d grenades %d ammo %d", throws, g.grenades, g.ammunition)
						}
						return
					}
				}
				t.Fatalf("ordinary approach did not disable the plane: player %+v aircraft %+v spawned %t throws %d", g.player, g.aircraft, spawned, throws)
			})
		}
	}
}

func TestRememberedHelicopterFlankRemainsSafeWithoutAmmunition(t *testing.T) {
	for _, known := range [][4]float64{{6000, 152, 104, 9270}, {6710, 102, 120, 8550}} {
		t.Run(fmt.Sprintf("event%.0f", known[0]), func(t *testing.T) {
			g := prepareHelicopterFlankScene(t, 0, 0, known[2], known[3])
			g.ammunition = 0
			found := false
			for i, event := range g.resources.events {
				if event.kind == 4 && event.progress == int(known[0]) {
					g.eventIndex, found = i, true
					break
				}
			}
			if !found {
				t.Fatal("missing original aircraft scroll event")
			}
			spawned, passedAt := false, -1
			for tick := 0; tick < 380; tick++ {
				control, handled := g.demoFlyingAircraftFlank()
				if !handled {
					control = playerControls{}
				}
				if control.fire || control.action {
					t.Fatal("an empty weapon was requested instead of the safe flank")
				}
				g.stepPlay([2]playerControls{control})
				spawned = spawned || g.aircraft.flying
				assertHelicopterFlankDidNotTakeAHit(t, g, tick)
				if g.grenades != 0 || g.ammunition != 0 || g.score != 0 {
					t.Fatal("the empty-weapon approach changed finite resources or score")
				}
				if spawned && !handled && g.aircraft.y > g.player.y+16 && passedAt < 0 {
					passedAt = tick
				}
				if spawned && !g.aircraft.flying {
					if passedAt < 0 || g.aircraft.alive != ([5]bool{}) {
						t.Fatal("the flank did not release after the undamaged plane passed")
					}
					t.Logf("ordinary empty-weapon pass released at %d, plane departed at %d, hero %.0f/%.0f", passedAt, tick, g.player.x, g.player.y)
					return
				}
			}
			t.Fatalf("the real scroll-triggered plane did not pass safely: player %+v aircraft %+v", g.player, g.aircraft)
		})
	}
}

func TestRememberedHelicopterFlankDecisionPreservesThePhysicalWorld(t *testing.T) {
	g := prepareHelicopterFlankScene(t, 2, 10, 80, 9104)
	g.aircraft = helicopter{x: 152, y: 9006, flying: true, alive: [5]bool{true, true, true, true, true}, owner: -1}
	player, aircraft := g.player, g.aircraft
	ammo, grenades, lives, score, random := g.ammunition, g.grenades, g.lives, g.score, g.random
	for range 4 {
		control, handled := g.demoFlyingAircraftFlank()
		if !handled || !control.action {
			t.Fatalf("safe ordinary flank throw was not proposed: %+v/%t", control, handled)
		}
		if g.player != player || g.aircraft != aircraft || g.ammunition != ammo || g.grenades != grenades || g.lives != lives || g.score != score || g.random != random || len(g.projectiles) != 0 || len(g.enemies) != 0 {
			t.Fatal("a decision advanced a physical object, spent resources or consumed random input")
		}
	}
}

func TestHelicopterPrestageYieldsAfterItsOriginalEventWasConsumed(t *testing.T) {
	g := prepareHelicopterFlankScene(t, 0, 50, 98, 9122)
	g.cameraY, g.cameraBackY = 9028, 9024
	g.secondaryTerrain = g.cameraY < secondaryMapRows*tileSize
	for i, event := range g.resources.events {
		if event.kind == 4 && event.progress == 6000 {
			g.eventIndex = i + 1
			break
		}
	}
	g.aircraft = helicopter{x: 152, y: 9042}
	if control, handled := g.demoFlyingAircraftFlank(); handled {
		t.Fatalf("a consumed aircraft event reclaimed the ordinary route: %+v", control)
	}
}

func TestHelicopterPrestageLeavesTheEarlierRiverGunToItsOrdinaryGuide(t *testing.T) {
	g := prepareHelicopterFlankScene(t, 0, 50, 132, 9299)
	g.cameraY, g.cameraBackY = 9204, 9216
	g.secondaryTerrain = g.cameraY < secondaryMapRows*tileSize
	for i, event := range g.resources.events {
		if event.kind == 4 && event.progress == 6000 {
			g.eventIndex = i
			break
		}
	}
	g.enemies = []actor{{x: 112, y: 9190, health: 1, kind: 2, slot: 3}}
	if control, handled := g.demoFlyingAircraftFlank(); handled {
		t.Fatalf("preparing for a later aircraft bypassed the nearer gun: %+v", control)
	}
}

// The actual cold approach reaches this odd coordinate before two alternating
// terrain barriers. Moving left too early seals the route below the first one.
func TestRememberedHelicopterApproachCrossesTheAlternatingMapBarriers(t *testing.T) {
	g := prepareHelicopterFlankScene(t, 2, 10, 129, 8590)
	g.demo = true
	found := false
	for index, event := range g.resources.events {
		if event.kind == 4 && event.progress == 6710 {
			g.eventIndex, found = index, true
			break
		}
	}
	if !found {
		t.Fatal("missing original second aircraft event")
	}
	spawned, disabledAt := false, -1
	for tick := 0; tick < 600; tick++ {
		if err := g.Update(); err != nil {
			t.Fatal(err)
		}
		assertHelicopterFlankDidNotTakeAHit(t, g, tick)
		spawned = spawned || g.aircraft.flying
		if spawned && !g.aircraft.alive[0] && disabledAt < 0 {
			disabledAt = tick
		}
		if disabledAt >= 0 && tick >= disabledAt+40 && g.player.y < 8380 {
			assertHelicopterFlankCompletedItsCascade(t, g)
			if g.grenades != 9 || g.ammunition != 99 {
				t.Fatalf("the map detour changed finite resources: ammo %d grenades %d", g.ammunition, g.grenades)
			}
			return
		}
	}
	t.Fatalf("the actual ordinary approach never passed both barriers: player %+v aircraft %+v", g.player, g.aircraft)
}

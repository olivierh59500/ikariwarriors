package game

import (
	"fmt"
	"testing"
)

// Controlled native scenes use the unchanged initial map, ordinary firing
// allocator and real owner weapon pass. Existing occupancy/timers are setup
// only; no model rule, oracle or live demonstration state is altered.
func allSlotForecastScene(t *testing.T, owner, slot, remaining int) (*Game, projectile) {
	t.Helper()
	g := pendingShotGame(t)
	g.playerCount = 2
	g.player = actor{x: 170, y: 14964, health: 1}
	g.player2 = g.player
	source := &g.player
	if owner == 1 {
		source = &g.player2
	}
	source.x, source.y, source.direction = 72, 14964, 0
	g.enemies = []actor{{x: 78, y: 14930, health: 1, kind: 2, slot: 3}}
	g.projectiles, g.pickups, g.mines = nil, nil, nil
	g.eventIndex, g.randomSpawn, g.redSpawn = len(g.resources.events), false, false
	g.bulletPower = [2]int{1, 1}
	for _, occupied := range []int{1, 0} {
		if occupied == slot {
			break
		}
		g.projectiles = append(g.projectiles, projectile{x: -40, y: 14900, vx: -6, life: 10, owner: owner, slot: occupied})
	}
	*g.pendingOwnedFootImpact(owner, slot) = remaining
	beforeAmmo := g.ammunition
	if owner == 1 {
		beforeAmmo = g.ammunition2
	}
	g.stepPlayer(owner, playerControls{fire: true})
	afterAmmo := g.ammunition
	if owner == 1 {
		afterAmmo = g.ammunition2
	}
	if afterAmmo != beforeAmmo-1 {
		t.Fatal("ordinary allocator did not spend exactly one finite bullet")
	}
	for _, shot := range g.projectiles {
		if shot.owner == owner && shot.slot == slot {
			if shot.x != 78 || shot.y != 14968 || shot.vy != -6 || shot.life != 10 {
				t.Fatalf("unexpected ordinary allocated shot:%+v", shot)
			}
			if *g.pendingOwnedFootImpact(owner, slot) != remaining {
				t.Fatal("ordinary replacement incorrectly reset its retained cleanup")
			}
			return g, shot
		}
	}
	t.Fatal("ordinary allocator did not choose requested free record")
	return nil, projectile{}
}

func TestDemoEveryOwnedSlotForecastStopsAtActualCleanup(t *testing.T) {
	for owner := 0; owner < 2; owner++ {
		for slot := 0; slot < 3; slot++ {
			t.Run(fmt.Sprintf("player%d-slot%d", owner+1, slot), func(t *testing.T) {
				g, shot := allSlotForecastScene(t, owner, slot, 2)
				enemyBefore := g.enemies[0]
				index, hit, lethal := g.demoBulletImpact(shot)
				_, landing, lands := g.demoShotLanding(shot)
				mask := g.demoPendingBulletTargets()
				for tick := 1; tick <= 3; tick++ {
					g.advancePlayerProjectiles(owner)
					active := false
					for _, live := range g.projectiles {
						if !live.enemy && live.owner == owner && live.slot == slot {
							active = true
							if tick < 2 && live.impact != 0 {
								t.Fatal("a terrain impact invalidated the clear-lane timer diagnostic")
							}
						}
					}
					if active != (tick < 2) || *g.pendingOwnedFootImpact(owner, slot) != max(0, 2-tick) {
						t.Fatalf("real pass differs at%d: active%v timer%d", tick, active, *g.pendingOwnedFootImpact(owner, slot))
					}
				}
				if g.enemies[0].health != enemyBefore.health || g.score != 0 || g.score2 != 0 || g.player.health != 1 || g.player2.health != 1 {
					t.Fatal("cleared ordinary bullet unexpectedly damaged actors or scored")
				}
				if lethal || index >= 0 || lands || mask != 0 {
					t.Fatalf("forecast promises hit after actual tick2 cleanup: target%d hit%d lethal%v landing%d/%v pending%b; actual target health%d", index, hit, lethal, landing, lands, mask, g.enemies[0].health)
				}
			})
		}
	}
}

func TestDemoEveryOwnedSlotIncludesImpactBeforeItsSameFrameCleanup(t *testing.T) {
	for owner := 0; owner < 2; owner++ {
		for slot := 0; slot < 3; slot++ {
			t.Run(fmt.Sprintf("player%d-slot%d", owner+1, slot), func(t *testing.T) {
				g, shot := allSlotForecastScene(t, owner, slot, 3)
				target, hit, lethal := g.demoBulletImpact(shot)
				if target != 0 || hit != 3 || !lethal {
					t.Fatalf("forecast lost real contact before cleanup: target%d hit%d lethal%v", target, hit, lethal)
				}
				for tick := 1; tick <= 3; tick++ {
					g.advancePlayerProjectiles(owner)
					if tick < 3 && g.enemies[0].health != 1 {
						t.Fatal("target was hit before its ordinary contact tick")
					}
				}
				score := g.score
				if owner == 1 {
					score = g.score2
				}
				if g.enemies[0].health != 0 || score != 100 {
					t.Fatalf("ordinary same-frame impact was suppressed: health%d score%d", g.enemies[0].health, score)
				}
			})
		}
	}
}

func TestDemoCleanupReadoutRejectsNonownedAndInvalidRecords(t *testing.T) {
	g := pendingShotGame(t)
	g.pendingFootImpact = [2]int{1, 2}
	g.pendingOtherFootImpacts = [2][2]int{{3, 4}, {5, 6}}
	for _, shot := range []projectile{{enemy: true, owner: 0, slot: 1}, {owner: -1}, {owner: 2}, {owner: 0, slot: -1}, {owner: 1, slot: 3}} {
		if g.ownedSlotCleanupDelay(shot) != 0 {
			t.Fatal("invalid record acquired an owned cleanup timer")
		}
	}
}

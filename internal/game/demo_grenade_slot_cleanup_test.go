package game

import "testing"

// Original CA20..CAF4 allocates a thrown grenade in the same first flying
// record as the bullet/cannon; late 84C8 clears that record's activity/mask.
// A surviving impact timer can therefore clear a newly thrown grenade too.
// These controlled ordinary-step scenes spend one finite grenade and preserve
// the source timer, rather than treating the constructor's artwork as a pool.
func TestDemoRetainedFirstSlotCleanupAlsoBoundsAThrownGrenade(t *testing.T) {
	for player := 0; player < 2; player++ {
		t.Run(string(rune('1'+player)), func(t *testing.T) {
			g := newSlotCleanupScene(t, 2, false)
			source := &g.player
			if player == 1 {
				g.playerCount = 2
				g.player2 = g.player
				g.pendingFootImpact = [2]int{0, 2}
				source = &g.player2
			}
			shot := g.projectileFor(source, false, player, true, 4)
			if shot.slot != 0 {
				t.Fatal("original grenade constructor did not select the shared first record")
			}
			if _, _, lands := g.demoShotLanding(shot); lands {
				t.Fatal("forecast invented grenade expiry beyond the known shared-record cleanup")
			}
			for tick := 1; tick <= 18; tick++ {
				controls := [2]playerControls{}
				controls[player].action = tick == 1
				g.stepPlay(controls)
				ammo, grenades, lives, value := g.ammunition, g.grenades, g.lives, g.player
				if player == 1 {
					ammo, grenades, lives, value = g.ammunition2, g.grenades2, g.lives2, g.player2
				}
				if grenades != 49 || ammo != 99 || lives != 5 || value.health != 1 {
					t.Fatalf("unrelated finite resources/actor changed at%d: ammo%d grenades%d lives%d player%+v", tick, ammo, grenades, lives, value)
				}
				if g.pendingFootImpact[player] != max(0, 2-tick) {
					t.Fatalf("shared first-record timer did not advance at%d: %v", tick, g.pendingFootImpact)
				}
				active := false
				for _, s := range g.projectiles {
					if !s.enemy && s.owner == player && s.grenade {
						active = true
						if tick == 1 && s.life != 17 {
							t.Fatalf("grenade skipped its real first movement: %+v", s)
						}
					}
				}
				if active != (tick == 1) {
					t.Fatalf("shared-record activity at%d is%v; expected one movement then retained cleanup", tick, active)
				}
				if g.blasts[player].impact != 0 {
					t.Fatalf("cleared grenade invented a quad at%d: %+v", tick, g.blasts[player])
				}
			}
		})
	}
}

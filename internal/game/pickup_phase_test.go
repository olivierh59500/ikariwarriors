package game

import (
	"compress/gzip"
	"encoding/json"
	"os"
	"reflect"
	"sort"
	"testing"
)

type originalPickupState struct {
	Players [2]struct {
		X, Y             float64
		Active           bool
		Fall, Protection int
	}
	Items []struct {
		Kind int
		X, Y float64
	}
	Ammo, Grenades, Score, BulletPower, GrenadePower [2]int
}

type originalPickupCase struct {
	Name          string
	ScanPlayers   []int
	Before, After originalPickupState
}

func readOriginalPickupCases(t *testing.T) []originalPickupCase {
	t.Helper()
	file, err := os.Open("testdata/pickup-and-live-power.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	z, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	defer z.Close()
	var fixture struct{ PickupCases []originalPickupCase }
	if err = json.NewDecoder(z).Decode(&fixture); err != nil {
		t.Fatal(err)
	}
	if len(fixture.PickupCases) != 11 {
		t.Fatal("missing controlled original pickup cases")
	}
	return fixture.PickupCases
}

// Original C0F6/C282 directly check activity, not fall/protection counters.
// A newly hit actor remains active until its already-entered input returns.
// The separate inactive fixture enters through stepPlayer's eligibility gate.
func TestOriginalControlledPickupContactScans(t *testing.T) {
	for _, source := range readOriginalPickupCases(t) {
		t.Run(source.Name, func(t *testing.T) {
			g, err := New(Options{Mute: true})
			if err != nil {
				t.Fatal(err)
			}
			g.startGamePlayers(false, 2)
			g.playerCount = 2
			for i, p := range source.Before.Players {
				value := actor{x: p.X, y: p.Y, health: 1, deadTicks: p.Fall, invulnerable: p.Protection}
				if !p.Active || p.Fall > 0 {
					value.health = 0
				}
				if i == 0 {
					g.player = value
				} else {
					g.player2 = value
				}
			}
			g.ammunition, g.ammunition2 = source.Before.Ammo[0], source.Before.Ammo[1]
			g.grenades, g.grenades2 = source.Before.Grenades[0], source.Before.Grenades[1]
			g.score, g.score2 = source.Before.Score[0], source.Before.Score[1]
			g.bulletPower, g.grenadePower = source.Before.BulletPower, source.Before.GrenadePower
			g.pickups = nil
			for _, p := range source.Before.Items {
				g.pickups = append(g.pickups, pickup{x: p.X, y: p.Y, kind: p.Kind})
			}
			for _, player := range source.ScanPlayers {
				if source.Before.Players[player].Active {
					g.collectPlayerPickup(player)
				} else {
					g.stepPlayer(player, playerControls{})
				}
			}
			if [2]int{g.ammunition, g.ammunition2} != source.After.Ammo || [2]int{g.grenades, g.grenades2} != source.After.Grenades || [2]int{g.score, g.score2} != source.After.Score || g.bulletPower != source.After.BulletPower || g.grenadePower != source.After.GrenadePower {
				t.Fatalf("pickup effects differ from the independent original: ammo%d/%d grenades%d/%d score%d/%d bullet%v grenade%v", g.ammunition, g.ammunition2, g.grenades, g.grenades2, g.score, g.score2, g.bulletPower, g.grenadePower)
			}
			got := append([]pickup(nil), g.pickups...)
			sort.Slice(got, func(i, j int) bool { return got[i].kind < got[j].kind })
			if len(got) != len(source.After.Items) {
				t.Fatal("original first-slot scan consumed a different number of pickups")
			}
			for i, p := range source.After.Items {
				if got[i].kind != p.Kind || got[i].x != p.X || got[i].y != p.Y {
					t.Fatal("remaining original pickup slots differ")
				}
			}
			for i, p := range source.Before.Players {
				value := g.player
				if i == 1 {
					value = g.player2
				}
				if value.deadTicks != p.Fall || value.invulnerable != p.Protection {
					t.Fatal("pickup scan changed unrelated fall/protection state")
				}
			}
		})
	}
}

// The original calls pickup before mine and vertical movement at B6A6/B83E.
// These controlled positions reuse the independently observed wet-step edge:
// the pickup touches after horizontal movement, then misses after vertical.
func TestPickupUsesTheEnteredIntermediatePoseForEachPlayer(t *testing.T) {
	for _, player := range []int{0, 1} {
		for _, mode := range []string{"living", "protected", "hit-during-input", "already-falling"} {
			t.Run(string(rune('1'+player))+"/"+mode, func(t *testing.T) {
				g, err := New(Options{Mute: true})
				if err != nil {
					t.Fatal(err)
				}
				g.startGamePlayers(false, 2)
				// This isolated pickup encounter has no retained landing requests.
				g.cooperativeScroll = [2]cooperativeScrollState{}
				g.cameraY, g.cameraBackY = 8931, 8960
				g.enemies, g.projectiles, g.mines = nil, nil, nil
				value := &g.player
				other := &g.player2
				if player == 1 {
					value, other = other, value
				}
				*value = actor{x: 96, y: 9025, health: 1}
				*other = actor{x: 180, y: 9100, health: 1}
				if mode == "protected" {
					value.invulnerable = 100
				}
				if mode == "already-falling" {
					value.health, value.deadTicks = 0, 5
				}
				if mode == "hit-during-input" || mode == "protected" {
					g.projectiles = []projectile{{x: 97, y: 9029, enemy: true, owner: -1, slot: 0, life: 10}}
				}
				g.pickups = []pickup{{x: 112, y: 9048, kind: 6}}
				before := *value
				g.stepPlayer(player, playerControls{x: 1, y: -1})
				if mode == "already-falling" {
					if !reflect.DeepEqual(*value, before) || len(g.pickups) != 1 || g.bulletPower[player] != 1 {
						t.Fatal("pre-existing fall entered pickup processing")
					}
					return
				}
				if value.x != 97 || value.y != 9024 || len(g.pickups) != 0 || g.bulletPower[player] != 2 {
					t.Fatalf("pickup did not use the original intermediate wet pose: player%+v power%v remaining%v", *value, g.bulletPower, g.pickups)
				}
				if mode == "hit-during-input" && value.deadTicks != 24 {
					t.Fatal("new hit prevented the already-entered pickup scan")
				}
				if mode == "protected" && (value.deadTicks != 0 || value.invulnerable != 100) {
					t.Fatal("protection prevented collection or advanced during input")
				}
				if g.bulletPower[1-player] != 1 {
					t.Fatal("pickup changed the other player's weapon")
				}
			})
		}
	}
}

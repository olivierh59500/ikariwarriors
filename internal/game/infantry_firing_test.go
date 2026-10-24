package game

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

type originalInfantryShot struct {
	Slot, X, Y, Art, Active int
	Control                 [4]int
}

type originalInfantryFiring struct {
	Allocations []struct {
		Index, PC, OnlyFreeSlot, ReturnD0 int
		PoolFull                          bool
		RNG                               []uint32
		Shot                              originalInfantryShot
	}
	Shooting []struct {
		Index, Kind, Slot, Row, Column, RawX, RawY int
		MarchX, Upper, Lower, ShotsRemaining       int
		Wet                                        bool
		RNG                                        []uint32
		Shots                                      []originalInfantryShot
	}
}

func readOriginalInfantryFiring(t *testing.T) originalInfantryFiring {
	t.Helper()
	file, err := os.Open("testdata/infantry-fire.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	packed, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	defer packed.Close()
	var original originalInfantryFiring
	if err := json.NewDecoder(packed).Decode(&original); err != nil {
		t.Fatal(err)
	}
	if len(original.Allocations) != 926 || len(original.Shooting) != 6521 {
		t.Fatal("incomplete original infantry firing observations")
	}
	return original
}

// These are actual original $b506 returns, not a replacement runtime RNG.
// They isolate the allocator/control rules from the original beam-derived
// random sequence; the ordinary game continues to use its native Go RNG.
func originalInfantryRandom(t *testing.T, values []uint32) (func() uint32, *int) {
	t.Helper()
	used := new(int)
	return func() uint32 {
		if *used >= len(values) {
			t.Fatal("native infantry consumed an unrecorded original random return")
		}
		value := values[*used]
		*used++
		return value
	}, used
}

func originalInfantryShotType(shot projectile) int {
	if shot.fragments {
		return 2
	}
	if shot.grenade {
		return 1
	}
	return 0
}

func TestOriginalInfantryShotAllocation(t *testing.T) {
	original := readOriginalInfantryFiring(t)
	g := &Game{}
	for _, source := range original.Allocations {
		t.Run(fmt.Sprintf("case-%03d", source.Index), func(t *testing.T) {
			g.projectiles = g.projectiles[:0]
			if source.PoolFull || source.OnlyFreeSlot != 0 {
				for slot := range 12 {
					if !source.PoolFull && slot+65 == source.OnlyFreeSlot {
						continue
					}
					g.projectiles = append(g.projectiles, projectile{enemy: true, owner: -1, slot: slot})
				}
			}
			before := len(g.projectiles)
			random, used := originalInfantryRandom(t, source.RNG)
			var shot projectile
			var ok bool
			if source.PC == 0xadf2 {
				shot, ok = g.allocateInfantryShot(random)
				if len(g.projectiles) != before {
					t.Fatal("the original allocation helper does not activate a projectile")
				}
			} else {
				// $ae14 is the existing ordinary-bullet path. Its caller supplies
				// the original sixty-unit range before adding the active shot.
				g.addEnemyProjectile(projectile{enemy: true, owner: -1, life: 15, power: 1})
				ok = len(g.projectiles) != before
				if ok {
					shot = g.projectiles[len(g.projectiles)-1]
				}
			}
			if ok != (source.ReturnD0 != 0) || *used != len(source.RNG) {
				t.Fatalf("allocation success/random draws %t/%d, original %d/%d", ok, *used, source.ReturnD0, len(source.RNG))
			}
			if !ok {
				return
			}
			art := 0x4a2e8 + g.projectileFrame(shot)*bobBytes
			if shot.slot != source.Shot.Slot-65 || shot.life*4 != source.Shot.Control[2] || originalInfantryShotType(shot) != source.Shot.Control[3] || art != source.Shot.Art {
				t.Fatalf("allocated native %+v/art%x, original %+v", shot, art, source.Shot)
			}
		})
	}
}

// The expected positions, graphics and counters were independently decoded
// from original RAM after 6,521 complete $a7e0 updates. In particular, the wet
// red two-shot observation verifies an actual repeated attempt in one update.
func TestOriginalInfantryFiring(t *testing.T) {
	original := readOriginalInfantryFiring(t)
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	failures := 0
	for _, source := range original.Shooting {
		if !t.Run(fmt.Sprintf("case-%04d", source.Index), func(t *testing.T) {
			g.projectiles = g.projectiles[:0]
			// The controlled source actor stands eight pixels before this map
			// row, with raw Y100 and the source player's coherent progress.
			y := float64(source.Row*tileSize - 8)
			g.cameraY = y - float64(source.RawY-4)
			value := actor{x: float64(source.RawX - 48), y: y, health: 1, kind: source.Kind, slot: source.Slot, marchX: source.MarchX, shots: 10}
			code := g.terrainCode(value.x, value.y)
			if (code == 157 || code == 158) != source.Wet {
				t.Fatal("the native actor does not stand on the actual original map cell")
			}
			random, used := originalInfantryRandom(t, source.RNG)
			g.marchEnemyWithRandom(&value, random)
			upper, lower := value.infantryArtAddresses()
			if *used != len(source.RNG) || value.shots != source.ShotsRemaining || upper != source.Upper || lower != source.Lower || len(g.projectiles) != len(source.Shots) {
				t.Fatalf("random/counter/art/shots %d/%d/%x/%x/%d, original %d/%d/%x/%x/%d", *used, value.shots, upper, lower, len(g.projectiles), len(source.RNG), source.ShotsRemaining, source.Upper, source.Lower, len(source.Shots))
			}
			for i, expected := range source.Shots {
				shot := g.projectiles[i]
				art := 0x4a2e8 + g.projectileFrame(shot)*bobBytes
				if shot.slot != expected.Slot-65 || shot.x != float64(expected.X-56) || shot.y != g.cameraY+float64(expected.Y) || shot.vx != float64(int8(expected.Control[0])) || shot.vy != float64(int8(expected.Control[1])) || shot.life*4 != expected.Control[2] || originalInfantryShotType(shot) != expected.Control[3] || art != expected.Art {
					t.Fatalf("shot %d: native %+v/art%x, original %+v at camera%g", i, shot, art, expected, g.cameraY)
				}
			}
		}) {
			failures++
		}
	}
	t.Logf("original complete infantry updates: %d cases, %d differences", len(original.Shooting), failures)
}

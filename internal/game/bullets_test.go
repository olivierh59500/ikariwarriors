package game

import (
	"encoding/json"
	"os"
	"strconv"
	"testing"
)

func TestOriginalOwnedWeaponTrajectories(t *testing.T) {
	for _, file := range []string{"weapons.json", "cannons.json"} {
		t.Run(file, func(t *testing.T) { compareOriginalWeapons(t, file) })
	}
}

// The original allocator receives a fire request on every tick. Compare its
// real slot occupancy, ammunition and visible shots across repeated shells,
// upgrades, the range extension and a transition from tank to foot controls.
func TestOriginalRepeatedCannonFire(t *testing.T) {
	data, err := os.ReadFile("testdata/cannon-cadence.json")
	if err != nil {
		t.Fatal(err)
	}
	var rows []struct {
		Kind                                      string
		Tick, CameraY, PlayerX, PlayerY, ExitTick int
		RangeBonus, Ammunition, Score             int
		BulletUpgrade, GrenadeUpgrade             bool
		ShotMask                                  uint8
		Slots                                     []struct {
			Slot, X, Y, Active int
			Bob                string
		}
		Targets []struct{ Slot, X, Y, Damaged, Active int }
	}
	if err := json.Unmarshal(data, &rows); err != nil {
		t.Fatal(err)
	}
	var g *Game
	kind := ""
	for _, source := range rows {
		if source.Kind != kind {
			kind = source.Kind
			g, err = New(Options{Mute: true})
			if err != nil {
				t.Fatal(err)
			}
			g.cameraY, g.score, g.ammunition = float64(source.CameraY), source.Score, 99
			g.playerCount = 1
			g.player = actor{x: float64(source.PlayerX - 48), y: g.cameraY + float64(source.PlayerY-4), health: 1, direction: 2, inTank: true}
			g.bulletPower[0], g.grenadePower[0], g.rangeBonus[0] = 1, 1, source.RangeBonus
			if source.BulletUpgrade {
				g.bulletPower[0] = 2
			}
			if source.GrenadeUpgrade {
				g.grenadePower[0] = 2
			}
			for _, target := range source.Targets {
				g.enemies = append(g.enemies, actor{x: float64(target.X - 48), y: g.cameraY + float64(target.Y-4), health: 1, kind: 2, slot: target.Slot})
			}
		}
		if source.Tick == source.ExitTick {
			g.player.inTank = false
		}
		if g.bulletAvailable(0) {
			g.fireProjectile(&g.player, false, 0, false, 6)
			g.ammunition--
		}
		if source.Tick > 0 {
			g.updateProjectiles()
		}
		if g.ownedShotMask(0) != source.ShotMask || g.ammunition != source.Ammunition || g.score != source.Score {
			t.Fatalf("%s tick%d native mask%x ammo%d score%d, original mask%x ammo%d score%d", kind, source.Tick, g.ownedShotMask(0), g.ammunition, g.score, source.ShotMask, source.Ammunition, source.Score)
		}
		for _, slot := range source.Slots {
			var visible *projectile
			for i := range g.projectiles {
				shot := &g.projectiles[i]
				if !shot.enemy && shot.owner == 0 && shot.slot == slot.Slot-34 && !(shot.cannon && shot.impact > 0) {
					visible = shot
				}
			}
			if (visible != nil) != (slot.Active != 0) {
				t.Fatalf("%s tick%d slot%d native visibility%v original%d", kind, source.Tick, slot.Slot, visible != nil, slot.Active)
			}
			if visible != nil {
				bob, err := strconv.ParseInt(slot.Bob, 0, 32)
				if err != nil {
					t.Fatal(err)
				}
				if int(visible.x+56) != slot.X || int(visible.y-g.cameraY) != slot.Y || 0x4a2e8+g.projectileFrame(*visible)*160 != int(bob) {
					t.Fatalf("%s tick%d slot%d native%+v original%+v", kind, source.Tick, slot.Slot, visible, slot)
				}
			}
		}
	}
}

func compareOriginalWeapons(t *testing.T, file string) {
	t.Helper()
	data, err := os.ReadFile("testdata/" + file)
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Kind                                   string
		Tick, Score, CameraY, PlayerX, PlayerY int
		ShotMask                               uint8
		Shot                                   struct {
			X, Y, Active int
			Bob          string
		}
		Targets                                 []struct{ Slot, X, Y, Damaged, Active int }
		QuadActive, BurstActive, BurstRemaining int
		Quad, Burst                             []struct {
			X, Y, Active int
			Bob          string
		}
	}
	if err = json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	var g *Game
	kind := ""
	for _, original := range cases {
		if kind != original.Kind {
			kind = original.Kind
			g, err = New(Options{Mute: true})
			if err != nil {
				t.Fatal(err)
			}
			g.cameraY, g.score = float64(original.CameraY), original.Score
			g.playerCount = 1
			g.player = actor{x: float64(original.PlayerX - 48), y: g.cameraY + float64(original.PlayerY-4), health: 1, direction: 2, inTank: len(kind) >= 6 && kind[:6] == "cannon"}
			g.bulletPower[0], g.grenadePower[0] = 1, 1
			if kind == "super_far" || kind == "cannon_bullet" || kind == "cannon_both" {
				g.bulletPower[0] = 2
			}
			if kind == "cannon_grenade" || kind == "cannon_both" {
				g.grenadePower[0] = 2
			}
			for _, target := range original.Targets {
				g.enemies = append(g.enemies, actor{x: float64(target.X - 48), y: g.cameraY + float64(target.Y-4), health: 1, kind: 2, slot: target.Slot})
			}
			g.fireProjectile(&g.player, false, 0, false, 6)
		}
		if original.Tick > 0 {
			g.updateProjectiles()
		}
		if g.ownedShotMask(0) != original.ShotMask {
			t.Fatalf("%s tick%d native occupied slots%x original%x", kind, original.Tick, g.ownedShotMask(0), original.ShotMask)
		}
		var flight, blast *projectile
		for i := range g.projectiles {
			flight = &g.projectiles[i]
		}
		if g.blasts[0].impact > 0 {
			blast = &g.blasts[0]
		}
		if (flight != nil) != (original.Shot.Active != 0) {
			t.Fatalf("%s tick%d original projectile lifetime differs", kind, original.Tick)
		}
		if flight != nil {
			shot := *flight
			address, err := strconv.ParseInt(original.Shot.Bob, 0, 32)
			if err != nil {
				t.Fatal(err)
			}
			if int(shot.x+56) != original.Shot.X || int(shot.y-g.cameraY) != original.Shot.Y || 0x4a2e8+g.projectileFrame(shot)*160 != int(address) {
				t.Fatalf("%s tick%d native%+v differs from original%+v", kind, original.Tick, shot, original.Shot)
			}
		}
		if (blast != nil) != (original.QuadActive != 0) {
			t.Fatalf("%s tick%d quad lifetime differs", kind, original.Tick)
		}
		if blast != nil {
			for i, part := range original.Quad {
				address, err := strconv.ParseInt(part.Bob, 0, 32)
				if err != nil {
					t.Fatal(err)
				}
				if int(blast.x+48)+i%2*16 != part.X || int(blast.y-g.cameraY)-8+i/2*16 != part.Y || 0x4a2e8+grenadeImpactFrame(blast.impact, i)*160 != int(address) {
					t.Fatalf("%s tick%d quad%d differs", kind, original.Tick, i)
				}
			}
		}
		if g.burst.active != (original.BurstActive != 0) {
			t.Fatalf("%s tick%d burst lifetime differs", kind, original.Tick)
		}
		if g.burst.active {
			if g.burst.remaining != original.BurstRemaining {
				t.Fatalf("%s tick%d burst remaining differs", kind, original.Tick)
			}
			for i, part := range original.Burst {
				point := g.burst.parts[i]
				if int(point.x+56) != part.X || int(point.y-g.cameraY) != part.Y {
					t.Fatalf("%s tick%d fragment%d differs", kind, original.Tick, i)
				}
			}
		}
		if g.score != original.Score {
			t.Fatalf("%s tick%d score%d, original%d", kind, original.Tick, g.score, original.Score)
		}
		for i, target := range original.Targets {
			if (g.enemies[i].health <= 0) != (target.Damaged != 0 || target.Active == 0) {
				t.Fatalf("%s tick%d target%d damage differs", kind, original.Tick, target.Slot)
			}
		}
	}
}

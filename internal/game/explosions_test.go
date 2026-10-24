package game

import (
	"encoding/json"
	"os"
	"strconv"
	"testing"
)

// Complete original gameplay frames verify simultaneous landings, replacement
// on the expiry tick, independent player pools, damage and fragment timing.
func TestOriginalSharedQuadTimelines(t *testing.T) {
	data, err := os.ReadFile("testdata/shared-quads.json")
	if err != nil {
		t.Fatal(err)
	}
	type part struct {
		X, Y, Slot int
		Bob        string
	}
	var rows []struct {
		Kind                                                   string
		Tick, CameraY, BossHealth, BurstActive, BurstRemaining int
		Pools                                                  []struct {
			Owner, Active int
			Actors        []part
		}
		Burst, Shots []part
	}
	if err = json.Unmarshal(data, &rows); err != nil {
		t.Fatal(err)
	}
	var g *Game
	kind := ""
	enemy := func(slot, x int, heavy bool) {
		frame := 9
		if heavy {
			frame = 8
		}
		g.projectiles = append(g.projectiles, projectile{x: float64(x - 56), y: g.cameraY + 60, vy: 4, life: 1, enemy: true, owner: -1, grenade: true, slot: slot, frame: frame, fragments: heavy})
	}
	owned := func(owner, x int) {
		g.projectiles = append(g.projectiles, projectile{x: float64(x - 56), y: g.cameraY + 60, vy: 4, life: 1, owner: owner, grenade: true})
	}
	for _, source := range rows {
		if kind != source.Kind {
			kind = source.Kind
			g, err = New(Options{Mute: true})
			if err != nil {
				t.Fatal(err)
			}
			g.cameraY = float64(source.CameraY)
			g.playerCount = 2
			g.player = actor{x: 52, y: g.cameraY + 78, health: 1, inTank: true}
			g.player2 = actor{x: 52, y: g.cameraY + 78, health: 1, inTank: true}
			g.spawnMiddleBoss(levelEvent{kind: 30, progress: 8200})
			g.boss.y = g.cameraY + 28
			g.boss.health = 10
			g.grenadePower = [2]int{1, 1}
			switch kind {
			case "all-three-pools":
				owned(0, 150)
				owned(1, 180)
				enemy(0, 210, false)
			case "owner-replace":
				owned(0, 150)
			default:
				if kind == "enemy-same-tick" || kind == "enemy-heavy-last" {
					enemy(1, 180, kind == "enemy-heavy-last")
				}
				enemy(0, 150, true)
			}
		}
		if source.Tick > 0 {
			if source.Tick == 2 && kind == "owner-replace" {
				owned(0, 180)
			}
			if source.Tick == 2 && kind == "enemy-replace" || source.Tick == 4 && kind == "enemy-expiring-replaced" {
				enemy(1, 180, false)
			}
			g.updateProjectiles()
			g.updateMiddleBoss()
			g.boss.applyDamage()
		}
		if g.boss.health != source.BossHealth {
			t.Fatalf("%s tick%d boss%d, original%d", kind, source.Tick, g.boss.health, source.BossHealth)
		}
		for _, pool := range source.Pools {
			index := pool.Owner
			if index < 0 {
				index = 2
			}
			shot := g.blasts[index]
			if (shot.impact > 0) != (pool.Active != 0) {
				t.Fatalf("%s tick%d pool%d visibility differs", kind, source.Tick, index)
			}
			for i, part := range pool.Actors {
				bob, err := strconv.ParseInt(part.Bob, 0, 32)
				if err != nil {
					t.Fatal(err)
				}
				x, y := shot.x+48+float64(i%2*16), shot.y-g.cameraY-8+float64(i/2*16)
				if int(x) != part.X || int(y) != part.Y || 0x4a2e8+grenadeImpactFrame(shot.impact, i)*160 != int(bob) {
					t.Fatalf("%s tick%d pool%d quadrant%d differs", kind, source.Tick, index, i)
				}
			}
		}
		if g.burst.active != (source.BurstActive != 0) {
			t.Fatalf("%s tick%d fragment visibility differs", kind, source.Tick)
		}
		if g.burst.active {
			if g.burst.remaining != source.BurstRemaining {
				t.Fatalf("%s tick%d fragment timing differs", kind, source.Tick)
			}
			for i, part := range source.Burst {
				point := g.burst.parts[i]
				bob, err := strconv.ParseInt(part.Bob, 0, 32)
				if err != nil {
					t.Fatal(err)
				}
				if int(point.x+56) != part.X || int(point.y-g.cameraY) != part.Y || int(bob) != 0x4a2e8+burstFrames[i]*160 {
					t.Fatalf("%s tick%d fragment%d differs", kind, source.Tick, i)
				}
			}
		}
		// Both supplied flying shots must disappear on landing. Later source
		// frames can also contain a newly fired boss shell; its random firing
		// decision is separate from these controlled impact observations.
		if source.Tick > 1 {
			continue
		}
		actualShots := 0
		for _, shot := range g.projectiles {
			if shot.enemy {
				actualShots++
			}
		}
		if actualShots != len(source.Shots) {
			t.Fatalf("%s tick%d enemy flight count%d, original%d", kind, source.Tick, actualShots, len(source.Shots))
		}
		for _, part := range source.Shots {
			found := false
			for _, shot := range g.projectiles {
				if !shot.enemy || shot.slot != part.Slot {
					continue
				}
				found = true
				bob, err := strconv.ParseInt(part.Bob, 0, 32)
				if err != nil {
					t.Fatal(err)
				}
				if int(shot.x+56) != part.X || int(shot.y-g.cameraY) != part.Y || int(bob) != 0x4a2e8+g.projectileFrame(shot)*160 {
					t.Fatalf("%s tick%d enemy slot%d differs", kind, source.Tick, part.Slot)
				}
			}
			if !found {
				t.Fatalf("%s tick%d enemy slot%d missing", kind, source.Tick, part.Slot)
			}
		}
	}
}

// $11290 queues a busy fragment request and gives it the shared enemy quad.
// The queued burst starts only after the first burst's six moving phases.
func TestOriginalBusyFragmentTimeline(t *testing.T) {
	data, err := os.ReadFile("testdata/busy-fragments.json")
	if err != nil {
		t.Fatal(err)
	}
	type part struct {
		X, Y int
		Bob  string
	}
	var rows []struct {
		Tick, CameraY, BurstActive, BurstRemaining, Queued, QuadActive int
		Burst, Quad                                                    []part
	}
	if err = json.Unmarshal(data, &rows); err != nil {
		t.Fatal(err)
	}
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.cameraY = float64(rows[0].CameraY)
	g.player.health = 0
	g.createBurst(144, g.cameraY, -1)
	g.createBurst(164, g.cameraY+4, -1)
	for _, source := range rows {
		if source.Tick > 0 {
			g.updateProjectiles()
		}
		if g.burst.active != (source.BurstActive != 0) || len(g.burstQueue) != source.Queued {
			t.Fatalf("tick%d original fragment queue differs", source.Tick)
		}
		if g.burst.active {
			if g.burst.remaining != source.BurstRemaining {
				t.Fatalf("tick%d original fragment timing differs", source.Tick)
			}
			for i, part := range source.Burst {
				point := g.burst.parts[i]
				bob, err := strconv.ParseInt(part.Bob, 0, 32)
				if err != nil {
					t.Fatal(err)
				}
				if int(point.x+56) != part.X || int(point.y-g.cameraY) != part.Y || 0x4a2e8+burstFrames[i]*160 != int(bob) {
					t.Fatalf("tick%d original fragment%d differs", source.Tick, i)
				}
			}
		}
		shot := g.blasts[2]
		if (shot.impact > 0) != (source.QuadActive != 0) {
			t.Fatalf("tick%d original queued quad visibility differs", source.Tick)
		}
		for i, part := range source.Quad {
			bob, err := strconv.ParseInt(part.Bob, 0, 32)
			if err != nil {
				t.Fatal(err)
			}
			if int(shot.x+48+float64(i%2*16)) != part.X || int(shot.y-g.cameraY-8+float64(i/2*16)) != part.Y || 0x4a2e8+grenadeImpactFrame(shot.impact, i)*160 != int(bob) {
				t.Fatalf("tick%d original queued quadrant%d differs", source.Tick, i)
			}
		}
	}
}

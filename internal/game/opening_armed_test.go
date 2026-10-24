package game

import (
	"encoding/json"
	"os"
	"testing"
)

// This controlled encounter comes from the actual second natural cycle before
// its early bullet death. Both live infantry and already-fired shots remain;
// future unrelated births are excluded only from this focused regression.
func TestOpeningApproachFightsObservedInfantry(t *testing.T) {
	var source struct {
		X, Y, CameraY, CameraBackY                       float64
		Direction, Ammo, Grenades, Lives, Score          int
		Random                                           uint32
		BulletPower, GrenadePower, RangeBonus, FireDelay [2]int
		EnemyActorsNearby                                []struct {
			X, Y                                                            float64
			Direction, Health, Kind, Slot, MarchX, MarchY, MarchTime, Shots int
		}
		Shots []struct {
			X, Y, Vx, Vy              float64
			Life, Impact, Owner, Slot int
			Enemy, Grenade, Cannon    bool
		}
	}
	b, err := os.ReadFile("testdata/opening-armed-approach.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &source); err != nil {
		t.Fatal(err)
	}
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGame(true)
	g.scene = playScene
	g.player = actor{x: source.X, y: source.Y, direction: source.Direction, health: 1}
	g.cameraY, g.cameraBackY = source.CameraY, source.CameraBackY
	g.ammunition, g.grenades, g.lives, g.score, g.random = source.Ammo, source.Grenades, source.Lives, source.Score, source.Random
	g.bulletPower, g.grenadePower, g.rangeBonus, g.fireDelay = source.BulletPower, source.GrenadePower, source.RangeBonus, source.FireDelay
	g.enemies, g.projectiles, g.mines, g.pickups = nil, nil, nil, nil
	for _, e := range source.EnemyActorsNearby {
		if e.Health <= 0 {
			continue
		}
		g.enemies = append(g.enemies, actor{x: e.X, y: e.Y, direction: e.Direction, health: e.Health, kind: e.Kind, slot: e.Slot, marchX: e.MarchX, marchY: e.MarchY, marchTime: e.MarchTime, shots: e.Shots})
	}
	for _, s := range source.Shots {
		g.projectiles = append(g.projectiles, projectile{x: s.X, y: s.Y, vx: s.Vx, vy: s.Vy, life: s.Life, impact: s.Impact, owner: s.Owner, slot: s.Slot, enemy: s.Enemy, grenade: s.Grenade, cannon: s.Cannon, power: 1, frame: 6})
	}
	g.eventIndex, g.randomSpawn, g.redSpawn = len(g.resources.events), false, false
	for tick := 0; tick < 70; tick++ {
		g.stepPlay([2]playerControls{})
		if g.player.health != 1 || g.player.deadTicks != 0 || g.player.invulnerable != 0 || g.lives != source.Lives {
			t.Fatalf("opening encounter took a hit at%d: hero%+v ammo%d", tick, g.player, g.ammunition)
		}
	}
	for _, e := range g.enemies {
		if (e.slot == 12 || e.slot == 13) && e.health > 0 {
			t.Fatal("opening guide never cleared its live infantry")
		}
	}
	if g.ammunition >= source.Ammo || g.grenades > source.Grenades {
		t.Fatalf("approach did not fight with finite ordinary weapons: ammo%d grenades%d", g.ammunition, g.grenades)
	}
	t.Logf("survived70ordinary updates, ammo%d->%d grenades%d->%d", source.Ammo, g.ammunition, source.Grenades, g.grenades)
}

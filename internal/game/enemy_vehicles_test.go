package game

import (
	"encoding/json"
	"os"
	"strconv"
	"testing"
)

// These checkpoints come from the original event initializer and $99f8;
// they cover all four paths and both forty-update direction boundaries.
func TestOriginalEnemyVehicleMotionAndAnimation(t *testing.T) {
	data, err := os.ReadFile("testdata/enemy-vehicles.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Mode, Tick int
		Control    []int
		Actors     []struct {
			Slot, X, Y, Active, Step, Delay int
			Bob                             string
		}
	}
	if err = json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.cameraY = levelHeight - 5700
	mode, tick := -1, 0
	for _, original := range cases {
		if mode != original.Mode {
			mode, tick = original.Mode, 0
			g.enemyVehicles = [2]enemyVehicle{}
			kind := 2
			if mode != 0 {
				kind = mode + 20
			}
			g.spawnEnemyVehicles(levelEvent{kind: kind, count: 1, x: 140, spacing: 40, progress: 5700})
		}
		v := &g.enemyVehicles[0]
		for tick < original.Tick {
			v.advance(g.cameraY)
			tick++
		}
		if v.active != (original.Control[0] != 0) || v.mode != original.Control[1] || int(v.turn) != original.Control[2] || int(v.reverse) != original.Control[3] {
			t.Fatalf("mode%d tick%d control%+v, original%+v", mode, tick, v, original.Control)
		}
		for _, object := range original.Actors {
			i := object.Slot - 26
			part := v.part(0, i)
			address, err := strconv.ParseInt(object.Bob, 0, 32)
			if err != nil {
				t.Fatal(err)
			}
			if int(part.x+48) != object.X || int(part.y-g.cameraY+4) != object.Y || part.bob+0x42b88 != int(address) || v.alive[i] != (object.Active != 0) {
				t.Fatalf("mode%d tick%d slot%d: %+v, original%+v", mode, tick, object.Slot, part, object)
			}
			if i == 0 && (v.chassis.step != object.Step || v.chassis.delay != object.Delay) {
				t.Fatalf("mode%d tick%d chassis phase%+v, original%+v", mode, tick, v.chassis, object)
			}
			if g.resources.bob(part.bob, vehicleBobHeight) == nil {
				t.Fatalf("missing original enemy vehicle bob%#x", address)
			}
		}
	}
}

func TestEnemyVehicleEventsRespectTheSharedWideSlots(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.applyLevelEvent(levelEvent{kind: 21, x: 100, count: 2, spacing: 50, progress: 500})
	if !g.enemyVehicles[0].active || !g.enemyVehicles[1].active || g.enemyVehicles[1].x != 102 {
		t.Fatal("the two original tank groups were not spawned with the event spacing")
	}
	first := g.enemyVehicles
	g.applyLevelEvent(levelEvent{kind: 22, x: 200, count: 2, progress: 520})
	if g.enemyVehicles != first {
		t.Fatal("an occupied original wide slot was overwritten")
	}
	g.applyLevelEvent(levelEvent{kind: 4, x: 90, progress: 5700})
	if g.enemyVehicles != ([2]enemyVehicle{}) || !g.aircraft.flying {
		t.Fatal("the helicopter did not take over the original wide-object groups")
	}
}

func TestOriginalNormalEnemyVehicleDamageAndBurstOrigin(t *testing.T) {
	data, err := os.ReadFile("testdata/enemy-vehicle-damage.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Hit, Tick int
		Control   []int
		Actors    []struct{ Slot, X, Y, Active int }
		Burst     []struct{ X, Y int }
	}
	if err = json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.cameraY = levelHeight - 5700
	hit := -1
	for _, original := range cases {
		if hit != original.Hit {
			hit = original.Hit
			g.enemyVehicles, g.burst = [2]enemyVehicle{}, radialBurst{}
			g.spawnEnemyVehicles(levelEvent{kind: 21, count: 1, x: 140, progress: 5700})
			g.enemyVehicles[0].y = g.cameraY + 46
			g.enemyVehicles[0].damaged = 1 << uint(hit-26)
		}
		g.updateBurst()
		g.updateEnemyVehicles()
		v := &g.enemyVehicles[0]
		if v.active != (original.Control[0] != 0) {
			t.Fatalf("normal original tank hit%d tick%d controller differs", hit, original.Tick)
		}
		for i, object := range original.Actors {
			part := v.part(0, i)
			if v.alive[i] != (object.Active != 0) || int(part.x+48) != object.X || int(part.y-g.cameraY+4) != object.Y {
				t.Fatalf("normal original tank hit%d tick%d part%d differs", hit, original.Tick, i)
			}
		}
		if hit == 26 {
			point, expected := g.burst.parts[0], original.Burst[0]
			if !g.burst.active || int(point.x+56) != expected.X || int(point.y-g.cameraY) != expected.Y {
				t.Fatalf("tank burst hit%d tick%d origin%+v differs from%+v", hit, original.Tick, point, expected)
			}
		}
	}
}

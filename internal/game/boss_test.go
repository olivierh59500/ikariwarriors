package game

import (
	"encoding/json"
	"os"
	"strconv"
	"testing"
)

func TestOriginalMiddleBossMotion(t *testing.T) {
	data, err := os.ReadFile("testdata/boss-motion.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Tick    int
		Control []int
		Actors  []struct {
			Slot, X, Y, Active int
			Bob                string
		}
	}
	if err = json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.cameraY = levelHeight - 8200
	g.applyLevelEvent(levelEvent{kind: 30, progress: 8200})
	b, tick := &g.boss, 0
	for _, original := range cases {
		for tick < original.Tick {
			b.advance(g.cameraY)
			tick++
		}
		if b.active != (original.Control[0] != 0) || b.turn != original.Control[1] || byte(b.direction) != byte(original.Control[2]) || b.health != original.Control[3] {
			t.Fatalf("original middle boss tick%d control differs: %+v original%+v", tick, b, original.Control)
		}
		for i, object := range original.Actors {
			part := b.part(i)
			address, err := strconv.ParseInt(object.Bob, 0, 32)
			if err != nil {
				t.Fatal(err)
			}
			if int(part.x+48) != object.X || int(part.y-g.cameraY+4) != object.Y || part.bob+0x42b88 != int(address) || b.alive[i] != (object.Active != 0) {
				t.Fatalf("original middle boss tick%d part%d differs: %+v original%+v", tick, i, part, object)
			}
			if g.resources.bob(part.bob, vehicleBobHeight) == nil {
				t.Fatal("missing original middle boss graphic")
			}
		}
	}
}

func TestOriginalMiddleBossHitRounds(t *testing.T) {
	data, err := os.ReadFile("testdata/boss-health.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Hit     int
		Control []int
		Damage  []int
	}
	if err = json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.spawnMiddleBoss(levelEvent{kind: 30, progress: 8200})
	for _, original := range cases {
		g.boss.damaged = 5
		g.boss.applyDamage()
		if g.boss.active != (original.Control[0] != 0) || g.boss.health != original.Control[3] {
			t.Fatalf("middle boss health after original round%d differs", original.Hit)
		}
		for i := range 5 {
			if (g.boss.damaged>>i)&1 != byte(original.Damage[i]) {
				t.Fatal("the boss's last hit flags were not retained for ordinary cleanup")
			}
		}
	}
}

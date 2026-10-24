package game

import (
	"encoding/json"
	"os"
	"strconv"
	"testing"
)

func TestOriginalFinalEncounterLayout(t *testing.T) {
	data, err := os.ReadFile("testdata/final-encounter.json")
	if err != nil {
		t.Fatal(err)
	}
	var original struct {
		Actors []struct {
			Slot, X, Y int
			Bob        string
		}
	}
	if err = json.Unmarshal(data, &original); err != nil {
		t.Fatal(err)
	}
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.cameraY = levelHeight - 14640
	g.applyLevelEvent(levelEvent{kind: 25, progress: 14640})
	for i, object := range original.Actors {
		part := g.final.part(i)
		address, err := strconv.ParseInt(object.Bob, 0, 32)
		if err != nil {
			t.Fatal(err)
		}
		if int(part.x+48) != object.X || int(part.y-g.cameraY+4) != object.Y || part.bob+0x42b88 != int(address) || part.slot != object.Slot {
			t.Fatalf("final encounter part%d: %+v, original%+v", i, part, object)
		}
		if !g.slotActive(part.slot) || g.resources.bob(part.bob, vehicleBobHeight) == nil {
			t.Fatal("missing original final encounter bob or shared slot")
		}
	}
}

func TestOriginalFinalEncounterBecomesBackgroundAfterAPlayerBlast(t *testing.T) {
	data, err := os.ReadFile("testdata/final-defeated.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Tick, Active int
		Actors       []struct {
			X, Y, Active int
			Bob          string
		}
	}
	if err = json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.cameraY = levelHeight - 14640
	g.spawnFinalEncounter(levelEvent{kind: 25, progress: 14640})
	g.final.y += 120
	shot := projectile{x: 170 - 56, y: g.cameraY + 60, enemy: true, owner: -1}
	g.damageFinalEncounter(shot, true)
	if !g.final.active {
		t.Fatal("an enemy-owned fragment triggered the player's final-encounter action")
	}
	shot.enemy, shot.owner = false, 0
	g.damageFinalEncounter(shot, true)
	for _, original := range cases {
		if g.final.active != (original.Active != 0) || !g.final.defeated {
			t.Fatal("the final encounter did not switch to the original background image")
		}
		for i, object := range original.Actors {
			if object.Active != 0 || g.slotActive(5+i) {
				t.Fatal("a defeated final-encounter bob remained a live enemy")
			}
			if original.Tick == 0 && i < 8 {
				address, err := strconv.ParseInt(object.Bob, 0, 32)
				if err != nil {
					t.Fatal(err)
				}
				part := g.final.part(i)
				if int(part.x+48) != object.X || int(part.y-g.cameraY+4) != object.Y || part.bob+0x42b88 != int(address) {
					t.Fatal("the original defeated graphic layout was changed")
				}
			}
		}
	}
}

func TestOriginalEndingAnimationAndWeaponState(t *testing.T) {
	data, err := os.ReadFile("testdata/ending.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Tick, Phase, Delay int
		Actors             []struct {
			Slot, X, Y int
			Bob        string
		}
	}
	if err = json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.cameraY = levelHeight - 14864
	g.player = actor{x: 104, y: g.cameraY + 96, health: 1}
	g.playerCount, g.lives = 1, 5
	g.ammunition, g.grenades = 99, 50
	g.eventIndex = len(g.resources.events)
	g.applyLevelEvent(levelEvent{kind: 24, progress: 14864})
	g.ending.y += 120
	tick := 0
	for _, original := range cases {
		for tick < original.Tick {
			g.stepPlay([2]playerControls{})
			tick++
		}
		if g.ending.ticks%2 != original.Delay || (g.ending.ticks/2&1 != 0) != (original.Phase != 0) {
			t.Fatal("the original ending phase or two-tick delay was changed")
		}
		for i, object := range original.Actors {
			part := g.endingPart(i)
			address, err := strconv.ParseInt(object.Bob, 0, 32)
			if err != nil {
				t.Fatal(err)
			}
			if int(part.x+48) != object.X || int(part.y-g.cameraY+4) != object.Y || part.bob+0x42b88 != int(address) || !g.slotActive(object.Slot) {
				t.Fatalf("ending tick%d part%d differs", tick, i)
			}
		}
		g.hitPlayer(0)
		if g.player.health != 1 || g.player.deadTicks != 0 || tick > 0 && (g.ammunition != 0 || g.grenades != 0) {
			t.Fatal("the original ending protection or weapon state was changed")
		}
	}
}

func TestOrdinaryBulletStopsAtTheProtectedFinalEncounter(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.spawnFinalEncounter(levelEvent{kind: 25, progress: 14640})
	for row := range g.destroyed {
		for column := range g.destroyed[row] {
			g.destroyed[row][column] = true
		}
	}
	part := g.final.part(1)
	g.projectiles = []projectile{{x: part.x, y: part.y + 4, owner: 0, power: 1, life: 2}}
	g.updateProjectiles()
	if len(g.projectiles) != 0 || !g.final.active || g.final.defeated {
		t.Fatal("the ordinary bullet passed through or defeated the final encounter")
	}
}

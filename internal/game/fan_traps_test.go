package game

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

type originalFanState struct {
	Actors         []originalConstructorActor
	Shots          []originalInfantryShot
	Guards         [2]int
	Resources      string
	ResourceValues originalConstructorResources
}

type originalFanObservations struct {
	CameraY float64
	Cases   []struct {
		Index         int
		Case          struct{ Group string }
		Before, After originalFanState
		Events        []struct{ PC, A0, A1, D0, D7 int }
	}
	Phase []struct {
		Label string
		PC    int
		State originalFanState
	}
}

func readOriginalFanTraps(t *testing.T) originalFanObservations {
	t.Helper()
	file, err := os.Open("testdata/fan-traps.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	packed, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	defer packed.Close()
	var original originalFanObservations
	if err := json.NewDecoder(packed).Decode(&original); err != nil {
		t.Fatal(err)
	}
	if len(original.Cases) != 303 || len(original.Phase) != 4 {
		t.Fatal("incomplete original fan-trap observations")
	}
	return original
}

func supplyOriginalFanState(g *Game, source originalFanState, cameraY float64) {
	source.ResourceValues.supply(g)
	g.cameraY = cameraY
	for _, value := range source.Actors {
		x, y := float64(value.X-48), cameraY+float64(value.Y-4)
		if value.Slot >= 87 {
			g.fanTraps[value.Slot-87] = fanTrap{x: x, y: y, active: value.Active != 0, cooldown: value.Cooldown}
			continue
		}
		player, guard := &g.player, source.Guards[1]
		if value.Slot == 22 {
			player, guard = &g.player2, source.Guards[0]
		}
		player.x, player.y, player.health = x, y, 1
		// $c768 reads activity and the separate death guard, independently
		// of the fall/protection bytes varied by these controlled cases.
		if value.Active == 0 || guard != 0 {
			player.health = 0
		}
	}
	g.projectiles = g.projectiles[:0]
	for _, shot := range source.Shots {
		if shot.Active == 0 {
			continue
		}
		g.projectiles = append(g.projectiles, projectile{
			x: float64(shot.X - 56), y: cameraY + float64(shot.Y),
			vx: float64(int8(shot.Control[0])), vy: float64(int8(shot.Control[1])),
			enemy: true, owner: -1, slot: shot.Slot - 65,
			life: shot.Control[2] / 4, frame: 1, power: 1,
		})
	}
}

func checkOriginalFanState(t *testing.T, g *Game, source originalFanState, cameraY float64) {
	t.Helper()
	for _, value := range source.Actors {
		if value.Slot < 87 {
			continue
		}
		got := g.fanTraps[value.Slot-87]
		if got.x != float64(value.X-48) || got.y != cameraY+float64(value.Y-4) || got.cooldown != value.Cooldown || got.active != (value.Active != 0) {
			t.Errorf("trap %d: native %+v, original %+v", value.Slot, got, value)
		}
	}
	var native [12]*projectile
	for i := range g.projectiles {
		shot := &g.projectiles[i]
		if !shot.enemy || shot.slot < 0 || shot.slot >= len(native) || native[shot.slot] != nil {
			t.Fatalf("unexpected native projectile %+v", *shot)
		}
		native[shot.slot] = shot
	}
	for _, value := range source.Shots {
		got := native[value.Slot-65]
		if (got != nil) != (value.Active != 0) {
			t.Errorf("shot %d activity: native %v, original %d", value.Slot, got != nil, value.Active)
			continue
		}
		if got == nil {
			continue
		}
		if value.Art != 0x4a388 || got.x != float64(value.X-56) || got.y != cameraY+float64(value.Y) || got.vx != float64(int8(value.Control[0])) || got.vy != float64(int8(value.Control[1])) || got.life*4 != value.Control[2] || value.Control[3] != 0 || got.frame != 1 || got.grenade || got.fragments || got.cannon || got.owner != -1 || got.power != 1 {
			t.Errorf("shot %d: native %+v, original %+v", value.Slot, *got, value)
		}
	}
}

// The expected countdowns, slots, positions, vectors, artwork and range were
// recorded from actual $9330/$ae14 execution. The original RNG breakpoint
// was never entered by these calls; occupied shots and resources survived.
func TestOriginalFanTrapFiring(t *testing.T) {
	original := readOriginalFanTraps(t)
	g := &Game{}
	for _, source := range original.Cases {
		t.Run(fmt.Sprintf("case-%03d-%s", source.Index, source.Case.Group), func(t *testing.T) {
			supplyOriginalFanState(g, source.Before, original.CameraY)
			resources, random := constructorResources(g), g.random
			g.updateFanTraps()
			checkOriginalFanState(t, g, source.After, original.CameraY)
			if source.Before.Resources != source.After.Resources || constructorResources(g) != resources || g.random != random {
				t.Fatal("fan contact changed player resources or consumed randomness")
			}
			for _, event := range source.Events {
				if event.PC == 0xb506 || event.PC == 0x20374 && event.D0&255 != 13 {
					t.Fatal("unexpected original RNG or sound event")
				}
			}
		})
	}
}

// A source active bob still contacts $9330 while the fall flag is set: $c768
// never reads the fall/protection counters. Represent that same active source
// bob as a falling native actor, rather than treating zero health as removal.
func TestOriginalFanTrapFallingContact(t *testing.T) {
	original := readOriginalFanTraps(t)
	source := original.Cases[6]
	g := &Game{}
	supplyOriginalFanState(g, source.Before, original.CameraY)
	g.player.health, g.player.deadTicks = 0, 24
	g.updateFanTraps()
	checkOriginalFanState(t, g, source.After, original.CameraY)
}

// The actual original late-main interval advances enemy bullets at $907c,
// fires fans at $90a8 and then runs vehicle firing at $90ac. Its new bullets
// retain their muzzle coordinates and sixty-unit range through presentation.
// Exercise the complete native step so moving this call earlier is detected.
func TestOriginalFanTrapBirthPhase(t *testing.T) {
	original := readOriginalFanTraps(t)
	for i, pc := range [...]int{0x90a8, 0x90ac, 0x90b0, 0x9178} {
		if original.Phase[i].PC != pc {
			t.Fatal("original main-phase fixture is out of order")
		}
	}
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	supplyOriginalFanState(g, original.Phase[0].State, original.CameraY)
	g.scene, g.eventIndex = playScene, len(g.resources.events)
	g.stepPlay([2]playerControls{})
	checkOriginalFanState(t, g, original.Phase[3].State, original.CameraY)
}

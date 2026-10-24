package game

import (
	"encoding/json"
	"os"
	"strconv"
	"testing"
)

type originalTankPlayer struct {
	X, Y, Fuel, Phase, Burn, Fall, Protection, Lives, Ammo, Grenades int
	Boarded, Burning, Dead, TankActive                               bool
	TankX, TankY, FlameStep, FlameDelay                              int
	Chassis, Legs, Turret                                            uint32
}

type originalTankState struct {
	CameraY        float64
	Players        [2]originalTankPlayer
	BurstActive    bool
	BurstRemaining int
	Burst          []struct{ X, Y int }
}

func (s originalTankState) supply(g *Game) {
	g.scene, g.demo, g.playerCount = playScene, false, 2
	g.cameraY, g.cameraBackY = s.CameraY, s.CameraY
	g.enemies, g.projectiles, g.pickups, g.mines = nil, nil, nil, nil
	g.burst, g.blasts, g.burstQueue = radialBurst{}, [3]projectile{}, nil
	g.randomSpawn, g.redSpawn, g.eventIndex = false, false, len(g.resources.events)
	for p, src := range s.Players {
		value := &g.player
		if p == 1 {
			value = &g.player2
		}
		*value = actor{x: float64(src.X - 48), y: s.CameraY + float64(src.Y-4), health: 1,
			inTank: src.Boarded, tankFuel: src.Fuel, tankSlot: 19 + p, tankBurning: src.Burning,
			tankBurn: src.Burn, deadTicks: src.Fall, invulnerable: src.Protection,
			tankFlame: walkAnimation{step: src.FlameStep, delay: src.FlameDelay}}
		if src.Dead {
			value.health = 0
		}
		if src.Turret >= 0x5a6e8 && src.Turret < 0x5a6e8+4*480 {
			value.tankFlame.frame = int(src.Turret-0x5a6e8) / 480
		}
		g.tankFuelPhase[p] = uint8(src.Phase)
		if src.Burning && !src.Boarded {
			g.enemies = append(g.enemies, actor{x: float64(src.TankX - 48), y: s.CameraY + float64(src.TankY-4),
				kind: 4, slot: 19 + p, health: 1, tankBurning: true, tankBurn: src.Burn,
				tankDriver: p, tankFuel: src.Fuel, tankFlame: value.tankFlame, bob: 0x5a328 - 0x42b88})
		}
	}
	g.lives, g.lives2 = s.Players[0].Lives, s.Players[1].Lives
	g.ammunition, g.ammunition2 = s.Players[0].Ammo, s.Players[1].Ammo
	g.grenades, g.grenades2 = s.Players[0].Grenades, s.Players[1].Grenades
}

func (s originalTankState) compareLifecycle(t *testing.T, g *Game) {
	t.Helper()
	for p, want := range s.Players {
		value := &g.player
		if p == 1 {
			value = &g.player2
		}
		burning, remaining, flame := value.inTank && value.tankBurning, value.tankBurn, value.tankFlame
		for _, tank := range g.enemies {
			if tank.kind == 4 && tank.tankBurning && tank.tankDriver == p {
				burning, remaining, flame = true, tank.tankBurn, tank.tankFlame
			}
		}
		if value.inTank != want.Boarded || burning != want.Burning || value.tankFuel != want.Fuel || int(g.tankFuelPhase[p]) != want.Phase {
			t.Fatalf("player%d tank state native boarded%v burning%v fuel%d phase%d original%+v", p, value.inTank, burning, value.tankFuel, g.tankFuelPhase[p], want)
		}
		if burning && remaining != want.Burn {
			t.Fatalf("player%d burning countdown%d original%d", p, remaining, want.Burn)
		}
		if burning && want.Burn != 0 && (flame.step != want.FlameStep || flame.delay != want.FlameDelay || 0x5a6e8+uint32(flame.frame)*480 != want.Turret) {
			t.Fatalf("player%d flame native%+v original step%d delay%d bob%x", p, flame, want.FlameStep, want.FlameDelay, want.Turret)
		}
	}
	if g.burst.active != s.BurstActive {
		t.Fatalf("tank expiry burst native%v original%v", g.burst.active, s.BurstActive)
	}
	if s.BurstActive {
		if g.burst.remaining != s.BurstRemaining {
			t.Fatalf("tank burst remaining%d original%d", g.burst.remaining, s.BurstRemaining)
		}
		for i, want := range s.Burst {
			part := g.burst.parts[i]
			if int(part.x+56) != want.X || int(part.y-g.cameraY) != want.Y {
				t.Fatalf("tank burst%d native%+v original%+v", i, part, want)
			}
		}
	}
}

func TestOriginalTankFuelAndBurningLifecycle(t *testing.T) {
	data, err := os.ReadFile("testdata/tank-lifecycle.json")
	if err != nil {
		t.Fatal(err)
	}
	var rows []struct {
		Kind                string
		Before, Early, Tail originalTankState
	}
	if err = json.Unmarshal(data, &rows); err != nil {
		t.Fatal(err)
	}
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range rows {
		t.Run(row.Kind, func(t *testing.T) {
			row.Before.supply(g)
			g.advanceTanks()
			row.Early.compareLifecycle(t, g)
			for p, want := range row.Early.Players {
				value := &g.player
				if p == 1 {
					value = &g.player2
				}
				if value.deadTicks != want.Fall || value.invulnerable != want.Protection {
					t.Fatalf("player%d early fall/protection native%d/%d original%d/%d", p, value.deadTicks, value.invulnerable, want.Fall, want.Protection)
				}
			}
			g.finishTankTick()
			row.Tail.compareLifecycle(t, g)
		})
	}
}

// Each sequence resumes the live original CPU normally after one controlled
// tank-state setup. These complete frames include context input, expiry,
// radial fragments, the infantry fall and ordinary reserve-life consumption.
func TestOriginalTankLifecycleThroughCompleteFrames(t *testing.T) {
	data, err := os.ReadFile("testdata/tank-lived.json")
	if err != nil {
		t.Fatal(err)
	}
	var rows []struct {
		Kind      string
		Tick      int
		State     originalTankState
		Destroyed []struct{ Row, Column int }
	}
	if err = json.Unmarshal(data, &rows); err != nil {
		t.Fatal(err)
	}
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range rows {
		if row.Tick == 0 {
			row.State.supply(g)
			g.playerCount = 1
			g.destroyed = [mapRows][mapColumns]bool{}
			for _, tile := range row.Destroyed {
				g.destroyed[tile.Row][tile.Column] = true
			}
			continue
		}
		controls := [2]playerControls{}
		if row.Kind == "leave" && row.Tick == 1 {
			controls[0].action = true
		}
		switch row.Kind {
		case "burn-right":
			controls[0].x = 1
		case "burn-left":
			controls[0].x = -1
		case "burn-up":
			controls[0].y = -1
		case "burn-down":
			controls[0].y = 1
		}
		g.stepPlay(controls)
		t.Run(row.Kind+"-"+strconv.Itoa(row.Tick), func(t *testing.T) {
			row.State.compareLifecycle(t, g)
			want, value := row.State.Players[0], &g.player
			if int(value.x+48) != want.X || int(value.y-g.cameraY+4) != want.Y || value.deadTicks != want.Fall || value.invulnerable != want.Protection || g.lives != want.Lives || g.ammunition != want.Ammo || g.grenades != want.Grenades {
				t.Fatalf("complete original tank frame differs: native pos%.0f/%.0f fall%d protection%d life%d ammo%d grenades%d original%+v", value.x+48, value.y-g.cameraY+4, value.deadTicks, value.invulnerable, g.lives, g.ammunition, g.grenades, want)
			}
		})
		if t.Failed() {
			t.FailNow()
		}
	}
}

func TestOriginalTankHitBoardingAndRefuelPhases(t *testing.T) {
	data, err := os.ReadFile("testdata/tank-context-phase.json")
	if err != nil {
		t.Fatal(err)
	}
	var rows []struct {
		Kind, Operation string
		Player          int
		Before, After   originalTankState
		SavedFuel       [3]int
	}
	if err = json.Unmarshal(data, &rows); err != nil {
		t.Fatal(err)
	}
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range rows {
		t.Run(row.Kind, func(t *testing.T) {
			row.Before.supply(g)
			switch row.Operation {
			case "hit":
				g.hitPlayer(row.Player)
			case "refuel":
				g.collectPickup(row.Player, 2)
			case "board":
				for p, tank := range row.Before.Players {
					if !tank.Boarded && tank.TankActive {
						g.enemies = append(g.enemies, actor{x: float64(tank.TankX - 48), y: row.Before.CameraY + float64(tank.TankY-4), kind: 4, slot: 19 + p, health: 1, tankFuel: row.SavedFuel[p]})
					}
				}
				if !g.useTank(row.Player) {
					t.Fatal("the native actor could not board its original available vehicle")
				}
			}
			row.After.compareLifecycle(t, g)
		})
	}
}

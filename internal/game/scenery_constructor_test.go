package game

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

type originalConstructorActor struct {
	Slot, X, Y, Art, Active, Cooldown int
}

type originalConstructorResources struct {
	Scores, Lives, Ammunition, Grenades [2]int
	Tank, Burning, Dead                 [2]bool
	Fall, Protection                    [2]int
	BurnP1                              int
}

type originalConstructorState struct {
	Actors                                         []originalConstructorActor
	Guard, Columns, Rows, OriginX, OriginY         int
	Active, Remaining, BurstRemaining, Busy, Owner int
	Phase                                          int
	ResourceValues                                 originalConstructorResources
}

func (s originalConstructorState) actor(slot int) originalConstructorActor {
	for _, value := range s.Actors {
		if value.Slot == slot {
			return value
		}
	}
	panic("missing original constructor actor")
}

type originalConstructorObservations struct {
	CameraY, InitialCameraY, BeforeInitialCameraY float64
	Initial, BeforeInitial                        []originalConstructorActor
	ColdCooldowns                                 []int
	Cases                                         []struct {
		Index int
		Case  struct {
			Group   string
			GunDead bool
		}
		Before, After originalConstructorState
	}
	Natural []struct {
		Tick, CameraY int
		Origin        [2]int
		Actors        []originalConstructorActor
	}
}

func readOriginalConstructor(t *testing.T) originalConstructorObservations {
	t.Helper()
	file, err := os.Open("testdata/scenery-constructor-effects.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	packed, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	defer packed.Close()
	var original originalConstructorObservations
	if err := json.NewDecoder(packed).Decode(&original); err != nil {
		t.Fatal(err)
	}
	if len(original.Cases) != 364 || len(original.Initial) != 5 || len(original.BeforeInitial) != 5 || len(original.ColdCooldowns) != 5 || len(original.Natural) != 3 {
		t.Fatal("incomplete original constructor observations")
	}
	return original
}

func (s originalConstructorResources) supply(g *Game) {
	g.score, g.score2 = s.Scores[0], s.Scores[1]
	g.lives, g.lives2 = s.Lives[0], s.Lives[1]
	g.ammunition, g.ammunition2 = s.Ammunition[0], s.Ammunition[1]
	g.grenades, g.grenades2 = s.Grenades[0], s.Grenades[1]
	g.playerCount = 2
	for i := range 2 {
		value := &g.player
		if i == 1 {
			value = &g.player2
		}
		*value = actor{health: 1, inTank: s.Tank[i], tankBurning: s.Burning[i], deadTicks: s.Fall[i], invulnerable: s.Protection[i]}
		if s.Dead[i] {
			value.health = 0
		}
	}
	g.player.tankBurn = s.BurnP1
}

func constructorResources(g *Game) originalConstructorResources {
	return originalConstructorResources{
		Scores: [2]int{g.score, g.score2}, Lives: [2]int{g.lives, g.lives2},
		Ammunition: [2]int{g.ammunition, g.ammunition2}, Grenades: [2]int{g.grenades, g.grenades2},
		Tank: [2]bool{g.player.inTank, g.player2.inTank}, Burning: [2]bool{g.player.tankBurning, g.player2.tankBurning},
		Dead: [2]bool{g.player.health == 0, g.player2.health == 0},
		Fall: [2]int{g.player.deadTicks, g.player2.deadTicks}, Protection: [2]int{g.player.invulnerable, g.player2.invulnerable},
		BurnP1: g.player.tankBurn,
	}
}

func TestOriginalFanTrapInitialState(t *testing.T) {
	original := readOriginalConstructor(t)
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	for i, source := range original.BeforeInitial {
		if g.fanTraps[i].cooldown != original.ColdCooldowns[i] {
			t.Fatal("cold fan-trap cooldown differs from the original executable")
		}
		// The actual source F1 restart retains the preceding cooldown while
		// restoring each record's original coordinate and active flag.
		g.fanTraps[i] = fanTrap{x: float64(source.X - 48), y: original.BeforeInitialCameraY + float64(source.Y-4), active: source.Active != 0, cooldown: source.Cooldown}
	}
	g.startGame(false)
	for i, source := range original.Initial {
		value := g.fanTraps[i]
		if value.x != float64(source.X-48) || value.y != original.InitialCameraY+float64(source.Y-4) || value.active != (source.Active != 0) || value.cooldown != source.Cooldown {
			t.Fatalf("fan trap %d: native %+v, original %+v", i, value, source)
		}
	}
}

// Original $113a6 calls measure every edge, shared-pool state, active mask,
// phantom guard and owner. The expected removals and player resources come
// from original RAM, including the early return after a slot32 contact.
func TestOriginalSceneryConstructorSideEffects(t *testing.T) {
	original := readOriginalConstructor(t)
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	failures := 0
	for _, source := range original.Cases {
		if !t.Run(fmt.Sprintf("case-%03d-%s", source.Index, source.Case.Group), func(t *testing.T) {
			before := source.Before
			g.cameraY = original.CameraY
			g.enemies = g.enemies[:0]
			before.ResourceValues.supply(g)
			for _, slot := range [...]int{32, 33} {
				value := before.actor(slot)
				if value.Active != 0 {
					enemy := actor{x: float64(value.X - 48), y: g.cameraY + float64(value.Y-4), kind: 6, slot: slot, health: 1}
					if slot == 32 && source.Case.GunDead {
						enemy.health = 0
					}
					g.enemies = append(g.enemies, enemy)
				}
			}
			for i := range g.fanTraps {
				value := before.actor(87 + i)
				g.fanTraps[i] = fanTrap{x: float64(value.X - 48), y: g.cameraY + float64(value.Y-4), active: value.Active != 0, cooldown: value.Cooldown}
			}
			origin := navPoint{float64(before.OriginX - 56), g.cameraY + float64(before.OriginY)}
			g.sceneryBurst = sceneryBurst{active: before.Active != 0, remaining: before.Remaining, phase: before.Phase, owner: before.Owner - 1, origin: origin}
			g.burst = radialBurst{active: before.Busy != 0, remaining: before.BurstRemaining, owner: before.Owner - 1}
			for i := range g.burst.parts {
				part := before.actor(49 + i)
				point := navPoint{float64(part.X - 56), g.cameraY + float64(part.Y)}
				g.burst.parts[i], g.sceneryBurst.parts[i] = point, point
			}
			value := sceneryBurst{active: true, enabled: 0xff, remaining: 8, phase: before.Phase, frame: 49, owner: before.Owner - 1, origin: origin}
			value.center = navPoint{origin.x + float64(before.Columns*8), origin.y + float64(before.Rows*8)}
			for i := range value.parts {
				group, quadrant := i/4, i%4
				value.parts[i] = navPoint{origin.x + float64(group*(before.Columns-1)*16+quadrant%2*16), origin.y + float64(group*(before.Rows-1)*16+quadrant/2*16)}
			}
			g.startSceneryBurstWithContactGuard(value, before.Guard != 0)
			for _, slot := range [...]int{32, 33} {
				present := false
				for _, enemy := range g.enemies {
					present = present || enemy.slot == slot
				}
				if present != (source.After.actor(slot).Active != 0) {
					t.Fatalf("gun %d native active%v, original active%d", slot, present, source.After.actor(slot).Active)
				}
			}
			for i, trap := range g.fanTraps {
				expected := source.After.actor(87 + i)
				if trap.active != (expected.Active != 0) || trap.cooldown != expected.Cooldown || trap.x != float64(expected.X-48) || trap.y != g.cameraY+float64(expected.Y-4) {
					t.Fatalf("fan trap %d native %+v, original %+v", i, trap, expected)
				}
			}
			if actual := constructorResources(g); actual != source.After.ResourceValues {
				t.Fatalf("constructor changed player resources: native %+v, original %+v", actual, source.After.ResourceValues)
			}
			if g.burst.active != (source.After.Busy != 0) || g.sceneryBurst.remaining != source.After.Remaining || g.sceneryBurst.phase != source.After.Phase || g.sceneryBurst.owner != source.After.Owner-1 {
				t.Fatal("the constructor changed its existing shared effect state differently")
			}
		}) {
			failures++
		}
	}
	t.Logf("original constructor updates: %d cases, %d differences", len(original.Cases), failures)
}

// The normally played original keeps this gun alive through tick1525 and
// removes it by tick1550 when the nearby composite detonates. No enemy death
// sequence or extra score is involved in the constructor's flag clear.
func TestOriginalPlayedSceneryClearsTheGun(t *testing.T) {
	original := readOriginalConstructor(t)
	before, after := original.Natural[1], original.Natural[2]
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.cameraY = float64(before.CameraY)
	for _, source := range before.Actors {
		if source.Slot == 32 || source.Slot == 33 {
			if source.Active != 0 {
				g.enemies = append(g.enemies, actor{x: float64(source.X - 48), y: g.cameraY + float64(source.Y-4), kind: 6, slot: source.Slot, health: 1})
			}
		} else if source.Slot >= 87 {
			g.fanTraps[source.Slot-87] = fanTrap{x: float64(source.X - 48), y: g.cameraY + float64(source.Y-4), active: source.Active != 0, cooldown: source.Cooldown}
		}
	}
	g.startSceneryBurst(sceneryBurst{origin: navPoint{float64(after.Origin[0] - 56), g.cameraY + float64(after.Origin[1])}})
	for _, source := range after.Actors {
		if source.Slot != 32 && source.Slot != 33 {
			continue
		}
		present := false
		for _, enemy := range g.enemies {
			present = present || enemy.slot == source.Slot
		}
		if present != (source.Active != 0) {
			t.Fatalf("normally played source gun%d visibility differs", source.Slot)
		}
	}
}

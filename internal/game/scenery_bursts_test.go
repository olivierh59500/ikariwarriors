package game

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

type originalSceneryPart struct {
	X, Y   int
	Bob    uint32
	Active uint8
}

type originalSceneryPlayer struct {
	X, Y, Progress, Burn, Fall, Protection int
	Active, Tank, Burning, Dead            bool
}

type originalSceneryState struct {
	Active, BurstActive, QuadActive                 bool
	Remaining, BurstRemaining, Owner, Phase, Queued int
	Columns, Rows, OriginX, OriginY                 int
	CameraY                                         float64
	Parts                                           []originalSceneryPart
	QuadParts                                       []originalSceneryPart
	Players                                         []originalSceneryPlayer
}

type originalSceneryObservations struct {
	Phases []struct {
		SourceTick int
		Combined   bool
		Before     originalSceneryState
		Frames     []originalSceneryState
	}
	Geometry []struct {
		Kind          string
		X, Y          int
		Before, After originalSceneryState
	}
	Contacts []struct {
		Case struct {
			Player, DX, DY, Owner int
			Kind                  string
		}
		Before, After originalSceneryState
	}
}

func readOriginalSceneryObservations(t *testing.T) originalSceneryObservations {
	t.Helper()
	file, err := os.Open("testdata/scenery-bursts.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	stream, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	var observations originalSceneryObservations
	if err := json.NewDecoder(stream).Decode(&observations); err != nil {
		t.Fatal(err)
	}
	return observations
}

func (source originalSceneryState) supply(g *Game) {
	g.startGamePlayers(false, 2)
	g.enemies, g.projectiles, g.pickups, g.mines, g.burstQueue = nil, nil, nil, nil, nil
	g.boss, g.final, g.aircraft = middleBoss{}, finalEncounter{}, helicopter{}
	g.cameraY, g.cameraBackY = source.CameraY, source.CameraY
	for p, original := range source.Players {
		value := &g.player
		if p == 1 {
			value = &g.player2
		}
		*value = actor{x: float64(original.X - 48), y: source.CameraY + float64(original.Y-4), health: 1, inTank: original.Tank, tankBurning: original.Burning, tankBurn: original.Burn, deadTicks: original.Fall, invulnerable: original.Protection}
		if !original.Active || original.Dead {
			value.health = 0
		}
	}
	origin := navPoint{float64(source.OriginX - 56), source.CameraY + float64(source.OriginY)}
	g.sceneryBurst = sceneryBurst{active: source.Active, shared: source.BurstActive, remaining: source.Remaining, phase: source.Phase, frame: (int(source.Parts[0].Bob) - 0x4a2e8) / 160, owner: source.Owner - 1, origin: origin, center: navPoint{origin.x + float64(source.Columns*8), origin.y + float64(source.Rows*8)}}
	g.burst = radialBurst{active: source.BurstActive, remaining: source.BurstRemaining, owner: source.Owner - 1}
	for range source.Queued {
		g.burstQueue = append(g.burstQueue, burstRequest{})
	}
	if source.QuadActive {
		first := source.QuadParts[0]
		g.blasts[2] = projectile{x: float64(first.X - 48), y: source.CameraY + float64(first.Y+8), enemy: true, owner: -1, grenade: true, impact: 4}
	}
	for i, part := range source.Parts {
		point := navPoint{float64(part.X - 56), source.CameraY + float64(part.Y)}
		g.sceneryBurst.parts[i], g.burst.parts[i] = point, point
		if part.Active != 0 {
			g.sceneryBurst.enabled |= 1 << i
		}
	}
}

func (source originalSceneryState) compare(t *testing.T, g *Game) {
	t.Helper()
	if g.sceneryBurst.active != source.Active || g.sceneryBurst.remaining != source.Remaining || g.sceneryBurst.phase != source.Phase || g.burst.active != source.BurstActive || g.burst.remaining != source.BurstRemaining {
		t.Fatalf("native scenery active%v remaining%d phase%d radial%v/%d; original%v/%d/%d/%v/%d", g.sceneryBurst.active, g.sceneryBurst.remaining, g.sceneryBurst.phase, g.burst.active, g.burst.remaining, source.Active, source.Remaining, source.Phase, source.BurstActive, source.BurstRemaining)
	}
	if len(g.burstQueue) != source.Queued {
		t.Fatalf("queued bursts native%d original%d", len(g.burstQueue), source.Queued)
	}
	if (g.blasts[2].impact > 0) != source.QuadActive {
		t.Fatal("shared enemy quad visibility differs")
	}
	if source.QuadActive {
		shot := g.blasts[2]
		for i, part := range source.QuadParts {
			if int(shot.x+48+float64(i%2*16)) != part.X || int(shot.y-g.cameraY-8+float64(i/2*16)) != part.Y || uint32(0x4a2e8+grenadeImpactFrame(shot.impact, i)*160) != part.Bob {
				t.Fatalf("queued enemy quad part%d differs from source%+v", i, part)
			}
		}
	}
	for i, part := range source.Parts {
		point, frame := g.sceneryBurst.parts[i], g.sceneryBurst.frame+i%4
		if g.burst.active || part.Bob == uint32(0x4a2e8+burstFrames[i]*160) {
			point, frame = g.burst.parts[i], burstFrames[i]
		}
		if source.Active && !source.BurstActive && (g.sceneryBurst.enabled&(1<<i) != 0) != (part.Active != 0) {
			t.Fatalf("scenery part%d active mask differs", i)
		}
		if int(point.x+56) != part.X || int(point.y-g.cameraY) != part.Y || uint32(0x4a2e8+frame*160) != part.Bob {
			t.Fatalf("part%d native%g/%g frame%d differs from original%+v", i, point.x+56, point.y-g.cameraY, frame, part)
		}
	}
	for p, expected := range source.Players {
		actual := &g.player
		if p == 1 {
			actual = &g.player2
		}
		if actual.deadTicks != expected.Fall || actual.invulnerable != expected.Protection || actual.inTank != expected.Tank || actual.tankBurning != expected.Burning || actual.tankBurn != expected.Burn {
			t.Fatalf("player%d native fall%d protection%d tank%v burning%v/%d original%+v", p, actual.deadTicks, actual.invulnerable, actual.inTank, actual.tankBurning, actual.tankBurn, expected)
		}
	}
}

// These timelines execute the original complete $885a-$895c phase from three
// states reached by normal play, plus three real shared-pool timelines starting
// at $8748. Values were exported directly from source RAM.
func TestOriginalSceneryBurstPhases(t *testing.T) {
	observations := readOriginalSceneryObservations(t)
	if len(observations.Phases) != 6 {
		t.Fatal("missing original scenery timelines")
	}
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, sequence := range observations.Phases {
		t.Run(fmt.Sprint(sequence.SourceTick), func(t *testing.T) {
			sequence.Before.supply(g)
			for tick, original := range sequence.Frames {
				if sequence.Combined {
					g.updateBurst()
				}
				g.updateSceneryBurst()
				t.Run(fmt.Sprint(tick), func(t *testing.T) { original.compare(t, g) })
			}
		})
	}
}

// The original $11776 impact routine supplies all four horizontal and both
// vertical parity variants. The crowded constructor additionally measures real
// $11290/$113a6 sharing rather than manufacturing expected native state.
func TestOriginalSceneryBurstConstruction(t *testing.T) {
	observations := readOriginalSceneryObservations(t)
	if len(observations.Geometry) != 9 {
		t.Fatal("missing original construction observations")
	}
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	for i, original := range observations.Geometry {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			original.Before.supply(g)
			x, y := float64(original.X), float64(original.Y)
			if original.Kind == "busy-constructor" {
				x, y = 85, 12336
			}
			value, ok := g.sceneryBurstAt(x, y)
			if !ok {
				t.Fatal("original impact has no recovered scenery")
			}
			g.startSceneryBurst(value)
			original.After.compare(t, g)
		})
	}
}

// Each case runs all original quad/contact calls in $885a-$895c. Only one
// source bob is enabled, so the rectangle edges remain independently visible.
func TestOriginalSceneryBurstContactBoundaries(t *testing.T) {
	observations := readOriginalSceneryObservations(t)
	if len(observations.Contacts) != 220 {
		t.Fatal("missing original scenery contact matrix")
	}
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	for i, original := range observations.Contacts {
		t.Run(fmt.Sprintf("%d-%s-p%d-%d-%d", i, original.Case.Kind, original.Case.Player, original.Case.DX, original.Case.DY), func(t *testing.T) {
			original.Before.supply(g)
			g.updateSceneryBurst()
			original.After.compare(t, g)
		})
	}
}

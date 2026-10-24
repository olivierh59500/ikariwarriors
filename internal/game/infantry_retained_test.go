package game

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

type originalRetainedInfantryState struct {
	Upper, Lower, UpperStep, UpperDelay, LowerStep, LowerDelay, Wet int
}

func originalRetainedInfantryActor(kind int, source originalRetainedInfantryState) actor {
	value := originalWetInfantryActor(kind, originalWetInfantryState{Upper: source.Upper, Lower: source.Lower, Step: source.LowerStep, Delay: source.LowerDelay})
	value.upper.step, value.upper.delay = source.UpperStep, source.UpperDelay
	upperBase := 0x4cb88
	if kind == 5 {
		upperBase = 0x51e08
	}
	if (source.Upper-upperBase)%actorBobBytes != 0 {
		value.upper.frame = infantryRawFrame + source.Upper - upperBase
	}
	return value
}

// F1 preserves infantry phase bytes during an unfinished death animation.
// The next short walking table therefore receives phases beyond its first
// terminator. The original routines read the following original data words
// and reset only when the selected word equals -1.
func TestOriginalControlledRetainedInfantryPhases(t *testing.T) {
	file, err := os.Open("testdata/infantry-retained-phases.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	packed, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	defer packed.Close()
	var original struct {
		Menu []struct {
			Slot, Kind    int
			Before, After originalRetainedInfantryState
		}
		Cases []struct {
			Index, Kind, Direction int
			Before                 originalRetainedInfantryState
			Frames                 []originalRetainedInfantryState
		}
	}
	if err := json.NewDecoder(packed).Decode(&original); err != nil {
		t.Fatal(err)
	}
	if len(original.Menu) != 14 || len(original.Cases) != 192 {
		t.Fatal("incomplete controlled original retained-phase observations")
	}
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, source := range original.Menu {
		value := originalRetainedInfantryActor(source.Kind, source.Before)
		value.slot = source.Slot
		g.enemies = append(g.enemies, value)
	}
	g.startGamePlayers(false, 1)
	for _, source := range original.Menu {
		pose := g.infantryPoses[source.Slot]
		value := actor{kind: source.Kind, upper: pose.upper, lower: pose.lower}
		upper, lower := value.infantryArtAddresses()
		if upper != source.After.Upper || lower != source.After.Lower || value.upper.step != source.After.UpperStep || value.upper.delay != source.After.UpperDelay || value.lower.step != source.After.LowerStep || value.lower.delay != source.After.LowerDelay {
			t.Fatalf("original F1 restart changed slot %d differently: native %+v, original %+v", source.Slot, pose, source.After)
		}
	}
	frames := 0
	for _, source := range original.Cases {
		t.Run(fmt.Sprintf("case-%03d-kind-%d-direction-%d", source.Index, source.Kind, source.Direction), func(t *testing.T) {
			if len(source.Frames) != 8 {
				t.Fatal("incomplete original retained-phase interval")
			}
			value := originalRetainedInfantryActor(source.Kind, source.Before)
			for tick, expected := range source.Frames {
				g.animateInfantryWalk(&value, source.Direction, expected.Wet != 0)
				upper, lower := value.infantryArtAddresses()
				if upper != expected.Upper || lower != expected.Lower || value.upper.step != expected.UpperStep || value.upper.delay != expected.UpperDelay || value.lower.step != expected.LowerStep || value.lower.delay != expected.LowerDelay {
					t.Fatalf("tick %d: art %x/%x phases %d/%d/%d/%d; original %+v", tick, upper, lower, value.upper.step, value.upper.delay, value.lower.step, value.lower.delay, expected)
				}
				frames++
			}
		})
	}
	if frames != 1536 {
		t.Fatalf("original retained-phase frames: got %d, want 1536", frames)
	}
}

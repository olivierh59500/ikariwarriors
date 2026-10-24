package game

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"testing"
)

type originalWetInfantryState struct {
	Upper, Lower, Step, Delay, Wet int
}

// Decode observed graphic pointers into the native actor representation.
// Expected pointers and phases always come from independently recorded Amiga
// memory; this helper does not produce any expected animation results.
func originalWetInfantryActor(kind int, source originalWetInfantryState) actor {
	upperBase, lowerBase := 0x4cb88, 0x50288
	if kind == 5 {
		upperBase, lowerBase = 0x51e08, 0x54608
	}
	value := actor{kind: kind, health: 1}
	value.upper.frame = (source.Upper - upperBase) / actorBobBytes
	value.lower = walkAnimation{step: source.Step, delay: source.Delay}
	switch source.Lower {
	case 0:
		value.lower.frame = -1
	case 0x4a1a8:
		value.lower.frame = -2
	case 0x4a248:
		value.lower.frame = -3
	default:
		value.lower.frame = (source.Lower - lowerBase) / overlayBobBytes
	}
	return value
}

// These are controlled original $fb82 calls, with the original sequence
// tables and explicit wet/dry inputs. They verify both banks, death overlays,
// phase wrap and terrain changes during delay-only ticks. They do not claim
// that the source player reached or crossed water during ordinary gameplay.
func TestOriginalControlledWetInfantryAnimation(t *testing.T) {
	file, err := os.Open("testdata/infantry-wet.json.gz")
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
		Cases []struct {
			Index, Kind, Direction, InitialDelay int
			Label                                string
			Sequence, Pattern                    []int
			Before                               originalWetInfantryState
			Frames                               []originalWetInfantryState
		}
	}
	if err := json.NewDecoder(packed).Decode(&original); err != nil {
		t.Fatal(err)
	}
	if len(original.Cases) != 56 {
		t.Fatalf("controlled original leg sequences: got %d, want 56", len(original.Cases))
	}
	g := &Game{}
	frames := 0
	for _, source := range original.Cases {
		t.Run(fmt.Sprintf("case-%02d-kind-%d-%s", source.Index, source.Kind, source.Label), func(t *testing.T) {
			sequence := enemyDeathLegSequence
			if source.Direction >= 0 {
				sequence = enemyPartSequences[source.Direction]
			}
			if !slices.Equal(sequence, source.Sequence) {
				t.Fatalf("leg table: got %v, want original %v", sequence, source.Sequence)
			}
			if len(source.Frames) != 16 || len(source.Pattern) != len(source.Frames) || source.Before.Delay != source.InitialDelay {
				t.Fatal("invalid controlled original leg observations")
			}
			value := originalWetInfantryActor(source.Kind, source.Before)
			for tick, expected := range source.Frames {
				if expected.Wet != source.Pattern[tick] {
					t.Fatal("original wet input differs from the recorded control pattern")
				}
				g.animateInfantryLegs(&value, sequence, expected.Wet != 0)
				upper, lower := value.infantryArtAddresses()
				if upper != expected.Upper || lower != expected.Lower || value.lower.step != expected.Step || value.lower.delay != expected.Delay {
					t.Fatalf("tick %d: upper/lower %x/%x, step/delay %d/%d; original %x/%x, %d/%d", tick, upper, lower, value.lower.step, value.lower.delay, expected.Upper, expected.Lower, expected.Step, expected.Delay)
				}
				frames++
			}
		})
	}
	if frames != 896 {
		t.Fatalf("controlled original leg calls: got %d, want 896", frames)
	}
}

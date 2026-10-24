package game

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

type originalInfantryReplacementActor struct {
	Slot, Upper, Lower, Active                   int
	UpperStep, UpperDelay, LowerStep, LowerDelay int
}

func replacementInfantryKind(slot int) int {
	if slot >= 15 {
		return 5
	}
	return 0
}

func replacementInfantryActor(source originalInfantryReplacementActor) actor {
	value := originalRetainedInfantryActor(replacementInfantryKind(source.Slot), originalRetainedInfantryState{
		Upper: source.Upper, Lower: source.Lower,
		UpperStep: source.UpperStep, UpperDelay: source.UpperDelay,
		LowerStep: source.LowerStep, LowerDelay: source.LowerDelay,
	})
	value.slot = source.Slot
	return value
}

// Actual $e3ee/$93b6 calls overwrite the final/ending body roles and clear
// their leg pointers, while preserving all four independent animation arrays.
// Seeded source phases and before/after values come from original RAM only.
func TestOriginalInfantrySlotReplacement(t *testing.T) {
	file, err := os.Open("testdata/infantry-slot-replacement.json.gz")
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
			Index, PC     int
			Label         string
			Before, After []originalInfantryReplacementActor
		}
	}
	if err := json.NewDecoder(packed).Decode(&original); err != nil {
		t.Fatal(err)
	}
	if len(original.Cases) != 4 {
		t.Fatal("incomplete original slot-replacement observations")
	}
	for _, source := range original.Cases {
		t.Run(fmt.Sprintf("%s-%d", source.Label, source.Index), func(t *testing.T) {
			g, err := New(Options{Mute: true})
			if err != nil {
				t.Fatal(err)
			}
			if len(source.Before) != 19 || len(source.After) != 19 {
				t.Fatal("incomplete original infantry slot range")
			}
			for _, before := range source.Before {
				value := replacementInfantryActor(before)
				g.infantryPoses[before.Slot] = infantryPose{value.upper, value.lower}
				if before.Active != 0 {
					g.enemies = append(g.enemies, value)
				}
			}
			switch source.PC {
			case 0xe3ee:
				g.spawnFinalEncounter(levelEvent{progress: 100})
			case 0x93b6:
				g.spawnEnding(levelEvent{progress: 100})
			default:
				t.Fatal("unexpected original slot initializer")
			}
			check := func() {
				for _, after := range source.After {
					pose := g.infantryPoses[after.Slot]
					if pose.upper.step != after.UpperStep || pose.upper.delay != after.UpperDelay || pose.lower.step != after.LowerStep || pose.lower.delay != after.LowerDelay {
						t.Errorf("slot%d lost retained animation counters", after.Slot)
					}
					kind := replacementInfantryKind(after.Slot)
					_, lower := (actor{kind: kind, upper: pose.upper, lower: pose.lower}).infantryArtAddresses()
					if lower != after.Lower {
						t.Errorf("slot%d retained leg pointer%x, original%x", after.Slot, lower, after.Lower)
					}
				}
			}
			check()
			for _, enemy := range g.enemies {
				if source.After[enemy.slot].Lower == 0 {
					t.Errorf("overwritten slot%d still retains an old infantry actor", enemy.slot)
				}
			}
			// Exercise the real menu restart, which remembers surviving actors.
			// Removed infantry must not restore the leg pointer just cleared.
			g.startGame(false)
			check()
			for slot := 5; slot <= 18; slot++ {
				g.placeEnemy(levelEvent{progress: 100}, slot, replacementInfantryKind(slot), 100, 0)
				value := g.enemies[len(g.enemies)-1]
				_, lower := value.infantryArtAddresses()
				if lower != source.After[slot].Lower || value.upper.step != source.After[slot].UpperStep || value.upper.delay != source.After[slot].UpperDelay || value.lower.step != source.After[slot].LowerStep || value.lower.delay != source.After[slot].LowerDelay {
					t.Errorf("reused slot%d lost the original post-initializer leg/counter state", slot)
				}
			}
		})
	}
}

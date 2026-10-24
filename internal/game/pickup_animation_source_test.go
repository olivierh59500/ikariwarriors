package game

import (
	"compress/gzip"
	"encoding/json"
	"os"
	"testing"
)

type sourcePickupAnimationFixture struct {
	Tables       [8][3]int `json:"tables"`
	Constructors []struct {
		Frame, Kind       int
		Initial, Retained bool
		Before            [4]int
		After             [][4]int
	} `json:"constructors"`
}

func readSourcePickupAnimations(t *testing.T) sourcePickupAnimationFixture {
	t.Helper()
	f, err := os.Open("testdata/pickup-animation-constructors.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	z, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	defer z.Close()
	var fixture sourcePickupAnimationFixture
	if err := json.NewDecoder(z).Decode(&fixture); err != nil {
		t.Fatal(err)
	}
	if len(fixture.Constructors) != 40 {
		t.Fatal("incomplete original pickup animation observations")
	}
	return fixture
}

// These before/after observations cover all eight original pickup tables and
// 32 actual retained-phase rebirths. Only the isolated starting phase is supplied.
func TestOriginalPickupAnimationConstructorsAndRetainedPhases(t *testing.T) {
	fixture := readSourcePickupAnimations(t)
	for kind, table := range fixture.Tables {
		if table[2] != -1 || len(pickupAnimationFrames[kind]) != 2 {
			t.Fatal("original pickup table terminator differs")
		}
		for frame := range 2 {
			if pickupAnimationFrames[kind][frame]*bobBytes != table[frame] {
				t.Fatalf("kind%d native pickup table differs", kind)
			}
		}
	}
	for _, row := range fixture.Constructors {
		g, err := New(Options{Mute: true})
		if err != nil {
			t.Fatal(err)
		}
		g.pickupAnimations[row.Kind] = walkAnimation{step: row.Before[1], delay: row.Before[2], frame: (row.Before[3] - 0x4a2e8) / bobBytes}
		g.addPickup(pickup{kind: row.Kind})
		for offset, want := range row.After {
			if want[0] == 0 {
				break
			}
			g.animatePickups()
			got := g.pickupAnimations[row.Kind]
			if got.step != want[1] || got.delay != want[2] || 0x4a2e8+g.pickupImage(row.Kind)*bobBytes != want[3] {
				t.Fatalf("source birth%d+%d kind%d native%+v image%x original%v", row.Frame, offset, row.Kind, got, 0x4a2e8+g.pickupImage(row.Kind)*bobBytes, want)
			}
		}
		saved := g.pickupAnimations
		g.pickups = nil
		g.animatePickups()
		if g.pickupAnimations != saved {
			t.Fatal("inactive pickup phases advanced")
		}
		g.startGame(false)
		if g.pickupAnimations != saved {
			t.Fatal("new game cleared retained pickup phases")
		}
	}
}

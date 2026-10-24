package game

import (
	"compress/gzip"
	"encoding/json"
	"os"
	"strconv"
	"testing"
)

// Actual consecutive original frames provide the movement, independent image
// pointers and animation counters. These intervals contain no firing pause.
func TestOriginalNaturallySpawnedInfantryTransitions(t *testing.T) {
	file, err := os.Open("testdata/infantry-transitions.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	reader, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	type state struct {
		X, Y, Upper, Lower, UpperStep, UpperDelay, LowerStep, LowerDelay, Shots int
		Path                                                                    [3]int
	}
	var cases []struct {
		Tick, Slot    int
		Before, After state
		Terrain       [][3]int
	}
	if err := json.NewDecoder(reader).Decode(&cases); err != nil {
		t.Fatal(err)
	}
	if len(cases) < 1000 {
		t.Fatalf("only %d original infantry transitions", len(cases))
	}
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	initial := g.resources.levelCodes
	for _, original := range cases {
		for _, tile := range original.Terrain {
			g.resources.levelCodes[tile[0]][tile[1]] = byte(tile[2])
		}
		before := original.Before
		kind, upperBase, lowerBase := 0, 0x4cb88, 0x50288
		if original.Slot >= 15 {
			kind, upperBase, lowerBase = 5, 0x51e08, 0x54608
		}
		value := actor{x: float64(before.X), y: float64(before.Y), health: 1, kind: kind, slot: original.Slot, marchY: before.Path[0], marchX: before.Path[1], marchTime: before.Path[2], shots: before.Shots}
		value.upper = walkAnimation{step: before.UpperStep, delay: before.UpperDelay, frame: (before.Upper - upperBase) / actorBobBytes}
		value.lower = walkAnimation{step: before.LowerStep, delay: before.LowerDelay, frame: (before.Lower - lowerBase) / overlayBobBytes}
		switch before.Lower {
		case 0:
			value.lower.frame = -1
		case 0x4a1a8:
			value.lower.frame = -2
		case 0x4a248:
			value.lower.frame = -3
		}
		g.marchEnemy(&value)
		after := original.After
		upper, lower := value.infantryArtAddresses()
		if int(value.x) != after.X || int(value.y) != after.Y || value.marchY != after.Path[0] || value.marchX != after.Path[1] || value.marchTime != after.Path[2] || value.shots != after.Shots || upper != after.Upper || lower != after.Lower || value.upper.step != after.UpperStep || value.upper.delay != after.UpperDelay || value.lower.step != after.LowerStep || value.lower.delay != after.LowerDelay {
			t.Errorf("source tick%d slot%d: raw native position %.0f/%.0f, path%d/%d/%d shots%d upper%06x/%d/%d lower%06x/%d/%d; original%+v", original.Tick, original.Slot, value.x, value.y, value.marchY, value.marchX, value.marchTime, value.shots, upper, value.upper.step, value.upper.delay, lower, value.lower.step, value.lower.delay, after)
		}
		for _, tile := range original.Terrain {
			g.resources.levelCodes[tile[0]][tile[1]] = initial[tile[0]][tile[1]]
		}
	}
}

// These real births retain the preceding slot's leg image and both delayed
// animation phases. Menu restart clears activity without clearing that state.
func TestOriginalInfantrySlotReuse(t *testing.T) {
	data, err := os.ReadFile("testdata/infantry-slot-reuse.json")
	if err != nil {
		t.Fatal(err)
	}
	var births []struct {
		Tick, Slot   int
		UpperBefore  string `json:"upper_before"`
		LowerBefore  string `json:"lower_before"`
		BeforePhases [4]int `json:"before_phases"`
		UpperAfter   string `json:"upper_after"`
		LowerAfter   string `json:"lower_after"`
		AfterPhases  [4]int `json:"after_phases"`
	}
	if err := json.Unmarshal(data, &births); err != nil {
		t.Fatal(err)
	}
	if len(births) != 27 {
		t.Fatalf("original retained-phase births%d, expected27", len(births))
	}
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	address := func(text string) int {
		value, e := strconv.ParseInt(text, 0, 32)
		if e != nil {
			t.Fatal(e)
		}
		return int(value)
	}
	for _, birth := range births {
		kind, upperBase, lowerBase := 0, 0x4cb88, 0x50288
		if birth.Slot >= 15 {
			kind, upperBase, lowerBase = 5, 0x51e08, 0x54608
		}
		value := actor{slot: birth.Slot, kind: kind, health: 1,
			upper: walkAnimation{step: birth.BeforePhases[0], delay: birth.BeforePhases[1], frame: (address(birth.UpperBefore) - upperBase) / actorBobBytes},
			lower: walkAnimation{step: birth.BeforePhases[2], delay: birth.BeforePhases[3], frame: (address(birth.LowerBefore) - lowerBase) / overlayBobBytes}}
		g.enemies = []actor{value}
		g.startGamePlayers(false, 1)
		g.placeEnemy(levelEvent{progress: 100}, birth.Slot, kind, 120, -10)
		next := &g.enemies[len(g.enemies)-1]
		next.marchTime = 30
		g.marchEnemy(next)
		upper, lower := next.infantryArtAddresses()
		phases := [4]int{next.upper.step, next.upper.delay, next.lower.step, next.lower.delay}
		if upper != address(birth.UpperAfter) || lower != address(birth.LowerAfter) || phases != birth.AfterPhases {
			t.Fatalf("source tick%d slot%d birth art%06x/%06x phases%v, original%s/%s phases%v", birth.Tick, birth.Slot, upper, lower, phases, birth.UpperAfter, birth.LowerAfter, birth.AfterPhases)
		}
	}
}

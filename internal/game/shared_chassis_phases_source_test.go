package game

import (
	"compress/gzip"
	"encoding/json"
	"os"
	"testing"
)

type sourceChassisGroup struct {
	Controller                       [4]int
	Active                           [3]bool
	X, Y, Base, Partner, Step, Delay int
}
type sourceChassisFrame struct {
	Frame, Camera, Helicopter int
	Groups                    [2]sourceChassisGroup
}
type sourceChassisEvent struct{ Progress, X, Count, Kind, Spacing int }
type sourceChassisTransition struct {
	Group         int
	Before, After sourceChassisFrame
	Event         sourceChassisEvent
}
type sourceChassisFixture struct {
	Constructors, FlightBirths, FlightExits []sourceChassisTransition
	Sentinel                                sourceChassisTransition
	Phases                                  [][4]int
}

func readSourceChassisPhases(t *testing.T) sourceChassisFixture {
	t.Helper()
	f, err := os.Open("testdata/source-shared-chassis-phases.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	z, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	defer z.Close()
	var rows sourceChassisFixture
	if err := json.NewDecoder(z).Decode(&rows); err != nil {
		t.Fatal(err)
	}
	if len(rows.Constructors) != 18 || len(rows.FlightBirths) != 8 || len(rows.FlightExits) != 8 || len(rows.Phases) != 12683 {
		t.Fatal("incomplete original shared chassis observations")
	}
	return rows
}

// Every constructor starts once from actual original phases and occupancy.
// Images, coordinates and the next phase come only from the original snapshots.
func TestOriginalSharedVehicleConstructorPhases(t *testing.T) {
	for _, row := range readSourceChassisPhases(t).Constructors {
		g, err := New(Options{Mute: true})
		if err != nil {
			t.Fatal(err)
		}
		g.cameraY = float64(row.After.Camera)
		for group, prior := range row.Before.Groups {
			g.vehicleChassisPhases[group] = vehicleChassisPhase{prior.Step, prior.Delay}
			g.enemyVehicles[group] = enemyVehicle{x: float64(prior.X - 48), y: float64(row.Before.Camera + prior.Y - 4), active: prior.Controller[0] != 0, alive: prior.Active, mode: prior.Controller[1], turn: byte(prior.Controller[2]), reverse: byte(prior.Controller[3]), chassis: walkAnimation{step: prior.Step, delay: prior.Delay, frame: (prior.Base - 0x60e08) / vehicleBobBytes}}
		}
		e := row.Event
		g.spawnEnemyVehicles(levelEvent{progress: e.Progress, x: e.X, count: e.Count, kind: e.Kind, spacing: e.Spacing})
		g.updateEnemyVehicles()
		v := g.enemyVehicles[row.Group]
		want := row.After.Groups[row.Group]
		if v.chassis.step != want.Step || v.chassis.delay != want.Delay || v.part(row.Group, 0).bob+0x42b88 != want.Base || v.part(row.Group, 1).bob+0x42b88 != want.Partner || int(v.x+48) != want.X || int(v.y-g.cameraY+4) != want.Y || g.vehicleChassisPhases[row.Group] != (vehicleChassisPhase{want.Step, want.Delay}) {
			t.Fatalf("source constructor%d group%d native%+v cache%+v original%+v", row.After.Frame, row.Group, v, g.vehicleChassisPhases[row.Group], want)
		}
	}
}

func TestOriginalRotorSharedPhaseBoundaries(t *testing.T) {
	rows := readSourceChassisPhases(t)
	for _, row := range rows.FlightBirths {
		g, err := New(Options{Mute: true})
		if err != nil {
			t.Fatal(err)
		}
		for group, s := range row.Before.Groups {
			g.vehicleChassisPhases[group] = vehicleChassisPhase{s.Step, s.Delay}
		}
		e := row.Event
		g.spawnAircraft(levelEvent{progress: e.Progress, x: e.X, count: e.Count, kind: e.Kind, spacing: e.Spacing})
		if g.vehicleChassisPhases != [2]vehicleChassisPhase{} {
			t.Fatal("aircraft initialization retained phases explicitly cleared by the source")
		}
		g.advanceAircraft()
		for group, want := range row.After.Groups {
			if g.vehicleChassisPhases[group] != (vehicleChassisPhase{want.Step, want.Delay}) {
				t.Fatalf("source rotor birth%d cache%+v original%+v", row.After.Frame, g.vehicleChassisPhases, row.After.Groups)
			}
		}
	}
	for _, row := range rows.FlightExits {
		g, err := New(Options{Mute: true})
		if err != nil {
			t.Fatal(err)
		}
		before := row.Before.Groups[0]
		// The source exit branch is reached only with an inactive anchor. Its
		// rotor pass still runs before that branch, including after early damage.
		g.aircraft = helicopter{phase: (3 - before.Step) % 3, flying: true}
		g.advanceAircraft()
		for group, want := range row.After.Groups {
			if g.vehicleChassisPhases[group] != (vehicleChassisPhase{want.Step, want.Delay}) {
				t.Fatalf("source rotor exit%d cache%+v original%+v", row.After.Frame, g.vehicleChassisPhases, row.After.Groups)
			}
		}
	}
}

// Original12223 reads the padded -1 sentinel after retained step2 enters a
// two-step table. It resets to0 rather than using modulo to select step1.
func TestOriginalVehicleShortTableSentinel(t *testing.T) {
	row := readSourceChassisPhases(t).Sentinel
	before, want := row.Before.Groups[0], row.After.Groups[0]
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.cameraY = float64(row.After.Camera)
	g.enemyVehicles[0] = enemyVehicle{x: float64(before.X - 48), y: float64(row.Before.Camera + before.Y - 4), active: true, mode: before.Controller[1], turn: byte(before.Controller[2]), reverse: byte(before.Controller[3]), chassis: walkAnimation{step: before.Step, delay: before.Delay, frame: (before.Base - 0x60e08) / vehicleBobBytes}}
	g.updateEnemyVehicles()
	v := g.enemyVehicles[0]
	if v.chassis.step != want.Step || v.chassis.delay != want.Delay || v.part(0, 0).bob+0x42b88 != want.Base {
		t.Fatalf("original short-table transition native%+v original%+v", v.chassis, want)
	}
}

// The original start path clears controllers but not these animation counters;
// its decoder clears BOB activity only. No new runtime F1 capture is claimed.
func TestNewGameRetainsSharedVehicleChassisPhases(t *testing.T) {
	before := readSourceChassisPhases(t).FlightBirths[6].Before
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	for group, s := range before.Groups {
		g.vehicleChassisPhases[group] = vehicleChassisPhase{s.Step, s.Delay}
	}
	saved := g.vehicleChassisPhases
	g.startGame(false)
	for range 116 {
		if err := g.Update(); err != nil {
			t.Fatal(err)
		}
	}
	if g.vehicleChassisPhases != saved {
		t.Fatal("new game/arrival reset the statically retained shared phases")
	}
}

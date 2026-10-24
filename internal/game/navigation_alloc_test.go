//go:build !rendercheck

package game

import "testing"

// AllocsPerRun counts allocations across the entire process. Keep this check
// in the ordinary suite, where the concurrent GPU loop cannot add allocations
// from pending texture commands to a navigation-only measurement.
func TestNavigationRebuildReusesItsBuffers(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGame(true)
	if allocations := testing.AllocsPerRun(10, g.buildDemoPath); allocations != 0 {
		t.Fatalf("navigation rebuild allocates %.1f times", allocations)
	}
}

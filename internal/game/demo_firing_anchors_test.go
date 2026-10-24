package game

import (
	"reflect"
	"slices"
	"testing"
)

func TestDemoAnchorForecastDoesNotChangeLiveState(t *testing.T) {
	g, _ := loadUnsafeAttackState(t)
	before := *g
	before.enemies = slices.Clone(g.enemies)
	before.projectiles = slices.Clone(g.projectiles)
	before.pickups = slices.Clone(g.pickups)
	before.mines = slices.Clone(g.mines)
	before.burstQueue = slices.Clone(g.burstQueue)
	before.demoPath = slices.Clone(g.demoPath)
	before.navParents = slices.Clone(g.navParents)
	before.navQueue = slices.Clone(g.navQueue)
	g.demoAnchorReady(g.player.x, g.player.y, 0)
	if !reflect.DeepEqual(*g, before) {
		t.Fatal("prospective firing changed live physics, resources, RNG or route")
	}
}

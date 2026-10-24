package game

import "testing"

// Original9357's enemy cannon expires after the chassis cleanup pass. E6F2
// then clears slots26..30 before9358, retaining only turret31's hit mark.
func TestVehicleEnemyExplosionUsesLateDamageCleanup(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.cameraY = 4084
	g.enemyVehicles = [2]enemyVehicle{
		{x: 77, y: 4268, active: true, alive: [3]bool{true, true, true}, mode: 1, reverse: 254, chassis: walkAnimation{frame: 6}},
		{x: 117, y: 4224, active: true, alive: [3]bool{true, true, true}, mode: 1, reverse: 254, chassis: walkAnimation{frame: 6}},
	}
	g.updateEnemyVehicles()
	g.damageEnemyVehicles(projectile{x: 117, y: 4222, enemy: true, owner: -1, grenade: true}, false)
	if g.enemyVehicles[1].damaged != 7 {
		t.Fatal("late original blast did not reach all second-group parts")
	}
	g.finishEnemyVehicleDamage()
	if g.enemyVehicles[0].damaged != 0 || g.enemyVehicles[1].damaged != 4 {
		t.Fatal("late five-part cleanup differs from original9357 hit marks")
	}
	g.updateEnemyVehicles()
	if !g.enemyVehicles[1].active || g.enemyVehicles[1].alive != [3]bool{true, true, true} {
		t.Fatal("late enemy blast incorrectly destroyed the next chassis update")
	}
	// The retained final turret must not award another hit on the same part.
	part := g.enemyVehicles[1].part(1, 2)
	g.damageEnemyVehicles(projectile{x: part.x + 15, y: part.y - 2, owner: 0}, true)
	if g.score != 100 {
		t.Fatalf("fragment award%d, expected only the newly hit chassis and no repeated turret award", g.score)
	}
	if g.enemyVehicles[1].damaged&4 == 0 {
		t.Fatal("retained original final-turret mark disappeared")
	}
}

// The early owned collision still reaches the original chassis cleanup in
// the same gameplay update; late cleanup must not delay its destruction.
func TestVehicleOwnedBlastRetainsEarlyChassisDestruction(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.cameraY = 4084
	g.enemyVehicles[1] = enemyVehicle{x: 117, y: 4224, active: true, alive: [3]bool{true, true, true}, mode: 1, reverse: 254, chassis: walkAnimation{frame: 6}}
	g.damageEnemyVehicles(projectile{x: 117, y: 4224, owner: 0, grenade: true}, false)
	if g.enemyVehicles[1].damaged&1 == 0 {
		t.Fatal("original early chassis collision was not exercised")
	}
	g.updateEnemyVehicles()
	if g.enemyVehicles[1].active || g.enemyVehicles[1].alive != [3]bool{} || !g.burst.active {
		t.Fatal("early owned blast missed its original chassis destruction phase")
	}
	g.finishEnemyVehicleDamage()
	if g.enemyVehicles[1].damaged != 0 {
		t.Fatal("destroyed group retained a spurious late turret mark")
	}
}

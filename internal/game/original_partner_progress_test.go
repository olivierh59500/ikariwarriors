package game

import "testing"

// E554 reads both progress words without testing the number of entered
// players. Keep this inactive-partner counterexample outside the cold1P trace.
func TestHorizontalGunsScanAnUnenteredPartnersLogicalProgress(t *testing.T) {
	g, err := New(Options{Mute: true, RandomSource: func(uint64) uint32 { return 0 }})
	if err != nil {
		t.Fatal(err)
	}
	g.startGame(false)
	g.player.y = levelHeight - 38
	beforePose := g.player2
	// Represent source partner progress11660 using the existing terrain
	// offset, while leaving its unused visual landing pose unchanged.
	g.cooperativeScroll[1].terrainOffset = levelHeight - 16 - g.player2.y - 11660
	before := g.randomSourceIndex
	g.updateHorizontalGuns()
	if g.playerCount != 1 || g.randomSourceIndex != before+1 || len(g.projectiles) != 1 {
		t.Fatal("unentered partner progress was skipped by the original two-value scan")
	}
	if g.player2 != beforePose {
		t.Fatal("logical progress scan changed the unused visual partner")
	}
	shot := g.projectiles[0]
	if shot.x != 160 || shot.y != levelHeight-11678 || shot.vx != -4 || shot.vy != 0 || shot.life != 28 {
		t.Fatal("partner progress did not use the original first horizontal gun")
	}
}

func TestUnenteredPartnerRetainsOriginalNegativeProgress(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGame(false)
	progress := func() int { return levelHeight - 16 - int(g.player2.y+g.cooperativeScroll[1].terrainOffset) }
	if progress() != -210 {
		t.Fatal("source one-player initialization did not retain partner progress-210")
	}
	for range 116 {
		if err := g.Update(); err != nil {
			t.Fatal(err)
		}
	}
	if progress() != -210 {
		t.Fatal("landing replaced inactive logical progress with a visual pose")
	}
	forecast := *g
	forecast.cooperativeScroll[1].terrainOffset -= 11870
	if progress() != -210 {
		t.Fatal("forecast changed live partner progress")
	}
}

func TestDepartedPartnersLogicalProgressRemainsOutsideGunBands(t *testing.T) {
	g, err := New(Options{Mute: true, RandomSource: func(uint64) uint32 { return 0 }})
	if err != nil {
		t.Fatal(err)
	}
	g.startGamePlayers(false, 2)
	g.player2.health, g.player2.deadTicks, g.lives2 = 0, 1, 0
	g.advancePlayerFall(1)
	progress := levelHeight - 16 - int(g.player2.y+g.cooperativeScroll[1].terrainOffset)
	if g.lives2 != -1 || progress != 0 {
		t.Fatal("departed partner lost its original separate terrain progress")
	}
	g.player.y = levelHeight - 16 - 11660
	before := g.randomSourceIndex
	g.updateHorizontalGuns()
	if g.randomSourceIndex != before+1 || len(g.projectiles) != 1 {
		t.Fatal("departed visual pose incorrectly added another gun-band scan")
	}
}

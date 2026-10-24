package game

import (
	"crypto/sha256"
	"fmt"
	"testing"

	gameassets "github.com/olivierh59500/ikariwarriors/assets"
)

func TestRecoveredRuntimeAssets(t *testing.T) {
	tests := []struct {
		name   string
		bytes  int
		digest string
	}{
		{"audio-driver.bin", 26130, "9c3bf15884f52565626c6ef17a8e862f62c0d6d3b79cd2b3c642978ff009d20d"},
		{"bobs-primary.bin", 10400, "c78e502a71c16abb9099dedc04568772c8c762a064271ba48e4fd7fb1dd2d54b"},
		{"bobs-player2-vehicles.bin", 7040, "029ce213b07ddeaf62c17d463f88118333026656846e50776508fe849c875ec8"},
		{"bobs-tank-enemy.bin", 11520, "bc3556f4081ba52b38d8ffcffe6764b714c691803961e8e2c4f4e297c3d8e651"},
		{"bobs-tank-player1.bin", 11520, "09025095d5a7068abf5902cdaa29c8c577297de587b99fb3de6c183c485da616"},
		{"bobs-tank-player2.bin", 11520, "9061fb7535fcd3e0a5d3d1bb4bac2ba571bd04e484e04e20081eed8e47acf030"},
		{"level.bin", 24388, "d54867fb279b654f3bbadb77158c5c4f0320469c707ba47f282a7320c3fe8aff"},
		{"tiles-primary.bin", 40960, "15d147873063edf8a52857212cc6667e1084fe77dfc5b8362dcff0245c388958"},
		{"tiles-secondary.bin", 40960, "10e6f835a68060fdf3a83bdfdaf5ca331999d77727fdff9dd7ff0b5ae9e5d24d"},
		{"title-actors.png", 723, "7406910942359c60914bc5480026a81702c0060a345e06da12ec090b5b2f7941"},
	}
	for _, test := range tests {
		data, err := gameassets.Original.ReadFile("original/" + test.name)
		if err != nil {
			t.Fatal(err)
		}
		if len(data) != test.bytes || fmt.Sprintf("%x", sha256.Sum256(data)) != test.digest {
			t.Fatalf("%s does not match its recovered source", test.name)
		}
	}
}

func TestGameStartsWithRecoveredMapAndSprites(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(g.resources.players[0]) != 22 || len(g.resources.players[1]) != 22 || len(g.resources.enemies) != 44 || len(g.resources.effects) != 65 || len(g.resources.vehicles[0]) != 24 || len(g.resources.vehicles[1]) != 24 || len(g.resources.vehicles[2]) != 24 || len(g.resources.levelChunks) != 15 {
		t.Fatalf("sprite counts player=%d enemy=%d effects=%d", len(g.resources.players[0]), len(g.resources.enemies), len(g.resources.effects))
	}
	g.startGame(true)
	for range 200 {
		g.updatePlay()
	}
	if g.player.y >= levelHeight || g.cameraY >= levelHeight {
		t.Fatal("automatic player left the recovered level")
	}
}

func TestAutomaticDemoCompletesRecoveredLevel(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGame(true)
	minimumY := g.player.y
	for tick := 0; tick < 20000 && g.scene != victoryScene && g.scene != titleScene; tick++ {
		g.tick++
		g.updatePlay()
		minimumY = min(minimumY, g.player.y)
	}
	// The original enters its message during the final map-strip request,
	// with the player still around Y124, rather than walking to the top edge.
	if g.scene != victoryScene || g.cameraY != 30 || !g.ending.active || g.score < 100000 {
		t.Fatalf("automatic demo stopped at minimumY%.1f in scene%d, score%d; player%.0f/%.0f lives%d ammo%d grenades%d tank%v fuel%d target%+v path%d/%d", minimumY, g.scene, g.score, g.player.x, g.player.y, g.lives, g.ammunition, g.grenades, g.player.inTank, g.player.tankFuel, g.demoTarget, g.demoPathIndex, len(g.demoPath))
	}
	t.Logf("ordinary demonstration reached the ending after%d ticks with%d reserve lives, score%d", g.gameplayTicks, g.lives, g.score)
}

func TestTwoPlayerSessionKeepsIndependentState(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGamePlayers(false, 2)
	if g.playerCount != 2 || g.player.x == g.player2.x || g.ammunition2 != 99 || g.grenades2 != 50 {
		t.Fatal("two-player state was not initialized independently")
	}
	previousHealth := g.player2.health
	g.hitPlayer(1)
	if g.player2.health != previousHealth-1 || g.player.health != 1 || g.player2.deadTicks != 24 {
		t.Fatal("player two damage changed the wrong commando")
	}
	g.enemies = []actor{{x: g.player2.x, y: g.player2.y - 4, health: 1}}
	g.projectiles = []projectile{{x: g.enemies[0].x, y: g.enemies[0].y, owner: 1, life: 2}}
	g.updateProjectiles()
	g.updateEnemies()
	if g.score2 != 100 || g.score != 0 {
		t.Fatalf("scores player1=%d player2=%d", g.score, g.score2)
	}
}

func TestPickupsAndDestructibleTerrainUseRecoveredObjects(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startGame(false)
	g.ammunition, g.grenades = 0, 0
	g.pickups = []pickup{{x: g.player.x, y: g.player.y, kind: 0}, {x: g.player.x, y: g.player.y, kind: 1}}
	g.updatePickups()
	if g.ammunition != 0 || g.grenades != 99 || len(g.pickups) != 1 {
		t.Fatal("pickup collection did not stop after the first original object slot")
	}
	g.updatePickups()
	if g.ammunition != 99 || g.grenades != 99 || g.score != 200 || len(g.pickups) != 0 {
		t.Fatalf("pickup ammo=%d grenades=%d score=%d remaining=%d", g.ammunition, g.grenades, g.score, len(g.pickups))
	}
	g.enemies = []actor{{x: g.player.x, y: g.player.y, kind: 4, slot: 19, health: 1, tankFuel: 64}}
	if !g.useTank(0) || !g.player.inTank || g.player.health != 1 || g.player.tankFuel != 64 {
		t.Fatal("the original tank context action did not board the vehicle")
	}
	for _, kind := range []int{2, 4, 6, 7} {
		g.collectPickup(0, kind)
	}
	if g.rangeBonus[0] != 20 || g.bulletPower[0] != 2 || g.grenadePower[0] != 2 {
		t.Fatal("weapon upgrades did not preserve their individual effects")
	}
	// The first bunker spans five columns and three rows. A shot anywhere
	// in it clears the entire object, exactly as observed in original play.
	if !g.damageTerrain(5*16+8, 927*16-7, 0) {
		t.Fatal("the first bunker did not break")
	}
	for row := 926; row <= 928; row++ {
		for column := 3; column <= 7; column++ {
			if !g.destroyed[row][column] || g.displayTile(row, column) != 0 || g.solidAt(float64(column*16+8), float64(row*16-7)) {
				t.Fatalf("bunker tile %d,%d did not become original ground code zero", row, column)
			}
		}
	}
	if g.damageTerrain(5*16+8, 927*16-7, 0) {
		t.Fatal("the same bunker was destroyed twice")
	}
}

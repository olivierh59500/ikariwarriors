//go:build rendercheck

package game

import (
	"github.com/hajimehoshi/ebiten/v2"
	"testing"
"os"
"encoding/json"
)

// This is one ordinary transition from the independently played original:
// the preceding complete frame, actual input, entry state and complete message.
func TestNormallyPlayedOriginalVictoryTransition(t *testing.T) {
	var source struct {
		Before                                                        originalPlayedScene
		UpperStep, UpperDelay, LowerStep, LowerDelay, BeforeIdleTicks int
		Input                                                         struct {
			InputDirection int
			InputFire      bool
		}
		ExpectedCamera, ExpectedX, ExpectedY, ExpectedUpper, ExpectedLower, ExpectedScore, ExpectedEndingTicks int
		Pixels                                                                                                 []byte
		BeforeDirectionLatch                                                                                   uint8
		BeforeDirectionSourceSHA256                                                                            string
		SourceSHA256                                                                                           string
	}
	readOriginalPaintFixture(t, "testdata/victory-transition.json.gz", &source)
	g := observedPlayedSceneGame(t, source.Before)
    // The isolated scene importer must also restore the original ending's
    // existing contact mark. Source piece9 was already touched before this
    // input; omitting it would award that same harmless contact twice.
    var initialHits struct { Version int; SourceSHA256 string; HitFlags []byte; HitParts uint16 }
    initialData,err:=os.ReadFile("testdata/original-victory-initial-hits.json")
    if err!=nil {t.Fatal(err)};if err:=json.Unmarshal(initialData,&initialHits);err!=nil {t.Fatal(err)}
    if initialHits.Version!=1 ||initialHits.SourceSHA256!=source.Before.SourceSHA256 ||len(initialHits.HitFlags)!=12 {t.Fatal("incomplete original pre-transition hit state")}
    var mask uint16
    for index,flag:=range initialHits.HitFlags {if flag!=0 {mask|=1<<index}}
    if mask!=initialHits.HitParts {t.Fatal("original ending hit flags differ")}
    g.ending.hitParts=mask
	g.eventIndex = len(g.resources.events)
	g.randomSpawn, g.redSpawn = false, false
	g.secondaryTerrain = true
	g.player.upper.step, g.player.upper.delay = source.UpperStep, source.UpperDelay
	g.player.lower.step, g.player.lower.delay = source.LowerStep, source.LowerDelay
	g.idleArtilleryTicks = uint32(source.BeforeIdleTicks)
	g.cooperativeDirection = source.BeforeDirectionLatch
	beforeRandom := g.random
	screen := ebiten.NewImage(ScreenWidth, ScreenHeight)
	check := &originalSmallDrawCheck{g: g, target: screen, background: make([]byte, ScreenWidth*ScreenHeight*4), expected: source.Pixels}
	check.draw = func() {
		control := playerControls{fire: source.Input.InputFire}
		if source.Input.InputDirection&0x10 != 0 {
			control.y = -1
		}
		g.stepPlay([2]playerControls{control})
		if g.scene != victoryScene || g.cameraY != float64(source.ExpectedCamera) || g.player.x != float64(source.ExpectedX) || g.player.y != float64(source.ExpectedY) || g.score != source.ExpectedScore || g.ending.ticks != source.ExpectedEndingTicks || g.random != beforeRandom {
			t.Errorf("original transition differs: scene%d camera%.0f hero%.0f/%.0f score%d phase%d", g.scene, g.cameraY, g.player.x, g.player.y, g.score, g.ending.ticks)
		}
		if 0x42b88+g.player.upper.frame*actorBobBytes != source.ExpectedUpper || 0x46288+g.player.lower.frame*overlayBobBytes != source.ExpectedLower {
			t.Error("original entry pose differs")
		}
		g.drawVictory(screen)
	}
	smallDrawScenes <- check
	if err := <-smallDrawResults; err != nil {
		t.Fatal(err)
	}
	// The original keeps the same picture and physical state through its delay.
	tick, random, hero, phase, score := g.gameplayTicks, g.random, g.player, g.ending.ticks, g.score
	for n := 1; n < victoryMessageTicks; n++ {
		if err := g.Update(); err != nil {
			t.Fatal(err)
		}
		if g.scene != victoryScene || g.gameplayTicks != tick || g.random != random || g.player != hero || g.ending.ticks != phase || g.score != score {
			t.Fatal("victory wait advanced gameplay")
		}
	}
	if err := g.Update(); err != nil {
		t.Fatal(err)
	}
	if g.scene != titleScene || g.score != score {
		t.Fatal("victory message did not exit with its single bonus")
	}
}

package game

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

// The original CPU delay keeps this message visible for about 9.6 seconds in
// the reference recording. Use the native game clock for the same presentation.
const victoryMessageTicks = TicksPerSecond * 96 / 10

// The original victory message overlays the last gameplay image. Its bonus
// changes the score while the HUD underneath keeps its previous value.
func (g *Game) enterVictory() {
	if g.scene == victoryScene {
		return
	}
	g.ending.displayScores = [2]int{g.score, g.score2}
	if g.lives >= 0 {
		g.score += 100000
	}
	if g.playerCount > 1 && g.lives2 >= 0 {
		g.score2 += 100000
	}
	g.scene, g.sceneTick = victoryScene, 0
}

func (g *Game) drawVictoryMessage(screen *ebiten.Image) {
	for _, line := range [...]struct {
		x, y int
		text string
	}{
		{104, 48, "CONGRATULATIONS"},
		{72, 64, "YOU HAVE RESCUED COLONEL"},
		{96, 80, "COOK AND RECEIVED"},
		{112, 96, "100000 POINTS"},
		{88, 112, "CONTINUE TO FIGHT HARD"},
		{96, 128, "AND BRING FREEDOM TO"},
		{88, 144, "THE PEOPLE YOU PROTECT"},
		{128, 160, "GOOD LUCK"},
		{104, 176, "AND GOOD HUNTING"},
	} {
		for index, character := range line.text {
			glyph := int(character) - 0x2f
			if character == ' ' {
				glyph = 0
			}
			source := g.resources.font.SubImage(image.Rect(glyph*8, 0, glyph*8+8, 8)).(*ebiten.Image)
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(line.x+index*8), float64(line.y))
			screen.DrawImage(source, op)
		}
	}
}

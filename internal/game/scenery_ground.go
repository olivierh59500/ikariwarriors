package game

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

// A destruction paints original ground graphics before later actor passes.
// Keep its exact origin and order so odd impacts preserve both borders and
// adjacent clears can overlap without replacing the logical collision grid.
type sceneryGroundPatch struct {
	row, column, rows, columns int
	yOffset                    int
	clipTop                    int
}

func (g *Game) drawDestroyedTerrain(cameraY float64) {
	first := max(0, int(cameraY)/tileSize-1)
	last := min(mapRows-1, int(cameraY+playfieldHeight)/tileSize+1)
	var covered [playfieldHeight/tileSize + 4][mapColumns]bool
	for _, p := range g.sceneryPaint {
		for row := max(first, p.row); row <= min(last, p.row+p.rows-1); row++ {
			for column := max(0, p.column); column < min(mapColumns, p.column+p.columns); column++ {
				if row*tileSize >= p.clipTop {
					covered[row-first][column] = true
				}
			}
		}
	}
	// Cells outside the painted strip enter later from the cleared map.
	// Only retained brush pixels replace their original terrain background.
	for row := first; row <= last; row++ {
		bank := 0
		if row < secondaryMapRows {
			bank = 1
		}
		for column := range mapColumns {
			if !g.destroyed[row][column] || covered[row-first][column] {
				continue
			}
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(column*tileSize), float64(row*tileSize)-cameraY)
			g.playfield.DrawImage(g.resources.tiles[bank].images[0], op)
		}
	}
	g.drawSceneryGround(cameraY)
}

func (g *Game) drawSceneryGround(cameraY float64) {
	for _, p := range g.sceneryPaint {
		top := p.row*tileSize + p.yOffset
		if float64(top+p.rows*tileSize) <= cameraY || float64(top) >= cameraY+playfieldHeight {
			continue
		}
		for row := 0; row < p.rows; row++ {
			worldY := top + row*tileSize
			cut := max(0, p.clipTop-worldY)
			if cut >= tileSize {
				continue
			}
			y := float64(worldY+cut) - cameraY
			if y+float64(tileSize-cut) <= 0 || y >= playfieldHeight {
				continue
			}
			for column := 0; column < p.columns; column++ {
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Translate(float64((p.column+column)*tileSize), y)
				brush := g.resources.effects[0]
				if cut != 0 {
					brush = brush.SubImage(image.Rect(0, cut, tileSize, tileSize)).(*ebiten.Image)
				}
				g.playfield.DrawImage(brush, op)
			}
		}
	}
}

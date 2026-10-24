package game

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

// The original uses the full sixteen-pixel '?' font image for the green
// gauge and the first row of '@' for its red overlay. Ordinary text uses
// only the left eight pixels; both images are already in the recovered font.
func decodeFuelGauge(data []byte, palette color.Palette) (*ebiten.Image, *ebiten.Image) {
	decode := func(glyph, height int) *image.NRGBA {
		result := image.NewNRGBA(image.Rect(0, 0, 16, height))
		for y := 0; y < height; y++ {
			for x := 0; x < 16; x++ {
				index := byte(0)
				for plane := 0; plane < 4; plane++ {
					index |= ((data[glyph*64+plane*16+y*2+x/8] >> uint(7-x%8)) & 1) << plane
				}
				if index != 0 {
					result.Set(x, y, palette[index])
				}
			}
		}
		return result
	}
	tile, line := decode(16, 8), decode(17, 1)
	gauge := image.NewNRGBA(image.Rect(0, 0, 16, 64))
	for y := 0; y < 64; y++ {
		copy(gauge.Pix[y*gauge.Stride:y*gauge.Stride+16*4], tile.Pix[(y%8)*tile.Stride:(y%8)*tile.Stride+16*4])
	}
	return ebiten.NewImageFromImage(gauge), ebiten.NewImageFromImage(line)
}

func (g *Game) drawFuelGauges(screen *ebiten.Image) {
	for player, value := range []*actor{&g.player, &g.player2} {
		if !value.inTank {
			continue
		}
		x := float64(32 + player*256)
		base := &ebiten.DrawImageOptions{}
		base.GeoM.Translate(x, 80)
		screen.DrawImage(g.resources.fuelGauge, base)

		// $c1f4/$c382 refill with the complete green bitmap. The ordinary
		// depletion overlay returns only when fuel is next consumed.
		if value.fuelGaugeFull {
			continue
		}

		// $8b08/$8ba8 start the burnout at one without repainting the gauge;
		// the preceding two-unit indication remains. $c912 paints 66-fuel
		// red rows, including the original two-row margin at a full tank.
		fuel := value.tankFuel
		if fuel == 1 {
			fuel = 2
		}
		empty := max(0, 66-int(int8(uint8(fuel))))
		if empty == 0 {
			continue
		}
		overlay := &ebiten.DrawImageOptions{}
		overlay.GeoM.Scale(1, float64(empty))
		overlay.GeoM.Translate(x, 80)
		screen.DrawImage(g.resources.fuelEmpty, overlay)
	}
}

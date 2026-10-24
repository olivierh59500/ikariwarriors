// Ikari Warriors runs the native Go game with Ebitengine presentation.
package main

import (
	"flag"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/olivierh59500/ikariwarriors/internal/game"
)

func main() {
	fullscreen := flag.Bool("fullscreen", false, "start in fullscreen mode")
	touch := flag.Bool("touch", false, "preview the Android touch layout")
	demo := flag.Bool("demo", false, "start the automatic demonstration")
	mute := flag.Bool("mute", false, "disable sound")
	scale := flag.Int("scale", 3, "initial window scale from 1 to 8")
	smoke := flag.Int("smoke", 0, "run the selected scene for this many ticks")
	capture := flag.String("capture", "", "write the final smoke frame as PNG")
	flag.Parse()
	if *scale < 1 || *scale > 8 || *smoke < 0 || *capture != "" && *smoke == 0 {
		log.Fatal("invalid scale or smoke capture options")
	}
	if *smoke > 0 {
		*demo = true
		*mute = true
		ebiten.SetRunnableOnUnfocused(true)
		ebiten.SetWindowVisible(false)
	}
	application, err := game.New(game.Options{Touch: *touch, Demo: *demo, Mute: *mute, Smoke: *smoke, Capture: *capture})
	if err != nil {
		log.Fatal(err)
	}
	width := game.ScreenWidth
	if *touch {
		width = game.TouchWidth
	}
	ticksPerSecond := game.TicksPerSecond
	if *smoke > 0 {
		ticksPerSecond = 6000
	}
	ebiten.SetTPS(ticksPerSecond)
	ebiten.SetScreenFilterEnabled(false)
	ebiten.SetWindowTitle("Ikari Warriors — Go / Ebitengine")
	ebiten.SetWindowSize(width**scale, game.ScreenHeight**scale)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetFullscreen(*fullscreen)
	if err = ebiten.RunGame(application); err != nil {
		log.Fatal(err)
	}
}

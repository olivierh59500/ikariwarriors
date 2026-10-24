//go:build rendercheck

package game

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

var smallDrawScenes = make(chan *originalSmallDrawCheck)
var smallDrawResults = make(chan error)
var smallDrawFinished = make(chan int, 1)

// Cocoa must run on the process main thread. Tests prepare scenes on their
// ordinary worker goroutine; the actual Ebitengine painter stays on main.
func TestMain(m *testing.M) {
	ebiten.SetWindowVisible(false)
	ebiten.SetWindowSize(640, 400)
	ebiten.SetWindowTitle("Ikari Warriors sprite verification")
	go func() { smallDrawFinished <- m.Run() }()
	runner := &originalSmallDrawRunner{}
	if err := ebiten.RunGame(runner); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(runner.exitCode)
}

type originalSmallDrawRunner struct {
	scene    *originalSmallDrawCheck
	exitCode int
}

func (r *originalSmallDrawRunner) Update() error {
	select {
	case r.exitCode = <-smallDrawFinished:
		return ebiten.Termination
	case r.scene = <-smallDrawScenes:
	default:
	}
	return nil
}
func (r *originalSmallDrawRunner) Layout(int, int) (int, int) { return 320, 200 }
func (r *originalSmallDrawRunner) Draw(dst *ebiten.Image) {
	if r.scene == nil {
		return
	}
	r.scene.Draw(dst)
	smallDrawResults <- r.scene.err
	r.scene = nil
}

type originalSmallDrawCheck struct {
	g                    *Game
	target               *ebiten.Image
	background, expected []byte
	ignore               []bool
	draw                 func()
	done                 bool
	err                  error
}

func (c *originalSmallDrawCheck) Update() error {
	if c.done {
		return ebiten.Termination
	}
	return nil
}
func (c *originalSmallDrawCheck) Layout(int, int) (int, int) { return 320, 200 }
func (c *originalSmallDrawCheck) Draw(*ebiten.Image) {
	if c.done {
		return
	}
	c.done = true
	target := c.g.playfield
	if c.target != nil {
		target = c.target
	}
	target.WritePixels(c.background)
	if c.draw != nil {
		c.draw()
	} else {
		c.g.drawSmallObjects()
	}
	actual := make([]byte, len(c.background))
	target.ReadPixels(actual)
	different, first := 0, -1
	for i, index := range c.expected {
		if c.ignore != nil && c.ignore[i] {
			continue
		}
		r, g, b, _ := c.g.resources.colors[index].RGBA()
		if actual[i*4] != byte(r>>8) || actual[i*4+1] != byte(g>>8) || actual[i*4+2] != byte(b>>8) {
			different++
			if first < 0 {
				first = i
			}
		}
	}
	if different != 0 {
		c.err = fmt.Errorf("%d pixels differ from the original painter; first at %d/%d", different, first%target.Bounds().Dx(), first/target.Bounds().Dx())
	}
}

// Run this graphics test separately with -tags rendercheck. The golden scene
// comes from the original $1224a painter, including its unchanged background,
// masks and overlapping shots, three quad pools, a fragment, pickup and mine.
func TestOriginalSmallBobCompositing(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile("testdata/small-before.bin")
	if err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile("testdata/small-after.bin")
	if err != nil {
		t.Fatal(err)
	}
	if len(before) != playfieldWidth*playfieldHeight || len(after) != len(before) {
		t.Fatal("invalid original painter fixture")
	}
	data, err := os.ReadFile("testdata/small-scene.json")
	if err != nil {
		t.Fatal(err)
	}
	var scene []struct {
		Slot, X, Y int
		Bob        string
	}
	if err = json.Unmarshal(data, &scene); err != nil {
		t.Fatal(err)
	}
	for i := range g.burst.parts {
		g.burst.parts[i] = navPoint{-1000, -1000}
	}
	for _, part := range scene {
		bob, err := strconv.ParseInt(part.Bob, 0, 32)
		if err != nil {
			t.Fatal(err)
		}
		frame := (int(bob) - 0x4a2e8) / 160
		switch {
		case part.Slot >= 41 && part.Slot <= 48 || part.Slot >= 77 && part.Slot <= 80:
			first, pool := 41, 0
			if part.Slot >= 77 {
				first, pool = 77, 2
			} else if part.Slot >= 45 {
				first, pool = 45, 1
			}
			if part.Slot != first {
				continue
			}
			owner := pool
			if owner == 2 {
				owner = -1
			}
			g.blasts[pool] = projectile{x: float64(part.X - 48), y: float64(part.Y + 8), owner: owner, enemy: pool == 2, grenade: true, impact: 4 - (frame-49)/4}
		case part.Slot == 49:
			g.burst.active = true
			g.burst.parts[0] = navPoint{float64(part.X - 56), float64(part.Y)}
		case part.Slot == 57:
			g.pickups = append(g.pickups, pickup{x: float64(part.X - 56), y: float64(part.Y), kind: 0})
		case part.Slot == 81:
			g.mines = append(g.mines, landmine{x: float64(part.X - 56), y: float64(part.Y), kind: 19, slot: 81})
			// Retain the image observed in this controlled source painter slot.
			g.mineAnimations[0].frame = frame
		default:
			enemy := part.Slot >= 65
			owner, slot := 0, part.Slot-34
			if part.Slot >= 37 && !enemy {
				owner, slot = 1, part.Slot-37
			}
			if enemy {
				owner, slot = -1, part.Slot-65
			}
			g.projectiles = append(g.projectiles, projectile{x: float64(part.X - 56), y: float64(part.Y), owner: owner, slot: slot, enemy: enemy, frame: frame})
		}
	}
	rgba := make([]byte, len(before)*4)
	for i, index := range before {
		r, g, b, _ := g.resources.colors[index].RGBA()
		rgba[i*4], rgba[i*4+1], rgba[i*4+2], rgba[i*4+3] = byte(r>>8), byte(g>>8), byte(b>>8), 255
	}
	check := &originalSmallDrawCheck{g: g, background: rgba, expected: after}
	smallDrawScenes <- check
	if err := <-smallDrawResults; err != nil {
		t.Fatal(err)
	}
}

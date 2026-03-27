// Package game provides the Ebitengine wrapper for human play.
// All game logic lives in the engine package.
package game

import (
	"image/color"

	"manicminer/engine"
	"manicminer/input"
	"manicminer/screen"

	"github.com/hajimehoshi/ebiten/v2"
)

// Game implements ebiten.Game as a thin wrapper around engine.GameEnv.
type Game struct {
	env         *engine.GameEnv
	renderer    *screen.Renderer
	display     *ebiten.Image
	accumulator float64
	paused      bool
	lastObs     engine.Observation
}

// New creates a new Game instance for human play.
func New() *Game {
	env := engine.NewGameEnv()
	g := &Game{
		env:      env,
		renderer: screen.NewRenderer(),
		display:  ebiten.NewImage(ScreenWidth, ScreenHeight),
		lastObs:  env.GetObservation(),
	}
	return g
}

// Update is called every tick (60 FPS by Ebitengine).
func (g *Game) Update() error {
	g.accumulator += 1.0 / 60.0
	for g.accumulator >= LogicFrameTime {
		g.logicTick()
		g.accumulator -= LogicFrameTime
	}
	return nil
}

func (g *Game) logicTick() {
	inp := input.Read()

	// Pause (UI concern — not sent to engine).
	if inp.Pause && !g.paused {
		g.paused = true
		return
	}
	if g.paused {
		if inp.Left || inp.Right || inp.Jump || inp.MusicToggle {
			g.paused = false
		}
		return
	}

	// Quit (UI concern — restart cavern).
	if inp.Quit {
		g.lastObs = g.env.Reset(g.env.CavernNumber)
		return
	}

	// Convert keyboard state to pure Action and step the engine.
	result := g.env.Step(inp.ToAction())
	g.lastObs = result.Obs
}

// Draw renders the current frame.
func (g *Game) Draw(scr *ebiten.Image) {
	scr.Fill(color.Black)

	// Render cavern area from the observation's buffers.
	g.renderer.RenderBuffer(g.display, g.lastObs.Attrs[:], g.lastObs.Pixels[:])

	// Render HUD (bottom 64 pixels).
	g.renderHUD()

	scr.DrawImage(g.display, &ebiten.DrawImageOptions{})
}

func (g *Game) renderHUD() {
	var hudAttr byte // Black background.

	screen.PrintMessage(g.display, 0, 128, g.lastObs.CavernName, hudAttr)
	screen.PrintMessage(g.display, 0, 136, "AIR", hudAttr)
	g.drawAirBar()

	highScoreText := "High Score " + string(g.env.HighScore[:]) + "   Score " + string(g.lastObs.Score[:])
	screen.PrintMessage(g.display, 0, 152, highScoreText, hudAttr)
}

func (g *Game) drawAirBar() {
	airLength := g.lastObs.Air - 0x24
	if airLength < 0 {
		airLength = 0
	}

	startX := 4 * 8
	green := color.RGBA{0, 215, 0, 255}

	for row := 0; row < 4; row++ {
		for cell := 0; cell < airLength; cell++ {
			for bit := 0; bit < 8; bit++ {
				x := startX + cell*8 + bit
				y := 136 + row
				if x < ScreenWidth {
					g.display.Set(x, y, green)
				}
			}
		}
	}
}

// Layout returns the logical screen size.
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return ScreenWidth, ScreenHeight
}

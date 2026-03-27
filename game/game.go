// Package game provides the Ebitengine wrapper for human play.
// All game logic lives in the engine package.
package game

import (
	"image/color"

	"manicminer/audio"
	"manicminer/data"
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
	audioPlayer *audio.Player
	accumulator float64
	paused      bool
	lastObs     engine.Observation
	cheat       CheatState
}

// New creates a new Game instance for human play.
func New() *Game {
	env := engine.NewGameEnv()
	g := &Game{
		env:         env,
		renderer:    screen.NewRenderer(),
		display:     ebiten.NewImage(ScreenWidth, ScreenHeight),
		audioPlayer: audio.NewPlayer(),
		lastObs:     env.GetObservation(),
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

	// Pause handling (UI-only, not sent to engine).
	if g.env.State == engine.StatePlaying {
		if inp.Pause && !g.paused {
			g.paused = true
			g.audioPlayer.Silence()
			return
		}
		if g.paused {
			if inp.Left || inp.Right || inp.Jump || inp.MusicToggle {
				g.paused = false
			}
			return
		}
		// Quit: restart cavern.
		if inp.Quit {
			g.lastObs = g.env.Reset(g.env.CavernNumber)
			return
		}
	}

	// Check cheat code (6031769).
	g.cheat.Update()

	// Check teleport (cheat mode + 6 held + 1-5).
	if g.env.State == engine.StatePlaying {
		if dest := g.cheat.CheckTeleport(); dest >= 0 {
			g.lastObs = g.env.Reset(dest)
			return
		}
	}

	result := g.env.Step(inp.ToAction())
	g.lastObs = result.Obs

	// Handle audio based on engine state.
	g.updateAudio()
}

// updateAudio manages sound based on game state.
func (g *Game) updateAudio() {
	switch g.env.State {
	case engine.StateTitle:
		if g.env.TitlePhase == 0 {
			// Start the title tune if not already playing.
			if !g.audioPlayer.IsTunePlaying() && g.env.TitleFrame <= 1 {
				g.audioPlayer.PlayTune(data.TitleTuneData[:])
			}
			// Sync the engine's TuneNoteIndex from the audio player
			// (for piano key animation).
			g.env.TuneNoteIndex = g.audioPlayer.TuneNoteIndex()
			// Check if tune finished — advance to banner phase.
			if !g.audioPlayer.IsTunePlaying() && g.env.TitleFrame > 1 {
				g.env.TitlePhase = 1
				g.env.BannerOffset = 0
			}
		} else {
			g.audioPlayer.Silence()
		}

	case engine.StatePlaying:
		// Stop the title tune if it was still playing.
		if g.audioPlayer.IsTunePlaying() {
			g.audioPlayer.Silence()
		}
		if g.env.MusicEnabled {
			noteIdx := g.env.MusicNoteIndex & 63
			freq := data.InGameTuneData[noteIdx]
			g.audioPlayer.PlayInGameNote(freq)
		} else {
			g.audioPlayer.Silence()
		}

	default:
		g.audioPlayer.Silence()
	}
}

// Draw renders the current frame.
func (g *Game) Draw(scr *ebiten.Image) {
	scr.Fill(color.Black)

	switch g.env.State {
	case engine.StateTitle:
		g.drawTitle()
	case engine.StatePlaying, engine.StateDemo:
		g.drawPlaying()
	case engine.StateDying, engine.StateNextCavern:
		g.drawPlaying()
	case engine.StateGameOver:
		g.drawGameOver()
	}

	scr.DrawImage(g.display, &ebiten.DrawImageOptions{})
}

func (g *Game) drawTitle() {
	g.renderer.RenderBuffer(g.display, g.lastObs.Attrs[:], g.lastObs.Pixels[:])

	if g.env.TitlePhase == 1 {
		// Draw the scrolling banner at row 19 (y=152).
		bannerStart := g.env.BannerOffset
		var bannerText [32]byte
		for i := 0; i < 32; i++ {
			idx := bannerStart + i
			if idx >= 0 && idx < len(data.TitleScreenBanner) {
				bannerText[i] = data.TitleScreenBanner[idx]
			} else {
				bannerText[i] = ' '
			}
		}
		screen.PrintMessage(g.display, 0, 152, string(bannerText[:]), 0)
	}
}

func (g *Game) drawPlaying() {
	g.renderer.RenderBuffer(g.display, g.lastObs.Attrs[:], g.lastObs.Pixels[:])
	g.renderHUD()
}

func (g *Game) drawGameOver() {
	g.display.Fill(color.Black)
	screen.PrintMessage(g.display, 10*8, 6*8, "Game", 0)
	screen.PrintMessage(g.display, 18*8, 6*8, "Over", 0)

	highScoreText := "High Score " + string(g.env.HighScore[:]) + "   Score " + string(g.lastObs.Score[:])
	screen.PrintMessage(g.display, 0, 152, highScoreText, 0)
}

func (g *Game) renderHUD() {
	var hudAttr byte
	// Row 16 (y=128): Cavern name.
	screen.PrintMessage(g.display, 0, 128, g.lastObs.CavernName, hudAttr)

	// Row 17 (y=136): "AIR" + air bar.
	screen.PrintMessage(g.display, 0, 136, "AIR", hudAttr)
	g.drawAirBar()

	// Row 19 (y=152): High score and score.
	highScoreText := "High Score " + string(g.env.HighScore[:]) + "   Score " + string(g.lastObs.Score[:])
	screen.PrintMessage(g.display, 0, 152, highScoreText, hudAttr)

	// Row 20-21 (y=160-175): Lives display (small Willy sprites).
	g.drawLives()
}

func (g *Game) drawAirBar() {
	airLength := g.lastObs.Air - 0x24
	if airLength < 0 {
		airLength = 0
	}

	// The air bar spans from column 4 (after "AIR ") to column 31.
	// Total bar width = 27 cells. Remaining air fills from left, depleted area is red.
	startX := 4 * 8 // Pixel x = 32.
	barWidthCells := 27

	red := color.RGBA{215, 0, 0, 255}
	green := color.RGBA{0, 215, 0, 255}

	for row := 0; row < 4; row++ {
		for cell := 0; cell < barWidthCells; cell++ {
			var c color.RGBA
			if cell < airLength {
				c = green // Remaining air.
			} else {
				c = red // Depleted air.
			}
			for bit := 0; bit < 8; bit++ {
				x := startX + cell*8 + bit
				y := 136 + row
				if x < ScreenWidth {
					g.display.Set(x, y, c)
				}
			}
		}
	}
}

func (g *Game) drawLives() {
	lives := g.env.Lives
	if lives <= 0 {
		return
	}
	// Draw small Willy sprites at the bottom of the screen.
	// In the original, lives are drawn at row 20 (y=160), 2 cells apart.
	// We use the current music note index to pick the animation frame.
	animIdx := (g.env.MusicNoteIndex >> 2) & 3
	spriteData := data.WillySprites[animIdx]

	for i := 0; i < lives && i < 8; i++ {
		px := i * 16 // Each Willy is 16 pixels wide.
		// Draw directly to the display image.
		for row := 0; row < 16; row++ {
			leftByte := spriteData[row*2]
			rightByte := spriteData[row*2+1]
			for bit := 7; bit >= 0; bit-- {
				if leftByte&(1<<uint(bit)) != 0 {
					x := px + (7 - bit)
					y := 160 + row
					if x < ScreenWidth && y < ScreenHeight {
						g.display.Set(x, y, color.RGBA{215, 215, 0, 255}) // Yellow.
					}
				}
			}
			for bit := 7; bit >= 0; bit-- {
				if rightByte&(1<<uint(bit)) != 0 {
					x := px + 8 + (7 - bit)
					y := 160 + row
					if x < ScreenWidth && y < ScreenHeight {
						g.display.Set(x, y, color.RGBA{215, 215, 0, 255})
					}
				}
			}
		}
	}
}

// Layout returns the logical screen size.
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return ScreenWidth, ScreenHeight
}

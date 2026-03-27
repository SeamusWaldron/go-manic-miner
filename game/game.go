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

// updateAudio plays the appropriate sound for the current engine state.
func (g *Game) updateAudio() {
	switch g.env.State {
	case engine.StateTitle:
		if g.env.TitlePhase == 0 {
			// Piano phase: play the current tune note.
			tuneData := data.TitleTuneData[:]
			noteOffset := g.env.TuneNoteIndex * 3
			if noteOffset+2 < len(tuneData) && tuneData[noteOffset] != 0xFF {
				freq1 := tuneData[noteOffset+1]
				freq2 := tuneData[noteOffset+2]
				g.audioPlayer.PlayNote(freq1, freq2)
			} else {
				g.audioPlayer.Silence()
			}
		} else {
			g.audioPlayer.Silence()
		}

	case engine.StatePlaying:
		// In-game music: play current note from Mountain King.
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

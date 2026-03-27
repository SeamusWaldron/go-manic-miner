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
	// All positions decoded from the original display file addresses:
	// $5000 = charRow 16, y=128: Cavern name
	// $5020 = charRow 17, y=136: "AIR" text
	// $5224 = charRow 17, pixRow 2, col 4, y=138: Air bar (4 pixel rows)
	// $5060 = charRow 19, y=152: "High Score ... Score ..."
	// $50A0 = charRow 21, y=168: Lives

	// Cavern name — yellow text on black.
	screen.PrintMessage(g.display, 0, 128, g.lastObs.CavernName, 0x06)

	// "AIR" text at (17,0) — white on red.
	screen.PrintMessage(g.display, 0, 136, "AIR", 0x17)

	// Air bar at (17, pixRow 2, col 4) — y=138, 4 pixel rows high.
	g.drawAirBar()

	// "High Score ... Score ..." at (19,0) — yellow on black.
	highScoreText := "High Score " + string(g.env.HighScore[:]) + "   Score " + string(g.lastObs.Score[:])
	screen.PrintMessage(g.display, 0, 152, highScoreText, 0x06)

	// Lives at (21,0) — y=168.
	g.drawLives()
}

func (g *Game) drawAirBar() {
	airLength := g.lastObs.Air - 0x24
	if airLength < 0 {
		airLength = 0
	}

	// The original draws the air bar at display address $52xx-$55xx:
	// MSB $52 = charRow 17, pixel row 2 → screen y=138
	// MSB $53 = pixel row 3 → y=139
	// MSB $54 = pixel row 4 → y=140
	// MSB $55 = pixel row 5 → y=141
	// Starting at column $24 & $1F = 4, spanning airLength cells.
	// Pixels are $FF (white), with green/red background from attributes.

	startX := 4 * 8 // Column 4 = pixel x=32.
	barY := 138      // Pixel row 2 of char row 17.
	barWidthCells := 28

	green := color.RGBA{0, 215, 0, 255}
	red := color.RGBA{215, 0, 0, 255}
	white := color.RGBA{215, 215, 215, 255}

	// Draw 4 pixel rows of the bar.
	for row := 0; row < 4; row++ {
		for cell := 0; cell < barWidthCells; cell++ {
			for bit := 0; bit < 8; bit++ {
				x := startX + cell*8 + bit
				y := barY + row
				if x >= ScreenWidth {
					continue
				}
				if cell < airLength {
					// Remaining air: white fill ($FF pixels, INK colour).
					g.display.Set(x, y, white)
				} else {
					// Depleted: red background (PAPER colour, $00 pixels).
					g.display.Set(x, y, red)
				}
			}
		}
	}

	// Fill the background areas above and below the bar within the char row
	// (pixel rows 0-1 and 6-7 of char row 17) with green for remaining, red for depleted.
	for _, rowSet := range [][2]int{{136, 138}, {142, 144}} {
		for y := rowSet[0]; y < rowSet[1]; y++ {
			for cell := 0; cell < barWidthCells; cell++ {
				var c color.RGBA
				if cell < airLength {
					c = green
				} else {
					c = red
				}
				for bit := 0; bit < 8; bit++ {
					x := startX + cell*8 + bit
					if x < ScreenWidth {
						g.display.Set(x, y, c)
					}
				}
			}
		}
	}
}

func (g *Game) drawLives() {
	// Clear the lives area (y=168-183) every frame.
	for y := 168; y < 184; y++ {
		for x := 0; x < ScreenWidth; x++ {
			g.display.Set(x, y, color.Black)
		}
	}

	lives := g.env.Lives
	if lives <= 0 {
		return
	}

	// Original: (MusicNoteIndex RLCA x3) AND $60 gives 0, 32, 64, or 96.
	// Dividing by 32 gives frame 0-3.
	animIdx := ((g.env.MusicNoteIndex << 3) & 0x60) >> 5
	spriteData := data.WillySprites[animIdx]

	cyan := color.RGBA{0, 215, 215, 255}

	for i := 0; i < lives && i < 8; i++ {
		// Original: lives start at $50A0, each 2 cells apart (INC HL twice).
		// $50A0 = row 20, column 0 in bottom third. Each life shifts 2 columns right.
		px := i * 16

		for row := 0; row < 16; row++ {
			leftByte := spriteData[row*2]
			rightByte := spriteData[row*2+1]
			for bit := 7; bit >= 0; bit-- {
				if leftByte&(1<<uint(bit)) != 0 {
					x := px + (7 - bit)
					y := 168 + row
					if x < ScreenWidth && y < ScreenHeight {
						g.display.Set(x, y, cyan)
					}
				}
			}
			for bit := 7; bit >= 0; bit-- {
				if rightByte&(1<<uint(bit)) != 0 {
					x := px + 8 + (7 - bit)
					y := 168 + row
					if x < ScreenWidth && y < ScreenHeight {
						g.display.Set(x, y, cyan)
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

// Package game provides the Ebitengine wrapper for human play.
// All game logic lives in the engine package.
package game

import (
	"fmt"
	"image/color"

	"manicminer/audio"
	"manicminer/config"
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
	cfg         *config.Config
	accumulator float64
	paused      bool
	lastObs     engine.Observation
	cheat       CheatState
	frameCount  int

	// Music.
	musicStep       int
	keyDebounce     int
	musicToggleHeld bool

	// Sub-screens.
	settingsScreen *SettingsScreen
	highScoreScr   *HighScoreScreen
	nameEntryScr   *NameEntryScreen
}

// New creates a new Game instance for human play.
func New() *Game {
	cfg := config.Load()
	env := engine.NewGameEnv()

	// Apply feature flags from config.
	applyFeatures(env, &cfg.Features)

	g := &Game{
		env:         env,
		renderer:    screen.NewRenderer(),
		display:     ebiten.NewImage(ScreenWidth, ScreenHeight),
		audioPlayer: audio.NewPlayer(),
		cfg:         cfg,
		lastObs:     env.GetObservation(),
		musicStep:   60,
	}
	return g
}

func applyFeatures(env *engine.GameEnv, f *config.Features) {
	env.InfiniteLives = f.InfiniteLives
	env.InfiniteAir = f.InfiniteAir
	env.HarmlessHeights = f.HarmlessHeights
	env.NoNasties = f.NoNasties
	env.NoGuardians = f.NoGuardians
	env.WarpMode = f.WarpMode
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
	g.frameCount++
	inp := input.Read(g.cfg.ControlScheme)

	// Handle sub-screens that bypass the engine.
	switch g.env.State {
	case engine.StateSettings:
		if g.settingsScreen == nil {
			g.settingsScreen = newSettingsScreen()
		}
		if g.settingsScreen.update(g.cfg) {
			// Returned from settings — save and apply.
			g.cfg.Save()
			applyFeatures(g.env, &g.cfg.Features)
			g.settingsScreen = nil
			g.env.State = engine.StateTitle
			g.env.TitleFrame = 0
		}
		return

	case engine.StateHighScores:
		if g.highScoreScr == nil {
			g.highScoreScr = newHighScoreScreen()
		}
		if g.highScoreScr.update() {
			g.highScoreScr = nil
			g.env.State = engine.StateTitle
			g.env.TitleFrame = 0
		}
		return

	case engine.StateNameEntry:
		if g.nameEntryScr == nil {
			return
		}
		g.nameEntryScr.update()
		if g.nameEntryScr.Done {
			name := g.nameEntryScr.nameString()
			g.cfg.AddHighScore(name, g.nameEntryScr.Score, g.nameEntryScr.Cavern)
			g.cfg.PlayerName = name
			g.cfg.Save()
			g.nameEntryScr = nil
			g.env.State = engine.StateHighScores
			g.highScoreScr = newHighScoreScreen()
		}
		return
	}

	// Pause handling.
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
		if inp.Quit {
			g.lastObs = g.env.Reset(g.env.CavernNumber)
			return
		}
	}

	// Music toggle.
	if inp.MusicToggle {
		if !g.musicToggleHeld {
			g.env.MusicEnabled = !g.env.MusicEnabled
			if !g.env.MusicEnabled {
				g.audioPlayer.StopInGameMusic()
			}
			g.musicToggleHeld = true
		}
	} else {
		g.musicToggleHeld = false
	}

	// Cheat code.
	g.cheat.Update()
	if g.env.State == engine.StatePlaying {
		if g.env.WarpMode || g.cheat.Active {
			if dest := g.cheat.CheckTeleport(); dest >= 0 {
				g.lastObs = g.env.Reset(dest)
				return
			}
		}
	}

	// Music tempo tuning (debug).
	if g.keyDebounce > 0 {
		g.keyDebounce--
	}
	if g.keyDebounce == 0 {
		if ebiten.IsKeyPressed(ebiten.KeyMinus) && g.musicStep < 200 {
			g.musicStep += 5
			g.keyDebounce = 5
			g.audioPlayer.SetInGameMusicTempo(g.musicStep)
			fmt.Printf("Music note duration: %dms\n", g.musicStep)
		}
		if ebiten.IsKeyPressed(ebiten.KeyEqual) && g.musicStep > 10 {
			g.musicStep -= 5
			g.keyDebounce = 5
			g.audioPlayer.SetInGameMusicTempo(g.musicStep)
			fmt.Printf("Music note duration: %dms\n", g.musicStep)
		}
	}

	// Check if game over should transition to name entry.
	prevState := g.env.State
	result := g.env.Step(inp.ToAction())
	g.lastObs = result.Obs

	// Detect transition from GameOver to Title — check for high score.
	if prevState == engine.StateGameOver && g.env.State == engine.StateTitle {
		score := g.lastObs.ScoreInt
		if g.cfg.QualifiesForHighScore(score) {
			g.env.State = engine.StateNameEntry
			g.nameEntryScr = newNameEntryScreen(score, g.env.CavernNumber, g.cfg.PlayerName)
			return
		}
	}

	g.updateAudio()
}

// updateAudio manages sound based on game state.
func (g *Game) updateAudio() {
	switch g.env.State {
	case engine.StateTitle:
		if g.env.TitlePhase == 0 {
			if !g.audioPlayer.IsTunePlaying() && g.env.TitleFrame <= 1 {
				g.audioPlayer.PlayTune(data.TitleTuneData[:])
			}
			g.env.TuneNoteIndex = g.audioPlayer.TuneNoteIndex()
			if !g.audioPlayer.IsTunePlaying() && g.env.TitleFrame > 1 {
				g.env.TitlePhase = 1
				g.env.BannerOffset = 0
			}
		} else {
			g.audioPlayer.Silence()
		}

	case engine.StatePlaying:
		if g.audioPlayer.IsTunePlaying() {
			g.audioPlayer.Silence()
		}
		if g.env.MusicEnabled {
			if !g.audioPlayer.IsInGameMusicPlaying() {
				g.audioPlayer.StartInGameMusic(data.InGameTuneData[:], g.musicStep)
			}
		} else {
			if g.audioPlayer.IsInGameMusicPlaying() {
				g.audioPlayer.StopInGameMusic()
			}
		}
		if g.lastObs.SoundRequest == 1 || g.lastObs.SoundRequest == 2 {
			g.audioPlayer.PlaySFX(g.lastObs.SoundPitch)
		}

	case engine.StateDying:
		g.audioPlayer.StopInGameMusic()
		pitch := 7 + g.env.AnimCounter*8
		if pitch > 63 {
			pitch = 63
		}
		g.audioPlayer.PlaySFX(pitch)

	case engine.StateGameOver:
		if g.lastObs.SoundRequest == 5 {
			g.audioPlayer.PlaySFX(g.lastObs.SoundPitch)
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
	case engine.StateSettings:
		if g.settingsScreen != nil {
			g.settingsScreen.draw(g.display, g.cfg, g.frameCount)
		}
	case engine.StateHighScores:
		if g.highScoreScr != nil {
			g.highScoreScr.draw(g.display, g.cfg, g.frameCount)
		}
	case engine.StateNameEntry:
		if g.nameEntryScr != nil {
			g.nameEntryScr.draw(g.display, g.frameCount)
		}
	}

	scr.DrawImage(g.display, &ebiten.DrawImageOptions{})
}

func (g *Game) drawTitle() {
	g.renderer.RenderBuffer(g.display, g.lastObs.Attrs[:], g.lastObs.Pixels[:])

	if g.env.TitlePhase == 1 {
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

	// Help text at bottom.
	screen.PrintMessage(g.display, 0, 184, "ENTER Start  ESC Settings", 0x06)
}

func (g *Game) drawPlaying() {
	g.renderer.RenderBuffer(g.display, g.lastObs.Attrs[:], g.lastObs.Pixels[:])
	g.renderHUD()
}

func (g *Game) drawGameOver() {
	g.renderer.RenderBuffer(g.display, g.lastObs.Attrs[:], g.lastObs.Pixels[:])
	if g.env.GameOverPhase >= 1 {
		screen.PrintMessage(g.display, 10*8, 6*8, "Game", 0x47)
		screen.PrintMessage(g.display, 18*8, 6*8, "Over", 0x47)
	}
}

func (g *Game) renderHUD() {
	// Cavern name row — yellow background, black text.
	for y := 128; y < 136; y++ {
		for x := 0; x < ScreenWidth; x++ {
			g.display.Set(x, y, color.RGBA{215, 215, 0, 255})
		}
	}
	screen.PrintMessage(g.display, 0, 128, g.lastObs.CavernName, 0x30)

	g.drawAirBar()

	highScoreText := "High Score " + string(g.env.HighScore[:]) + "   Score " + string(g.lastObs.Score[:])
	screen.PrintMessage(g.display, 0, 152, highScoreText, 0x06)

	g.drawLives()
}

func (g *Game) drawAirBar() {
	airLength := g.lastObs.Air - 0x24
	if airLength < 0 {
		airLength = 0
	}
	green := color.RGBA{0, 215, 0, 255}
	red := color.RGBA{215, 0, 0, 255}
	white := color.RGBA{215, 215, 215, 255}

	for y := 136; y < 144; y++ {
		for col := 0; col < 32; col++ {
			var c color.RGBA
			if col >= 4 {
				c = green
			} else {
				c = red
			}
			for bit := 0; bit < 8; bit++ {
				g.display.Set(col*8+bit, y, c)
			}
		}
	}
	for row := 0; row < 4; row++ {
		for cell := 0; cell < airLength; cell++ {
			for bit := 0; bit < 8; bit++ {
				g.display.Set((cell+4)*8+bit, 138+row, white)
			}
		}
	}
	screen.PrintMessage(g.display, 0, 136, "AIR", 0x17)
}

func (g *Game) drawLives() {
	for y := 168; y < 184; y++ {
		for x := 0; x < ScreenWidth; x++ {
			g.display.Set(x, y, color.Black)
		}
	}

	lives := g.env.Lives
	if lives <= 0 {
		return
	}

	animIdx := ((g.env.MusicNoteIndex << 3) & 0x60) >> 5
	spriteData := data.WillySprites[animIdx]
	cyan := color.RGBA{0, 215, 215, 255}

	for i := 0; i < lives && i < 8; i++ {
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

	// Draw boot sprite when infinite lives or cheat mode is active.
	if g.env.InfiniteLives || g.cheat.Active {
		bootPx := lives * 16
		for row := 0; row < 16; row++ {
			leftByte := data.BootGraphic[row*2]
			rightByte := data.BootGraphic[row*2+1]
			for bit := 7; bit >= 0; bit-- {
				if leftByte&(1<<uint(bit)) != 0 {
					x := bootPx + (7 - bit)
					y := 168 + row
					if x < ScreenWidth && y < ScreenHeight {
						g.display.Set(x, y, cyan)
					}
				}
			}
			for bit := 7; bit >= 0; bit-- {
				if rightByte&(1<<uint(bit)) != 0 {
					x := bootPx + 8 + (7 - bit)
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

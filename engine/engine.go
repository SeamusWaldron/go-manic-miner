// Package engine implements the Manic Miner game logic as a headless state
// machine with no graphics dependencies. It exposes a Gym-like Step/Reset API
// suitable for AI training, testing, and debug tooling.
package engine

import (
	"manicminer/action"
	"manicminer/cavern"
	"manicminer/data"
	"manicminer/entity"
	"manicminer/screen"
)

// State represents the overall game state.
type State int

const (
	StateTitle    State = iota // Title screen with scrolling banner.
	StatePlaying               // Active gameplay.
	StateDying                 // Death animation in progress.
	StateGameOver              // Game over sequence.
	StateDemo                  // Demo mode (auto-cycling caverns).
	StateNextCavern            // Cavern transition animation.
)

// GameEnv is the headless game environment.
type GameEnv struct {
	State State

	CurrentCavern *cavern.Cavern
	CavernNumber  int

	EmptyAttr   [AttrBufSize]byte
	EmptyPixels [PixelBufSize]byte
	WorkAttr    [AttrBufSize]byte
	WorkPixels  [PixelBufSize]byte

	Willy          *entity.Willy
	HorizGuardians []entity.HorizGuardian
	VertGuardians  []entity.VertGuardian
	Items          []entity.Item
	Portal         *entity.Portal

	// Special entities.
	Eugene  *entity.Eugene
	Kong    *entity.Kong
	Skylabs []entity.Skylab

	Score        [10]byte
	HighScore    [6]byte
	Lives        int
	Air          byte
	GameClock    byte
	BorderColour byte
	FlashCounter byte
	LastItemAttr byte

	// Title screen state.
	BannerOffset    int  // Scroll position for title banner.
	TitleFrame      int  // Frame counter for title screen.
	TitlePhase      int  // 0 = tune/piano, 1 = banner scroll.
	TuneNoteIndex   int  // Current note in the title tune.
	TuneFrameCount  int  // Frames spent on current note.
	titleBasePixels [AttrBufSize * 8]byte // Base title pixels.
	titleBaseAttrs  [AttrBufSize]byte     // Base title attributes.

	// Death/transition animation state.
	AnimCounter int

	// Music state.
	MusicNoteIndex int
	MusicEnabled   bool

	// Demo mode.
	DemoCounter int

	// Internal tracking.
	prevScoreInt int
	levelDone    bool
	died         bool
}

// NewGameEnv creates a new game environment.
func NewGameEnv() *GameEnv {
	e := &GameEnv{
		State:        StateTitle,
		Lives:        2,
		MusicEnabled: true,
	}
	for i := range e.Score {
		e.Score[i] = '0'
	}
	for i := range e.HighScore {
		e.HighScore[i] = '0'
	}
	e.initTitle()
	return e
}

// initTitle sets up the title screen state.
func (e *GameEnv) initTitle() {
	e.State = StateTitle
	e.BannerOffset = 0
	e.TitleFrame = 0
	e.CavernNumber = 0
	e.MusicNoteIndex = 0

	// Build title screen buffers.
	// The title screen graphic data is in raw ZX Spectrum display file format
	// (interleaved thirds). Convert to our linearised pixel buffer layout.
	screen.SpectrumDisplayToLinear(data.TitleScreenPixels[:], e.titleBasePixels[:])

	// Attributes: top third from The Final Barrier cavern data,
	// bottom two-thirds from BottomAttributes.
	finalBarrier := cavern.Load(19)
	if finalBarrier != nil {
		copy(e.titleBaseAttrs[0:256], finalBarrier.Attributes[0:256])
	}
	copy(e.titleBaseAttrs[256:], data.TitleScreenBottomAttrs[:])

	// Copy base to work buffers.
	copy(e.WorkPixels[:], e.titleBasePixels[:])
	copy(e.WorkAttr[:], e.titleBaseAttrs[:])

	// Draw initial Willy sprite at (9,29) — pixel y=72.
	willySprite := data.WillySprites[2]
	screen.DrawSprite(e.WorkPixels[:], 72, 29, willySprite[:], screen.DrawOverwrite)

	e.TitlePhase = 0
	e.TuneNoteIndex = 0
	e.TuneFrameCount = 0
}

// Reset initialises a cavern for gameplay. Returns the initial observation.
func (e *GameEnv) Reset(cavernNum int) Observation {
	e.CavernNumber = cavernNum
	e.CurrentCavern = cavern.Load(cavernNum)
	if e.CurrentCavern == nil {
		return Observation{}
	}

	copy(e.EmptyAttr[:], e.CurrentCavern.Attributes[:])
	screen.DrawCavernToBuffer(e.CurrentCavern, e.EmptyAttr[:], e.EmptyPixels[:])

	e.Air = e.CurrentCavern.Air
	e.GameClock = e.CurrentCavern.GameClock
	e.BorderColour = e.CurrentCavern.BorderColour

	e.Willy = entity.NewWilly(e.CurrentCavern)
	e.HorizGuardians = entity.NewHorizGuardians(e.CurrentCavern)
	e.VertGuardians = entity.NewVertGuardians(e.CurrentCavern)
	e.Items = entity.NewItems(e.CurrentCavern)
	e.Portal = entity.NewPortal(e.CurrentCavern)
	e.LastItemAttr = 0xFF
	e.levelDone = false
	e.died = false
	e.MusicNoteIndex = 0

	// Special entities.
	e.Eugene = nil
	e.Kong = nil
	e.Skylabs = nil
	if cavernNum == 4 {
		e.Eugene = entity.NewEugene()
	}
	if cavernNum == 7 || cavernNum == 11 {
		e.Kong = entity.NewKong()
	}
	if cavernNum == 13 {
		e.Skylabs = entity.NewSkylabs(e.CurrentCavern)
	}

	e.State = StatePlaying
	return e.buildObservation()
}

// Step advances the game by one logic frame.
func (e *GameEnv) Step(act action.Action) StepResult {
	e.prevScoreInt = e.scoreToInt()
	e.levelDone = false
	e.died = false

	switch e.State {
	case StateTitle:
		e.stepTitle(act)
	case StatePlaying:
		e.stepPlaying(act)
	case StateDying:
		e.stepDying()
	case StateGameOver:
		e.stepGameOver()
	case StateNextCavern:
		e.stepNextCavern()
	case StateDemo:
		e.stepDemo()
	}

	obs := e.buildObservation()
	reward := e.computeReward()
	done := e.died || e.levelDone

	obs.Done = done
	obs.LevelDone = e.levelDone
	obs.GameOver = e.Lives < 0

	return StepResult{Obs: obs, Reward: reward, Done: done}
}

// GetObservation returns the current state without advancing.
func (e *GameEnv) GetObservation() Observation {
	return e.buildObservation()
}

// stepTitle handles one frame of the title screen.
// Phase 0: Piano keys animate through the Blue Danube tune data.
// Phase 1: Banner scrolls with Willy animating at (9,29).
func (e *GameEnv) stepTitle(act action.Action) {
	e.TitleFrame++

	// Enter/fire starts the game (passed via act.Enter).
	if act.Enter {
		e.startGame()
		return
	}

	if e.TitlePhase == 0 {
		e.stepTitleTune()
	} else {
		e.stepTitleBanner()
	}
}

// stepTitleTune animates piano keys through the Blue Danube.
// stepTitleTune animates piano keys based on the currently-playing note.
// The audio system manages note timing internally — TuneNoteIndex is synced
// from the audio player by the game wrapper.
func (e *GameEnv) stepTitleTune() {
	tuneData := data.TitleTuneData[:]

	// Reset attributes to base (clears previous key highlights).
	copy(e.WorkAttr[:], e.titleBaseAttrs[:])

	// Get the current note from the audio-synced index.
	noteOffset := e.TuneNoteIndex * 3
	if noteOffset+2 >= len(tuneData) || tuneData[noteOffset] == 0xFF {
		return // Tune finished — wrapper will switch to banner phase.
	}

	freq1 := tuneData[noteOffset+1]
	freq2 := tuneData[noteOffset+2]

	// Highlight the two piano keys for this note.
	if freq1 > 0 {
		key1 := pianoKeyColumn(freq1)
		if key1 >= 0 && key1 < 32 {
			e.WorkAttr[15*32+key1] = 80 // INK 0, PAPER 2, BRIGHT 1.
		}
	}
	if freq2 > 0 {
		key2 := pianoKeyColumn(freq2)
		if key2 >= 0 && key2 < 32 {
			e.WorkAttr[15*32+key2] = 40 // INK 0, PAPER 5.
		}
	}
}

// pianoKeyColumn converts a frequency parameter to a piano key column (0-31).
// Matches the original CalcAFAForPianoKey: key = 31 - ((freq - 8) / 8).
func pianoKeyColumn(freq byte) int {
	if freq < 8 {
		return -1
	}
	return 31 - int((freq-8)/8)
}

// stepTitleBanner scrolls the banner and animates Willy.
func (e *GameEnv) stepTitleBanner() {
	// Restore base pixels (clears previous Willy frame).
	copy(e.WorkPixels[:], e.titleBasePixels[:])
	// Restore base attributes (clears any leftover piano highlights).
	copy(e.WorkAttr[:], e.titleBaseAttrs[:])

	// Animate Willy at (9,29) — pixel y=72, cellX=29.
	// Animation frame cycles based on BannerOffset bits 1-2.
	animIdx := (e.BannerOffset & 0x06) >> 1
	willySprite := data.WillySprites[animIdx]
	screen.DrawSprite(e.WorkPixels[:], 72, 29, willySprite[:], screen.DrawOverwrite)

	// Advance banner every frame (original pauses ~0.1s per character).
	e.BannerOffset++
	if e.BannerOffset >= 224 {
		// Banner finished — enter demo mode.
		e.State = StateDemo
		e.DemoCounter = 64
		e.Reset(e.CavernNumber)
		e.State = StateDemo
	}
}

func (e *GameEnv) startGame() {
	e.Lives = 2
	for i := range e.Score {
		e.Score[i] = '0'
	}
	e.CavernNumber = 0
	e.Reset(0)
}

// stepPlaying handles one frame of active gameplay.
func (e *GameEnv) stepPlaying(act action.Action) {
	if e.CurrentCavern == nil || e.Willy == nil {
		return
	}

	// Copy empty buffers into working buffers.
	copy(e.WorkAttr[:], e.EmptyAttr[:])
	copy(e.WorkPixels[:], e.EmptyPixels[:])

	// Move horizontal guardians.
	entity.MoveHorizGuardians(e.HorizGuardians, e.GameClock)

	// Update Willy.
	e.Willy.Update(act, e.CurrentCavern, e.EmptyAttr[:], e.EmptyPixels[:], e.EmptyAttr[:])

	// Re-copy after crumbling.
	copy(e.WorkAttr[:], e.EmptyAttr[:])
	copy(e.WorkPixels[:], e.EmptyPixels[:])

	// Check nasties, set attributes, draw Willy.
	if e.Willy.Alive {
		e.Willy.CheckNasties(e.CurrentCavern, e.WorkAttr[:])
	}
	if e.Willy.Alive {
		e.Willy.SetAttributes(e.CurrentCavern, e.WorkAttr[:])
		e.Willy.Draw(e.WorkPixels[:])
	}

	// Draw horizontal guardians.
	if e.Willy.Alive {
		if entity.DrawHorizGuardians(e.HorizGuardians, e.CurrentCavern, e.CavernNumber,
			e.WorkAttr[:], e.WorkPixels[:]) {
			e.Willy.Kill()
		}
	}

	// Move conveyor.
	e.moveConveyor()

	// Draw and collect items.
	e.LastItemAttr = entity.DrawAndCollectItems(e.Items, e.CurrentCavern,
		e.WorkAttr[:], e.WorkPixels[:], e.Score[:])

	// Special entity: Eugene (cavern 4).
	if e.Eugene != nil && e.Willy.Alive {
		if e.Eugene.MoveAndDraw(e.CurrentCavern, e.LastItemAttr, e.GameClock,
			e.WorkAttr[:], e.WorkPixels[:]) {
			e.Willy.Kill()
		}
	}

	// Special entity: Skylabs (cavern 13).
	if e.Skylabs != nil && e.Willy.Alive {
		if entity.MoveAndDrawSkylabs(e.Skylabs, e.CurrentCavern,
			e.WorkAttr[:], e.WorkPixels[:]) {
			e.Willy.Kill()
		}
	}

	// Vertical guardians (caverns >= 8, except 13).
	if e.CavernNumber >= 8 && e.CavernNumber != 13 {
		entity.MoveVertGuardians(e.VertGuardians)
		if e.Willy.Alive {
			if entity.DrawVertGuardians(e.VertGuardians, e.CurrentCavern,
				e.WorkAttr[:], e.WorkPixels[:]) {
				e.Willy.Kill()
			}
		}
	}

	// Kong Beast (caverns 7, 11).
	if e.Kong != nil && e.Willy.Alive {
		if e.Kong.MoveAndDraw(e.CurrentCavern, e.GameClock,
			e.WorkAttr[:], e.WorkPixels[:], e.Score[:]) {
			e.Willy.Kill()
		}
	}

	// Light Beam (cavern 18).
	if e.CavernNumber == 18 && e.Willy.Alive {
		extraDrain := entity.DrawLightBeam(e.CurrentCavern, e.WorkAttr[:])
		for i := 0; i < extraDrain; i++ {
			e.decreaseAir()
		}
	}

	// Portal.
	if e.LastItemAttr == 0 && e.Portal != nil {
		e.Portal.ActivateFlash()
	}
	if e.Portal != nil {
		if e.Portal.CheckEntry(e.Willy) {
			e.moveToNextCavern()
			return
		}
		e.Portal.Draw(e.WorkAttr[:], e.WorkPixels[:])
	}

	// Screen flash.
	if e.FlashCounter > 0 {
		e.FlashCounter--
		flashAttr := (e.FlashCounter << 3) & 0x38
		for i := range e.WorkAttr {
			e.WorkAttr[i] = flashAttr
		}
	}

	// Decrease air.
	e.decreaseAir()

	// Advance in-game music note.
	// Original increments a 0-255 counter each frame at ~20 FPS, then maps
	// to notes 0-63 via (counter AND 126) >> 1. Each note plays 2 frames.
	// At our 12 FPS, advance by 3 to get the right tempo (~2.7s per cycle).
	e.MusicNoteIndex = (e.MusicNoteIndex + 3) & 63

	// Check death.
	if !e.Willy.Alive {
		e.died = true
		e.State = StateDying
		e.AnimCounter = 0
	}
}

// stepDying handles the death animation (colour cycling).
func (e *GameEnv) stepDying() {
	e.AnimCounter++

	// Flash the screen with cycling colours for ~8 frames.
	attr := byte(0x47 - (e.AnimCounter % 8))
	for i := range e.WorkAttr {
		e.WorkAttr[i] = attr
	}

	if e.AnimCounter >= 32 {
		if e.Lives > 0 {
			e.Lives--
			e.Reset(e.CavernNumber)
		} else {
			e.State = StateGameOver
			e.AnimCounter = 0
		}
	}
}

// stepGameOver handles the game over sequence.
func (e *GameEnv) stepGameOver() {
	e.AnimCounter++

	// Update high score.
	currentScore := string(e.Score[4:])
	highScore := string(e.HighScore[:])
	if currentScore > highScore {
		copy(e.HighScore[:], e.Score[4:])
	}

	if e.AnimCounter >= 96 {
		// Return to title screen.
		e.initTitle()
	}
}

// stepNextCavern handles the cavern transition animation.
func (e *GameEnv) stepNextCavern() {
	e.AnimCounter++

	// Colour cycling transition.
	attr := byte(0x3F - (e.AnimCounter % 64))
	for i := range e.WorkAttr {
		e.WorkAttr[i] = attr
	}

	if e.AnimCounter >= 64 {
		next := e.CavernNumber + 1
		if next >= NumCaverns {
			next = 0
		}
		e.Reset(next)
	}
}

// stepDemo handles demo mode (auto-cycling caverns with no player control).
func (e *GameEnv) stepDemo() {
	if e.CurrentCavern == nil || e.Willy == nil {
		return
	}

	e.DemoCounter--
	if e.DemoCounter <= 0 {
		// Move to next cavern in demo.
		next := (e.CavernNumber + 1) % NumCaverns
		e.Reset(next)
		e.State = StateDemo
		e.DemoCounter = 64
		return
	}

	// Run the game logic but with no input (Willy stands still).
	e.stepPlaying(action.Action{})
	e.State = StateDemo // Ensure we stay in demo state.
}

func (e *GameEnv) moveConveyor() {
	if e.CurrentCavern == nil {
		return
	}
	cav := e.CurrentCavern
	convAttr := cav.Conveyor.Attr
	dir := cav.ConveyorDir

	for cellRow := 0; cellRow < CavernRows; cellRow++ {
		for cellCol := 0; cellCol < CavernCols; cellCol++ {
			if e.EmptyAttr[cellRow*CavernCols+cellCol] != convAttr {
				continue
			}
			row0Idx := cellRow*256 + 0*32 + cellCol
			row2Idx := cellRow*256 + 2*32 + cellCol
			if dir == 0 {
				e.EmptyPixels[row0Idx] = rotateLeft(e.EmptyPixels[row0Idx], 2)
				e.EmptyPixels[row2Idx] = rotateRight(e.EmptyPixels[row2Idx], 2)
			} else {
				e.EmptyPixels[row0Idx] = rotateRight(e.EmptyPixels[row0Idx], 2)
				e.EmptyPixels[row2Idx] = rotateLeft(e.EmptyPixels[row2Idx], 2)
			}
		}
	}
}

func rotateLeft(b byte, n uint) byte  { return (b << n) | (b >> (8 - n)) }
func rotateRight(b byte, n uint) byte { return (b >> n) | (b << (8 - n)) }

func (e *GameEnv) decreaseAir() {
	e.GameClock -= 4
	if e.GameClock == 0xFC {
		if e.Air <= 0x24 {
			if e.Willy != nil {
				e.Willy.Kill()
			}
			return
		}
		e.Air--
	}
}

func (e *GameEnv) moveToNextCavern() {
	// Convert remaining air to score.
	for e.Air > 0x24 {
		e.Air--
		entity.AddToScore(e.Score[:], 9, 1)
	}
	e.levelDone = true
	e.State = StateNextCavern
	e.AnimCounter = 0
}

func (e *GameEnv) buildObservation() Observation {
	obs := Observation{
		CavernNum: e.CavernNumber,
		Lives:     e.Lives,
		Air:       int(e.Air),
		GameClock: e.GameClock,
	}

	copy(obs.Attrs[:], e.WorkAttr[:])
	copy(obs.Pixels[:], e.WorkPixels[:])
	copy(obs.Score[:], e.Score[4:10])
	obs.ScoreInt = e.scoreToInt()

	if e.CurrentCavern != nil {
		obs.CavernName = e.CurrentCavern.Name
	}

	if e.Willy != nil {
		obs.WillyX = e.Willy.CellX
		obs.WillyY = e.Willy.PixelY()
		obs.WillyCellY = e.Willy.CellY
		obs.WillyDir = e.Willy.Direction()
		obs.WillyFrame = e.Willy.AnimFrame
		obs.Airborne = e.Willy.Airborne
	}

	return obs
}

func (e *GameEnv) computeReward() float64 {
	reward := float64(e.scoreToInt()-e.prevScoreInt) * 0.01
	if e.levelDone {
		reward += 10.0
	}
	if e.died {
		reward -= 1.0
	}
	reward -= 0.001
	return reward
}

func (e *GameEnv) scoreToInt() int {
	result := 0
	for _, ch := range e.Score[4:10] {
		result = result*10 + int(ch-'0')
	}
	return result
}

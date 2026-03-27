// Package engine implements the Manic Miner game logic as a headless state
// machine with no graphics dependencies. It exposes a Gym-like Step/Reset API
// suitable for AI training, testing, and debug tooling.
package engine

import (
	"manicminer/action"
	"manicminer/cavern"
	"manicminer/entity"
	"manicminer/screen"
)

// GameEnv is the headless game environment.
type GameEnv struct {
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

	Score        [10]byte // ASCII digits. [4]-[9] are the visible 6.
	HighScore    [6]byte  // ASCII digits.
	Lives        int
	Air          byte
	GameClock    byte
	BorderColour byte
	FlashCounter byte
	LastItemAttr byte

	// Internal tracking for reward computation.
	prevScoreInt int
	levelDone    bool
	died         bool
}

// NewGameEnv creates a new game environment starting at cavern 0 with 3 lives.
func NewGameEnv() *GameEnv {
	e := &GameEnv{Lives: 2}
	for i := range e.Score {
		e.Score[i] = '0'
	}
	for i := range e.HighScore {
		e.HighScore[i] = '0'
	}
	e.Reset(0)
	return e
}

// Reset initialises (or re-initialises) a cavern. Returns the initial observation.
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

	return e.buildObservation()
}

// Step advances the game by one logic frame with the given action.
// Returns the resulting observation, reward, and done flag.
func (e *GameEnv) Step(act action.Action) StepResult {
	e.prevScoreInt = e.scoreToInt()
	e.levelDone = false
	e.died = false

	e.step(act)

	obs := e.buildObservation()
	reward := e.computeReward()
	done := e.died || e.levelDone

	obs.Done = done
	obs.LevelDone = e.levelDone
	obs.GameOver = e.Lives < 0

	return StepResult{
		Obs:    obs,
		Reward: reward,
		Done:   done,
	}
}

// GetObservation returns the current state without advancing.
func (e *GameEnv) GetObservation() Observation {
	return e.buildObservation()
}

// step runs one logic frame — this is the extracted logicUpdate().
func (e *GameEnv) step(act action.Action) {
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

	// Re-copy empty buffers (crumbling may have modified them).
	copy(e.WorkAttr[:], e.EmptyAttr[:])
	copy(e.WorkPixels[:], e.EmptyPixels[:])

	// Check nasties, set Willy's attributes, draw Willy (before guardians for collision).
	if e.Willy.Alive {
		e.Willy.CheckNasties(e.CurrentCavern, e.WorkAttr[:])
	}
	if e.Willy.Alive {
		e.Willy.SetAttributes(e.CurrentCavern, e.WorkAttr[:])
		e.Willy.Draw(e.WorkPixels[:])
	}

	// Draw horizontal guardians (blend mode detects collision with Willy's pixels).
	if e.Willy.Alive {
		if entity.DrawHorizGuardians(e.HorizGuardians, e.CurrentCavern, e.CavernNumber,
			e.WorkAttr[:], e.WorkPixels[:]) {
			e.Willy.Kill()
		}
	}

	// Move conveyor animation.
	e.moveConveyor()

	// Draw and collect items.
	e.LastItemAttr = entity.DrawAndCollectItems(e.Items, e.CurrentCavern,
		e.WorkAttr[:], e.WorkPixels[:], e.Score[:])

	// Move and draw vertical guardians (caverns >= 8, except 13 = Skylabs).
	if e.CavernNumber >= 8 && e.CavernNumber != 13 {
		entity.MoveVertGuardians(e.VertGuardians)
		if e.Willy.Alive {
			if entity.DrawVertGuardians(e.VertGuardians, e.CurrentCavern,
				e.WorkAttr[:], e.WorkPixels[:]) {
				e.Willy.Kill()
			}
		}
	}

	// Activate portal flash when all items collected.
	if e.LastItemAttr == 0 && e.Portal != nil {
		e.Portal.ActivateFlash()
	}

	// Draw portal and check entry.
	if e.Portal != nil {
		if e.Portal.CheckEntry(e.Willy) {
			e.moveToNextCavern()
			return
		}
		e.Portal.Draw(e.WorkAttr[:], e.WorkPixels[:])
	}

	// Handle screen flash.
	if e.FlashCounter > 0 {
		e.FlashCounter--
		flashAttr := (e.FlashCounter << 3) & 0x38
		for i := range e.WorkAttr {
			e.WorkAttr[i] = flashAttr
		}
	}

	// Decrease air supply.
	e.decreaseAir()

	// Check if Willy died.
	if !e.Willy.Alive {
		e.died = true
		if e.Lives > 0 {
			e.Lives--
			e.Reset(e.CavernNumber)
		} else {
			e.handleGameOver()
		}
	}
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
	for e.Air > 0x24 {
		e.Air--
		entity.AddToScore(e.Score[:], 9, 1)
	}

	next := e.CavernNumber + 1
	if next >= NumCaverns {
		next = 0
	}
	e.levelDone = true
	e.Reset(next)
}

func (e *GameEnv) handleGameOver() {
	currentScore := string(e.Score[4:])
	highScore := string(e.HighScore[:])
	if currentScore > highScore {
		copy(e.HighScore[:], e.Score[4:])
	}
	e.Lives = 2
	for i := range e.Score {
		e.Score[i] = '0'
	}
	e.Reset(0)
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
	reward := float64(e.scoreToInt()-e.prevScoreInt) * 0.01 // Score delta.
	if e.levelDone {
		reward += 10.0 // Level completion bonus.
	}
	if e.died {
		reward -= 1.0 // Death penalty.
	}
	reward -= 0.001 // Small time penalty to encourage speed.
	return reward
}

func (e *GameEnv) scoreToInt() int {
	result := 0
	for _, ch := range e.Score[4:10] {
		result = result*10 + int(ch-'0')
	}
	return result
}

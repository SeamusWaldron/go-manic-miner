package game

import "github.com/hajimehoshi/ebiten/v2"

// CheatState tracks the 6031769 cheat code entry.
type CheatState struct {
	KeyCounter int  // 0-7. When 7, cheat mode is active.
	Active     bool
}

// The 6031769 sequence mapped to modern keyboard keys.
var cheatKeys = [7]ebiten.Key{
	ebiten.KeyDigit6,
	ebiten.KeyDigit0,
	ebiten.KeyDigit3,
	ebiten.KeyDigit1,
	ebiten.KeyDigit7,
	ebiten.KeyDigit6,
	ebiten.KeyDigit9,
}

// UpdateCheat checks for the 6031769 key sequence each frame.
func (cs *CheatState) Update() {
	if cs.Active {
		return
	}

	idx := cs.KeyCounter
	if idx >= 7 {
		cs.Active = true
		return
	}

	expected := cheatKeys[idx]

	// Check if the expected key is pressed.
	if ebiten.IsKeyPressed(expected) {
		cs.KeyCounter++
		if cs.KeyCounter >= 7 {
			cs.Active = true
		}
		return
	}

	// If any digit key OTHER than expected is pressed, reset.
	for k := ebiten.KeyDigit0; k <= ebiten.KeyDigit9; k++ {
		if ebiten.IsKeyPressed(k) && k != expected {
			cs.KeyCounter = 0
			return
		}
	}
}

// CheckTeleport returns a cavern number (0-19) if teleporting, or -1 if not.
// Requires cheat mode active + key 6 held + keys 1-5 as binary cavern number.
func (cs *CheatState) CheckTeleport() int {
	if !cs.Active {
		return -1
	}

	if !ebiten.IsKeyPressed(ebiten.KeyDigit6) {
		return -1
	}

	cavern := 0
	if ebiten.IsKeyPressed(ebiten.KeyDigit1) {
		cavern |= 1
	}
	if ebiten.IsKeyPressed(ebiten.KeyDigit2) {
		cavern |= 2
	}
	if ebiten.IsKeyPressed(ebiten.KeyDigit3) {
		cavern |= 4
	}
	if ebiten.IsKeyPressed(ebiten.KeyDigit4) {
		cavern |= 8
	}
	if ebiten.IsKeyPressed(ebiten.KeyDigit5) {
		cavern |= 16
	}

	if cavern == 0 || cavern >= 20 {
		return -1
	}

	return cavern
}

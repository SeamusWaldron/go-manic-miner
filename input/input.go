package input

import (
	"manicminer/action"

	"github.com/hajimehoshi/ebiten/v2"
)

// ToAction converts the keyboard state to an engine Action (pure, no Ebitengine dependency).
func (s State) ToAction() action.Action {
	return action.Action{Left: s.Left, Right: s.Right, Jump: s.Jump}
}

// State holds the current input state for one frame.
type State struct {
	Left  bool
	Right bool
	Jump  bool
	Pause bool

	MusicToggle bool
	Quit        bool // SHIFT+SPACE
}

// Read reads the current keyboard state and returns an input State.
func Read() State {
	var s State

	// Left: Q, W, E, R, T, or numpad 5
	s.Left = ebiten.IsKeyPressed(ebiten.KeyQ) ||
		ebiten.IsKeyPressed(ebiten.KeyW) ||
		ebiten.IsKeyPressed(ebiten.KeyE) ||
		ebiten.IsKeyPressed(ebiten.KeyR) ||
		ebiten.IsKeyPressed(ebiten.KeyT) ||
		ebiten.IsKeyPressed(ebiten.KeyDigit5)

	// Right: P, O, I, U, Y, or numpad 8
	s.Right = ebiten.IsKeyPressed(ebiten.KeyP) ||
		ebiten.IsKeyPressed(ebiten.KeyO) ||
		ebiten.IsKeyPressed(ebiten.KeyI) ||
		ebiten.IsKeyPressed(ebiten.KeyU) ||
		ebiten.IsKeyPressed(ebiten.KeyY) ||
		ebiten.IsKeyPressed(ebiten.KeyDigit8)

	// Jump: Space, Shift, Z, X, C, V, B, N, M, 0, 7
	s.Jump = ebiten.IsKeyPressed(ebiten.KeySpace) ||
		ebiten.IsKeyPressed(ebiten.KeyShiftLeft) ||
		ebiten.IsKeyPressed(ebiten.KeyShiftRight) ||
		ebiten.IsKeyPressed(ebiten.KeyZ) ||
		ebiten.IsKeyPressed(ebiten.KeyX) ||
		ebiten.IsKeyPressed(ebiten.KeyC) ||
		ebiten.IsKeyPressed(ebiten.KeyV) ||
		ebiten.IsKeyPressed(ebiten.KeyB) ||
		ebiten.IsKeyPressed(ebiten.KeyN) ||
		ebiten.IsKeyPressed(ebiten.KeyM) ||
		ebiten.IsKeyPressed(ebiten.KeyDigit0) ||
		ebiten.IsKeyPressed(ebiten.KeyDigit7)

	// Pause: A-G
	s.Pause = ebiten.IsKeyPressed(ebiten.KeyA) ||
		ebiten.IsKeyPressed(ebiten.KeyS) ||
		ebiten.IsKeyPressed(ebiten.KeyD) ||
		ebiten.IsKeyPressed(ebiten.KeyF) ||
		ebiten.IsKeyPressed(ebiten.KeyG)

	// Music toggle: H-L or Enter
	s.MusicToggle = ebiten.IsKeyPressed(ebiten.KeyH) ||
		ebiten.IsKeyPressed(ebiten.KeyJ) ||
		ebiten.IsKeyPressed(ebiten.KeyK) ||
		ebiten.IsKeyPressed(ebiten.KeyL) ||
		ebiten.IsKeyPressed(ebiten.KeyEnter)

	// Quit: both Shift and Space pressed
	shift := ebiten.IsKeyPressed(ebiten.KeyShiftLeft) || ebiten.IsKeyPressed(ebiten.KeyShiftRight)
	s.Quit = shift && ebiten.IsKeyPressed(ebiten.KeySpace)

	// If quit is active, don't also register jump
	if s.Quit {
		s.Jump = false
	}

	return s
}

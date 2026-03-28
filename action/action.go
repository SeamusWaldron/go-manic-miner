// Package action defines the input action type used by the game engine.
// This is a leaf package with zero dependencies, allowing both the engine
// and entity packages to import it without circular dependencies.
package action

// Action represents a single frame's player input.
type Action struct {
	Left   bool
	Right  bool
	Jump   bool
	Enter  bool // Start game / confirm selection.
	Up     bool // Menu navigation.
	Down   bool // Menu navigation.
	Escape bool // Back / return to previous screen.
}

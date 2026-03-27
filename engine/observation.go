package engine

// Observation captures the full game state after a step.
type Observation struct {
	// Screen buffers (ZX Spectrum format).
	Attrs  [AttrBufSize]byte  // 512 bytes: attribute buffer.
	Pixels [PixelBufSize]byte // 4096 bytes: pixel buffer.

	// Player state.
	WillyX     int  // Cell X (0-31).
	WillyY     int  // Pixel Y (0-127).
	WillyCellY int  // Cell Y (0-15).
	WillyDir   int  // 0=right, 1=left.
	WillyFrame byte // Animation frame 0-3.
	Airborne   int  // 0=grounded, 1=jumping, 2+=falling, 255=dead.

	// Game state.
	Score      [6]byte // ASCII digits (the visible 6).
	ScoreInt   int     // Numeric score for convenience.
	Lives      int
	Air        int // Raw air value (0x24=empty, 0x3F=full).
	CavernNum  int
	CavernName string
	GameClock  byte

	// Sound request from the engine (for the wrapper to play).
	// 0 = silence, 1 = jump sound, 2 = fall sound.
	SoundRequest int
	SoundPitch   int // Pitch parameter for the sound.

	// Episode signals.
	Done      bool // True if life lost or level complete.
	LevelDone bool // True if portal entered (level completed).
	GameOver  bool // True if no lives remaining.
}

// StepResult is returned by GameEnv.Step().
type StepResult struct {
	Obs    Observation
	Reward float64
	Done   bool
}

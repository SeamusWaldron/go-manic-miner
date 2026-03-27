package engine

const (
	CavernRows = 16
	CavernCols = 32

	AttrBufSize  = CavernRows * CavernCols // 512
	PixelBufSize = AttrBufSize * 8         // 4096

	// The original Spectrum main loop runs at roughly 15-18 FPS depending on
	// CPU workload. 16 FPS balances gameplay speed and music tempo.
	LogicFPS       = 16.0
	LogicFrameTime = 1.0 / LogicFPS

	NumCaverns     = 20
	CavernDataSize = 1024
)

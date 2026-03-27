package engine

const (
	CavernRows = 16
	CavernCols = 32

	AttrBufSize  = CavernRows * CavernCols // 512
	PixelBufSize = AttrBufSize * 8         // 4096

	// The original Spectrum main loop runs at roughly 17-20 FPS depending on
	// how much work the CPU does per frame. 18 FPS is a good approximation.
	LogicFPS       = 18.0
	LogicFrameTime = 1.0 / LogicFPS

	NumCaverns     = 20
	CavernDataSize = 1024
)

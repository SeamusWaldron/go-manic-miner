package engine

const (
	CavernRows = 16
	CavernCols = 32

	AttrBufSize  = CavernRows * CavernCols // 512
	PixelBufSize = AttrBufSize * 8         // 4096

	LogicFPS       = 12.0
	LogicFrameTime = 1.0 / LogicFPS

	NumCaverns     = 20
	CavernDataSize = 1024
)

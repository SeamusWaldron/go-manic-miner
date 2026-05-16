package game

import "manicminer/engine"

const (
	ScreenWidth  = 256
	ScreenHeight = 192
	ScaleFactor  = 3
	WindowWidth  = ScreenWidth * ScaleFactor
	WindowHeight = ScreenHeight * ScaleFactor

	// Re-export engine constants for backward compatibility within game package.
	LogicFrameTime = engine.LogicFrameTime

	// Version is the build version reported on the loading screen.
	Version = "1.0.0"
)

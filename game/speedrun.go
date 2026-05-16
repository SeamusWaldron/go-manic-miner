package game

import (
	"fmt"

	"manicminer/engine"
)

// startSpeedrun(re)starts a single-cavern speedrun on the given cavern.
// Called when the player presses S during gameplay.
func (g *Game) startSpeedrun(cavern int) {
	g.env.SpeedrunActive = true
	g.env.SpeedrunIsOverall = false
	g.env.SpeedrunCavern = cavern
	g.env.SpeedrunFinalCs = 0
	g.lastObs = g.env.Reset(cavern)
	g.env.BeginSpeedrunCountdown()
	g.speedrunPending = false
}

// startSpeedrunOverall begins a full-game overall speedrun from cavern 0.
func (g *Game) startSpeedrunOverall() {
	g.env.SpeedrunActive = true
	g.env.SpeedrunIsOverall = true
	g.env.SpeedrunCavern = 0
	g.env.SpeedrunFinalCs = 0
	g.env.Lives = 2
	g.lastObs = g.env.Reset(0)
	g.env.BeginSpeedrunCountdown()
	g.speedrunPending = false
}

// stopSpeedrun cancels an active speedrun and drops back into normal
// gameplay on the current cavern (used by the S toggle).
func (g *Game) stopSpeedrun() {
	g.env.SpeedrunActive = false
	g.env.SpeedrunPhase = engine.SpeedrunPhaseCountdown
	g.env.SpeedrunRunTicks = 0
	g.env.SpeedrunFinalCs = 0
	g.env.SpeedrunCountdownLeft = 0
	g.env.SpeedrunBootFlashLeft = 0
	g.speedrunPending = false
	g.lastObs = g.env.Reset(g.env.CavernNumber)
}

// endSpeedrunForTitle clears speedrun state on the way back to the
// title screen (overall-mode completion or abort).
func (g *Game) endSpeedrunForTitle() {
	g.stopSpeedrun()
}

// speedrunCountdownDisplayLives returns how many mini-Willy lives to
// show during the countdown. Start = 3, step down each second; once
// the counter hits zero the lives renderer shows the boot ("GO").
func (g *Game) speedrunCountdownDisplayLives() int {
	remaining := g.env.SpeedrunCountdownLeft
	if remaining <= 0 {
		return 0
	}
	step := (remaining + engine.SpeedrunCountdownPerStep - 1) / engine.SpeedrunCountdownPerStep
	return step
}

// speedrunShouldShowBoot reports whether the boot icon should be drawn
// during the brief post-countdown flash.
func (g *Game) speedrunShouldShowBoot() bool {
	return g.env.SpeedrunActive &&
		g.env.SpeedrunPhase == engine.SpeedrunPhaseRunning &&
		g.env.SpeedrunBootFlashLeft > 0
}

// formatSpeedrunTime renders an elapsed centisecond count as MM:SS:cc.
func formatSpeedrunTime(cs int64) string {
	if cs < 0 {
		cs = 0
	}
	minutes := cs / 6000
	cs -= minutes * 6000
	seconds := cs / 100
	cs -= seconds * 100
	return fmt.Sprintf("%02d:%02d:%02d", minutes, seconds, cs)
}

// currentSpeedrunCs returns the live elapsed time during a speedrun.
// Once the portal has frozen the run (FinalCs > 0) the display stays
// pinned there through the air-drain animation and records flow.
func (g *Game) currentSpeedrunCs() int64 {
	if g.env.SpeedrunFinalCs > 0 {
		return g.env.SpeedrunFinalCs
	}
	return engine.SpeedrunTicksToCs(g.env.SpeedrunRunTicks)
}

// finishSpeedrun is called when the engine reports Phase==Finished
// (single-cavern: after the air-drain animation; overall: after the
// final-barrier ending). For single-cavern we loop: record the time
// (if it qualifies) then immediately start a fresh attempt on the
// same cavern. For overall we go back to the title.
func (g *Game) finishSpeedrun() {
	isOverall := g.env.SpeedrunIsOverall
	cavern := g.env.SpeedrunCavern
	cs := g.env.SpeedrunFinalCs

	if cs > 0 && g.cfg.QualifiesAsSpeedrun(isOverall, cavern, cs) {
		g.speedrunNameOK = true
		g.nameEntryScr = newSpeedrunNameEntryScreen(cs, cavern, isOverall, g.cfg.PlayerName)
		g.env.State = engine.StateNameEntry
		return
	}

	// No qualifying time — go straight to the next attempt (single) or
	// back to the title (overall).
	if isOverall {
		g.endSpeedrunForTitle()
		g.cfg.Save()
		g.env.InitTitle()
		g.env.BannerLength = len(extendedBanner) - 31
		return
	}
	// Single mode: loop on the same cavern.
	g.startSpeedrun(cavern)
}

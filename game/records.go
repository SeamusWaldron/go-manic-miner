package game

import (
	"fmt"
	"image/color"

	"manicminer/config"
	"manicminer/screen"

	"github.com/hajimehoshi/ebiten/v2"
)

// RecordsScreen is the combined high-scores + speedrun-records view
// opened by pressing S on the title screen.
//
// Three tabs, switched with Left/Right:
//
//   0 HIGH SCORES   — existing top-10 score table.
//   1 FASTEST TIMES — one row per cavern showing the single best
//                     speedrun time (the "one big list" requested by
//                     the user), plus the overall best at the top.
//   2 PER CAVERN    — top-5 board for the cavern selected with
//                     Up/Down (single-cavern speedrun detail view).
//
// ESC returns to the title.
type RecordsScreen struct {
	tab          int // 0..2
	cavernCursor int // 0..19, used by tab 2.
	debounce     int
}

func newRecordsScreen() *RecordsScreen {
	return &RecordsScreen{debounce: 12}
}

func (r *RecordsScreen) update() bool {
	if r.debounce > 0 {
		r.debounce--
		return false
	}
	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		return true
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowLeft) {
		r.tab--
		if r.tab < 0 {
			r.tab = 2
		}
		r.debounce = 8
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowRight) {
		r.tab++
		if r.tab > 2 {
			r.tab = 0
		}
		r.debounce = 8
	}
	if r.tab == 2 {
		if ebiten.IsKeyPressed(ebiten.KeyArrowUp) {
			r.cavernCursor--
			if r.cavernCursor < 0 {
				r.cavernCursor = 19
			}
			r.debounce = 6
		}
		if ebiten.IsKeyPressed(ebiten.KeyArrowDown) {
			r.cavernCursor++
			if r.cavernCursor > 19 {
				r.cavernCursor = 0
			}
			r.debounce = 6
		}
	}
	return false
}

func (r *RecordsScreen) draw(display *ebiten.Image, cfg *config.Config, frameCount int) {
	display.Fill(color.Black)
	cyan := byte(0x45)
	yellow := byte(0x46)
	white := byte(0x47)
	green := byte(0x44)
	dim := byte(0x07)

	// Header.
	screen.PrintMessage(display, 7*8, 0, "MANIC MINER", cyan)

	// Tab strip.
	tabs := []string{"HIGH SCORES", "FASTEST", "PER CAVERN"}
	x := 0
	for i, t := range tabs {
		attr := dim
		if i == r.tab {
			attr = yellow | 0x80 // BRIGHT + FLASH.
		}
		screen.PrintMessage(display, x, 16, t, attr)
		x += (len(t) + 1) * 8
	}

	switch r.tab {
	case 0:
		r.drawHighScores(display, cfg, white, yellow, cyan, green)
	case 1:
		r.drawFastest(display, cfg, white, yellow, cyan, green)
	case 2:
		r.drawPerCavern(display, cfg, white, yellow, cyan, green)
	}

	// Footer hint.
	footer := "ARROWS  ESC BACK"
	hint := byte(yellow)
	if frameCount/12%2 == 0 {
		hint = yellow | 0x80
	}
	screen.PrintMessage(display, (256-len(footer)*8)/2, 23*8, footer, hint)
}

func (r *RecordsScreen) drawHighScores(display *ebiten.Image, cfg *config.Config, white, yellow, cyan, green byte) {
	colours := []byte{white, yellow, cyan, green, white, yellow, cyan, green, white, yellow}
	for i, hs := range cfg.HighScores {
		row := 4 + i
		attr := colours[i%len(colours)]
		cavernName := config.CavernName(hs.Cavern)
		if len(cavernName) > 12 {
			cavernName = cavernName[:12]
		}
		line := fmt.Sprintf("%2d. %06d %s %-12s", i+1, hs.Score, hs.Name, cavernName)
		screen.PrintMessage(display, 0, row*8, line, attr)
	}
	for i := len(cfg.HighScores); i < 10; i++ {
		row := 4 + i
		line := fmt.Sprintf("%2d. ------ --- ------------", i+1)
		screen.PrintMessage(display, 0, row*8, line, 0x40)
	}
}

func (r *RecordsScreen) drawFastest(display *ebiten.Image, cfg *config.Config, white, yellow, cyan, green byte) {
	// Overall best at top.
	screen.PrintMessage(display, 0, 4*8, "OVERALL", yellow)
	ov := cfg.FastestOverall()
	if ov.Centiseconds > 0 {
		line := fmt.Sprintf("    %s  %s", formatSpeedrunTime(ov.Centiseconds), ov.Initials)
		screen.PrintMessage(display, 0, 5*8, line, white)
	} else {
		screen.PrintMessage(display, 0, 5*8, "    --:--:--  ---", 0x40)
	}

	// Per-cavern fastest. 20 caverns into 16 rows of screen → split
	// into two columns of 10.
	screen.PrintMessage(display, 0, 7*8, "FASTEST PER CAVERN", yellow)
	colours := []byte{white, cyan, green}
	for i := 0; i < 20; i++ {
		col := i / 10 // 0 left half, 1 right half.
		row := 8 + (i % 10)
		rec := cfg.FastestForCavern(i)
		attr := colours[i%len(colours)]
		var line string
		if rec.Centiseconds > 0 {
			line = fmt.Sprintf("%02d %s %s", i+1, formatSpeedrunTime(rec.Centiseconds), rec.Initials)
		} else {
			line = fmt.Sprintf("%02d --:--:-- ---", i+1)
			attr = 0x40
		}
		screen.PrintMessage(display, col*16*8, row*8, line, attr)
	}
}

func (r *RecordsScreen) drawPerCavern(display *ebiten.Image, cfg *config.Config, white, yellow, cyan, green byte) {
	cav := r.cavernCursor
	name := config.CavernName(cav)
	if len(name) > 30 {
		name = name[:30]
	}
	hdr := fmt.Sprintf("%02d %s", cav+1, name)
	screen.PrintMessage(display, 0, 4*8, hdr, yellow)

	colours := []byte{white, cyan, green, white, cyan}
	board := cfg.SpeedrunRecords.PerCavern[cav]
	for i := 0; i < config.MaxSpeedrunRecords; i++ {
		row := 7 + i
		rec := board[i]
		var line string
		attr := colours[i%len(colours)]
		if rec.Centiseconds > 0 {
			line = fmt.Sprintf("%d. %s  %s", i+1, formatSpeedrunTime(rec.Centiseconds), rec.Initials)
		} else {
			line = fmt.Sprintf("%d. --:--:--  ---", i+1)
			attr = 0x40
		}
		screen.PrintMessage(display, 2*8, row*8, line, attr)
	}

	screen.PrintMessage(display, 0, 16*8, "UP/DOWN  Change cavern", green)
}

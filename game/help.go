package game

import (
	"image/color"

	"manicminer/screen"

	"github.com/hajimehoshi/ebiten/v2"
)

// HelpLine is one line of the help text with its colour attribute.
type HelpLine struct {
	text string
	attr byte
}

var (
	helpCyan   byte = 0x45 // INK 5, BRIGHT 1.
	helpYellow byte = 0x46 // INK 6, BRIGHT 1.
	helpWhite  byte = 0x47 // INK 7, BRIGHT 1.
	helpGreen  byte = 0x44 // INK 4, BRIGHT 1.
)

var helpLines = []HelpLine{
	{"", 0},
	{"  GAMEPLAY CONTROLS", helpYellow},
	{"", 0},
	{"  Original scheme:", helpWhite},
	{"  QWERT = Move left", helpWhite},
	{"  POIUY = Move right", helpWhite},
	{"  Space/Shift/ZXCVBNM = Jump", helpWhite},
	{"  A-G   = Pause game", helpWhite},
	{"  H-L   = Toggle music", helpWhite},
	{"", 0},
	{"  ALTERNATE CONTROLS", helpYellow},
	{"", 0},
	{"  Set in Settings screen:", helpWhite},
	{"  Arrows + Space", helpGreen},
	{"  O/P + Space", helpGreen},
	{"", 0},
	{"  TITLE SCREEN", helpYellow},
	{"", 0},
	{"  ENTER  Start new game", helpWhite},
	{"  DOWN   Continue last cave", helpWhite},
	{"  ESC    Settings", helpWhite},
	{"  UP     High scores", helpWhite},
	{"  ?      This help screen", helpWhite},
	{"", 0},
	{"  DURING GAMEPLAY", helpYellow},
	{"", 0},
	{"  ESC      Exit to title", helpWhite},
	{"  SHIFT+SPACE  Restart cave", helpWhite},
	{"  SHIFT+8  Screenshot (PNG)", helpWhite},
	{"", 0},
	{"  WARP MODE", helpYellow},
	{"", 0},
	{"  Enable in Settings, then", helpWhite},
	{"  press 6 during gameplay", helpWhite},
	{"  to open cavern selector.", helpWhite},
	{"", 0},
	{"  CHEAT CODE", helpYellow},
	{"", 0},
	{"  Type 6031769 during", helpWhite},
	{"  gameplay to enable cheat", helpWhite},
	{"  mode and teleport.", helpWhite},
	{"", 0},
	{"  SETTINGS", helpYellow},
	{"", 0},
	{"  Infinite Lives", helpGreen},
	{"  Infinite Air", helpGreen},
	{"  Harmless Heights", helpGreen},
	{"  No Nasties", helpGreen},
	{"  No Guardians", helpGreen},
	{"  Warp Mode", helpGreen},
	{"", 0},
	{"  All settings are saved", helpWhite},
	{"  and remembered.", helpWhite},
	{"", 0},
	{"  HIGH SCORES", helpYellow},
	{"", 0},
	{"  Top 10 scores saved.", helpWhite},
	{"  Enter your 3-letter name", helpWhite},
	{"  when you qualify.", helpWhite},
	{"", 0},
	{"", 0},
	{"  ACKNOWLEDGEMENTS", helpYellow},
	{"", 0},
	{"  Original game by", helpWhite},
	{"  Matthew Smith", helpCyan},
	{"  (C) 1983 Bug-Byte Ltd.", helpWhite},
	{"", 0},
	{"  Based on the Bug-Byte", helpWhite},
	{"  version of Manic Miner", helpWhite},
	{"  for the ZX Spectrum.", helpWhite},
	{"", 0},
	{"  Z80 disassembly by", helpWhite},
	{"  William Humphreys", helpCyan},
	{"  with Simon Brattel.", helpWhite},
	{"  github.com/WHumphreys/", helpGreen},
	{"  Manic-Miner-Source-Code", helpGreen},
	{"", 0},
	{"  Go implementation by", helpWhite},
	{"  Seamus Waldron", helpCyan},
	{"  with Claude AI.", helpWhite},
	{"", 0},
	{"", 0},
	{"", 0},
}

// HelpScreen displays scrolling help/instructions.
type HelpScreen struct {
	scrollY  int // Current scroll position in pixel rows.
	debounce int
}

func newHelpScreen() *HelpScreen {
	return &HelpScreen{debounce: 12}
}

func (h *HelpScreen) update() bool {
	if h.debounce > 0 {
		h.debounce--
		return false
	}

	if ebiten.IsKeyPressed(ebiten.KeyEscape) || ebiten.IsKeyPressed(ebiten.KeyBackspace) {
		return true
	}

	maxScroll := len(helpLines)*8 - 160 // Leave room for header + footer.
	if maxScroll < 0 {
		maxScroll = 0
	}

	// Manual scroll with up/down.
	if ebiten.IsKeyPressed(ebiten.KeyArrowUp) || ebiten.IsKeyPressed(ebiten.KeyK) {
		h.scrollY -= 4
		if h.scrollY < 0 {
			h.scrollY = 0
		}
		h.debounce = 2
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowDown) || ebiten.IsKeyPressed(ebiten.KeyJ) {
		h.scrollY += 4
		if h.scrollY > maxScroll {
			h.scrollY = maxScroll
		}
		h.debounce = 2
	}

	// Auto-scroll slowly.
	h.scrollY++
	if h.scrollY > maxScroll {
		h.scrollY = maxScroll
	}

	return false
}

func (h *HelpScreen) draw(display *ebiten.Image, frameCount int) {
	display.Fill(color.Black)

	// Fixed header.
	screen.PrintMessage(display, 5*8, 0, "MANIC MINER HELP", helpCyan)

	// Scrolling content area: y=16 to y=172 (starts below header row).
	contentTop := 16
	contentBottom := 172

	for i, line := range helpLines {
		y := contentTop + i*8 - h.scrollY
		if y < contentTop-8 || y >= contentBottom {
			continue // Off-screen.
		}
		if line.text == "" {
			continue
		}
		screen.PrintMessage(display, 0, y, line.text, line.attr)
	}

	// Fixed footer.
	footerAttr := helpYellow
	if frameCount/12%2 == 0 {
		footerAttr = helpYellow | 0x80
	}
	screen.PrintMessage(display, 2*8, 23*8, "UP/DOWN Scroll  ESC Back", footerAttr)
}

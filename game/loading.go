package game

import (
	"image/color"

	"manicminer/data"
	"manicminer/screen"

	"github.com/hajimehoshi/ebiten/v2"
)

// LoadingScreen reproduces the original Bug-Byte tape loading screen.
//
// The trick: the original has no pixel data at all. The BASIC loader
// runs `CLEAR 30000 : PAPER 0 : INK 0 : CLS` to zero the bitmap, then
// LOADs 256 attribute bytes to $5900 (attribute rows 8..15). Every byte
// has FLASH set, with paper colour = one frame's letter colour and ink
// colour = the other frame's letter colour. Because the pixels are all
// zero, only the paper colour is visible; when the Spectrum's hardware
// FLASH bit auto-swaps paper↔ink, the alternate word appears.
//
// We replay this exactly: empty pixel buffer, the original 256-byte
// attribute block at the right offset, and the existing renderer (which
// already does the FLASH swap on attr bit 7).
//
// Inputs while on screen:
//   - Enter  → start a new game
//   - Down   → continue from last cavern
//   - Esc    → settings (returning brings the user back here)
//   - Up     → high scores
// After 5 seconds the wrapper auto-advances to the title screen.
type LoadingScreen struct {
	frame    int
	debounce int

	// Last-tick non-Esc key state, for in-logicTick edge detection.
	// We can't rely on inpututil here: it tracks transitions per ebiten
	// frame (60 Hz), but logicTick runs at 16 Hz, so most "just pressed"
	// events fall between ticks and are missed.
	anyKeyHeldLast bool

	// Pre-built buffers re-used every frame. Pixels are all zero; attrs
	// have rows 0..7 black and rows 8..15 filled from LoadingScreenAttrs.
	attrs  [512]byte
	pixels [4096]byte
}

// 5 seconds at the engine's 16 FPS logic tick.
const LoadingDurationFrames = 5 * 16

type loadingAction int

const (
	loadingContinue loadingAction = iota
	loadingTimeout // 5 s elapsed → title screen.
	loadingSkip    // Key press → title screen (skip the wait).
	loadingSettings
)

// titleRowOffset is the attribute-row index at which the original 8-row
// MANIC/MINER block is painted. The Bug-Byte loader puts it on rows
// 8..15; we shift up 5 rows for visual balance with the credits below.
const titleRowOffset = 3

func newLoadingScreen() *LoadingScreen {
	l := &LoadingScreen{debounce: 6}
	copy(l.attrs[titleRowOffset*32:], data.LoadingScreenAttrs[:])
	return l
}

func (l *LoadingScreen) update() loadingAction {
	l.frame++

	anyKey := anyNonEscKeyPressed()
	justPressed := anyKey && !l.anyKeyHeldLast
	l.anyKeyHeldLast = anyKey

	if l.debounce > 0 {
		l.debounce--
	} else {
		if ebiten.IsKeyPressed(ebiten.KeyEscape) {
			return loadingSettings
		}
		if justPressed {
			return loadingSkip
		}
	}

	if l.frame >= LoadingDurationFrames {
		return loadingTimeout
	}
	return loadingContinue
}

// anyNonEscKeyPressed reports whether any key besides Escape is
// currently held. Iterates the full ebiten.Key range and is called once
// per logic tick.
func anyNonEscKeyPressed() bool {
	for k := ebiten.Key(0); k <= ebiten.KeyMax; k++ {
		if k == ebiten.KeyEscape {
			continue
		}
		if ebiten.IsKeyPressed(k) {
			return true
		}
	}
	return false
}

func (l *LoadingScreen) draw(display *ebiten.Image, renderer *screen.Renderer) {
	display.Fill(color.Black)

	// Render the top 128 pixel rows: zero pixels + the original
	// attribute block. The renderer handles the FLASH bit auto-swap.
	renderer.RenderBuffer(display, l.attrs[:], l.pixels[:])

	// Additional information drawn below the title band, in the area
	// the loader didn't touch (pixel rows 128..191 = attr rows 16..23).
	const (
		inkWhite   byte = 0x47
		inkYellow  byte = 0x46
		inkCyan    byte = 0x45
		inkMagenta byte = 0x43
		inkGreen   byte = 0x44
	)

	printCentered(display, 104, "(C) 1983 BUG-BYTE LTD.", inkYellow)
	printCentered(display, 116, "ORIGINAL BY MATTHEW SMITH", inkCyan)
	printCentered(display, 136, "GO PORT BY SEAMUS WALDRON", inkMagenta)
	printCentered(display, 148, "VERSION "+Version, inkGreen)
	printCentered(display, 168, "ESC  SETUP", inkWhite)
}

// printCentered renders msg horizontally centred at pixel row y.
func printCentered(display *ebiten.Image, y int, msg string, attr byte) {
	x := (ScreenWidth - len(msg)*8) / 2
	if x < 0 {
		x = 0
	}
	screen.PrintMessage(display, x, y, msg, attr)
}

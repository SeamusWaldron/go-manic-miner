package game

import (
	"image/color"

	"manicminer/cavern"
	"manicminer/config"
	"manicminer/screen"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	warpCols = 5
	warpRows = 4
)

// WarpScreen shows a grid of cavern thumbnails for level selection.
type WarpScreen struct {
	cursor     int // 0-19 selected cavern.
	debounce   int
	thumbnails [20]*ebiten.Image
}

func newWarpScreen() *WarpScreen {
	w := &WarpScreen{debounce: 12}

	// Generate a 32x16 pixel thumbnail for each cavern.
	// Each attribute byte maps to one pixel using its PAPER colour.
	for i := 0; i < 20; i++ {
		cav := cavern.Load(i)
		if cav == nil {
			continue
		}
		img := ebiten.NewImage(32, 16)
		for row := 0; row < 16; row++ {
			for col := 0; col < 32; col++ {
				attr := cav.Attributes[row*32+col]
				// Use PAPER colour (bits 3-5) with BRIGHT (bit 6).
				paperIdx := (attr >> 3) & 0x07
				bright := attr&0x40 != 0
				c := spectrumColour(paperIdx, bright)
				img.Set(col, row, c)
			}
		}
		w.thumbnails[i] = img
	}

	return w
}

func spectrumColour(idx byte, bright bool) color.RGBA {
	palette := [16]color.RGBA{
		{0, 0, 0, 255},       {0, 0, 215, 255},
		{215, 0, 0, 255},     {215, 0, 215, 255},
		{0, 215, 0, 255},     {0, 215, 215, 255},
		{215, 215, 0, 255},   {215, 215, 215, 255},
		{0, 0, 0, 255},       {0, 0, 255, 255},
		{255, 0, 0, 255},     {255, 0, 255, 255},
		{0, 255, 0, 255},     {0, 255, 255, 255},
		{255, 255, 0, 255},   {255, 255, 255, 255},
	}
	i := int(idx & 0x07)
	if bright {
		i += 8
	}
	return palette[i]
}

// update returns: -1 = still on screen, -2 = escape (back to game), 0-19 = warp to cavern.
func (w *WarpScreen) update() int {
	if w.debounce > 0 {
		w.debounce--
		return -1
	}

	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		return -2
	}
	if ebiten.IsKeyPressed(ebiten.KeyEnter) {
		w.debounce = 10
		return w.cursor
	}

	col := w.cursor % warpCols
	row := w.cursor / warpCols

	if ebiten.IsKeyPressed(ebiten.KeyArrowLeft) {
		if col > 0 {
			w.cursor--
		}
		w.debounce = 5
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowRight) {
		if col < warpCols-1 {
			w.cursor++
		}
		w.debounce = 5
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowUp) {
		if row > 0 {
			w.cursor -= warpCols
		}
		w.debounce = 5
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowDown) {
		if row < warpRows-1 {
			w.cursor += warpCols
		}
		w.debounce = 5
	}

	if w.cursor < 0 {
		w.cursor = 0
	}
	if w.cursor > 19 {
		w.cursor = 19
	}

	return -1
}

func (w *WarpScreen) draw(display *ebiten.Image, frameCount int) {
	display.Fill(color.Black)

	cyan := byte(0x45)
	yellow := byte(0x46)
	white := byte(0x47)

	screen.PrintMessage(display, 5*8, 0, "WARP TO CAVERN", cyan)

	// Grid layout: 5 columns x 4 rows.
	// Each thumbnail: 32x16 pixels.
	// Horizontal spacing: (256 - 5*32) / 6 = 16px margin between each.
	// Vertical start at y=16 (row 2), spacing: 20px per slot (16 + 4 gap).
	const thumbW = 32
	const thumbH = 16
	const startX = 8
	const startY = 16
	const gapX = 16
	const gapY = 6

	for i := 0; i < 20; i++ {
		col := i % warpCols
		row := i / warpCols

		x := startX + col*(thumbW+gapX)
		y := startY + row*(thumbH+gapY)

		// Draw selection border.
		if i == w.cursor {
			borderCol := color.RGBA{255, 255, 255, 255}
			if frameCount/6%2 == 0 {
				borderCol = color.RGBA{255, 255, 0, 255}
			}
			for bx := x - 1; bx <= x+thumbW; bx++ {
				if bx >= 0 && bx < 256 {
					display.Set(bx, y-1, borderCol)
					display.Set(bx, y+thumbH, borderCol)
				}
			}
			for by := y - 1; by <= y+thumbH; by++ {
				if by >= 0 && by < 192 {
					display.Set(x-1, by, borderCol)
					display.Set(x+thumbW, by, borderCol)
				}
			}
		}

		// Draw thumbnail.
		if w.thumbnails[i] != nil {
			opts := &ebiten.DrawImageOptions{}
			opts.GeoM.Translate(float64(x), float64(y))
			display.DrawImage(w.thumbnails[i], opts)
		}
	}

	// Show selected cavern name.
	name := config.CavernName(w.cursor)
	if len(name) > 30 {
		name = name[:30]
	}
	screen.PrintMessage(display, 1*8, 106, name, white)

	// Help text.
	screen.PrintMessage(display, 1*8, 118, "Arrows Move ENTER Warp ESC Back", yellow)
}

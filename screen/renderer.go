package screen

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	cavernRows = 16
	cavernCols = 32
)

// ZX Spectrum colour palette. Indices 0-7 normal, 8-15 BRIGHT.
var spectrumPalette = [16]color.RGBA{
	{0, 0, 0, 255},       // 0: Black
	{0, 0, 215, 255},     // 1: Blue
	{215, 0, 0, 255},     // 2: Red
	{215, 0, 215, 255},   // 3: Magenta
	{0, 215, 0, 255},     // 4: Green
	{0, 215, 215, 255},   // 5: Cyan
	{215, 215, 0, 255},   // 6: Yellow
	{215, 215, 215, 255}, // 7: White
	{0, 0, 0, 255},       // 8: Bright Black
	{0, 0, 255, 255},     // 9: Bright Blue
	{255, 0, 0, 255},     // 10: Bright Red
	{255, 0, 255, 255},   // 11: Bright Magenta
	{0, 255, 0, 255},     // 12: Bright Green
	{0, 255, 255, 255},   // 13: Bright Cyan
	{255, 255, 0, 255},   // 14: Bright Yellow
	{255, 255, 255, 255}, // 15: Bright White
}

func inkFromAttr(attr byte) color.RGBA {
	idx := int(attr & 0x07)
	if attr&0x40 != 0 {
		idx += 8
	}
	return spectrumPalette[idx]
}

func paperFromAttr(attr byte) color.RGBA {
	idx := int((attr >> 3) & 0x07)
	if attr&0x40 != 0 {
		idx += 8
	}
	return spectrumPalette[idx]
}

// Renderer converts screen buffers to Ebitengine images.
type Renderer struct {
	flashState bool
	frameCount int
}

// NewRenderer creates a new Renderer.
func NewRenderer() *Renderer {
	return &Renderer{}
}

// RenderBuffer renders the cavern area (top 128 pixels) from attribute and
// pixel buffers onto the target image.
func (r *Renderer) RenderBuffer(target *ebiten.Image, attrs []byte, pixels []byte) {
	r.frameCount++
	if r.frameCount%16 == 0 {
		r.flashState = !r.flashState
	}

	for cellRow := 0; cellRow < cavernRows; cellRow++ {
		for cellCol := 0; cellCol < cavernCols; cellCol++ {
			attrIdx := cellRow*cavernCols + cellCol
			attr := attrs[attrIdx]

			ink := inkFromAttr(attr)
			paper := paperFromAttr(attr)

			if attr&0x80 != 0 && r.flashState {
				ink, paper = paper, ink
			}

			for pixRow := 0; pixRow < 8; pixRow++ {
				pixIdx := cellRow*256 + pixRow*32 + cellCol
				pixByte := pixels[pixIdx]

				y := cellRow*8 + pixRow
				for bit := 7; bit >= 0; bit-- {
					x := cellCol*8 + (7 - bit)
					var c color.RGBA
					if pixByte&(1<<uint(bit)) != 0 {
						c = ink
					} else {
						c = paper
					}
					target.Set(x, y, c)
				}
			}
		}
	}
}

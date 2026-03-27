package entity

import (
	"manicminer/cavern"
	"manicminer/screen"
)

// Kong is the special entity in caverns 7 and 11 (Kong Beast).
// Two switches control wall opening and floor removal. When the right switch
// is flipped, Kong falls. When he reaches y=100, he dies.
type Kong struct {
	Status int // 0 = on ledge, 1 = falling, 2 = dead.
	PixelY int // Pixel y-coordinate (only used when falling).
	Active bool
}

// NewKong creates a Kong Beast.
func NewKong() *Kong {
	return &Kong{
		Status: 0,
		PixelY: 0,
		Active: true,
	}
}

// MoveAndDraw updates and draws the Kong Beast.
// Returns true if Kong collided with Willy.
func (k *Kong) MoveAndDraw(cav *cavern.Cavern, gameClock byte,
	attrs []byte, pixels []byte, score []byte) bool {

	if !k.Active || k.Status == 2 {
		return false
	}

	if k.Status == 1 {
		// Kong is falling.
		if k.PixelY >= 100 {
			// Reached portal — Kong is dead.
			k.Status = 2
			return false
		}

		k.PixelY += 4

		// Draw falling Kong using alternating sprite (game clock bit 5).
		spriteOffset := int(gameClock&32) | 0x40
		if spriteOffset+32 > len(cav.GuardianGraphics) {
			return false
		}
		spriteData := cav.GuardianGraphics[spriteOffset : spriteOffset+32]

		cellX := 15
		screen.DrawSprite(pixels, k.PixelY, cellX, spriteData, screen.DrawOverwrite)

		// Add 100 to score while falling.
		AddToScore(score, 7, 1)

		// Set attribute bytes (yellow INK).
		cellY := k.PixelY / 8
		bgAttr := cav.Background.Attr
		attr := (bgAttr & 0xF8) | 0x06 // Yellow INK.
		for dy := 0; dy < 3; dy++ {
			for dx := 0; dx < 2; dx++ {
				cy := cellY + dy
				cx := cellX + dx
				if cx >= 0 && cx < 32 && cy >= 0 && cy < 16 {
					attrs[cy*32+cx] = attr
				}
			}
		}

		return false
	}

	// Kong is on the ledge (status 0). Draw at fixed position (0,15).
	spriteOffset := int(gameClock & 32) // Alternating sprite using bit 5.
	if spriteOffset+32 > len(cav.GuardianGraphics) {
		return false
	}
	spriteData := cav.GuardianGraphics[spriteOffset : spriteOffset+32]

	cellX := 15
	pixelY := 0
	collision := screen.DrawSprite(pixels, pixelY, cellX, spriteData, screen.DrawBlend)

	// Set attribute bytes (INK 4, PAPER 0, BRIGHT 1 = 0x44).
	positions := [][2]int{{15, 0}, {16, 0}, {15, 1}, {16, 1}}
	for _, p := range positions {
		cx, cy := p[0], p[1]
		if cx >= 0 && cx < 32 && cy >= 0 && cy < 16 {
			attrs[cy*32+cx] = 0x44
		}
	}

	return collision
}

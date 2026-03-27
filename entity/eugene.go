package entity

import (
	"manicminer/cavern"
	"manicminer/screen"
)

// Eugene is the special entity in Eugene's Lair (cavern 4).
// He bounces vertically between y=0 and y=0x58 (the portal position).
// While items remain, he's drawn with white INK and blocks the portal.
// When all items are collected, his INK cycles and the portal activates.
type Eugene struct {
	PixelY    int  // Pixel y-coordinate.
	Direction int  // 0 = moving down, 1 = moving up.
	Active    bool
}

// NewEugene creates Eugene for cavern 4.
func NewEugene() *Eugene {
	return &Eugene{
		PixelY:    0,
		Direction: 0, // Start moving down.
		Active:    true,
	}
}

// MoveAndDraw updates Eugene's position and draws him.
// Returns true if Eugene collided with Willy.
func (e *Eugene) MoveAndDraw(cav *cavern.Cavern, lastItemAttr byte,
	gameClock byte, attrs []byte, pixels []byte) bool {

	if !e.Active {
		return false
	}

	allCollected := lastItemAttr == 0

	// Move Eugene.
	if allCollected || e.Direction == 0 {
		// Move down.
		e.PixelY++
		if e.PixelY >= 0x58 {
			// Reached portal position — toggle direction.
			e.Direction ^= 1
		}
	} else {
		// Move up (items still remain, direction is up).
		e.PixelY--
		if e.PixelY <= 0 {
			// Reached top — toggle direction.
			e.Direction ^= 1
		}
	}

	// Draw Eugene using the guardian graphic data at offset 0xE0 (224).
	// The original uses DE=$80E0, where E=$E0 and D=high(GuardianGraphicData).
	// This points to frame 7 of the guardian graphics (offset 7*32 = 224).
	spriteOffset := 224
	if spriteOffset+32 > len(cav.GuardianGraphics) {
		return false
	}
	spriteData := cav.GuardianGraphics[spriteOffset : spriteOffset+32]

	// Draw at x=15 (OR $0F in original), y=PixelY.
	cellX := 15
	collision := screen.DrawSprite(pixels, e.PixelY, cellX, spriteData, screen.DrawBlend)

	// Set attribute bytes.
	cellY := e.PixelY / 8
	var inkColour byte
	if !allCollected {
		inkColour = 0x07 // White INK while items remain.
	} else {
		// Cycle INK colour using game clock bits 2-4.
		inkColour = (gameClock >> 2) & 0x07
	}

	bgAttr := cav.Background.Attr
	attr := (bgAttr & 0xF8) | inkColour

	// Set 2x3 attribute block.
	for dy := 0; dy < 3; dy++ {
		for dx := 0; dx < 2; dx++ {
			cy := cellY + dy
			cx := cellX + dx
			if cx >= 0 && cx < 32 && cy >= 0 && cy < 16 {
				attrs[cy*32+cx] = attr
			}
		}
	}

	return collision
}

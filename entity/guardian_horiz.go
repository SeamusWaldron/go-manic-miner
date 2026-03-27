package entity

import (
	"manicminer/cavern"
	"manicminer/screen"
)

// HorizGuardian holds the runtime state of a horizontal guardian.
type HorizGuardian struct {
	Attr       byte // Bit 7: speed flag, bits 0-6: attribute.
	PosLSB     byte // LSB of attribute buffer position (encodes x within row).
	PosRow     byte // Row portion of attribute buffer address (encodes y).
	ScreenHi   byte // MSB of screen buffer address.
	Frame      byte // Animation frame 0-7. Frames 0-3: moving right, 4-7: moving left.
	LeftBound  byte // LSB of leftmost point.
	RightBound byte // LSB of rightmost point.
	Active     bool
}

// NewHorizGuardians creates horizontal guardians from the cavern definition.
func NewHorizGuardians(cav *cavern.Cavern) []HorizGuardian {
	guards := make([]HorizGuardian, 0, 4)
	for i := 0; i < cav.NumHorizGuardians; i++ {
		def := cav.HorizGuardians[i]
		guards = append(guards, HorizGuardian{
			Attr:       def.Attr,
			PosLSB:     def.AttrBufLo,
			PosRow:     def.AttrBufHi,
			ScreenHi:   def.ScreenBufHi,
			Frame:      def.Frame,
			LeftBound:  def.LeftBound,
			RightBound: def.RightBound,
			Active:     true,
		})
	}
	return guards
}

// MoveHorizGuardians updates the position/frame of all horizontal guardians.
// gameClock is needed for the speed flag check.
func MoveHorizGuardians(guards []HorizGuardian, gameClock byte) {
	for i := range guards {
		g := &guards[i]
		if !g.Active {
			continue
		}

		// Speed check: if bit 7 of attr is set (slow speed), only move when
		// game clock bit 2 is clear.
		if g.Attr&0x80 != 0 {
			// Compute the speed gate: game clock bit 2 rotated to bit 7.
			speedBit := ((gameClock & 0x04) >> 2) << 7
			if speedBit&g.Attr != 0 {
				continue // Not this guardian's turn to move.
			}
		}

		frame := g.Frame
		if frame == 3 {
			// Terminal frame moving right — cross boundary or turn.
			if g.PosLSB == g.RightBound {
				// Reached rightmost point — turn around.
				g.Frame = 7
			} else {
				// Move right across cell boundary.
				g.Frame = 0
				g.PosLSB++
			}
		} else if frame == 4 {
			// Terminal frame moving left — cross boundary or turn.
			if g.PosLSB == g.LeftBound {
				// Reached leftmost point — turn around.
				g.Frame = 0
			} else {
				// Move left across cell boundary.
				g.Frame = 7
				g.PosLSB--
			}
		} else if frame < 3 {
			// Moving right — increment frame.
			g.Frame++
		} else {
			// Frame 5, 6, or 7 — moving left, decrement frame.
			g.Frame--
		}
	}
}

// DrawHorizGuardians draws all horizontal guardians into the attribute and
// pixel buffers. Returns true if any guardian collided with Willy.
func DrawHorizGuardians(guards []HorizGuardian, cav *cavern.Cavern, cavernNum int,
	attrs []byte, pixels []byte) bool {

	collision := false

	for i := range guards {
		g := &guards[i]
		if !g.Active {
			continue
		}

		// Decode position: PosLSB contains the x position within the row,
		// PosRow encodes the row.
		// In the original, the address is $5Cxx where xx = PosLSB.
		// Row = (PosRow - $5C) * 2 + high bits of PosLSB.
		// Simplified: cellX = PosLSB & 0x1F, cellY = (PosLSB >> 5) + (PosRow - $5C) * 8
		cellX := int(g.PosLSB & 0x1F)
		// The row is encoded across PosLSB bits 5-7 and PosRow.
		// PosRow is the MSB of the attr buffer address ($5C or $5D).
		rowOffset := int(g.PosLSB>>5) + (int(g.PosRow)-0x5C)*8
		cellY := rowOffset
		if cellY < 0 || cellY >= 16 || cellX < 0 || cellX >= 31 {
			continue
		}

		// Set attribute bytes in the buffer (2x2 cells).
		attr := g.Attr & 0x7F // Strip speed bit.
		setGuardianAttrs(attrs, cellX, cellY, attr)

		// Determine sprite data offset.
		// Frame 0-7, multiply by 32 to get offset into guardian graphics.
		frameOffset := int(g.Frame) * 32

		// In caverns >= 7 (except 9 and 15), guardians use frames 4-7 base.
		if cavernNum >= 7 && cavernNum != 9 && cavernNum != 15 {
			frameOffset += 128
		}

		// Clamp to 256-byte guardian graphic area.
		if frameOffset >= 256 {
			frameOffset -= 256
		}

		spriteData := cav.GuardianGraphics[frameOffset : frameOffset+32]

		// Calculate pixel Y from cell Y.
		pixelY := cellY * 8

		// Draw the guardian sprite in blend mode.
		if screen.DrawSprite(pixels, pixelY, cellX, spriteData, screen.DrawBlend) {
			collision = true
		}
	}

	return collision
}

// setGuardianAttrs sets the 2x2 attribute block for a guardian.
func setGuardianAttrs(attrs []byte, cellX, cellY int, attr byte) {
	positions := [][2]int{
		{cellX, cellY},
		{cellX + 1, cellY},
		{cellX, cellY + 1},
		{cellX + 1, cellY + 1},
	}
	for _, p := range positions {
		x, y := p[0], p[1]
		if x >= 0 && x < 32 && y >= 0 && y < 16 {
			attrs[y*32+x] = attr
		}
	}
}

package entity

import (
	"manicminer/cavern"
	"manicminer/screen"
)

// VertGuardian holds the runtime state of a vertical guardian.
type VertGuardian struct {
	Attr       byte
	Frame      byte // Animation frame 0-3.
	PixelY     int  // Pixel y-coordinate.
	CellX      int  // Cell x-coordinate.
	YIncrement int  // Signed y-increment per frame.
	MinY       int  // Minimum pixel y-coordinate.
	MaxY       int  // Maximum pixel y-coordinate.
	Active     bool
}

// NewVertGuardians creates vertical guardians from the cavern definition.
func NewVertGuardians(cav *cavern.Cavern) []VertGuardian {
	guards := make([]VertGuardian, 0, 4)
	for i := 0; i < cav.NumVertGuardians; i++ {
		def := cav.VertGuardians[i]
		guards = append(guards, VertGuardian{
			Attr:       def.Attr,
			Frame:      def.Frame,
			PixelY:     int(def.PixelY),
			CellX:      int(def.X),
			YIncrement: int(def.YIncrement),
			MinY:       int(def.MinY),
			MaxY:       int(def.MaxY),
			Active:     true,
		})
	}
	return guards
}

// MoveVertGuardians updates the position/frame of all vertical guardians.
func MoveVertGuardians(guards []VertGuardian) {
	for i := range guards {
		g := &guards[i]
		if !g.Active {
			continue
		}

		// Increment animation frame (0→1→2→3→0).
		g.Frame++
		if g.Frame >= 4 {
			g.Frame = 0
		}

		// Move vertically.
		newY := g.PixelY + g.YIncrement
		if newY < g.MinY || newY >= g.MaxY {
			// Reverse direction.
			g.YIncrement = -g.YIncrement
		} else {
			g.PixelY = newY
		}
	}
}

// DrawVertGuardians draws all vertical guardians into the attribute and pixel
// buffers. Returns true if any guardian collided with Willy.
func DrawVertGuardians(guards []VertGuardian, cav *cavern.Cavern,
	attrs []byte, pixels []byte) bool {

	collision := false

	for i := range guards {
		g := &guards[i]
		if !g.Active {
			continue
		}

		// Calculate sprite data offset from frame (0-3).
		frameOffset := int(g.Frame) * 32
		if frameOffset+32 > len(cav.GuardianGraphics) {
			continue
		}
		spriteData := cav.GuardianGraphics[frameOffset : frameOffset+32]

		// Draw the guardian sprite in blend mode.
		if screen.DrawSprite(pixels, g.PixelY, g.CellX, spriteData, screen.DrawBlend) {
			collision = true
		}

		// Set attribute bytes.
		// Calculate cell Y from pixel Y.
		cellY := g.PixelY / 8
		attr := g.Attr

		// Set 2x3 or 2x2 attribute block depending on pixel alignment.
		rows := 2
		if g.PixelY%8 != 0 {
			rows = 3
		}
		for dy := 0; dy < rows; dy++ {
			for dx := 0; dx < 2; dx++ {
				cy := cellY + dy
				cx := g.CellX + dx
				if cx >= 0 && cx < 32 && cy >= 0 && cy < 16 {
					// Combine guardian's INK with background PAPER.
					bgAttr := cav.Background.Attr
					newAttr := (bgAttr & 0xF8) | (attr & 0x07)
					attrs[cy*32+cx] = newAttr
				}
			}
		}
	}

	return collision
}

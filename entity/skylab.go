package entity

import (
	"manicminer/cavern"
	"manicminer/screen"
)

// Skylab is the special entity in Skylab Landing Bay (cavern 13).
// Skylabs fall downward, disintegrate on impact, then respawn shifted right.
// They use the vertical guardian slots but have different movement logic.
type Skylab struct {
	Attr   byte
	Frame  byte // Animation frame 0-7. 0 = falling, 1-7 = disintegrating.
	PixelY int  // Current pixel y-coordinate.
	CellX  int  // Current cell x-coordinate.
	Speed  int  // Y increment per frame.
	StartY int  // Starting pixel y-coordinate (for respawn).
	MaxY   int  // Crash site pixel y-coordinate.
	Active bool
}

// NewSkylabs creates Skylabs from vertical guardian slots for cavern 13.
func NewSkylabs(cav *cavern.Cavern) []Skylab {
	skylabs := make([]Skylab, 0, 4)
	for i := 0; i < cav.NumVertGuardians; i++ {
		def := cav.VertGuardians[i]
		skylabs = append(skylabs, Skylab{
			Attr:   def.Attr,
			Frame:  def.Frame,
			PixelY: int(def.PixelY),
			CellX:  int(def.X),
			Speed:  int(def.YIncrement),
			StartY: int(def.MinY),
			MaxY:   int(def.MaxY),
			Active: true,
		})
	}
	return skylabs
}

// MoveAndDrawSkylabs updates and draws all Skylabs.
// Returns true if any Skylab collided with Willy.
func MoveAndDrawSkylabs(skylabs []Skylab, cav *cavern.Cavern,
	attrs []byte, pixels []byte) bool {

	collision := false

	for i := range skylabs {
		s := &skylabs[i]
		if !s.Active {
			continue
		}

		if s.PixelY < s.MaxY {
			// Still falling.
			s.PixelY += s.Speed
		} else {
			// At crash site — disintegrate.
			s.Frame++
			if s.Frame >= 8 {
				// Fully disintegrated — respawn.
				s.PixelY = s.StartY
				s.CellX = (s.CellX + 8) & 0x1F // Shift right by 8, wrap at 32.
				s.Frame = 0
				continue
			}
		}

		// Draw the Skylab sprite.
		spriteOffset := int(s.Frame) * 32
		if spriteOffset+32 > len(cav.GuardianGraphics) {
			continue
		}
		spriteData := cav.GuardianGraphics[spriteOffset : spriteOffset+32]

		if screen.DrawSprite(pixels, s.PixelY, s.CellX, spriteData, screen.DrawBlend) {
			collision = true
		}

		// Set attribute bytes.
		cellY := s.PixelY / 8
		bgAttr := cav.Background.Attr
		attr := (bgAttr & 0xF8) | (s.Attr & 0x07)
		for dy := 0; dy < 3; dy++ {
			for dx := 0; dx < 2; dx++ {
				cy := cellY + dy
				cx := s.CellX + dx
				if cx >= 0 && cx < 32 && cy >= 0 && cy < 16 {
					attrs[cy*32+cx] = attr
				}
			}
		}
	}

	return collision
}

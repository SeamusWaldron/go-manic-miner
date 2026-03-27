package entity

import (
	"manicminer/cavern"
	"manicminer/screen"
)

// Portal holds the runtime state of the level exit portal.
type Portal struct {
	Attr          byte // Attribute byte. Bit 7 = flashing (active).
	CellX         int
	CellY         int
	Graphic       [32]byte
	AttrBufAddr   uint16
	ScreenBufAddr uint16
}

// NewPortal creates a portal from the cavern definition.
func NewPortal(cav *cavern.Cavern) *Portal {
	// Decode position from attribute buffer address.
	addr := cav.Portal.AttrBufAddr
	base := uint16(0x5C00)
	offset := addr - base
	cellY := int(offset / 32)
	cellX := int(offset % 32)

	p := &Portal{
		Attr:          cav.Portal.Attr,
		CellX:         cellX,
		CellY:         cellY,
		AttrBufAddr:   cav.Portal.AttrBufAddr,
		ScreenBufAddr: cav.Portal.ScreenBufAddr,
	}
	copy(p.Graphic[:], cav.Portal.Graphic[:])
	return p
}

// ActivateFlash sets the FLASH bit on the portal (called when all items collected).
func (p *Portal) ActivateFlash() {
	p.Attr |= 0x80
}

// IsFlashing returns true if the portal is flashing (all items collected).
func (p *Portal) IsFlashing() bool {
	return p.Attr&0x80 != 0
}

// CheckEntry returns true if Willy has entered the portal.
func (p *Portal) CheckEntry(willy *Willy) bool {
	if !p.IsFlashing() {
		return false
	}
	return willy.CellX == p.CellX && willy.CellY == p.CellY
}

// Draw draws the portal into the attribute and pixel buffers.
func (p *Portal) Draw(attrs []byte, pixels []byte) {
	// Set the 2x2 attribute block.
	positions := [][2]int{
		{p.CellX, p.CellY},
		{p.CellX + 1, p.CellY},
		{p.CellX, p.CellY + 1},
		{p.CellX + 1, p.CellY + 1},
	}
	for _, pos := range positions {
		x, y := pos[0], pos[1]
		if x >= 0 && x < 32 && y >= 0 && y < 16 {
			attrs[y*32+x] = p.Attr
		}
	}

	// Draw the portal graphic (16x16 sprite) into the pixel buffer.
	pixelY := p.CellY * 8
	screen.DrawSprite(pixels, pixelY, p.CellX, p.Graphic[:], screen.DrawOverwrite)
}

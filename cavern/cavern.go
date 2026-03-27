package cavern

// TileDef holds one tile definition: an attribute byte and 8 pixel-row bytes.
type TileDef struct {
	Attr   byte
	Pixels [8]byte
}

// ItemDef holds one item definition.
type ItemDef struct {
	Attr       byte   // Current attribute (0 = collected).
	AttrBufLo  byte   // LSB of attribute buffer address.
	AttrBufHi  byte   // MSB of attribute buffer address.
	ScreenBufHi byte  // MSB of screen buffer address.
}

// HorizGuardianDef holds one horizontal guardian definition.
type HorizGuardianDef struct {
	Attr       byte // Bit 7: speed, bits 0-6: attribute.
	AttrBufLo  byte
	AttrBufHi  byte
	ScreenBufHi byte
	Frame      byte
	LeftBound  byte
	RightBound byte
}

// VertGuardianDef holds one vertical guardian definition.
type VertGuardianDef struct {
	Attr       byte
	Frame      byte
	PixelY     byte
	X          byte
	YIncrement int8 // Signed.
	MinY       byte
	MaxY       byte
}

// PortalDef holds the portal definition.
type PortalDef struct {
	Attr        byte
	Graphic     [32]byte
	AttrBufAddr uint16
	ScreenBufAddr uint16
}

// Cavern holds all parsed data for a single cavern.
type Cavern struct {
	// Tile attribute grid: 16 rows x 32 cols.
	Attributes [512]byte

	// Cavern name (32 chars, padded with spaces).
	Name string

	// Tile definitions (8 tiles).
	Background    TileDef
	Floor         TileDef
	CrumblingFloor TileDef
	Wall          TileDef
	Conveyor      TileDef
	Nasty1        TileDef
	Nasty2        TileDef
	Extra         TileDef

	// All tiles as a slice for lookup.
	Tiles [8]*TileDef

	// Willy's initial state.
	WillyPixelY    byte
	WillyFrame     byte
	WillyDir       byte
	WillyAirborne  byte
	WillyAttrAddr  uint16
	WillyJumpCount byte

	// Conveyor.
	ConveyorDir    byte
	ConveyorAddr   uint16
	ConveyorLength byte

	// Border colour.
	BorderColour byte

	// Items (up to 5).
	Items [5]ItemDef
	NumItems int

	// Portal.
	Portal PortalDef

	// Item graphic (8 bytes).
	ItemGraphic [8]byte

	// Air supply and game clock.
	Air       byte
	GameClock byte

	// Horizontal guardians (up to 4).
	HorizGuardians [4]HorizGuardianDef
	NumHorizGuardians int

	// Vertical guardians (up to 4).
	VertGuardians [4]VertGuardianDef
	NumVertGuardians int

	// Guardian graphic data (256 bytes = 8 frames x 32 bytes).
	GuardianGraphics [256]byte
}

// FindTileByAttr returns the tile definition whose attribute matches, or nil.
func (c *Cavern) FindTileByAttr(attr byte) *TileDef {
	for _, t := range c.Tiles {
		if t.Attr == attr {
			return t
		}
	}
	return nil
}

// Load loads a cavern by number (0-19) from the embedded data.
func Load(num int) *Cavern {
	if num < 0 || num >= len(allCavernData) {
		return nil
	}
	data := allCavernData[num]
	if len(data) < 1024 {
		return nil
	}
	return parseCavern(data)
}

// parseCavern decodes a 1024-byte cavern definition.
func parseCavern(data []byte) *Cavern {
	c := &Cavern{}

	// Bytes 0-511: attribute grid.
	copy(c.Attributes[:], data[0:512])

	// Bytes 512-543: cavern name.
	c.Name = string(data[512:544])

	// Bytes 544-615: 8 tile definitions (9 bytes each).
	tiles := [8]*TileDef{
		&c.Background, &c.Floor, &c.CrumblingFloor, &c.Wall,
		&c.Conveyor, &c.Nasty1, &c.Nasty2, &c.Extra,
	}
	offset := 544
	for i := 0; i < 8; i++ {
		tiles[i].Attr = data[offset]
		copy(tiles[i].Pixels[:], data[offset+1:offset+9])
		offset += 9
	}
	c.Tiles = tiles

	// Bytes 616-622: Willy initial state.
	c.WillyPixelY = data[616]
	c.WillyFrame = data[617]
	c.WillyDir = data[618]
	c.WillyAirborne = data[619]
	c.WillyAttrAddr = uint16(data[620]) | uint16(data[621])<<8
	c.WillyJumpCount = data[622]

	// Bytes 623-626: conveyor.
	c.ConveyorDir = data[623]
	c.ConveyorAddr = uint16(data[624]) | uint16(data[625])<<8
	c.ConveyorLength = data[626]

	// Byte 627: border colour.
	c.BorderColour = data[627]
	// Byte 628: unused.

	// Bytes 629-653: items (5 slots of 5 bytes, terminated by $FF).
	offset = 629
	c.NumItems = 0
	for i := 0; i < 5; i++ {
		if data[offset] == 0xFF {
			break
		}
		c.Items[i] = ItemDef{
			Attr:        data[offset],
			AttrBufLo:   data[offset+1],
			AttrBufHi:   data[offset+2],
			ScreenBufHi: data[offset+3],
			// data[offset+4] is unused ($FF).
		}
		c.NumItems++
		offset += 5
	}

	// Bytes 655-691: portal (1 + 32 + 2 + 2 = 37 bytes).
	offset = 655
	c.Portal.Attr = data[offset]
	copy(c.Portal.Graphic[:], data[offset+1:offset+33])
	c.Portal.AttrBufAddr = uint16(data[offset+33]) | uint16(data[offset+34])<<8
	c.Portal.ScreenBufAddr = uint16(data[offset+35]) | uint16(data[offset+36])<<8

	// Bytes 692-699: item graphic.
	copy(c.ItemGraphic[:], data[692:700])

	// Bytes 700-701: air supply and game clock.
	c.Air = data[700]
	c.GameClock = data[701]

	// Bytes 702-729: horizontal guardians (4 x 7 bytes).
	offset = 702
	c.NumHorizGuardians = 0
	for i := 0; i < 4; i++ {
		if data[offset] == 0xFF {
			break
		}
		if data[offset] == 0x00 {
			offset += 7
			continue
		}
		c.HorizGuardians[c.NumHorizGuardians] = HorizGuardianDef{
			Attr:        data[offset],
			AttrBufLo:   data[offset+1],
			AttrBufHi:   data[offset+2],
			ScreenBufHi: data[offset+3],
			Frame:       data[offset+4],
			LeftBound:   data[offset+5],
			RightBound:  data[offset+6],
		}
		c.NumHorizGuardians++
		offset += 7
	}

	// Byte 730: horizontal guardian terminator.
	// Bytes 731-732: unused.
	// Byte 733: vertical guardian start or terminator.

	// Bytes 733-767: vertical guardians area.
	offset = 733
	c.NumVertGuardians = 0
	for i := 0; i < 4; i++ {
		if offset+7 > len(data) || data[offset] == 0xFF {
			break
		}
		if data[offset] == 0x00 {
			offset += 7
			continue
		}
		c.VertGuardians[c.NumVertGuardians] = VertGuardianDef{
			Attr:       data[offset],
			Frame:      data[offset+1],
			PixelY:     data[offset+2],
			X:          data[offset+3],
			YIncrement: int8(data[offset+4]),
			MinY:       data[offset+5],
			MaxY:       data[offset+6],
		}
		c.NumVertGuardians++
		offset += 7
	}

	// Bytes 768-1023: guardian graphics (256 bytes).
	if len(data) >= 1024 {
		copy(c.GuardianGraphics[:], data[768:1024])
	}

	return c
}

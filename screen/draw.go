package screen

import "manicminer/cavern"

// DrawCavernToBuffer fills the pixel buffer by looking up each attribute byte
// in the attribute buffer, finding the matching tile, and copying its 8 pixel
// rows into the correct position in the pixel buffer.
//
// This replicates the original DrawCurrentCavernToScreenBuffer routine.
func DrawCavernToBuffer(cav *cavern.Cavern, attrs []byte, pixels []byte) {
	for cellRow := 0; cellRow < 16; cellRow++ {
		for cellCol := 0; cellCol < 32; cellCol++ {
			attrIdx := cellRow*32 + cellCol
			attr := attrs[attrIdx]

			// Find the tile whose attribute byte matches.
			tile := cav.FindTileByAttr(attr)

			// Copy 8 pixel rows into the pixel buffer.
			for pixRow := 0; pixRow < 8; pixRow++ {
				pixIdx := cellRow*256 + pixRow*32 + cellCol
				if tile != nil {
					pixels[pixIdx] = tile.Pixels[pixRow]
				} else {
					pixels[pixIdx] = 0
				}
			}
		}
	}
}

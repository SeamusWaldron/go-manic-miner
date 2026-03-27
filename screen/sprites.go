package screen

// DrawMode controls how sprites interact with the background.
type DrawMode int

const (
	DrawOverwrite DrawMode = iota // C=0 in original: overwrite background.
	DrawBlend                     // C=1 in original: OR with background, AND to detect collision.
	DrawOR                        // OR onto background, no collision detection (used for Willy).
)

// DrawSprite draws a 16x16 pixel sprite into the pixel buffer at the given
// position. Returns true if a collision was detected (DrawBlend mode only).
//
// spriteData is 32 bytes: 2 bytes per row, 16 rows. Even bytes are the left
// column, odd bytes are the right column.
//
// pixelY is the pixel y-coordinate (0-127).
// cellX is the cell x-coordinate (0-31) for the left column of the sprite.
func DrawSprite(pixels []byte, pixelY int, cellX int, spriteData []byte, mode DrawMode) bool {
	if len(spriteData) < 32 {
		return false
	}

	for row := 0; row < 16; row++ {
		py := pixelY + row
		if py < 0 || py >= 128 {
			continue
		}

		leftByte := spriteData[row*2]
		rightByte := spriteData[row*2+1]

		leftIdx := YTable[py] + cellX
		rightIdx := YTable[py] + cellX + 1

		// Left cell.
		if cellX >= 0 && cellX < 32 && leftIdx >= 0 && leftIdx < len(pixels) {
			switch mode {
			case DrawOverwrite:
				pixels[leftIdx] = leftByte
			case DrawBlend:
				if pixels[leftIdx]&leftByte != 0 {
					return true // Collision detected.
				}
				pixels[leftIdx] |= leftByte
			case DrawOR:
				pixels[leftIdx] |= leftByte
			}
		}

		// Right cell.
		rightX := cellX + 1
		if rightX >= 0 && rightX < 32 && rightIdx >= 0 && rightIdx < len(pixels) {
			switch mode {
			case DrawOverwrite:
				pixels[rightIdx] = rightByte
			case DrawBlend:
				if pixels[rightIdx]&rightByte != 0 {
					return true // Collision detected.
				}
				pixels[rightIdx] |= rightByte
			case DrawOR:
				pixels[rightIdx] |= rightByte
			}
		}
	}

	return false
}

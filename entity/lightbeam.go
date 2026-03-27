package entity

import "manicminer/cavern"

// DrawLightBeam traces the light beam in Solar Power Generator (cavern 18).
// The beam starts at cell (0,23), travels downward, reflects off guardians
// (toggles between down and left), stops on floor/wall tiles, and drains
// air when touching Willy (attribute $27 = INK 7, PAPER 4).
//
// Returns the number of extra air decrements caused by the beam hitting Willy.
func DrawLightBeam(cav *cavern.Cavern, attrs []byte) int {
	airDrain := 0

	// Starting position: row 0, column 23.
	pos := 0*32 + 23 // Attribute buffer index.
	delta := 32       // Moving down (one row = 32 bytes in attr buffer).

	for {
		if pos < 0 || pos >= len(attrs) {
			break
		}

		attr := attrs[pos]

		// Stop on floor or wall.
		if attr == cav.Floor.Attr || attr == cav.Wall.Attr {
			break
		}

		// Check if hitting Willy (INK 7, PAPER 4 = 0x27).
		if attr == 0x27 {
			airDrain = 4 // Drain air 4 extra times.
			// Draw beam over Willy and continue.
			attrs[pos] = 0x77 // INK 7, PAPER 6, BRIGHT 1.
			pos += delta
			continue
		}

		// Check if background — beam passes through.
		if attr == cav.Background.Attr {
			attrs[pos] = 0x77 // Draw beam.
			pos += delta
			continue
		}

		// Hit something else (guardian, item, etc.) — reflect.
		// Toggle direction between down (delta=32) and left (delta=-1).
		if delta == 32 {
			delta = -1 // Switch to moving left.
		} else {
			delta = 32 // Switch to moving down.
		}

		attrs[pos] = 0x77 // Draw beam at reflection point.
		pos += delta
	}

	return airDrain
}

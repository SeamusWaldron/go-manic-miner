package entity

import (
	"manicminer/action"
	"manicminer/cavern"
	"manicminer/data"
	"manicminer/screen"
)

// Movement table: maps (currentState + inputOffset) to new state.
// States: 0=right/still, 1=left/still, 2=right/moving, 3=left/moving.
// Input offsets: 0=none, 4=left, 8=right, 12=both.
var movementTable = [16]byte{
	0, 1, 0, 1, // No input
	1, 3, 1, 3, // Left
	2, 0, 2, 0, // Right
	0, 1, 2, 3, // Both
}

// Willy holds Miner Willy's complete state.
type Willy struct {
	// Y2 is the y-coordinate in the original's doubled format (WillysPixelYCoord).
	// Actual pixel Y = Y2 / 2. Cell-aligned when Y2 & 0x0F == 0 (i.e. pixel Y multiple of 8).
	Y2        int
	AnimFrame byte // Animation frame 0-3.
	DirFlags  byte // Bit 0: direction (0=right, 1=left), bit 1: moving.
	Airborne  int  // 0=grounded, 1=jumping, 2-11=falling safe, 12+=fatal, 255=dead.
	CellX     int  // X position in cells (0-31).
	CellY     int  // Y position in cell rows (0-15). Derived from Y2.
	JumpCount int  // Jump animation counter (0-17).
	Alive     bool
}

// NewWilly creates Willy from the cavern's initial state.
func NewWilly(cav *cavern.Cavern) *Willy {
	addr := cav.WillyAttrAddr
	base := uint16(0x5C00)
	offset := addr - base
	cellY := int(offset / 32)
	cellX := int(offset % 32)

	return &Willy{
		Y2:        int(cav.WillyPixelY), // Already in y*2 format.
		AnimFrame: cav.WillyFrame,
		DirFlags:  cav.WillyDir,
		Airborne:  int(cav.WillyAirborne),
		CellX:     cellX,
		CellY:     cellY,
		JumpCount: int(cav.WillyJumpCount),
		Alive:     true,
	}
}

// PixelY returns the actual pixel y-coordinate (0-127).
func (w *Willy) PixelY() int { return w.Y2 / 2 }

// IsCellAligned returns true when Willy's top edge is on a cell boundary.
// In original: Y2 & 0x0F == 0, meaning pixel Y is a multiple of 8.
func (w *Willy) IsCellAligned() bool { return w.Y2&0x0F == 0 }

func (w *Willy) Direction() int  { return int(w.DirFlags & 1) }
func (w *Willy) IsMoving() bool  { return w.DirFlags&2 != 0 }

// SpriteData returns the 32-byte sprite for Willy's current frame/direction.
func (w *Willy) SpriteData() []byte {
	idx := w.Direction()*4 + int(w.AnimFrame&3)
	s := data.WillySprites[idx]
	return s[:]
}

// Update processes one frame of Willy logic.
func (w *Willy) Update(inp action.Action, cav *cavern.Cavern,
	emptyAttrs []byte, emptyPixels []byte, workAttrs []byte) {
	if !w.Alive {
		return
	}
	w.moveWilly1(cav, emptyAttrs, emptyPixels)
	w.moveWilly2(inp, cav, emptyAttrs)
}

// moveWilly1 handles jumping and falling.
func (w *Willy) moveWilly1(cav *cavern.Cavern, attrs []byte, pixels []byte) {
	if w.Airborne == 1 {
		w.handleJump(cav, attrs, pixels)
		return
	}
	if w.Airborne == 255 {
		return
	}
	w.checkGround(cav, attrs, pixels)
}

// handleJump processes one frame of the jump arc.
// Delta is applied to Y2 (the doubled coordinate), matching the original Z80 code.
func (w *Willy) handleJump(cav *cavern.Cavern, attrs []byte, pixels []byte) {
	// Original: A = (counter & ~1) - 8; WillysPixelYCoord += A
	delta := (w.JumpCount & ^1) - 8
	w.Y2 += delta
	if w.Y2 < 0 {
		w.Y2 = 0
	}
	w.syncCellY()

	// Check wall collision at top of sprite.
	if w.checkWallAbove(cav, attrs) {
		// Snap Y2 so pixel Y aligns to next cell boundary below wall.
		// Original: ADD A,16; AND 240 (applied to pixel Y, i.e. Y2/2)
		pxY := w.PixelY()
		pxY = ((pxY / 16) + 1) * 16
		if pxY > 112 {
			pxY = 112
		}
		w.Y2 = pxY * 2
		w.syncCellY()
		w.Airborne = 2
		w.DirFlags &^= 2
		return
	}

	w.JumpCount++
	if w.JumpCount >= 18 {
		w.Airborne = 6
		return
	}

	// At counter 13 or 16, check if Willy can land.
	if w.JumpCount == 13 || w.JumpCount == 16 {
		w.checkGround(cav, attrs, pixels)
	}
}

// checkGround determines if Willy is on solid ground or should fall.
func (w *Willy) checkGround(cav *cavern.Cavern, attrs []byte, pixels []byte) {
	if !w.IsCellAligned() {
		if w.Airborne == 1 {
			return // Mid-jump, not aligned — skip.
		}
		if w.Airborne >= 2 {
			w.continueFalling()
		}
		return
	}

	// Cells below Willy's sprite (Willy occupies 2 cell rows when aligned).
	belowY := w.CellY + 2
	if belowY >= 16 {
		w.Kill()
		return
	}

	leftIdx := belowY*32 + w.CellX
	rightIdx := belowY*32 + w.CellX + 1
	if rightIdx >= len(attrs) {
		return
	}

	leftAttr := attrs[leftIdx]
	rightAttr := attrs[rightIdx]

	// Animate crumbling floors.
	if leftAttr == cav.CrumblingFloor.Attr {
		AnimateCrumblingFloor(cav, attrs, pixels, leftIdx)
	}
	if rightAttr == cav.CrumblingFloor.Attr {
		AnimateCrumblingFloor(cav, attrs, pixels, rightIdx)
	}

	// Re-read after crumble.
	leftAttr = attrs[leftIdx]
	rightAttr = attrs[rightIdx]

	// A tile is "empty" if it's background, nasty, or crumbling (still counts as walkable
	// in the original — nasties kill via attribute check, not ground check).
	leftEmpty := leftAttr == cav.Background.Attr ||
		leftAttr == cav.Nasty1.Attr ||
		leftAttr == cav.Nasty2.Attr
	rightEmpty := rightAttr == cav.Background.Attr ||
		rightAttr == cav.Nasty1.Attr ||
		rightAttr == cav.Nasty2.Attr

	if leftEmpty && rightEmpty {
		if w.Airborne == 1 {
			return // Still in jump arc.
		}
		if w.Airborne == 0 {
			w.Airborne = 2
			w.DirFlags &^= 2
		} else {
			w.continueFalling()
		}
		return
	}

	// Solid ground below — land Willy.
	// In the original, this path jumps to MoveWilly2 which always resets
	// AirborneStatusIndicator to 0, regardless of whether Willy was jumping
	// (Airborne=1) or falling (Airborne>=2). This allows Willy to land on
	// platforms mid-jump at the JC=13 and JC=16 checkpoints.
	if w.Airborne >= 12 && w.Airborne != 255 {
		w.Kill()
		return
	}
	w.Airborne = 0
}

// continueFalling moves Willy down and increments airborne counter.
// Original: Y2 += 8 (i.e. pixel Y += 4).
func (w *Willy) continueFalling() {
	w.Airborne++
	w.Y2 += 8
	if w.Y2 > 240 {
		w.Y2 = 240
	}
	w.syncCellY()
}

// checkWallAbove returns true if the top cell row of Willy's sprite overlaps a wall.
func (w *Willy) checkWallAbove(cav *cavern.Cavern, attrs []byte) bool {
	if w.CellY < 0 || w.CellY >= 16 {
		return false
	}
	li := w.CellY*32 + w.CellX
	ri := w.CellY*32 + w.CellX + 1
	if li >= 0 && li < len(attrs) && attrs[li] == cav.Wall.Attr {
		return true
	}
	if ri >= 0 && ri < len(attrs) && attrs[ri] == cav.Wall.Attr {
		return true
	}
	return false
}

// moveWilly2 handles left/right movement and jump initiation.
func (w *Willy) moveWilly2(inp action.Action, cav *cavern.Cavern, attrs []byte) {
	if w.Airborne == 255 {
		return
	}

	// Conveyor effect (only when grounded and cell-aligned).
	convLeft, convRight := false, false
	if w.Airborne == 0 && w.IsCellAligned() {
		belowY := w.CellY + 2
		if belowY < 16 {
			for dx := 0; dx < 2; dx++ {
				idx := belowY*32 + w.CellX + dx
				if idx < len(attrs) && attrs[idx] == cav.Conveyor.Attr {
					if cav.ConveyorDir == 0 {
						convLeft = true
					} else {
						convRight = true
					}
				}
			}
		}
	}

	wantLeft := inp.Left || convLeft
	wantRight := inp.Right || convRight

	var inputOffset byte
	if wantLeft && wantRight {
		inputOffset = 12
	} else if wantRight {
		inputOffset = 8
	} else if wantLeft {
		inputOffset = 4
	}

	w.DirFlags = movementTable[w.DirFlags+inputOffset]

	// Jump initiation.
	if inp.Jump && w.Airborne == 0 {
		w.JumpCount = 0
		w.Airborne = 1
	}

	if !w.IsMoving() {
		return
	}

	if w.Direction() == 1 {
		w.moveLeft(cav, attrs)
	} else {
		w.moveRight(cav, attrs)
	}
}

func (w *Willy) moveLeft(cav *cavern.Cavern, attrs []byte) {
	if w.AnimFrame == 0 {
		newX := w.CellX - 1
		if newX < 0 || w.isWallAt(cav, attrs, newX, w.CellY) ||
			w.isWallAt(cav, attrs, newX, w.CellY+1) {
			return
		}
		if !w.IsCellAligned() && w.CellY+2 < 16 && w.isWallAt(cav, attrs, newX, w.CellY+2) {
			return
		}
		w.CellX = newX
		w.AnimFrame = 3
	} else {
		w.AnimFrame--
	}
}

func (w *Willy) moveRight(cav *cavern.Cavern, attrs []byte) {
	if w.AnimFrame == 3 {
		checkX := w.CellX + 2
		if checkX >= 32 || w.isWallAt(cav, attrs, checkX, w.CellY) ||
			w.isWallAt(cav, attrs, checkX, w.CellY+1) {
			return
		}
		if !w.IsCellAligned() && w.CellY+2 < 16 && w.isWallAt(cav, attrs, checkX, w.CellY+2) {
			return
		}
		w.CellX++
		w.AnimFrame = 0
	} else {
		w.AnimFrame++
	}
}

func (w *Willy) isWallAt(cav *cavern.Cavern, attrs []byte, x, y int) bool {
	if x < 0 || x >= 32 || y < 0 || y >= 16 {
		return false
	}
	return attrs[y*32+x] == cav.Wall.Attr
}

func (w *Willy) Kill() {
	w.Airborne = 255
	w.Alive = false
}

// CheckNasties checks if Willy's sprite overlaps any nasty tiles.
func (w *Willy) CheckNasties(cav *cavern.Cavern, attrs []byte) {
	rows := 2
	if !w.IsCellAligned() {
		rows = 3
	}
	for dy := 0; dy < rows; dy++ {
		for dx := 0; dx < 2; dx++ {
			cy := w.CellY + dy
			cx := w.CellX + dx
			if cy < 0 || cy >= 16 || cx < 0 || cx >= 32 {
				continue
			}
			a := attrs[cy*32+cx]
			if a == cav.Nasty1.Attr || a == cav.Nasty2.Attr {
				w.Kill()
				return
			}
		}
	}
}

// SetAttributes sets white INK on Willy's cells in the attribute buffer.
//
// The original always checks 3 rows of cells (6 cells). For the top 4 cells
// (rows 0-1), it forces white INK. For the bottom 2 cells (row 2), it only
// sets white INK if Willy is NOT cell-aligned (i.e. the sprite spills into
// the third row). When aligned, the sprite fits in exactly 2 cell rows so
// the third row is skipped.
func (w *Willy) SetAttributes(cav *cavern.Cavern, attrs []byte) {
	for dy := 0; dy < 3; dy++ {
		// Skip the third row when cell-aligned (sprite doesn't reach it).
		if dy == 2 && w.IsCellAligned() {
			continue
		}
		for dx := 0; dx < 2; dx++ {
			cy := w.CellY + dy
			cx := w.CellX + dx
			if cy < 0 || cy >= 16 || cx < 0 || cx >= 32 {
				continue
			}
			idx := cy*32 + cx
			if attrs[idx] == cav.Background.Attr {
				attrs[idx] = (cav.Background.Attr & 0xF8) | 0x07
			}
		}
	}
}

// Draw draws Willy's sprite into the pixel buffer using OR (no collision).
func (w *Willy) Draw(pixels []byte) {
	screen.DrawSprite(pixels, w.PixelY(), w.CellX, w.SpriteData(), screen.DrawOR)
}

// syncCellY recalculates CellY from Y2.
func (w *Willy) syncCellY() {
	w.CellY = w.PixelY() / 8
}

// AnimateCrumblingFloor shifts pixel rows down in the persistent buffer.
// When the bottom row is empty, the tile becomes background.
func AnimateCrumblingFloor(cav *cavern.Cavern, attrs []byte, pixels []byte, attrIdx int) {
	cellY := attrIdx / 32
	cellX := attrIdx % 32
	if cellY < 0 || cellY >= 16 || cellX < 0 || cellX >= 32 {
		return
	}

	// Shift pixel rows down.
	for row := 7; row >= 1; row-- {
		src := cellY*256 + (row-1)*32 + cellX
		dst := cellY*256 + row*32 + cellX
		if src >= 0 && src < len(pixels) && dst >= 0 && dst < len(pixels) {
			pixels[dst] = pixels[src]
		}
	}
	// Clear top row.
	top := cellY*256 + cellX
	if top >= 0 && top < len(pixels) {
		pixels[top] = 0
	}

	// Replace with background when bottom row is empty.
	bottom := cellY*256 + 7*32 + cellX
	if bottom >= 0 && bottom < len(pixels) && pixels[bottom] == 0 {
		attrs[attrIdx] = cav.Background.Attr
	}
}

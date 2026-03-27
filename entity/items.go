package entity

import (
	"manicminer/cavern"
	"manicminer/screen"
)

// Item holds the runtime state of a collectible item.
type Item struct {
	Attr      byte // Current attribute (0 = collected).
	CellX     int
	CellY     int
	ScreenHi  byte
	Collected bool
}

// NewItems creates items from the cavern definition.
func NewItems(cav *cavern.Cavern) []Item {
	items := make([]Item, 0, 5)
	for i := 0; i < cav.NumItems; i++ {
		def := cav.Items[i]
		cellX := int(def.AttrBufLo & 0x1F)
		rowOffset := int(def.AttrBufLo>>5) + (int(def.AttrBufHi)-0x5C)*8
		items = append(items, Item{
			Attr:     def.Attr,
			CellX:    cellX,
			CellY:    rowOffset,
			ScreenHi: def.ScreenBufHi,
		})
	}
	return items
}

// DrawAndCollectItems draws all items and checks if Willy is touching any.
// Returns the attribute of the last item drawn (0 if all collected).
func DrawAndCollectItems(items []Item, cav *cavern.Cavern,
	attrs []byte, pixels []byte, score []byte) byte {

	var lastAttrDrawn byte

	for i := range items {
		it := &items[i]
		if it.Collected {
			continue
		}

		attrIdx := it.CellY*32 + it.CellX
		if attrIdx < 0 || attrIdx >= len(attrs) {
			continue
		}

		// Check if Willy's white INK overlay touches the item.
		if attrs[attrIdx]&0x07 == 0x07 {
			it.Collected = true
			it.Attr = 0
			AddToScore(score, 7, 1) // +100 points (add 1 to hundreds digit).
			continue
		}

		// Cycle the item's INK colour: keep BRIGHT+PAPER, cycle INK 3→4→5→6.
		ink := it.Attr & 0x07
		base := it.Attr & 0xF8
		ink++
		if ink > 6 {
			ink = 3
		}
		it.Attr = base | ink

		attrs[attrIdx] = it.Attr
		lastAttrDrawn = it.Attr

		// Draw the item graphic into the pixel buffer.
		pixelY := it.CellY * 8
		if pixelY >= 0 && pixelY < 128 && it.CellX >= 0 && it.CellX < 32 {
			for row := 0; row < 8; row++ {
				idx := screen.YTable[pixelY+row] + it.CellX
				if idx >= 0 && idx < len(pixels) {
					pixels[idx] = cav.ItemGraphic[row]
				}
			}
		}
	}

	return lastAttrDrawn
}

// AddToScore adds a value to a specific digit position in the ASCII score,
// propagating carries leftward. pos is the digit index (0=leftmost).
func AddToScore(score []byte, pos int, value int) {
	for value > 0 && pos >= 0 {
		d := int(score[pos]-'0') + value
		score[pos] = byte(d%10) + '0'
		value = d / 10
		pos--
	}
}

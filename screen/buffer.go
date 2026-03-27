package screen

// YTable maps a pixel y-coordinate (0-127) to the offset within the pixel buffer
// where that pixel row begins. This replicates the ZX Spectrum's non-linear
// screen memory layout, but linearised for our buffer.
//
// In the original, YTable entries are absolute addresses into the screen buffer.
// Here we use offsets: for pixel row y, the offset is computed so that the 32
// bytes for that row are contiguous.
//
// The layout matches the Spectrum: the buffer is organised as 16 character rows
// of 8 pixel rows each, with 32 bytes per pixel row. The offset for pixel
// y-coordinate p is:
//
//	charRow = p / 8
//	pixelRow = p % 8
//	offset = charRow*256 + pixelRow*32
//
// This gives a total buffer size of 16*256 = 4096 bytes.
var YTable [128]int

func init() {
	for p := 0; p < 128; p++ {
		charRow := p / 8
		pixelRow := p % 8
		YTable[p] = charRow*256 + pixelRow*32
	}
}

// PixelBufOffset returns the byte offset in the pixel buffer for a given
// pixel y-coordinate and cell x-coordinate (0-31).
func PixelBufOffset(pixelY, cellX int) int {
	if pixelY < 0 || pixelY >= 128 || cellX < 0 || cellX >= 32 {
		return -1
	}
	return YTable[pixelY] + cellX
}

// AttrBufOffset returns the byte offset in the attribute buffer for a given
// cell row (0-15) and cell column (0-31).
func AttrBufOffset(cellRow, cellCol int) int {
	return cellRow*32 + cellCol
}

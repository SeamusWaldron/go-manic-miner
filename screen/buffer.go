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

// SpectrumDisplayToLinear converts raw ZX Spectrum display file data into
// our linearised pixel buffer format.
//
// The Spectrum display file for the top two-thirds (4096 bytes, $4000-$4FFF)
// uses an interleaved layout:
//
//	Address bits: 010T TRRR CCCL LLLL
//	  T  = third (0-1, covering char rows 0-7 and 8-15)
//	  R  = pixel row within character cell (0-7)
//	  C  = character row within third (0-7)
//	  L  = column (0-31)
//
// Our buffer uses: charRow*256 + pixelRow*32 + column
//
// spectrumData must be 4096 bytes. linearBuf must be 4096 bytes.
func SpectrumDisplayToLinear(spectrumData []byte, linearBuf []byte) {
	for i := 0; i < 4096; i++ {
		// Decode Spectrum address (offset from $4000).
		third := (i >> 11) & 1         // bit 11
		pixelRow := (i >> 8) & 7       // bits 8-10
		charRowInThird := (i >> 5) & 7 // bits 5-7
		column := i & 31               // bits 0-4

		charRow := third*8 + charRowInThird
		linearOffset := charRow*256 + pixelRow*32 + column

		if linearOffset < len(linearBuf) {
			linearBuf[linearOffset] = spectrumData[i]
		}
	}
}

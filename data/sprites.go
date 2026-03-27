package data

// Willy sprite data: 8 frames, each 32 bytes (2 bytes wide x 16 rows).
//
// The original stores these as right-facing frames 0-3, then left-facing frames 0-3.
// Frame layout in the original memory:
//   Offset 0x00: Right frame 0 (sprite shifted leftmost)
//   Offset 0x20: Right frame 1
//   Offset 0x40: Right frame 2 (WillySpriteData1)
//   Offset 0x60: Right frame 3 (WillySpriteData2)
//   Offset 0x80: Left frame 3
//   Offset 0xA0: Left frame 2
//   Offset 0xC0: Left frame 1
//   Offset 0xE0: Left frame 0
//
// In our Go code we index as:
//   WillySprites[direction*4 + frame]
// where direction 0=right, 1=left, frame 0-3.

var WillySprites [8][32]byte

func init() {
	// Right frame 0
	WillySprites[0] = dgFrame([][16]bool{
		parseDG("-----##---------"),
		parseDG("--#####---------"),
		parseDG("-#####----------"),
		parseDG("--##-#----------"),
		parseDG("--#####---------"),
		parseDG("--####----------"),
		parseDG("---##-----------"),
		parseDG("--####----------"),
		parseDG("-######---------"),
		parseDG("-######---------"),
		parseDG("####-###--------"),
		parseDG("#####-##--------"),
		parseDG("--####----------"),
		parseDG("-###-##---------"),
		parseDG("-##-###---------"),
		parseDG("-###-###--------"),
	})

	// Right frame 1
	WillySprites[1] = dgFrame([][16]bool{
		parseDG("-------##-------"),
		parseDG("----#####-------"),
		parseDG("---#####--------"),
		parseDG("----##-#--------"),
		parseDG("----#####-------"),
		parseDG("----####--------"),
		parseDG("-----##---------"),
		parseDG("----####--------"),
		parseDG("---##-###-------"),
		parseDG("---##-###-------"),
		parseDG("---##-###-------"),
		parseDG("---###-##-------"),
		parseDG("----####--------"),
		parseDG("-----##---------"),
		parseDG("-----##---------"),
		parseDG("-----###--------"),
	})

	// Right frame 2
	WillySprites[2] = dgFrame([][16]bool{
		parseDG("---------##-----"),
		parseDG("------#####-----"),
		parseDG("-----#####------"),
		parseDG("------##-#------"),
		parseDG("------#####-----"),
		parseDG("------####------"),
		parseDG("-------##-------"),
		parseDG("------####------"),
		parseDG("-----######-----"),
		parseDG("-----######-----"),
		parseDG("----####-###----"),
		parseDG("----#####-##----"),
		parseDG("------####------"),
		parseDG("-----###-##-----"),
		parseDG("-----##-###-----"),
		parseDG("-----###-###----"),
	})

	// Right frame 3
	WillySprites[3] = dgFrame([][16]bool{
		parseDG("-----------##---"),
		parseDG("--------#####---"),
		parseDG("-------#####----"),
		parseDG("--------##-#----"),
		parseDG("--------#####---"),
		parseDG("--------####----"),
		parseDG("---------##-----"),
		parseDG("--------####----"),
		parseDG("-------######---"),
		parseDG("------########--"),
		parseDG("-----##########-"),
		parseDG("-----##-####-##-"),
		parseDG("--------#####---"),
		parseDG("-------###-##-#-"),
		parseDG("------##----###-"),
		parseDG("------###----#--"),
	})

	// Left frame 0 (mirror of right frame 3 position)
	WillySprites[4] = dgFrame([][16]bool{
		parseDG("---##-----------"),
		parseDG("---#####--------"),
		parseDG("----#####-------"),
		parseDG("----#-##--------"),
		parseDG("---#####--------"),
		parseDG("----####--------"),
		parseDG("-----##---------"),
		parseDG("----####--------"),
		parseDG("---######-------"),
		parseDG("--########------"),
		parseDG("-##########-----"),
		parseDG("-##-####-##-----"),
		parseDG("---#####--------"),
		parseDG("-#-##-###-------"),
		parseDG("-###----##------"),
		parseDG("--#----###------"),
	})

	// Left frame 1
	WillySprites[5] = dgFrame([][16]bool{
		parseDG("-----##---------"),
		parseDG("-----#####------"),
		parseDG("------#####-----"),
		parseDG("------#-##------"),
		parseDG("-----#####------"),
		parseDG("------####------"),
		parseDG("-------##-------"),
		parseDG("------####------"),
		parseDG("-----######-----"),
		parseDG("-----######-----"),
		parseDG("----###-####----"),
		parseDG("----##-#####----"),
		parseDG("------####------"),
		parseDG("-----##-###-----"),
		parseDG("-----###-##-----"),
		parseDG("----###-###-----"),
	})

	// Left frame 2
	WillySprites[6] = dgFrame([][16]bool{
		parseDG("-------##-------"),
		parseDG("-------#####----"),
		parseDG("--------#####---"),
		parseDG("--------#-##----"),
		parseDG("-------#####----"),
		parseDG("--------####----"),
		parseDG("---------##-----"),
		parseDG("--------####----"),
		parseDG("-------######---"),
		parseDG("-------###-##---"),
		parseDG("-------###-##---"),
		parseDG("-------##-###---"),
		parseDG("--------####----"),
		parseDG("---------##-----"),
		parseDG("---------##-----"),
		parseDG("--------###-----"),
	})

	// Left frame 3
	WillySprites[7] = dgFrame([][16]bool{
		parseDG("---------##-----"),
		parseDG("---------#####--"),
		parseDG("----------#####-"),
		parseDG("----------#-##--"),
		parseDG("---------#####--"),
		parseDG("----------####--"),
		parseDG("-----------##---"),
		parseDG("----------####--"),
		parseDG("---------######-"),
		parseDG("---------######-"),
		parseDG("--------###-####"),
		parseDG("--------##-#####"),
		parseDG("----------####--"),
		parseDG("---------##-###-"),
		parseDG("---------###-##-"),
		parseDG("--------###-###-"),
	})
}

// parseDG converts a 16-char DG string to a row of 16 booleans.
func parseDG(s string) [16]bool {
	var row [16]bool
	for i := 0; i < 16 && i < len(s); i++ {
		row[i] = s[i] == '#'
	}
	return row
}

// dgFrame converts 16 rows of 16-bit pixel data into 32 bytes (2 bytes per row).
func dgFrame(rows [][16]bool) [32]byte {
	var frame [32]byte
	for r := 0; r < 16 && r < len(rows); r++ {
		var hi, lo byte
		for bit := 0; bit < 8; bit++ {
			if rows[r][bit] {
				hi |= 1 << uint(7-bit)
			}
		}
		for bit := 0; bit < 8; bit++ {
			if rows[r][8+bit] {
				lo |= 1 << uint(7-bit)
			}
		}
		frame[r*2] = hi
		frame[r*2+1] = lo
	}
	return frame
}

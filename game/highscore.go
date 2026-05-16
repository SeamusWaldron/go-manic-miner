package game

import (
	"fmt"
	"image/color"

	"manicminer/config"
	"manicminer/screen"

	"github.com/hajimehoshi/ebiten/v2"
)

// HighScoreScreen displays the high score table.
type HighScoreScreen struct {
	debounce int
}

func newHighScoreScreen() *HighScoreScreen {
	return &HighScoreScreen{debounce: 16} // Brief pause before accepting input.
}

func (h *HighScoreScreen) update() bool {
	if h.debounce > 0 {
		h.debounce--
		return false
	}
	// Any key returns to title.
	for k := ebiten.KeyA; k <= ebiten.KeyZ; k++ {
		if ebiten.IsKeyPressed(k) {
			return true
		}
	}
	if ebiten.IsKeyPressed(ebiten.KeyEnter) || ebiten.IsKeyPressed(ebiten.KeySpace) ||
		ebiten.IsKeyPressed(ebiten.KeyEscape) {
		return true
	}
	return false
}

func (h *HighScoreScreen) draw(display *ebiten.Image, cfg *config.Config, frameCount int) {
	display.Fill(color.Black)

	cyan := byte(0x45)
	yellow := byte(0x46)
	white := byte(0x47)
	green := byte(0x44)

	screen.PrintMessage(display, 7*8, 1*8, "MANIC MINER", cyan)
	screen.PrintMessage(display, 7*8, 3*8, "HIGH SCORES", yellow)

	colours := []byte{white, yellow, cyan, green, white, yellow, cyan, green, white, yellow}

	for i, hs := range cfg.HighScores {
		row := 5 + i
		attr := colours[i%len(colours)]
		cavernName := config.CavernName(hs.Cavern)
		if len(cavernName) > 12 {
			cavernName = cavernName[:12]
		}
		line := fmt.Sprintf("%2d. %06d %s %-12s", i+1, hs.Score, hs.Name, cavernName)
		screen.PrintMessage(display, 1*8, row*8, line, attr)
	}

	// Fill empty slots.
	for i := len(cfg.HighScores); i < 10; i++ {
		row := 5 + i
		line := fmt.Sprintf("%2d. ------ --- ------------", i+1)
		screen.PrintMessage(display, 1*8, row*8, line, 0x40) // Dim.
	}

	// Flash "press any key".
	flashAttr := yellow
	if frameCount/12%2 == 0 {
		flashAttr = yellow | 0x80
	}
	screen.PrintMessage(display, 3*8, 20*8, "PRESS ANY KEY TO CONTINUE", flashAttr)
}

// NameEntryScreen handles entering a name for a new high score OR a
// new speedrun record. When IsSpeedrun is true the screen displays the
// time (in centiseconds) instead of a score, and the wrapper routes
// the saved entry into the speedrun records table.
type NameEntryScreen struct {
	Name             [3]byte
	Cursor           int // 0-2.
	Score            int
	Cavern           int
	debounce         int
	Done             bool
	IsSpeedrun       bool
	SpeedrunIsOverall bool
	SpeedrunCs       int64
}

func newNameEntryScreen(score int, cavern int, defaultName string) *NameEntryScreen {
	n := &NameEntryScreen{
		Score:  score,
		Cavern: cavern,
	}
	for i := 0; i < 3; i++ {
		if i < len(defaultName) {
			n.Name[i] = defaultName[i]
		} else {
			n.Name[i] = 'A'
		}
	}
	return n
}

// newSpeedrunNameEntryScreen creates a name-entry screen for a new
// speedrun record. The wrapper distinguishes via IsSpeedrun.
func newSpeedrunNameEntryScreen(cs int64, cavern int, isOverall bool, defaultName string) *NameEntryScreen {
	n := newNameEntryScreen(0, cavern, defaultName)
	n.IsSpeedrun = true
	n.SpeedrunIsOverall = isOverall
	n.SpeedrunCs = cs
	return n
}

func (n *NameEntryScreen) update() {
	if n.debounce > 0 {
		n.debounce--
		return
	}

	if ebiten.IsKeyPressed(ebiten.KeyArrowLeft) && n.Cursor > 0 {
		n.Cursor--
		n.debounce = 6
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowRight) && n.Cursor < 2 {
		n.Cursor++
		n.debounce = 6
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowUp) {
		n.Name[n.Cursor]++
		if n.Name[n.Cursor] > 'Z' {
			n.Name[n.Cursor] = 'A'
		}
		n.debounce = 4
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowDown) {
		n.Name[n.Cursor]--
		if n.Name[n.Cursor] < 'A' {
			n.Name[n.Cursor] = 'Z'
		}
		n.debounce = 4
	}
	if ebiten.IsKeyPressed(ebiten.KeyEnter) {
		n.Done = true
		n.debounce = 8
	}
}

func (n *NameEntryScreen) draw(display *ebiten.Image, frameCount int) {
	display.Fill(color.Black)

	yellow := byte(0x46)
	cyan := byte(0x45)
	white := byte(0x47)

	// Flash header — adapts to the entry type.
	flashAttr := yellow
	if frameCount/8%2 == 0 {
		flashAttr = yellow | 0x80
	}
	if n.IsSpeedrun {
		header := "NEW SPEEDRUN TIME!"
		x := (256 - len(header)*8) / 2
		screen.PrintMessage(display, x, 3*8, header, flashAttr)
		body := fmt.Sprintf("TIME  %s", formatSpeedrunTime(n.SpeedrunCs))
		bx := (256 - len(body)*8) / 2
		screen.PrintMessage(display, bx, 6*8, body, white)
	} else {
		screen.PrintMessage(display, 6*8, 3*8, "NEW HIGH SCORE!", flashAttr)
		screen.PrintMessage(display, 9*8, 6*8, fmt.Sprintf("SCORE: %06d", n.Score), white)
	}
	screen.PrintMessage(display, 7*8, 9*8, "ENTER YOUR NAME", cyan)

	// Name characters.
	for i := 0; i < 3; i++ {
		charAttr := white
		if n.Cursor == i && frameCount/6%2 == 0 {
			charAttr = white | 0x80
		}
		screen.PrintMessage(display, (13+i*2)*8, 12*8, string(n.Name[i]), charAttr)
	}

	screen.PrintMessage(display, 2*8, 18*8, "LEFT/RIGHT Select position", yellow)
	screen.PrintMessage(display, 2*8, 19*8, "UP/DOWN    Change letter", yellow)
	screen.PrintMessage(display, 2*8, 20*8, "ENTER      Confirm", yellow)
}

func (n *NameEntryScreen) nameString() string {
	return string(n.Name[:])
}

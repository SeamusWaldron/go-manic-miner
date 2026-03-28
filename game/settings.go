package game

import (
	"image/color"

	"manicminer/config"
	"manicminer/screen"

	"github.com/hajimehoshi/ebiten/v2"
)

// SettingsScreen handles the settings menu state and rendering.
type SettingsScreen struct {
	cursor      int  // Currently selected item (0-9).
	debounce    int  // Key repeat debounce counter.
	nameCursor  int  // 0-2 for player name character selection.
	editingName bool // True when editing the player name.
}

const (
	settingsItemControls0 = 0 // Original
	settingsItemControls1 = 1 // Arrows + Space
	settingsItemControls2 = 2 // O/P + Space
	settingsItemInfLives  = 3
	settingsItemInfAir    = 4
	settingsItemHarmless  = 5
	settingsItemNoNasties = 6
	settingsItemNoGuards  = 7
	settingsItemWarp      = 8
	settingsItemName      = 9
	settingsItemCount     = 10
)

func newSettingsScreen() *SettingsScreen {
	return &SettingsScreen{debounce: 16} // Initial debounce to ignore key from title transition.
}

func (s *SettingsScreen) update(cfg *config.Config) bool {
	if s.debounce > 0 {
		s.debounce--
		return false
	}

	// Escape returns to title.
	if ebiten.IsKeyPressed(ebiten.KeyEscape) || ebiten.IsKeyPressed(ebiten.KeyBackspace) {
		s.debounce = 8
		return true
	}

	// Navigation — support both arrow keys and common alternatives.
	up := ebiten.IsKeyPressed(ebiten.KeyArrowUp) || ebiten.IsKeyPressed(ebiten.KeyK)
	down := ebiten.IsKeyPressed(ebiten.KeyArrowDown) || ebiten.IsKeyPressed(ebiten.KeyJ)

	if up && s.cursor > 0 {
		s.cursor--
		s.editingName = false
		s.debounce = 6
	}
	if down && s.cursor < settingsItemCount-1 {
		s.cursor++
		s.editingName = false
		s.debounce = 6
	}

	// Enter to toggle/select.
	if ebiten.IsKeyPressed(ebiten.KeyEnter) {
		s.debounce = 8
		switch s.cursor {
		case settingsItemControls0:
			cfg.ControlScheme = config.ControlOriginal
		case settingsItemControls1:
			cfg.ControlScheme = config.ControlArrows
		case settingsItemControls2:
			cfg.ControlScheme = config.ControlOP
		case settingsItemInfLives:
			cfg.Features.InfiniteLives = !cfg.Features.InfiniteLives
		case settingsItemInfAir:
			cfg.Features.InfiniteAir = !cfg.Features.InfiniteAir
		case settingsItemHarmless:
			cfg.Features.HarmlessHeights = !cfg.Features.HarmlessHeights
		case settingsItemNoNasties:
			cfg.Features.NoNasties = !cfg.Features.NoNasties
		case settingsItemNoGuards:
			cfg.Features.NoGuardians = !cfg.Features.NoGuardians
		case settingsItemWarp:
			cfg.Features.WarpMode = !cfg.Features.WarpMode
		case settingsItemName:
			s.editingName = !s.editingName
			s.nameCursor = 0
		}
	}

	// Name editing: type A-Z directly. Each letter advances to next position.
	if s.editingName && s.cursor == settingsItemName {
		name := []byte(cfg.PlayerName)
		for len(name) < 3 {
			name = append(name, 'A')
		}
		// Check for A-Z key presses.
		for k := ebiten.KeyA; k <= ebiten.KeyZ; k++ {
			if ebiten.IsKeyPressed(k) {
				letter := byte('A') + byte(k-ebiten.KeyA)
				name[s.nameCursor] = letter
				s.nameCursor++
				if s.nameCursor >= 3 {
					s.nameCursor = 0
					s.editingName = false
				}
				s.debounce = 6
				break
			}
		}
		// Backspace to go back one position.
		if ebiten.IsKeyPressed(ebiten.KeyBackspace) && s.nameCursor > 0 {
			s.nameCursor--
			s.debounce = 6
		}
		cfg.PlayerName = string(name[:3])
	}

	return false
}

func (s *SettingsScreen) draw(display *ebiten.Image, cfg *config.Config, frameCount int) {
	display.Fill(color.Black)

	yellow := byte(0x46)  // INK 6, BRIGHT 1.
	cyan := byte(0x45)    // INK 5, BRIGHT 1.
	green := byte(0x44)   // INK 4, BRIGHT 1.
	red := byte(0x42)     // INK 2, BRIGHT 1.
	white := byte(0x47)   // INK 7, BRIGHT 1.
	dim := byte(0x07)     // INK 7, normal brightness.

	// Selected item gets bright white, unselected gets dim.
	itemAttr := func(item int) byte {
		if s.cursor == item {
			return white
		}
		return dim
	}

	// Draw blinking cursor `>` next to selected item.
	cursorChar := ">"
	if frameCount/6%2 == 0 {
		cursorChar = " "
	}

	screen.PrintMessage(display, 2*8, 1*8, "MANIC MINER SETTINGS", cyan)

	// Controls section.
	screen.PrintMessage(display, 2*8, 3*8, "CONTROLS", yellow)

	schemes := []struct {
		label  string
		scheme config.ControlScheme
		item   int
	}{
		{"Original (QWERT/POIUY)", config.ControlOriginal, settingsItemControls0},
		{"Arrows + Space", config.ControlArrows, settingsItemControls1},
		{"O/P + Space", config.ControlOP, settingsItemControls2},
	}
	for _, sc := range schemes {
		row := 5 + sc.item
		active := " "
		if cfg.ControlScheme == sc.scheme {
			active = "*"
		}
		// Cursor.
		if s.cursor == sc.item {
			screen.PrintMessage(display, 1*8, row*8, cursorChar, yellow)
		}
		screen.PrintMessage(display, 2*8, row*8, "("+active+")", itemAttr(sc.item))
		screen.PrintMessage(display, 6*8, row*8, sc.label, itemAttr(sc.item))
	}

	// Cheats section.
	screen.PrintMessage(display, 2*8, 9*8, "CHEATS", yellow)

	toggles := []struct {
		label string
		on    bool
		item  int
	}{
		{"Infinite Lives", cfg.Features.InfiniteLives, settingsItemInfLives},
		{"Infinite Air", cfg.Features.InfiniteAir, settingsItemInfAir},
		{"Harmless Heights", cfg.Features.HarmlessHeights, settingsItemHarmless},
		{"No Nasties", cfg.Features.NoNasties, settingsItemNoNasties},
		{"No Guardians", cfg.Features.NoGuardians, settingsItemNoGuards},
		{"Warp Mode", cfg.Features.WarpMode, settingsItemWarp},
	}
	for _, t := range toggles {
		row := 11 + (t.item - settingsItemInfLives)
		// Cursor.
		if s.cursor == t.item {
			screen.PrintMessage(display, 1*8, row*8, cursorChar, yellow)
		}
		screen.PrintMessage(display, 2*8, row*8, t.label, itemAttr(t.item))
		if t.on {
			screen.PrintMessage(display, 22*8, row*8, "[ON ]", green)
		} else {
			screen.PrintMessage(display, 22*8, row*8, "[OFF]", red)
		}
	}

	// Player name.
	row := 18
	if s.cursor == settingsItemName {
		screen.PrintMessage(display, 1*8, row*8, cursorChar, yellow)
	}
	screen.PrintMessage(display, 2*8, row*8, "PLAYER NAME:", itemAttr(settingsItemName))
	name := cfg.PlayerName
	for len(name) < 3 {
		name += "A"
	}
	for i := 0; i < 3; i++ {
		charAttr := cyan
		if s.editingName && s.nameCursor == i {
			// Blinking cursor on active character.
			if frameCount/4%2 == 0 {
				charAttr = white
			} else {
				charAttr = red
			}
		}
		screen.PrintMessage(display, (16+i*2)*8, row*8, string(name[i]), charAttr)
	}

	// Help text.
	screen.PrintMessage(display, 2*8, 21*8, "UP/DOWN Navigate", yellow)
	screen.PrintMessage(display, 2*8, 22*8, "ENTER  Select/Toggle", yellow)
	screen.PrintMessage(display, 2*8, 23*8, "ESC    Back to title", yellow)
}

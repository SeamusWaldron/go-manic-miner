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
	return &SettingsScreen{}
}

func (s *SettingsScreen) update(cfg *config.Config) bool {
	if s.debounce > 0 {
		s.debounce--
		return false
	}

	// Escape returns to title.
	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		s.debounce = 8
		return true // Signal to exit settings.
	}

	// Navigation.
	if ebiten.IsKeyPressed(ebiten.KeyArrowUp) && s.cursor > 0 {
		s.cursor--
		s.editingName = false
		s.debounce = 6
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowDown) && s.cursor < settingsItemCount-1 {
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

	// Name editing.
	if s.editingName && s.cursor == settingsItemName {
		if ebiten.IsKeyPressed(ebiten.KeyArrowLeft) && s.nameCursor > 0 {
			s.nameCursor--
			s.debounce = 6
		}
		if ebiten.IsKeyPressed(ebiten.KeyArrowRight) && s.nameCursor < 2 {
			s.nameCursor++
			s.debounce = 6
		}
		name := []byte(cfg.PlayerName)
		for len(name) < 3 {
			name = append(name, 'A')
		}
		if ebiten.IsKeyPressed(ebiten.KeyArrowUp) {
			name[s.nameCursor]++
			if name[s.nameCursor] > 'Z' {
				name[s.nameCursor] = 'A'
			}
			s.debounce = 4
		}
		if ebiten.IsKeyPressed(ebiten.KeyArrowDown) {
			name[s.nameCursor]--
			if name[s.nameCursor] < 'A' {
				name[s.nameCursor] = 'Z'
			}
			s.debounce = 4
		}
		cfg.PlayerName = string(name[:3])
	}

	return false
}

func (s *SettingsScreen) draw(display *ebiten.Image, cfg *config.Config, frameCount int) {
	display.Fill(color.Black)

	white := byte(0x47)      // INK 7, BRIGHT 1.
	yellow := byte(0x46)     // INK 6, BRIGHT 1.
	cyan := byte(0x45)       // INK 5, BRIGHT 1.
	green := byte(0x44)      // INK 4, BRIGHT 1.
	red := byte(0x42)        // INK 2, BRIGHT 1.

	flash := func(attr byte, selected bool) byte {
		if selected && frameCount/8%2 == 0 {
			return attr | 0x80
		}
		return attr
	}

	screen.PrintMessage(display, 3*8, 1*8, "MANIC MINER SETTINGS", cyan)

	// Controls section.
	screen.PrintMessage(display, 2*8, 3*8, "CONTROLS", yellow)

	schemes := []struct {
		label  string
		scheme config.ControlScheme
	}{
		{"Original (QWERT/POIUY)", config.ControlOriginal},
		{"Arrows + Space", config.ControlArrows},
		{"O/P + Space", config.ControlOP},
	}
	for i, sc := range schemes {
		row := 5 + i
		prefix := "  "
		if cfg.ControlScheme == sc.scheme {
			prefix = "> "
		}
		attr := flash(white, s.cursor == i)
		screen.PrintMessage(display, 2*8, row*8, prefix+sc.label, attr)
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
	for i, t := range toggles {
		row := 11 + i
		status := "[ ]"
		statusAttr := red
		if t.on {
			status = "[*]"
			statusAttr = green
		}
		attr := flash(white, s.cursor == t.item)
		screen.PrintMessage(display, 4*8, row*8, t.label, attr)
		screen.PrintMessage(display, 24*8, row*8, status, statusAttr)
	}

	// Player name.
	nameAttr := flash(cyan, s.cursor == settingsItemName)
	screen.PrintMessage(display, 2*8, 18*8, "PLAYER NAME:", nameAttr)
	name := cfg.PlayerName
	for len(name) < 3 {
		name += "A"
	}
	for i := 0; i < 3; i++ {
		charAttr := cyan
		if s.editingName && s.nameCursor == i {
			charAttr = white | 0x80 // Flash the active character.
		}
		screen.PrintMessage(display, (16+i*2)*8, 18*8, string(name[i]), charAttr)
	}

	// Help text.
	screen.PrintMessage(display, 1*8, 22*8, "UP/DOWN Move  ENTER Toggle", yellow)
	screen.PrintMessage(display, 1*8, 23*8, "ESC Back", yellow)
}

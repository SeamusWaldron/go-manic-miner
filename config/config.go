// Package config handles persistent game settings and high scores.
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
)

// ControlScheme identifies the active control mapping.
type ControlScheme string

const (
	ControlOriginal ControlScheme = "original" // QWERT/POIUY/Space
	ControlArrows   ControlScheme = "arrows"   // Arrow keys + Space
	ControlOP       ControlScheme = "op"       // O/P + Space
)

// Features holds cheat/modifier flags.
type Features struct {
	InfiniteLives    bool `json:"infiniteLives"`
	Invulnerable     bool `json:"invulnerable"`
	InfiniteAir      bool `json:"infiniteAir"`
	HarmlessHeights  bool `json:"harmlessHeights"`
	NoNasties        bool `json:"noNasties"`
	NoGuardians      bool `json:"noGuardians"`
	WarpMode         bool `json:"warpMode"`
}

// HighScoreEntry is one entry in the high score table.
type HighScoreEntry struct {
	Name   string `json:"name"`
	Score  int    `json:"score"`
	Cavern int    `json:"cavern"` // Furthest cavern reached.
}

// Speedrun mode constants.
const (
	SpeedrunModeSingle  = 0 // Single cavern.
	SpeedrunModeOverall = 1 // Full game start to finish.
)

// SpeedrunRecord is one entry in a speedrun records table. A zero
// Centiseconds value means the slot is empty (i.e. no record set).
type SpeedrunRecord struct {
	Initials     string `json:"initials"`
	Centiseconds int64  `json:"centiseconds"`
}

// SpeedrunRecords holds top-5 boards for the overall game and for each
// individual cavern. PerCavern is pre-allocated to all 20 caverns.
type SpeedrunRecords struct {
	Overall   [5]SpeedrunRecord       `json:"overall"`
	PerCavern [20][5]SpeedrunRecord   `json:"perCavern"`
}

// Config is the persistent game configuration.
type Config struct {
	PlayerName       string           `json:"playerName"`
	ControlScheme    ControlScheme    `json:"controlScheme"`
	HighScores       []HighScoreEntry `json:"highScores"`
	Features         Features         `json:"features"`
	LastCavern       int              `json:"lastCavern"` // For continue feature.
	SpeedrunEnabled  bool             `json:"speedrunEnabled"`
	SpeedrunMode     int              `json:"speedrunMode"`
	SpeedrunRecords  SpeedrunRecords  `json:"speedrunRecords"`
}

const (
	maxHighScores      = 10
	MaxSpeedrunRecords = 5
)

// DefaultConfig returns a new Config with default values.
func DefaultConfig() *Config {
	return &Config{
		PlayerName:    "AAA",
		ControlScheme: ControlOriginal,
		HighScores:    []HighScoreEntry{},
		Features:      Features{},
	}
}

// configDir returns the directory for storing config files.
func configDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return filepath.Join(home, ".manicminer")
}

// configPath returns the full path to the config file.
func configPath() string {
	return filepath.Join(configDir(), "config.json")
}

// Load reads the config from disk. Returns default config if file doesn't exist.
func Load() *Config {
	data, err := os.ReadFile(configPath())
	if err != nil {
		return DefaultConfig()
	}
	cfg := DefaultConfig()
	if err := json.Unmarshal(data, cfg); err != nil {
		return DefaultConfig()
	}
	return cfg
}

// Save writes the config to disk.
func (c *Config) Save() error {
	dir := configDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath(), data, 0644)
}

// AddHighScore inserts a score into the table if it qualifies.
// Returns the position (0-9) or -1 if it didn't qualify.
func (c *Config) AddHighScore(name string, score int, cavern int) int {
	entry := HighScoreEntry{Name: name, Score: score, Cavern: cavern}
	c.HighScores = append(c.HighScores, entry)

	// Sort descending by score.
	sort.Slice(c.HighScores, func(i, j int) bool {
		return c.HighScores[i].Score > c.HighScores[j].Score
	})

	// Trim to max entries.
	if len(c.HighScores) > maxHighScores {
		c.HighScores = c.HighScores[:maxHighScores]
	}

	// Find position of the new entry.
	for i, hs := range c.HighScores {
		if hs.Name == name && hs.Score == score && hs.Cavern == cavern {
			return i
		}
	}
	return -1
}

// QualifiesForHighScore returns true if the score would make the table.
func (c *Config) QualifiesForHighScore(score int) bool {
	if len(c.HighScores) < maxHighScores {
		return score > 0
	}
	return score > c.HighScores[len(c.HighScores)-1].Score
}

// speedrunBoard returns a pointer to the table for the given mode.
func (c *Config) speedrunBoard(isOverall bool, cavern int) *[5]SpeedrunRecord {
	if isOverall {
		return &c.SpeedrunRecords.Overall
	}
	if cavern < 0 || cavern >= 20 {
		return nil
	}
	return &c.SpeedrunRecords.PerCavern[cavern]
}

// QualifiesAsSpeedrun reports whether the given time is fast enough to
// enter the relevant top-5 board.
func (c *Config) QualifiesAsSpeedrun(isOverall bool, cavern int, cs int64) bool {
	if cs <= 0 {
		return false
	}
	board := c.speedrunBoard(isOverall, cavern)
	if board == nil {
		return false
	}
	slowest := board[MaxSpeedrunRecords-1]
	if slowest.Centiseconds == 0 {
		return true // Empty slot.
	}
	return cs < slowest.Centiseconds
}

// AddSpeedrunRecord inserts a time into the appropriate board if it
// qualifies. Returns the slot index (0..4) or -1 if it did not qualify.
func (c *Config) AddSpeedrunRecord(isOverall bool, cavern int, initials string, cs int64) int {
	if !c.QualifiesAsSpeedrun(isOverall, cavern, cs) {
		return -1
	}
	board := c.speedrunBoard(isOverall, cavern)
	if board == nil {
		return -1
	}
	// Collect all valid entries, append new one, sort ascending by time
	// (empty slots fall to the bottom because their Centiseconds=0 is
	// treated as "missing").
	all := make([]SpeedrunRecord, 0, MaxSpeedrunRecords+1)
	for _, r := range board {
		if r.Centiseconds > 0 {
			all = append(all, r)
		}
	}
	newRec := SpeedrunRecord{Initials: initials, Centiseconds: cs}
	all = append(all, newRec)
	sort.Slice(all, func(i, j int) bool {
		return all[i].Centiseconds < all[j].Centiseconds
	})
	if len(all) > MaxSpeedrunRecords {
		all = all[:MaxSpeedrunRecords]
	}
	// Write back; pad empties.
	var pos = -1
	for i := 0; i < MaxSpeedrunRecords; i++ {
		if i < len(all) {
			board[i] = all[i]
			if pos == -1 && all[i] == newRec {
				pos = i
			}
		} else {
			board[i] = SpeedrunRecord{}
		}
	}
	return pos
}

// FastestForCavern returns the best per-cavern record (empty if none).
func (c *Config) FastestForCavern(cavern int) SpeedrunRecord {
	if cavern < 0 || cavern >= 20 {
		return SpeedrunRecord{}
	}
	return c.SpeedrunRecords.PerCavern[cavern][0]
}

// FastestOverall returns the best overall record (empty if none).
func (c *Config) FastestOverall() SpeedrunRecord {
	return c.SpeedrunRecords.Overall[0]
}

// CavernName returns the name for a cavern number.
func CavernName(num int) string {
	names := []string{
		"Central Cavern", "The Cold Room", "The Menagerie",
		"Abandoned Uranium Workings", "Eugene's Lair", "Processing Plant",
		"The Vat", "Miner Willy meets the Kong Beast", "Wacky Amoebatrons",
		"The Endorian Forest", "Attack of the Mutant Telephones",
		"Return of the Alien Kong Beast", "Ore Refinery", "Skylab Landing Bay",
		"The Bank", "The Sixteenth Cavern", "The Warehouse",
		"Amoebatrons' Revenge", "Solar Power Generator", "The Final Barrier",
	}
	if num >= 0 && num < len(names) {
		return names[num]
	}
	return "Unknown"
}

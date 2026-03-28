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

// Config is the persistent game configuration.
type Config struct {
	PlayerName    string           `json:"playerName"`
	ControlScheme ControlScheme    `json:"controlScheme"`
	HighScores    []HighScoreEntry `json:"highScores"`
	Features      Features         `json:"features"`
	LastCavern    int              `json:"lastCavern"` // For continue feature.
}

const maxHighScores = 10

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

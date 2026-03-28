# Manic Miner — Go Replication

A faithful Go replication of the ZX Spectrum classic Manic Miner (1983), built from the original Z80 assembly. Includes a headless game engine with a Gym-like API for AI training.

## Features

- All 20 caverns fully playable
- Accurate physics, movement, and collision from the original Z80 code
- In the Hall of the Mountain King and Blue Danube music
- Jump, fall, and death sound effects
- Title screen with piano key animation and scrolling banner
- Game over sequence with boot descent and glistening text
- Demo mode (cycles through caverns after banner scroll)
- Settings screen with alternate control schemes and cheat feature flags
- High score table with persistent save
- Continue from last cavern played
- Visual warp screen with cavern thumbnails
- Headless engine for AI training (`engine.GameEnv` with `Step(Action) → StepResult`)
- Screenshot capture (Shift+8)

## Controls

| Key | Action |
|---|---|
| Q/W/E/R/T | Move left (original scheme) |
| P/O/I/U/Y | Move right (original scheme) |
| Space | Jump |
| A-G | Pause |
| H-L / Enter | Toggle music |
| Shift+Space | Restart cavern |
| ESC | Exit to title / Settings (from title) |
| Enter | Start game (from title) |
| Down | Continue from last cavern (from title) |
| Up | High scores (from title) |
| ? (Shift+/) | Help screen (from title) |
| Shift+8 | Screenshot |

Alternate control schemes (Arrows+Space, O/P+Space) available in Settings.

## Building & Running

```bash
make run     # Run the game
make build   # Build binary
make test    # Run engine tests
```

Requires Go 1.24+ and Ebitengine v2.

## Architecture

```
engine/     Headless game logic (no graphics dependency)
game/       Ebitengine wrapper, settings, high scores, help
entity/     Willy, guardians, items, portal, special entities
screen/     ZX Spectrum buffer system, renderer, sprites, font
audio/      Direct oto audio (low-latency square wave synthesis)
cavern/     20 cavern definitions extracted from Z80 assembly
data/       Sprite data, title screen, music note data
action/     Pure input type (leaf package)
config/     Persistent settings and high scores (JSON)
input/      Keyboard input with multiple control schemes
```

## Credits

### Original Game
- **Manic Miner** © 1983 Matthew Smith — all rights reserved
- Published by Bug-Byte Software Ltd.

### Z80 Disassembly
- **William Humphreys** — created the disassembly from the original binary
- **Simon Brattel** — assistance and modifications; author of the Zeus Z80 Assembler
- Source: [WHumphreys/Manic-Miner-Source-Code](https://github.com/WHumphreys/Manic-Miner-Source-Code)

### Go Implementation
- **Seamus Waldron** — Go replication, engine architecture, quality-of-life features
- **Claude AI** (Anthropic) — pair programming, Z80 analysis, implementation

## Documentation

- `docs/z80-source-analysis.md` — Complete Z80 source code analysis
- `docs/go-implementation-plan.md` — Architecture and package design
- `docs/phased-implementation-tasks.md` — Implementation task breakdown
- `docs/development-journal.md` — Detailed development narrative with lessons learned

## License

The original Manic Miner game is copyright Matthew Smith. The Z80 disassembly is by William Humphreys. This Go replication is an educational project — a faithful recreation for preservation and study purposes.

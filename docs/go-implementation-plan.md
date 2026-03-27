# Manic Miner — Go Implementation Plan

## Technology Choices

- **Graphics:** [Ebitengine](https://ebitengine.org/) (formerly Ebiten) — the de facto Go 2D game library, cross-platform, handles input/audio/rendering
- **Resolution:** Render at native 256x192, scale up with Ebitengine's built-in scaling
- **Colour:** Replicate the ZX Spectrum's 15-colour palette (8 colours x bright/normal, minus duplicate blacks)
- **Audio:** Ebitengine's audio package with procedural square wave generation matching the Spectrum's beeper

## Package Structure

```
manic-miner/
├── main.go                  # Entry point, Ebitengine game loop
├── game/
│   ├── game.go              # Game struct implementing ebiten.Game interface
│   ├── state.go             # GameState enum (Title, Playing, Demo, GameOver)
│   └── constants.go         # Screen dimensions, timing, colours
├── cavern/
│   ├── cavern.go            # Cavern struct and loader
│   ├── tiles.go             # Tile types and rendering
│   ├── data.go              # All 20 cavern definitions (embedded data)
│   └── conveyor.go          # Conveyor animation logic
├── entity/
│   ├── willy.go             # Miner Willy: movement, jumping, falling, animation
│   ├── guardian_horiz.go    # Horizontal guardian movement and rendering
│   ├── guardian_vert.go     # Vertical guardian movement and rendering
│   ├── eugene.go            # Eugene special entity
│   ├── kong.go              # Kong Beast special entity
│   ├── skylab.go            # Skylab special entity
│   ├── lightbeam.go         # Solar Power Generator light beam
│   └── portal.go            # Portal logic and rendering
├── screen/
│   ├── buffer.go            # Attribute + pixel buffer (replicates ZX memory model)
│   ├── renderer.go          # Buffer → Ebitengine image conversion
│   ├── sprites.go           # Sprite drawing with collision detection
│   └── text.go              # Character rendering from ZX ROM font
├── input/
│   └── input.go             # Keyboard mapping (ZX keys → modern keyboard)
├── audio/
│   ├── music.go             # Blue Danube + In the Hall of the Mountain King
│   └── sfx.go               # Jump, fall, death, level complete sounds
├── title/
│   ├── title.go             # Title screen rendering and piano animation
│   └── banner.go            # Scrolling banner text
├── data/
│   ├── sprites.go           # Willy sprite data (8 frames x 2 directions)
│   ├── guardians.go         # Guardian sprite data per cavern
│   ├── titlescreen.go       # Title screen graphic data
│   └── embed.go             # go:embed for binary data files
└── cheat/
    └── cheat.go             # 6031769 cheat code detection + teleport
```

## Key Design Decisions

### 1. Faithful Buffer Model

Replicate the dual attribute+pixel buffer system. The game logic depends on attribute-byte comparisons for collision detection (floor type, nasty tiles, wall tiles). Converting to a tile-map abstraction would break subtle behaviours.

```go
type ScreenBuffer struct {
    Attributes [512]byte  // 16 rows x 32 cols, one attribute per 8x8 cell
    Pixels     [4096]byte // 16 rows x 32 cols x 8 pixel rows per cell
}
```

### 2. Fixed-Point Coordinates

Keep Willy's y-coordinate as the original "pixel y * 2" system. The jump table and fall mechanics depend on these exact values.

### 3. Frame-Accurate Timing

The original runs one main loop iteration per "frame". We target the same ~12 FPS logic rate inside Ebitengine's 60 FPS loop using a frame accumulator.

```go
const LogicFPS = 12.0
const LogicFrameTime = 1.0 / LogicFPS

func (g *Game) Update() error {
    g.accumulator += 1.0 / 60.0
    for g.accumulator >= LogicFrameTime {
        g.logicUpdate()
        g.accumulator -= LogicFrameTime
    }
    return nil
}
```

### 4. Attribute-Based Rendering

Each 8x8 cell has one attribute byte (ink colour, paper colour, bright flag, flash flag). Render by iterating pixel buffer, colouring set bits as INK and unset bits as PAPER, exactly as the Spectrum does.

```go
func (r *Renderer) RenderToImage(buf *ScreenBuffer, img *ebiten.Image) {
    for cellY := 0; cellY < 16; cellY++ {
        for cellX := 0; cellX < 32; cellX++ {
            attr := buf.Attributes[cellY*32+cellX]
            ink := spectrumColour(attr & 0x07, attr & 0x40)
            paper := spectrumColour((attr >> 3) & 0x07, attr & 0x40)
            flash := attr & 0x80
            for row := 0; row < 8; row++ {
                pixelByte := buf.Pixels[cellY*256 + row*32 + cellX]
                for bit := 7; bit >= 0; bit-- {
                    x := cellX*8 + (7 - bit)
                    y := cellY*8 + row
                    if pixelByte & (1 << bit) != 0 {
                        img.Set(x, y, ink)
                    } else {
                        img.Set(x, y, paper)
                    }
                }
            }
        }
    }
}
```

### 5. Data Extraction

All 20 cavern definitions, sprite graphics, and music data will be extracted from the assembly source and encoded as Go byte slices using `go:embed` for binary data files.

## ZX Spectrum Colour Palette

```go
var SpectrumPalette = [16]color.RGBA{
    {0, 0, 0, 255},       // 0: Black
    {0, 0, 215, 255},     // 1: Blue
    {215, 0, 0, 255},     // 2: Red
    {215, 0, 215, 255},   // 3: Magenta
    {0, 215, 0, 255},     // 4: Green
    {0, 215, 215, 255},   // 5: Cyan
    {215, 215, 0, 255},   // 6: Yellow
    {215, 215, 215, 255}, // 7: White
    {0, 0, 0, 255},       // 8: Bright Black
    {0, 0, 255, 255},     // 9: Bright Blue
    {255, 0, 0, 255},     // 10: Bright Red
    {255, 0, 255, 255},   // 11: Bright Magenta
    {0, 255, 0, 255},     // 12: Bright Green
    {0, 255, 255, 255},   // 13: Bright Cyan
    {255, 255, 0, 255},   // 14: Bright Yellow
    {255, 255, 255, 255}, // 15: Bright White
}
```

## Keyboard Mapping

| Original ZX Key | Modern Key | Action |
|---|---|---|
| Q/E/T/W/R | Q/E/T/W/R | Move left |
| P/I/U/O/Y | P/I/U/O/Y | Move right |
| Bottom row (SPACE-M) | Space | Jump |
| 0 / 7 | 0 / 7 | Jump (alternative) |
| A-G | A-G | Pause |
| H-L, ENTER | H-L, Enter | Toggle music |
| SHIFT+SPACE | Shift+Space | Quit to title |
| 1-5 (with 6 held) | 1-5 (with 6 held) | Teleport (cheat mode) |

## Implementation Phases

### Phase 1: Core Rendering (Static Cavern)
- Implement `ScreenBuffer` with attribute + pixel arrays
- Implement tile definitions and cavern data loading
- Parse first cavern (Central Cavern) from extracted data
- Implement `DrawCurrentCavernToScreenBuffer` — fill pixel buffer from attribute buffer using tile graphics
- Implement `Renderer` — convert buffer to Ebitengine image with Spectrum colours
- Implement ZX ROM font for text rendering
- **Goal:** Central Cavern visible on screen with correct tiles and colours

### Phase 2: Willy Movement
- Implement Willy sprite data (8 frames x 2 directions)
- Implement `DrawASprite` with overwrite and blend modes
- Implement `YTable` screen buffer address lookup
- Implement Willy's movement state machine (16-entry lookup table)
- Implement keyboard input mapping
- Implement `MoveWilly1` (jumping/falling) and `MoveWilly2` (left/right + conveyor)
- Implement crumbling floor animation
- Implement wall collision checks
- **Goal:** Willy can walk, jump, fall, and die in Central Cavern

### Phase 3: Guardians
- Implement horizontal guardian movement (`MoveHorzGuardians`)
- Implement horizontal guardian rendering (`DrawHorizontalGuardians`)
- Implement guardian animation frames (8 frames, speed flag)
- Implement collision detection (blend mode sprite draw → kill Willy)
- Implement vertical guardian movement and rendering
- **Goal:** All guardian types functioning correctly

### Phase 4: Items, Portal, Air, Scoring
- Implement item drawing with colour cycling
- Implement item collection (INK white detection)
- Implement portal drawing and entry detection
- Implement air supply decrease and bar rendering
- Implement score system (ASCII digit arithmetic, 10000-point extra life)
- Implement `MoveToNextCavern` with colour cycling transition
- **Goal:** Complete playable cavern loop

### Phase 5: Special Entities
- Implement Eugene (cavern 4): vertical bouncing, portal blocking, colour cycling
- Implement Kong Beast (caverns 7, 11): switches, wall dissolution, falling, death
- Implement Skylabs (cavern 13): falling, disintegration, respawn
- Implement Light Beam (cavern 18): tracing, reflection, air drain
- **Goal:** All 20 caverns fully playable

### Phase 6: Title Screen, Music, Game Over
- Implement title screen rendering from graphic data
- Implement piano key animation
- Implement Blue Danube (title) and Mountain King (in-game) music
- Implement scrolling banner text
- Implement game over sequence (boot descent, "Game Over" glistening text)
- Implement demo mode (auto-play through caverns)
- Implement death animation (screen flash, colour cycling, sound effects)
- **Goal:** Complete game flow from title to game over

### Phase 7: Polish
- Implement 6031769 cheat code detection
- Implement teleport functionality
- Implement Final Barrier completion sequence (swordfish, celebratory sound)
- Frame timing fine-tuning for authentic feel
- Pixel-perfect comparison testing against original
- **Goal:** Pixel-perfect, behaviour-perfect replication

## Critical Implementation Notes

- The **movement table** (16 entries at `WillyNotMoving0`) must be replicated exactly — it encodes the state machine for direction changes
- **Crumbling floor** animation works at pixel-row level within the screen buffer, not at tile level
- **Guardian collision** depends on pixel-level OR/AND operations during sprite drawing — must replicate `DrawASprite` blend mode exactly
- The **air supply** system uses the game clock (decrements by 4 each frame) — when it wraps past 0, one air unit is consumed
- **Conveyor** animation rotates pixel rows 0 and 2 of the tile (not all rows)
- The **jump arc** is symmetric: frames 0-8 up, 9-17 down, with wall-hit checks during ascent
- Guardian speed flag (bit 7): when set, guardian only moves on frames where game clock bit 2 matches
- Horizontal guardians in caverns 7+ (except 9, 15) use frames 4-7 only (bit 7 of sprite offset set)
- The Final Barrier (cavern 19) copies the title screen graphic data into the top half of the screen buffer
- Score is stored as ASCII characters ('0'-'9'), not binary — arithmetic is done character by character

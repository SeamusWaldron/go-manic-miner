# Manic Miner Go Replication — Development Journal

## Project Overview

A faithful replication of the ZX Spectrum game Manic Miner (1983, Bug-Byte Ltd., by Matthew Smith) written in Go, built from a Z80 assembly disassembly. The project also includes a headless game engine with a Gym-like API for AI training, and modern quality-of-life features.

**Final stats:** 36 Go source files, ~6,900 lines of code, 11 packages, 56 commits over 2 days.

---

## Phase 1: Analysis & Planning

### Starting Point
We began with the Z80 assembly source from [WHumphreys/Manic-Miner-Source-Code](https://github.com/WHumphreys/Manic-Miner-Source-Code) — a ~12,000 line disassembly of the original game for the Zeus Z80 Assembler.

### Approach
Rather than attempting a line-by-line assembly translation, we:
1. Analysed the complete source to understand every data structure, game mechanic, and rendering pipeline
2. Created three detailed documentation files:
   - `z80-source-analysis.md` — complete Z80 source breakdown
   - `go-implementation-plan.md` — Go architecture and package design
   - `phased-implementation-tasks.md` — 41 tasks across 7 phases
3. Defined a 7-phase implementation plan with verification gates

### Key Architectural Decisions
- **Replicate the buffer model**: The ZX Spectrum uses attribute bytes (colour) + pixel bytes (graphics) in separate buffers. We replicated this exactly because the game logic depends on attribute comparisons for collision detection.
- **Use y*2 coordinates**: The original stores Willy's Y position doubled. We initially used pixel Y and applied the doubled deltas, causing jumps to be double height. This was a critical early bug.
- **Ebitengine for graphics**: The de facto Go 2D game library, providing cross-platform rendering, input, and (initially) audio.

---

## Phase 2: Core Implementation (Phases 1-4)

### Rendering System
Built the ZX Spectrum rendering pipeline:
- `screen/buffer.go` — YTable lookup matching the Spectrum's non-linear display memory
- `screen/renderer.go` — Attribute-based pixel colouring (INK/PAPER/BRIGHT/FLASH)
- `screen/text.go` — Full ZX Spectrum ROM font (96 characters, 8x8 pixels)
- `screen/sprites.go` — Three draw modes: Overwrite, Blend (collision detection), OR (Willy)

### Cavern Data Extraction
Extracted all 20 cavern definitions (1,024 bytes each, 20,480 bytes total) from the assembly source using a Python script. Each cavern contains:
- 512-byte attribute grid (16×32 tiles)
- 32-byte name, 72-byte tile definitions
- Willy start position, conveyor, items, portal
- Guardian definitions and sprite graphics

**Issue encountered:** The initial manual extraction of Central Cavern had byte offset errors in the parser (portal at offset 654 instead of 655, cascading to air supply, guardians). Fixed by carefully recalculating all offsets from the assembly structure.

### Willy Movement & Physics
The most bug-prone area of the entire project:

**Bug 1 — Willy disappears when jumping:** `DrawSprite` in blend mode (AND check) was used for Willy. Any pixel overlap with floor tiles triggered a false "collision", aborting the draw. **Fix:** Added `DrawOR` mode — OR onto background without collision detection. Only guardians use blend mode.

**Bug 2 — Jump height doubled:** Jump deltas are designed for the y*2 coordinate system but were applied to pixel Y. **Fix:** Changed to store Y2 (the doubled coordinate) internally, matching the original exactly.

**Bug 3 — Can't land on platforms:** The mid-jump ground check at JC=13 and JC=16 found solid floor but didn't land Willy because the code only reset Airborne when `>= 2`, not when `== 1` (jumping). In the original Z80 code, this path jumps to `MoveWilly2` which always resets Airborne to 0. **Fix:** Changed `if w.Airborne >= 2` to unconditional `w.Airborne = 0`.

**Bug 4 — Lower half of Willy invisible:** `SetAttributes` skipped the second row of cells due to a wrong skip condition. The condition `dy == rows-1 && IsCellAligned()` with `rows=2` skipped `dy=1`. **Fix:** Always iterate 3 rows, only skip `dy=2` when cell-aligned.

**Bug 5 — Direction change mid-jump:** Our code called `moveWilly2` (full keyboard reading) every frame, even during jumps. The original skips keyboard input during jumps and only continues existing movement. **Fix:** Restructured `Update` to check airborne state — during jumps, only `continueExistingMovement` runs (no keyboard, no direction change).

**Bug 6 — Walking off edges allows horizontal movement:** When falling, the original resets the movement flag. Our code didn't. **Fix:** `DirFlags &^= 2` when starting to fall.

### Guardians, Items, Portal
Implemented horizontal guardians (speed flag, 8-frame animation, boundary patrol), vertical guardians, items (INK colour cycling 3→4→5→6, collection via white INK detection), portal (FLASH activation, entry check), and air supply.

---

## Phase 3: Engine Refactoring

### The Headless Engine
A major architectural pivot: we decoupled all game logic from Ebitengine into a pure `engine.GameEnv` with a Gym-like API:

```go
func (e *GameEnv) Step(action Action) StepResult
func (e *GameEnv) Reset(cavernNum int) Observation
```

This enables:
- **AI training**: Call `Step()` thousands of times per second, no window needed
- **Testing**: Deterministic action sequences with assertions
- **Debugging**: Full state inspection via `Observation` struct

The `action.Action` type lives in a leaf package to break circular dependencies between `entity` (which accepts actions) and `engine` (which imports entity).

---

## Phase 4: Special Entities (Phase 5)

Implemented the four cavern-specific entities:
- **Eugene** (cavern 4): Vertical bouncer, blocks portal until all items collected
- **Kong Beast** (caverns 7, 11): Switches, wall dissolution, falling death
- **Skylabs** (cavern 13): Falling debris with disintegration and respawn
- **Light Beam** (cavern 18): Traces from (0,23), reflects off guardians, drains air

---

## Phase 5: Title Screen & Audio (Phase 6)

### Title Screen
**Critical bug:** The title screen graphic data is stored in ZX Spectrum display file format (interleaved thirds). We copied it directly into our linear buffer, producing garbage. **Fix:** Wrote `SpectrumDisplayToLinear()` to remap the interleaved address layout.

The title screen has two phases matching the original:
1. Piano keys animate while the Blue Danube plays
2. Banner scrolls with Willy animating at (9,29)

### Audio — A Long Journey
Audio was the most iteratively difficult feature, going through multiple complete rewrites:

**Attempt 1 — Ebitengine audio with streaming:** Used `audio.NewPlayerF32` with a continuous `toneStream`. The frequency conversion was wrong (`counter * 8` instead of `counter * 112`), producing ultrasonic clicks instead of musical tones.

**Attempt 2 — Fixed frequencies, still slow:** The title tune played one note per game frame (12 FPS) instead of managing its own timing. The entire Blue Danube takes ~30 seconds but was playing over several minutes. **Fix:** Made the audio stream manage note advancement internally, calculating duration from Z80 T-state counts.

**Attempt 3 — In-game music too slow/fast:** Multiple iterations of adjusting the note counter increment, trying burst mode vs sustained mode, changing game FPS. The root issue: the in-game music counter in the original goes 0-255 with `(counter AND 126) >> 1` mapping to 64 notes. Each note plays for 2 frames. We initially cycled 0-63 directly.

**Attempt 4 — Audio latency (~500ms):** Ebitengine's audio pipeline has multiple internal buffering layers. Even with `SetBufferSize(1024)`, sounds played half a second late. **Fix:** Replaced Ebitengine's audio entirely with direct `oto` (the underlying library). Set oto player buffer to 4096 bytes (~12ms). This reduced latency to ~30ms.

**Attempt 5 — Music hanging on toggle:** `StopInGameMusic` set `igmPlaying=false` but didn't zero `freq1/freq2`, so the last note played forever. **Fix:** Zero frequencies in `StopInGameMusic`.

**Attempt 6 — Music speed tuning:** Added interactive `-`/`=` keys to adjust note duration by ear. The user found 60ms per note was correct. Initial attempts to adjust tempo by changing the note counter increment also changed the lives animation speed (they share `MusicNoteIndex`). **Fix:** Created a separate `musicCounter` in the game wrapper for audio, independent of the engine's `MusicNoteIndex`.

### HUD Rendering
The HUD went through many iterations to match the original:

**Air bar saga:** Multiple incorrect implementations — wrong colours, wrong position, wrong behaviour. The correct implementation (from the Z80 display file addresses):
- Fixed green background (cols 4-31), fixed red background (cols 0-3)
- White gauge pixels at y=138-141 (pixel rows 2-5 of char row 17)
- Gauge shrinks from right, revealing green underneath (NOT red)

**Key insight:** All HUD positions must be decoded from the original Z80 display file addresses, not guessed. The address format `010TTRRR CCCXXXXX` encodes third, pixel row, character row, and column.

---

## Phase 6: Game Over & Polish (Phase 7)

### Game Over Sequence
Implemented the three-phase sequence from the original:
1. Boot descent (49 steps with rising-pitch sound)
2. "Game Over" text
3. Glistening text (cycling INK colours per letter, ~1.5 seconds)

**Bug:** The init block was inside `stepGameOver` guarded by `AnimCounter==1`. When phase transitions reset `AnimCounter`, the init triggered again, restarting the boot descent infinitely. **Fix:** Moved init into a separate `InitGameOver()` called once from the transition.

### Z80 Timing Analysis
A comprehensive T-state analysis of the original main loop determined:
- Normal frame: ~232,000 T-states = **~15.1 FPS**
- Death animation: ~415,000 T-states = **~0.12 seconds**
- Boot descent: ~7,170,000 T-states = **~2.05 seconds**
- In-game note: ~31,000 T-states = **~8.8ms burst**

This corrected our death animation (was 2 seconds, should be 0.12 seconds) and cave transition (was 4 seconds, should be 0.43 seconds).

### Jump Mechanics Fix
The user reported Willy could change direction mid-jump and didn't fall straight down off edges. Tracing the original Z80 code revealed:
- During a jump, the code skips keyboard entirely and goes to `MV2x7` (continue existing movement only)
- When falling, movement flag is reset — Willy falls straight down
- Wall hit above: Y snapped via `(Y2+16) & 0xF0`, then Airborne=2, movement stopped, return immediately

---

## Phase 7: Quality of Life Features

### Settings & Persistence
Added a settings screen accessible from the title screen with:
- 3 control schemes (Original, Arrows+Space, O/P+Space)
- 6 cheat feature flags from popular POKEs (Infinite Lives, Infinite Air, Harmless Heights, No Nasties, No Guardians, Warp Mode)
- Player name entry (3 characters, type A-Z directly)
- All settings persisted to `~/.manicminer/config.json`

### High Score Table
Classic 80s arcade-style top 10 table with name, score, and cavern reached. Name entry screen with cycling letter selection appears after qualifying.

### Warp Screen
Visual cavern selection: 20 thumbnails in a 5×4 grid, each showing the cavern layout using attribute PAPER colours. Navigate with arrows, Enter to warp.

### Continue Mode
Press DOWN on title screen to restart from the last cavern played. Cavern number saved to config on every frame of gameplay and on ESC exit.

### Other Features
- Screenshot capture (Shift+8)
- Scrolling help screen (? on title)
- ESC exits gameplay with game over animation and high score check
- Demo mode exits on any key press (matching original)
- Boot sprite displayed next to lives when infinite lives or cheat mode active

---

## Lessons Learned

### 1. Don't guess — read the original code
The single most important lesson. Every time we guessed at a mechanic (jump physics, HUD layout, air bar behaviour, music timing), we got it wrong. The Z80 assembly is the definitive source of truth. Even when the user provided reference screenshots, the assembly code was needed to understand WHY things look the way they do.

### 2. The coordinate system matters enormously
Using pixel Y instead of the original's y*2 coordinate caused cascading bugs: wrong jump height, inability to land on platforms, alignment issues with ground checks. When replicating retro games, use the EXACT coordinate system the original uses, even if it seems redundant.

### 3. Audio is harder than gameplay
The game mechanics (movement, collision, guardians) were implemented relatively smoothly. Audio went through 6+ complete rewrites. Key issues:
- Frequency conversion requires understanding the exact Z80 timing loop T-states
- Frame-rate-driven vs audio-stream-driven note timing produces completely different results
- Ebitengine's audio pipeline has too much internal buffering for responsive game audio
- The character of the sound (staccato vs sustained, burst duration) matters as much as the pitch and tempo

### 4. Test with a human early and often
Every major bug was caught by the user playing the game, not by automated tests or code review. The user's feedback ("Willy disappears when jumping", "music is too slow", "floor disappears instantly") immediately identified issues that would have taken much longer to find through code inspection alone.

### 5. The ZX Spectrum display is intentionally weird
The interleaved display memory layout, the attribute-based colour system, the way addresses encode position — these all need to be understood and handled correctly. Don't try to "simplify" them into a modern rendering model; replicate them and convert at the rendering boundary.

### 6. Decouple the engine from the renderer
The headless `GameEnv` refactoring was one of the best architectural decisions. It enabled:
- Testing without graphics
- Future AI training capability
- Clean separation of concerns
- The ability to add sub-screens (settings, high scores) without touching game logic

### 7. Feature flags are better than code removal for cheats
Implementing POKEs as boolean feature flags in a config struct is cleaner than modifying game logic. Each flag is a single `if` check at the appropriate point, and they can be toggled at runtime through a settings UI.

---

## Approach for Replicating Another Retro Game

If attempting this process again with a different game:

### Step 1: Obtain and analyse the source
- Find a disassembly or original source code
- Create a comprehensive analysis document covering:
  - Memory layout and data structures
  - Main loop flow (exact order of operations)
  - All game mechanics with the specific values/formulas used
  - Rendering pipeline
  - Audio system
  - Input handling

### Step 2: Design the architecture
- Choose a modern language and 2D graphics library
- Plan a headless engine + thin renderer wrapper from the start
- Replicate the original's coordinate systems and buffer layouts exactly
- Define a phased implementation plan with verification at each gate

### Step 3: Implement bottom-up
- Phase 1: Static rendering (get something on screen)
- Phase 2: Player movement (the hardest part — get this right)
- Phase 3: Enemies and collision
- Phase 4: Items, scoring, level progression
- Phase 5: Special mechanics (level-specific features)
- Phase 6: Audio (expect this to take longer than gameplay)
- Phase 7: Polish, menus, persistence

### Step 4: Test with a human player continuously
- Every few commits, have someone play the game
- Their feedback will catch issues automated tests can't
- Be prepared to rewrite audio multiple times

### Step 5: Add modern features last
- Settings, high scores, alternate controls
- These are additive and don't affect the core game
- Use the original game's aesthetic for UI

### Key Technical Recommendations
- **Use the original's coordinate system** — don't convert to "simpler" coordinates
- **Use the original's buffer layout** — the game logic depends on it
- **Decode display addresses from the original** — don't guess pixel positions
- **For audio, bypass high-level libraries** — use the lowest-level audio API available for minimal latency
- **Trace T-states for timing** — the original game's speed is determined by CPU execution time, not frame sync
- **Keep a debug screenshot capability** — you'll need it when you can't see what the AI sees

---

## Project Statistics

| Metric | Value |
|---|---|
| Total Go files | 36 |
| Total Go lines | ~6,900 |
| Go packages | 11 |
| Git commits | 56 |
| Development time | ~2 days |
| Cavern definitions | 20 (20,480 bytes extracted from Z80 ASM) |
| Sprite frames | 8 Willy + per-cavern guardian sprites |
| Audio system rewrites | 6+ |
| Jump physics bugs fixed | 6 |
| Air bar rendering attempts | 7 |

---

## Repository

Private: [SeamusWaldron/go-manic-miner](https://github.com/SeamusWaldron/go-manic-miner)

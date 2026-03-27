# Manic Miner Z80 Source Code Analysis

## Game Architecture Overview

The original game runs on the ZX Spectrum (Z80 @ 3.5MHz, 48K RAM, 256x192 display with attribute-based colour). The code is a single monolithic assembly file (~12,000 lines) with a tight main loop that runs at approximately 12-15 FPS, gated by the air supply countdown and music timing.

## Core Data Structures

### Memory Layout (starting at $5C00)

- `AttributeBufferCWGI` ($200 bytes) — composite attribute buffer (cavern + Willy + guardians + items)
- `EmptyCavernAttributeBuffer` ($200 bytes) — clean cavern attributes, refreshed each frame
- `ScreenBufferCWGI` ($1000 bytes) — composite pixel buffer (32x16 tiles x 8 rows)
- `EmptyCavernScreenBuffer` ($1000 bytes) — clean cavern pixels
- `CavernName` (32 bytes)
- 8 tile definitions (9 bytes each: 1 attribute + 8 pixel rows): Background, Floor, CrumblingFloor, Wall, Conveyor, Nasty1, Nasty2, Extra

### Willy State

| Variable | Description |
|---|---|
| `WillysPixelYCoord` | y-coordinate x2 (LSB into Y lookup table) |
| `WillysAnimationFrame` | 0-3 (4 frames, 2-byte wide sprites) |
| `WillysDirAndMovFlags` | bit 0: direction (0=right, 1=left), bit 1: moving flag |
| `AirborneStatusIndicator` | 0=grounded, 1=jumping, 2-11=falling safe, 12+=fatal, 255=dead |
| `WillysLocInAttrBuffer` | 16-bit address in attribute buffer |
| `JumpingAnimationCounter` | 0-17 jump arc position |

### Guardian Definitions

**Horizontal (4 slots, 7 bytes each):**

| Byte | Content |
|---|---|
| 0 | Bit 7: animation speed (0=normal, 1=slow). Bits 0-6: attribute (BRIGHT, PAPER, INK) |
| 1,2 | Address of guardian's location in attribute buffer |
| 3 | MSB of address of guardian's location in screen buffer |
| 4 | Animation frame (0-7) |
| 5 | LSB of leftmost point of path in attribute buffer |
| 6 | LSB of rightmost point of path in attribute buffer |

**Vertical (4 slots, 7 bytes each):**

| Byte | Content |
|---|---|
| 0 | Attribute byte |
| 1 | Animation frame (0-3) |
| 2 | Pixel y-coordinate |
| 3 | x-coordinate |
| 4 | Pixel y-coordinate increment (signed) |
| 5 | Minimum pixel y-coordinate |
| 6 | Maximum pixel y-coordinate |

### Item Definitions (5 slots, 5 bytes each)

| Byte | Content |
|---|---|
| 0 | Current attribute (0 = collected) |
| 1,2 | Address in attribute buffer |
| 3 | MSB of address in screen buffer |
| 4 | Unused (always 255) |

### Portal Definition

| Field | Size | Content |
|---|---|---|
| Attribute | 1 byte | Attribute byte (bit 7 = flashing when active) |
| Graphic | 32 bytes | 16x16 pixel graphic data |
| AttrBufAddr | 2 bytes | Location in attribute buffer |
| ScreenBufAddr | 2 bytes | Location in screen buffer |

## Cavern Definitions (20 caverns, 1024 bytes each)

Each cavern is structured as:

| Offset | Size | Content |
|---|---|---|
| 0 | 512 bytes | 16x32 attribute grid (tile type per cell) |
| 512 | 32 bytes | Cavern name (padded to 32 chars) |
| 544 | 72 bytes | 8 tile definitions (attribute + 8x8 pixel graphic each) |
| 616 | 7 bytes | Willy initial state (y-coord, frame, direction, airborne, location, jump counter) |
| 623 | 4 bytes | Conveyor (direction, screen address, length) |
| 627 | 2 bytes | Border colour + unused |
| 629 | 25 bytes | Up to 5 items + $FF terminator |
| 654 | 37 bytes | Portal definition |
| 691 | 8 bytes | Item graphic |
| 699 | 2 bytes | Air supply + game clock |
| 701 | 28 bytes | Horizontal guardians (4 slots) |
| 729 | 3 bytes | Terminator + unused |
| 732 | 35 bytes | Vertical guardians area |
| 767 | 32 bytes | Special graphic (swordfish/plinth/boot per cavern) |
| 799 | 256 bytes | Guardian sprite graphics (8 frames x 32 bytes) |

### The 20 Caverns

| # | Name | Special Features |
|---|---|---|
| 0 | Central Cavern | Basic level |
| 1 | The Cold Room | Basic level |
| 2 | The Menagerie | Spider silk (extra tile) |
| 3 | Abandoned Uranium Workings | — |
| 4 | Eugene's Lair | Eugene special entity |
| 5 | Processing Plant | — |
| 6 | The Vat | — |
| 7 | Miner Willy meets the Kong Beast | Kong Beast + switches |
| 8 | Wacky Amoebatrons | First with vertical guardians |
| 9 | The Endorian Forest | Extra tile as floor |
| 10 | Attack of the Mutant Telephones | Extra tile as floor |
| 11 | Return of the Alien Kong Beast | Kong Beast + switches |
| 12 | Ore Refinery | Extra tile as floor |
| 13 | Skylab Landing Bay | Skylabs (special vertical entities) |
| 14 | The Bank | Extra tile as floor |
| 15 | The Sixteenth Cavern | — |
| 16 | The Warehouse | — |
| 17 | Amoebatrons' Revenge | — |
| 18 | Solar Power Generator | Light beam special entity |
| 19 | The Final Barrier | Uses title screen graphic for top half |

## Main Loop Flow

```
MainLoop:
  1.  Draw remaining lives sprites at bottom of screen
  2.  Draw boot if cheat mode active (6031769 entered)
  3.  Copy empty cavern buffers → working buffers (attr + screen)
  4.  MoveHorizontalGuardians()
  5.  MoveWilly() [if not demo mode]
  6.  CheckSetAttributeForWilly() [if not demo mode]
  7.  DrawHorizontalGuardians() — kills Willy on collision
  8.  MoveConveyor()
  9.  DrawAndCollectItems()
  10. Special entity logic per cavern:
      - Cavern 4: MoveDrawEugene()
      - Cavern 7, 11: MoveDrawKongBeast()
      - Cavern 8+: MoveDrawVerticalGuardians()
      - Cavern 13: MoveDrawSkylabs()
      - Cavern 18: MoveDrawLightBeam()
  11. DrawPortal() — or move to next cavern if Willy entered it
  12. Copy screen buffer → display file
  13. Handle screen flash (extra life effect)
  14. Copy attribute buffer → attribute file
  15. Print score + high score
  16. DecreaseAir() — return to death sequence if air is gone
  17. Check SHIFT+SPACE (quit to title)
  18. Check pause (A-G keys)
  19. Check fatal collision (airborne status = $FF)
  20. Toggle music (H-L-ENTER keys)
  21. Play music note (In the Hall of the Mountain King)
  22. Demo mode: check for keypress → return to title
  23. Teleport checks (6031769 cheat sequence)
  24. Loop back to MainLoop
```

## Key Game Mechanics

### Jump Physics

Fixed arc over 18 frames (counter 0-17):
- Vertical delta per frame = `(counter & ~1) - 8` pixels (applied to y*2 coordinate)
- Frames 0-8: rising
- Frames 9-17: falling
- Wall collision during ascent: snap y to next cell boundary below, start falling
- At frame 13: check if Willy can land (same as grounded check)
- At frame 16: check if Willy can land
- At frame 18: jump complete, set airborne status to 6 (falling, but safe if landing immediately)

### Falling

- Airborne status increments each frame while falling
- Status 2-11: safe to land
- Status 12+: fatal on landing (`CP $0C / JP NC,KillWilly1`)
- Falling speed: y increases by 8 (4 pixels) per frame

### Movement State Machine

16-entry lookup table maps (current_state, input) → new_state:

| State | Meaning |
|---|---|
| 0 | Facing right, not moving |
| 1 | Facing left, not moving |
| 2 | Facing right, moving |
| 3 | Facing left, moving |

Input offsets: +0 = no input, +4 = left, +8 = right, +12 = both

Key behaviour: pressing opposite direction first turns Willy around (without moving), then pressing again starts movement. This creates the characteristic "turn then walk" feel.

### Animation Frames

- Willy has 4 frames (0-3) per direction, 8 total sprite sets
- Moving right: frame increments 0→1→2→3, then crosses cell boundary and resets to 0
- Moving left: frame decrements 3→2→1→0, then crosses cell boundary and resets to 3
- Each frame shifts the sprite 2 pixels within the 16-pixel-wide sprite area

### Conveyor

- Affects Willy's movement input when standing on conveyor tiles
- Left conveyor: resets bit 1 of input (forces left movement)
- Right conveyor: resets bit 0 of input (forces right movement)
- Animation: rotates pixel rows 0 and 2 of the conveyor tile graphic (left=RLC x2, right=RRC x2)

### Crumbling Floors

- Triggered when Willy stands on them (checked each frame)
- Animation: each pixel row shifts down by one position, top row cleared
- When bottom pixel row becomes empty: tile replaced with background in attribute buffer
- This creates a progressive dissolution effect over 8 frames

### Collision Detection

The `DrawASprite` routine has two modes:
- **Overwrite** (C=0): Copies sprite bytes directly to screen buffer
- **Blend** (C=1): ANDs sprite with background first — if non-zero, collision detected (returns NZ). Then ORs sprite onto background.

Guardian/Willy collision uses blend mode. Any overlapping set bit = death.

### Air Supply

- `RemainingAirSupply` ranges from 36 ($24) to 63 ($3F)
- `GameClock` decrements by 4 each frame (values are multiples of 4)
- When GameClock wraps from 0 to $FC: one air unit consumed
- The air bar is drawn directly to the display file using the supply value as the LSB of the display address
- Visual: 4 pixel rows high, drawn from left ($24) to current supply value

### Scoring

- Collecting an item: +100 points (add 1 to hundreds digit)
- Converting remaining air: +1 point per air tick
- Kong Beast falling: +100 points per frame while falling
- Every 10,000 points: extra life + screen flash (counter set to 8)
- Score stored as ASCII digits, incremented character-by-character with carry propagation

## Special Entity Details

### Eugene (Cavern 4: Eugene's Lair)

- Bounces vertically between y=0 and y=$58 (portal position)
- Direction toggles at boundaries
- While items remain: drawn with white INK, blocks portal
- All items collected: INK colour cycles (game clock bits 2-4), portal activates
- Collision with Willy = death (blend mode sprite draw)

### Kong Beast (Caverns 7, 11)

- Two switches at fixed positions in attribute buffer
- Left switch: triggers wall dissolution at (11,17) — pixel rows cleared one per frame, then attributes changed to background
- Right switch: Kong Beast starts falling (status 0→1), floor beneath removed
- Falling: y increases by 4 per frame, drawn with alternating sprite (game clock bit 5)
- At y=100: Kong Beast dies (status 2), scoring +100 per frame while falling
- While on ledge: drawn at fixed position (0,15), collision checked with blend mode

### Skylabs (Cavern 13: Skylab Landing Bay)

- Use vertical guardian slots but different movement logic
- Fall downward at variable speed (y += increment)
- On reaching crash site (max y): animation frame increments 0-7 (disintegration)
- After full disintegration: y reset to start, x shifted right by 8 (wrapping at 32)
- Collision with Willy = death

### Light Beam (Cavern 18: Solar Power Generator)

- Starts at cell (0,23) in attribute buffer, travels downward
- Stops on floor tile or wall tile
- On background tile: continues in current direction
- On guardian: reflects (direction toggles between down and left)
- On Willy (attribute $27 = INK 7, PAPER 4): drains air (calls DecreaseAir x4 = 16 extra air units)
- Drawn with attribute $77 (INK 7, PAPER 6, BRIGHT 1)

## Sprite Rendering System

### ZX Spectrum Display Layout

The Spectrum's display file has a non-linear layout:
- 3 "thirds" of 8 character rows each
- Within each third, pixel rows are interleaved: row 0 of char 0, row 0 of char 1, ..., row 1 of char 0, etc.
- The `YTable` lookup table (128 entries, 2 bytes each) pre-computes screen buffer addresses for each pixel y-coordinate

### DrawASprite Routine

Draws a 16x16 pixel sprite (2 bytes wide, 16 rows):
1. For each of 16 rows:
   - Load left byte from sprite data → write/blend to (HL)
   - Load right byte from sprite data → write/blend to (HL+1)
   - Advance HL to next pixel row (complex address arithmetic handling cell/third boundaries)
2. In blend mode: AND check before OR write, return NZ on collision

### Character Rendering

Uses the ZX Spectrum ROM font at address $3C00-$3FFF:
- Character code → address: `$3C00 + (code * 8)`
- 8 bytes per character, 8x8 pixels
- Written directly to display file, advancing D (MSB) for each row

## Music Data

### Title Screen: The Blue Danube

- 95 notes, 3 bytes each (duration, freq1, freq2)
- Two simultaneous frequency counters produce a simple chord
- Duration byte in C register, counted down in outer loop
- Piano key visualisation: frequency → attribute file address mapping

### In-Game: In the Hall of the Mountain King

- 64 bytes, one frequency value per note
- Played one note per main loop iteration
- Note index cycles through 0-63
- Border colour XOR'd with 24 for speaker output

## Cheat System (6031769)

- 8 key pairs checked sequentially: none, 6, 0, 3, 1, 7, 6, 9
- Each pair is (keys 1-2-3-4-5 reading, keys 0-9-8-7-6 reading)
- Counter increments on correct key, resets to 0 on wrong key
- Counter = 7: cheat mode active, boot displayed, teleport enabled
- Teleport: hold 6 + press 1-5 keys (binary cavern number, 0-19)

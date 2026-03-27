# Manic Miner — Phased Implementation Tasks

Each phase ends with a mandatory verification task. Phases are sequential; tasks within a phase may be parallelised where noted.

---

## Phase 1: Project Scaffolding & Core Rendering

### Task 1.1: Project Init
- `go mod init manicminer`
- Add Ebitengine dependency (`github.com/hajimehoshi/ebiten/v2`)
- Create directory structure per go-implementation-plan.md
- Create `main.go` with minimal Ebitengine game loop (black screen, 256x192 scaled to 768x576)
- Create `game/constants.go` with screen dimensions, Spectrum colour palette, timing constants

### Task 1.2: Screen Buffer System
- Implement `screen/buffer.go`: `ScreenBuffer` struct with `Attributes [512]byte` and `Pixels [4096]byte`
- Implement `YTable` lookup (128 entries mapping pixel y → buffer offset)
- Implement `screen/renderer.go`: convert `ScreenBuffer` → `*ebiten.Image` using attribute-based colouring (INK/PAPER/BRIGHT/FLASH)

### Task 1.3: Tile System & Cavern Data
- Implement `cavern/tiles.go`: `TileDef` struct (1 attribute byte + 8 pixel bytes), `TileType` enum (Background, Floor, CrumblingFloor, Wall, Conveyor, Nasty1, Nasty2, Extra)
- Implement `cavern/cavern.go`: `Cavern` struct holding all parsed fields from the 1024-byte cavern definition
- Extract Central Cavern data from assembly source → `cavern/data.go` as Go byte slice
- Implement cavern parser: decode attribute grid, tile defs, willy start, conveyor, items, portal, guardians, etc.

### Task 1.4: Cavern Rendering
- Implement `DrawCurrentCavernToScreenBuffer`: iterate attribute grid, find matching tile, copy 8 pixel rows to screen buffer
- Implement `screen/text.go`: ZX Spectrum ROM font (96 printable characters, 8x8 pixels each) and `PrintMessage` function
- Render cavern name, "AIR" label, air bar, score/high-score text to bottom third of display
- Integrate into Ebitengine `Draw()`: render cavern buffer + HUD to screen

### Task 1.5: Verification — Phase 1
- Run `/simplify` on all files created in Phase 1
- Diff implementation against Phase 1 tasks: flag MISSING, PARTIAL, or STUB items
- Fix every issue found
- Run `/simplify` again on any files that changed during fixes
- Run `go vet ./...` and `go build ./...` to confirm compilation
- Visual check: Central Cavern renders correctly with tiles, colours, name, air bar, and score text

---

## Phase 2: Willy Movement & Physics

### Task 2.1: Sprite Data
- Extract all 8 Willy sprite frames (4 right-facing, 4 left-facing) from assembly `DG` directives → `data/sprites.go`
- Each frame: 32 bytes (2 bytes wide x 16 rows)

### Task 2.2: Sprite Drawing
- Implement `screen/sprites.go`: `DrawSprite(buf *ScreenBuffer, x, y int, spriteData []byte, mode DrawMode) bool`
- Overwrite mode (C=0): direct copy
- Blend mode (C=1): AND check for collision, then OR to merge — return true if collision detected
- Handle pixel row advancement across cell boundaries (replicate Z80 address arithmetic)

### Task 2.3: Willy State & Rendering
- Implement `entity/willy.go`: `Willy` struct with all state variables (pixelY, animFrame, dirFlags, airborne, attrBufAddr, jumpCounter)
- Implement `DrawWillyToScreenBuffer`: use YTable + sprite data + animation frame to draw Willy
- Implement `CheckSetAttributeForWilly`: set INK white on Willy's cells, check nasty tile collision

### Task 2.4: Input System
- Implement `input/input.go`: map modern keyboard to ZX Spectrum key matrix
- Left keys: Q, W, E, R, T (and 5)
- Right keys: P, O, I, U, Y (and 8)
- Jump: Space, Shift, Z, X, C, V, B, N, M, 0, 7
- Joystick: not needed (modern keyboard suffices)

### Task 2.5: Movement Logic
- Implement 16-entry movement table (`WillyNotMoving0` through `WillyMovingBoth3`)
- Implement `MoveWilly2`: read keyboard, apply conveyor effect, compute new direction/movement flags from table, handle cell boundary crossing, wall collision checks
- Implement jump initiation: set airborne=1, jumpCounter=0

### Task 2.6: Jump & Fall Physics
- Implement `MoveWilly1`: jumping logic (counter 0-17, delta = `(counter & ~1) - 8`, wall hit → snap to cell boundary + start falling)
- Implement falling: airborne status increment, y += 8 per frame, status >= 12 = fatal
- Implement grounded check: look at cells below Willy's sprite in attribute buffer
- Implement crumbling floor: `AnimateCrumblingFloor` — shift pixel rows down, clear top, replace with background when empty
- Implement `KillWilly`: set airborne = $FF

### Task 2.7: Verification — Phase 2
- Run `/simplify` on all files changed in Phase 2
- Diff implementation against Phase 2 tasks: flag MISSING, PARTIAL, or STUB items
- Fix every issue found
- Run `/simplify` again on any files that changed during fixes
- Run `go vet ./...` and `go build ./...` to confirm compilation
- Play test: Willy walks, turns, jumps (correct arc), falls, dies on nasties, crumbling floors work

---

## Phase 3: Guardians

### Task 3.1: Guardian Sprite Data
- Extract guardian sprite data from each cavern definition (256 bytes = 8 frames x 32 bytes per cavern)
- Store in `data/guardians.go` or embed per-cavern

### Task 3.2: Horizontal Guardians
- Implement `entity/guardian_horiz.go`: `HorizontalGuardian` struct (attribute, location, screenMSB, frame, leftBound, rightBound)
- Implement `MoveHorzGuardians`: frame 0-3 moving right, 4-7 moving left, boundary checks, speed flag (bit 7 + game clock bit 2)
- Implement `DrawHorizontalGuardians`: set attribute bytes in buffer, draw sprite in blend mode, kill Willy on collision
- Handle cavern-specific frame offset (caverns 7+ except 9,15 use frames 4-7 base offset)

### Task 3.3: Vertical Guardians
- Implement `entity/guardian_vert.go`: `VerticalGuardian` struct (attribute, frame, pixelY, x, yIncrement, minY, maxY)
- Implement `MoveDrawVerticalGuardians`: increment frame (0-3 cycle), add y-increment, reverse at min/max, draw in blend mode
- Only active in caverns >= 8 (except cavern 13 which uses Skylabs instead)

### Task 3.4: Verification — Phase 3
- Run `/simplify` on all files changed in Phase 3
- Diff implementation against Phase 3 tasks: flag MISSING, PARTIAL, or STUB items
- Fix every issue found
- Run `/simplify` again on any files that changed during fixes
- Run `go vet ./...` and `go build ./...` to confirm compilation
- Play test: guardians move correctly, collide with Willy, speed flag works, boundary patrol works

---

## Phase 4: Items, Portal, Air, Scoring

### Task 4.1: Items
- Implement `DrawCollectItemsWillyTouching`: iterate item slots, skip collected (attr=0), check INK white for collection
- Implement item colour cycling: maintain BRIGHT+PAPER, cycle INK through 3→4→5→6
- Implement item graphic rendering using `DrawItem` (8x8, 8 rows)
- Collection: set item attr to 0, add 100 to score

### Task 4.2: Portal
- Implement `entity/portal.go`: `DrawThePortal` — check if Willy's location matches portal, check flashing bit
- If entered + flashing: move to next cavern
- If not entered: draw portal graphic and set attribute bytes
- Activate flashing (bit 7) when all items collected (`AttrLastItemDrawn == 0`)

### Task 4.3: Air Supply
- Implement `DecreaseAirRemaining`: game clock -= 4, on wrap: air supply --, draw partial pixel fill at bar end
- Implement air bar rendering (4 pixel rows, from column $24 to current supply)
- Return zero flag when air exhausted (supply == $24)

### Task 4.4: Scoring System
- Implement `AddToTheScore`: ASCII digit increment with carry, 10000-point extra life (screen flash counter = 8)
- Implement score display: print 6 digits at fixed screen position
- Implement high score comparison and update on game over
- Implement screen flash effect: fill attributes with rotating colour

### Task 4.5: Cavern Transitions
- Implement `MoveToTheNextCavern`: increment cavern number, colour cycling transition (63→0 attribute sweep), convert remaining air to score
- Cavern 19 → cavern 0 (loop)
- Implement cavern reinit on death (`Start7`): reload cavern data, redraw, decrement lives

### Task 4.6: All Cavern Data
- Extract remaining 19 cavern definitions from assembly source
- Validate all parsing against assembly comments (item positions, guardian paths, tile attributes)

### Task 4.7: Verification — Phase 4
- Run `/simplify` on all files changed in Phase 4
- Diff implementation against Phase 4 tasks: flag MISSING, PARTIAL, or STUB items
- Fix every issue found
- Run `/simplify` again on any files that changed during fixes
- Run `go vet ./...` and `go build ./...` to confirm compilation
- Play test: items collectible, portal activates, air depletes, score increments, cavern transitions work, all 20 caverns load

---

## Phase 5: Special Entities

### Task 5.1: Eugene (Cavern 4)
- Implement `entity/eugene.go`: vertical bounce between y=0 and y=$58
- Direction toggle at boundaries
- While items remain: white INK, portal blocked
- All items collected: cycling INK colour (game clock bits 2-4), portal enabled
- Collision detection via blend mode draw

### Task 5.2: Kong Beast (Caverns 7, 11)
- Implement `entity/kong.go`: two switch positions, `FlipSwitchInKongBeastCavern` check
- Left switch: wall dissolution at (11,17) — pixel rows cleared one per frame
- Right switch: Kong starts falling (y += 4 per frame), floor removed, scoring +100 per frame
- At y=100: Kong dies (status=2)
- On ledge: drawn at (0,15), collision checked

### Task 5.3: Skylabs (Cavern 13)
- Implement `entity/skylab.go`: fall at variable speed, disintegration animation (frames 0-7)
- After disintegration: reset y, shift x right by 8 (wrapping)
- Collision = death
- Uses vertical guardian slots but separate movement logic
- Must jump to `MainLoop5` after processing (bypass vertical guardian logic)

### Task 5.4: Light Beam (Cavern 18)
- Implement `entity/lightbeam.go`: trace from (0,23), move down
- Stop on floor/wall
- Reflect on guardian (toggle direction between down and left)
- On Willy: call DecreaseAir x4
- Draw with attribute $77

### Task 5.5: Verification — Phase 5
- Run `/simplify` on all files changed in Phase 5
- Diff implementation against Phase 5 tasks: flag MISSING, PARTIAL, or STUB items
- Fix every issue found
- Run `/simplify` again on any files that changed during fixes
- Run `go vet ./...` and `go build ./...` to confirm compilation
- Play test each special cavern: Eugene blocks/unblocks portal, Kong switches and falls, Skylabs fall and respawn, light beam traces and drains air

---

## Phase 6: Title Screen, Music, Game Over, Demo Mode

### Task 6.1: Title Screen
- Extract title screen graphic data (`TitleScreenDataTop`, 4096 bytes)
- Extract bottom attributes (`BottomAttributes`, 512 bytes)
- Implement `title/title.go`: render title screen with Willy sprite at (9,29)
- Implement `title/banner.go`: scrolling banner text (224 characters, 32 visible at a time)

### Task 6.2: Music — Blue Danube (Title)
- Implement `audio/music.go`: parse 95-note tune data (duration, freq1, freq2)
- Generate square wave audio matching ZX Spectrum beeper output
- Implement piano key visualisation: frequency → attribute address mapping, colour highlighting

### Task 6.3: Music — In the Hall of the Mountain King (In-Game)
- Implement in-game music: 64-note cycle, one note per frame
- Implement music toggle (H-ENTER key group, bit 0 keypress flag, bit 1 music flag)

### Task 6.4: Sound Effects
- Implement `audio/sfx.go`: jumping sound (rising/falling pitch based on jump counter)
- Falling sound (pitch = 16 * airborne status)
- Death sound sequence (8 notes, descending pitch, colour cycling)
- Level complete: air conversion beeps (decreasing pitch with remaining air)
- Final Barrier completion: celebratory rising tone

### Task 6.5: Game Over Sequence
- Implement `DisplayGameOver`: high score check and update
- Boot descent animation: draw boot at increasing y using YTable, extending "trouser leg"
- Rising pitch sound during descent
- Attribute cycling (PAPER colour changes with distance variable bits 2-3)
- "Game Over" text printing at (6,10) and (6,18)
- Glistening text: cycle INK colours for each letter over 6*256 iterations

### Task 6.6: Death Sequence
- Implement `MainLoop19→MainLoop20`: attribute fill cycling (71→64), sound effects
- Lives check: decrement and reinit cavern, or jump to game over

### Task 6.7: Demo Mode
- Implement demo mode: GameModeIndicator = 64, decrements each frame
- No Willy movement in demo mode
- Keypress or joystick returns to title
- At indicator = 0: move to next cavern

### Task 6.8: Verification — Phase 6
- Run `/simplify` on all files changed in Phase 6
- Diff implementation against Phase 6 tasks: flag MISSING, PARTIAL, or STUB items
- Fix every issue found
- Run `/simplify` again on any files that changed during fixes
- Run `go vet ./...` and `go build ./...` to confirm compilation
- Play test: title screen displays, music plays, banner scrolls, game starts on ENTER, death animation plays, game over sequence runs, demo mode cycles caverns

---

## Phase 7: Cheat Codes, Final Barrier, Polish

### Task 7.1: Cheat Code (6031769)
- Implement `cheat/cheat.go`: 8-pair key sequence detection
- Key counter increments on correct pair, resets on wrong key (but not if previous pair still held)
- Counter = 7: display boot next to lives, enable teleport

### Task 7.2: Teleport
- Hold key 6 + press keys 1-5: binary value 0-19 selects cavern
- CPL + AND $1F: complement of keys 1-5 reading
- Jump to `Start7` (reinit cavern)

### Task 7.3: Final Barrier Completion
- Cavern 19 special: top half uses title screen graphic data
- On completion (not in demo/cheat mode): draw Willy at (2,19), swordfish at (4,19)
- Set specific attributes, celebratory sound, then reset to cavern 0

### Task 7.4: Pixel-Perfect Tuning
- Compare rendering output against Spectrum emulator screenshots
- Verify jump arc timing matches original
- Verify guardian patrol boundaries match assembly data
- Verify air depletion rate
- Verify score increments

### Task 7.5: Verification — Phase 7 (Final)
- Run `/simplify` on all files changed in Phase 7
- Diff implementation against Phase 7 tasks: flag MISSING, PARTIAL, or STUB items
- Fix every issue found
- Run `/simplify` again on any files that changed during fixes
- Run `go vet ./...` and `go build ./...` to confirm compilation
- Full play-through test: title → all 20 caverns → Final Barrier completion → title
- Cheat code test: enter 6031769, teleport to each cavern
- Demo mode test: leave on title, verify cavern cycling

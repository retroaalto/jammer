# jammer-go ↔ jammer Parity Plan

## Goals

1. Match the **TUI layout** of the original jammer
2. Match **all keybindings**, reading from `~/jammer/KeyData.ini`
3. Match the **user experience** — views, help screen, settings, dialogs

## Visual Comparison

| | Original Jammer | jammer-go (current) |
|---|---|---|
| **Main view** | Bordered box with prev/current/next 3-song snippet | Full scrollable list, no border |
| **Full list view** | `F` toggles to numbered full-list inside bordered box | Always shows full list |
| **Progress bar** | `❚❚ ⇌ ↻ 0:08 \|█████\| 6:42 5%` with state+shuffle+loop glyphs | `▶ 0:42 ━━━─── 3:12 ▁▂▃` inline |
| **Help** | Full-screen 4-column paginated table (3 pages, `←/→` to navigate) | One-line hint at bottom |
| **Settings** | Full-screen 3-page table with letter-keyed actions | Not implemented |
| **Mini help bar** | `h for help \| c for settings \| f for playlist` bordered row | Two lines of raw key hints |
| **Header** | Current song full path in bordered header row | Tab bar (jammer / Songs / Playlists) |
| **Visualizer** | Separate row above time bar | Inline next to progress bar |
| **Keymapping source** | `~/jammer/KeyData.ini` `[Keybinds]` section | Hardcoded in `Update()` switch |

## Decisions

- **Borders**: lipgloss (close match, stays in BubbleTea)
- **Default view**: 3-song snippet (prev/current/next); `F` toggles to full list; new `settings.json` field `"defaultView": "all"` makes full-list the startup default
- **Settings**: Core settings only
- **Unimplemented keys**: Silently ignore

---

## Phase 1 — `keybinds` package

**New file:** `internal/keybinds/keybinds.go`

- Parse `~/jammer/KeyData.ini` `[Keybinds]` section (simple line-by-line INI reader, no external dependency)
- Build a map of `actionName → KeySpec{key tea.Key, mod modifier}`
- Support modifier syntax: `Shift + N`, `Ctrl + F`, `Alt + S`, `Ctrl + Alt + Y`
- ConsoleKey name → bubbletea mapping table (e.g. `Spacebar→KeySpace`, `RightArrow→KeyRight`, `F3→KeyF3`, `PageUp→KeyPgUp`, `Delete→KeyDelete`, letter keys → rune)
- Expose `func Is(action string, msg tea.KeyMsg) bool` for use in `Update()`
- Fall back to all original defaults when file is absent or a key is missing

---

## Phase 2 — `settings.json` extension

Add `"defaultView"` field (`"default"` or `"all"`) to the settings struct in `main.go` and pass it into the UI model as `initialView viewKind`.

---

## Phase 3 — Layout overhaul (`internal/ui/model.go`)

### 3a. Two song-list view modes

Add `viewDefault` (3-song) alongside existing `viewSongs` (full list). `F` key toggles. Rename `viewSongs` → `viewAll` for clarity.

```
viewDefault   prev/current/next 3-song snippet
viewAll       full scrollable numbered list (current behavior)
```

### 3b. Bordered layout structure

Render as nested lipgloss borders matching original's anatomy:

```
╭── main (full width) ──────────────────────────────────────╮
│  [current song path/title, truncated to width-8]          │
├───────────────────────────────────────────────────────────┤
│  ╭── playlist box ──────────────────────────────────────╮ │
│  │ playlist <name>                                      │ │
│  │ previous: ...   /   or full numbered list            │ │
│  │ current : ...                                        │ │
│  │ next    : ...                                        │ │
│  ╰──────────────────────────────────────────────────────╯ │
│  [spacer rows]                                            │
│  ╭── mini-help ─────────────────────────────────────────╮ │
│  │ [H] for help | [C] for settings | [F] for playlist   │ │
│  ╰──────────────────────────────────────────────────────╯ │
│  [visualizer row — FFT bars]                              │
│  ╭── time bar ──────────────────────────────────────────╮ │
│  │ ❚❚ ⇌ ↻  0:08 |████████           | 6:42  5%          │ │
│  ╰──────────────────────────────────────────────────────╯ │
╰───────────────────────────────────────────────────────────╯
```

### 3c. Progress bar format

`[state] [shuffle] [loop]  [elapsed] |[bar]| [total]  [vol%]`

- State: `▶` playing, `❚❚` paused, `■` stopped
- Shuffle: `⇌` on / ` ` off
- Loop: `↻` all / `↺` one / ` ` off

### 3d. Visualizer row

Move FFT bars from inline next to progress bar to a dedicated row above the time bar.

---

## Phase 4 — Keybinding wiring

Replace all hardcoded key checks in `Update()` with `keybinds.Is(action, msg)`. Correct or add:

| Action | Original default | Change |
|---|---|---|
| `Shuffle` | `S` | Was `R` → fix to `S` |
| `PlayRandom` | `R` | Was `r` → fix to `R` |
| `VolumeUp` / `VolumeDown` | `UpArrow` / `DownArrow` | Add (was `+`/`-` only) |
| `VolumeUpByOne` / `VolumeDownByOne` | `Shift+Up/Down` | Add |
| `Mute` | `M` | Add (toggle mute in player) |
| `ShowHidePlaylist` | `F` | Add (toggle viewDefault↔viewAll) |
| `ListAllPlaylists` | `Shift+F` | Add (open playlist browser, same as current Tab) |
| `Help` | `H` | Add → `viewHelp` |
| `Settings` | `C` | Add → `viewSettings` |
| `ToSongStart` | `0` | Add (seek to 0) |
| `ToSongEnd` | `9` | Add (seek to end) |
| `SaveCurrentPlaylist` | `Shift+S` | Add (save playlist to disk) |
| `SearchInPlaylist` | `F3` | Add as alias for `/` filter |
| `RenameSong` | `F2` | Add → `viewRename` |
| `PlayOtherPlaylist` | `Shift+O` | Add (open playlist browser) |
| `RedownloadCurrentSong` | `Shift+B` | Add as alias for `d` |
| `ToggleInfo` | `I` | Add → `viewInfo` |
| `CurrentState` | `F12` | Add → same as `I` |
| `PlaylistViewScrollup/down` | `PageUp/PageDown` | Add to both viewAll and viewPlaylists |
| `ToMainMenu` | `Escape` | Ensure exits any view back to viewDefault |
| `AddSongToPlaylist` | `Shift+A` | Add → `viewAddSong` input |

---

## Phase 5 — Help screen (`viewHelp`)

**New view** (extracted to `internal/ui/help.go`)

- 4-column table: `Keybind | Description | Keybind | Description`
- 10 pairs per page; page indicator `(1/3)` in header
- `←`/`→` or `PageUp`/`PageDown` to paginate
- Keys displayed are read from the `Keybinds` singleton (respects user's ini)
- `Escape` / `ToMainMenu` → back to previous view

---

## Phase 6 — Settings screen (`viewSettings`)

**New view** (`internal/ui/settings.go`)

- Pages navigated with `→`/`←`
- Core settings rows: Forward/rewind seconds (`A`/`B`), volume step (`C`), auto-save (`D`), visualizer toggle (`G`), skip-errors (`L`), quick-search (`P`)
- Read from / write back to `~/jammer/settings.json`
- `Escape` → back

---

## Phase 7 — Additional views

| viewKind | Key | Description |
|---|---|---|
| `viewRename` | `F2` | Inline input to rename current song title in playlist |
| `viewInfo` | `I` / `F12` | Song tag info overlay (title, artist, album, duration) |
| `viewAddSong` | `Shift+A` | Input box to add a URL or file path to the playlist |

These reuse the existing filter-input pattern (text input field inline in BubbleTea model, `Enter` to confirm, `Escape` to cancel).

---

## File-level change map

| File | Change |
|---|---|
| `internal/keybinds/keybinds.go` | **New** — INI parser + key mapping |
| `main.go` | Add `defaultView` to settings struct; pass to UI |
| `internal/ui/model.go` | Layout overhaul, new viewKinds, replace key switches, progress bar format |
| `internal/ui/help.go` | **New** — help screen renderer |
| `internal/ui/settings.go` | **New** — settings screen renderer |
| `internal/player/player.go` | Add `Mute()` toggle if missing |
| `~/jammer/settings.json` schema | Add `"defaultView"` field |

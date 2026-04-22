# Jammer TUI Elements — Current State (Spectre.Console)

This document catalogues every TUI element in jammer as it exists today.
Its purpose is to serve as the reference for a future Terminal.Gui migration —
what needs to be ported, redesigned, or dropped.

---

## 16 Distinct TUI Elements

| # | Element | Source file | Interactive |
|---|---|---|---|
| 1 | Main Player View (default/all) | `TUI.cs:DrawPlayer()` | No |
| 2 | Playlist — prev/current/next | `Components/PlaylistComponent.cs` | No |
| 3 | Playlist — all songs scrollable list | `Components/PlaylistComponent.cs` | Keyboard scroll |
| 4 | Mini Help Bar | `TUI.cs:~115` | No |
| 5 | Visualizer (FFT bar) | `Components/VisualizerComponent.cs`, `Visual.cs` | No |
| 6 | Progress Bar / Time Row | `Components/PlayerTimeComponent.cs`, `TUI.cs:ProgressBar()` | No |
| 7 | Help Menu (paginated, 4-col) | `Components/HelpMenuComponent.cs` | Pagination keys |
| 8 | Settings (paginated, 3-col) | `Components/SettingsComponent.cs` | Pagination + action keys |
| 9 | Edit Keybindings | `TUI.cs:EditKeyBindings()` | Select + edit |
| 10 | Change Language | `TUI.cs:ChangeLanguage()` | Select |
| 11 | Input Box (modal) | `Message.cs:Input()` | Text entry |
| 12 | Data / Info Box (modal) | `Message.cs:Data()` | Dismiss (keypress or auto-close) |
| 13 | Custom Menu Select (modal) | `Message.cs:CustomMenuSelect()` | Arrow keys, Enter, Escape |
| 14 | Legacy MultiSelect (Spectre native) | `Message.cs:MultiSelect()` | Arrow keys, Enter |
| 15 | CLI Help Tables (non-interactive) | `TUI.cs:CliHelp()`, `TUI.cs:PlaylistHelp()` | No |
| 16 | Top-of-Player Inline Message | `TUI.cs:PrintToTopOfPlayer()` | No |

---

## Views (playerView states)

The app has a single global `Start.playerView` string that controls which screen is shown.
There is no router, no stack, and no animation between states.

| State | File | Description |
|---|---|---|
| `default` | `TUI.cs:DrawPlayer()` | Main player — prev/current/next song, progress bar |
| `all` | `TUI.cs:DrawPlayer()` | Same outer shell as default, full song list instead of prev/curr/next |
| `help` | `TUI.cs:DrawHelp()` | Keybinding reference, paginated |
| `settings` | `TUI.cs:DrawSettings()` | Settings table, paginated |
| `editkeybindings` | `TUI.cs:EditKeyBindings()` | 2-col table of all keybinds, editable |
| `changelanguage` | `TUI.cs:ChangeLanguage()` | List of locale files, selectable |
| `fake` | `TUI.cs:DrawPlayer()` | Internal transition state, renders like `default` |

---

## 1. Main Player View (`default` and `all`)

**Source:** `TUI.cs:DrawPlayer()`, `Jammer.Core/src/TUI.cs:24`

The outermost container is a single Spectre `Table` (`mainTable`) that spans the full
console width. Everything else is nested inside it as rows.

**Structure (top to bottom):**

```
╭─ mainTable ────────────────────────────────────────────────────────╮
│  [column header = current song path, truncated with ...]           │
├────────────────────────────────────────────────────────────────────┤
│  songsTable (PlaylistComponent)                          [centered] │
├────────────────────────────────────────────────────────────────────┤
│  (N empty rows to fill terminal height)                             │
├────────────────────────────────────────────────────────────────────┤
│  helpTable (mini key hint bar)                                      │
├────────────────────────────────────────────────────────────────────┤
│  (empty row, only when visualizer is enabled)                       │
├────────────────────────────────────────────────────────────────────┤
│  timeTable (PlayerTimeComponent)                                    │
╰────────────────────────────────────────────────────────────────────╯
```

**Observed output (default view, no playlist):**
```
╭─ /home/.../easy_mode.ogg ──────────────────────────────────────────╮
│ ╭──────────────────────────────────────────────────────────────╮   │
│ │ previous : -                                                  │   │
│ │ current  : /home/.../easy_mode.ogg                            │   │
│ │ next     : /home/.../Unity.wav                                │   │
│ ╰──────────────────────────────────────────────────────────────╯   │
│  (empty rows to fill height)                                        │
│ ╭─ h for help | c for settings | f for playlist ─╮                  │
│ │   ▃▂  ▆  ▇  ▃  ██  ▁       (visualizer bar)   │                  │
│ ╭──────────────────────────────────────────────────────────────╮   │
│ │ ❚❚ ⇌  ↻  0:05 |██████████            | 0:51   0%             │   │
│ ╰──────────────────────────────────────────────────────────────╯   │
╰────────────────────────────────────────────────────────────────────╯
```

**Key facts:**
- Entire screen is re-rendered on every timer tick via `AnsiConsole.Cursor.SetPosition(0,0)` + `AnsiConsole.Write(mainTable)`.
- No `LiveDisplay` or background render loop in Spectre — all manual.
- Height padding (empty rows) is calculated by `LayoutCalculator.CalculateTableRowCount()`.
- The song path header is a Spectre Table *column header*, not a row.

---

## 2. Playlist Sub-Component (Default View)

**Source:** `Components/PlaylistComponent.cs`, `PlaylistComponent.CreateNormalPlaylistTable()`

A single-column bordered table showing 3 lines: previous / current / next song.

**Sub-modes:**
- **No playlist:** Shows raw previous/current/next song paths.
- **With playlist:** Header row = playlist name + position (e.g., `Playlist: foo [3/10]`), then prev/curr/next.
- **RSS feed:** Header = feed title + author + "Exit with [key]" hint. Rows show `previous/current/next` pub dates.

**Theme section:** `GeneralPlaylist`

---

## 3. Playlist Sub-Component (All Songs View)

**Source:** `Components/PlaylistComponent.cs`, `PlaylistComponent.CreateAllSongsPlaylistTable()`

Shows the full song list with numbered rows. The visible window is managed by
`IniFileHandling.ScrollIndexLanguage` (a shared scroll offset).

**Structure:**
```
╭─ Playlist: foo ──────────────────────────────────────────────────╮
│  Move with PageUp, PageDown                                       │
│  Play with Enter. Delete with Delete.                             │
│  1. song-name-truncated                                           │
│  2. song-name-truncated         ← current song, highlighted      │
│  3. song-name-truncated                                           │
╰───────────────────────────────────────────────────────────────────╯
```

**Row states:**
- Current playing song: `WholePlaylist.CurrentSongColor`
- "Choosing" (cursor) song: `WholePlaylist.ChoosingColor`
- Favorite: prefixed with `★`
- Normal: `WholePlaylist.NormalSongColor`

Maximum 7 items visible at a time; window scrolls around current/cursor song.

**Theme section:** `WholePlaylist`

---

## 4. Mini Help Bar

**Source:** `TUI.cs:~115-132`

Single-column bordered table, always shown at the bottom of `default`/`all`/`editkeybindings`/`changelanguage` views.

**Content:**
```
╭─ [h] for help | [c] for settings | [f] for playlist ─╮
```
Each letter is individually colored (`HelpLetterColor`, `SettingsLetterColor`, `PlaylistLetterColor`).
Separator `|` has its own color (`ForSeperatorTextColor`).

**Theme section:** `Playlist.MiniHelp*`

---

## 5. Visualizer

**Source:** `Components/VisualizerComponent.cs`, `Visual.cs`

Rendered *outside* the main table — directly to the console via raw ANSI cursor
positioning (`AnsiConsole.Cursor.SetPosition(5, consoleHeight - 5)`).
This puts it in the empty row inside `mainTable` that was reserved for it.

**Mechanics:**
- FFT data sampled from BASS audio channel each frame.
- Values mapped to block characters: `▁▂▃▄▅▆▇█` (configurable via `Visualizer.UnicodeMap` in theme).
- When paused, amplitude gradually decays (`scaleFactor *= 0.95^...`).
- Full width of terminal (`consoleWidth + 35` for the raw character count).
- Playing color vs paused color from `Visualizer.PlayingColor` / `Visualizer.PausedColor`.
- Refresh timing controlled by `Visual.refreshTime` (default 35 ms), set in `Visualizer.ini`.
- Optional — toggled in settings, guarded by `Preferences.isVisualizer`.

**Theme section:** `Visualizer`

---

## 6. Progress Bar / Time Row

**Source:** `Components/PlayerTimeComponent.cs`, `TUI.cs:ProgressBar()`

A single-column bordered table containing one composite string line:

```
❚❚  ⇌  ↻  0:05 |███████████████          | 0:51   0%
^   ^   ^   ^       ^                      ^     ^
│   │   │   │       │                      │     └── Volume (muted = different color)
│   │   │   │       └── fill chars         └── total duration
│   │   │   └── elapsed time
│   │   └── loop state icon (↻ always / once / off)
│   └── shuffle icon (⇌ on / off)
└── playback state icon (❚❚ pause / ▶ play / ■ stop / → next / ← prev)
```

All icons, colors, and fill characters are configurable in the `Time` theme section.

**Inline indicators:**
- Volume: shows `%` value, color changes when muted (`VolumeColorMuted` vs `VolumeColorNotMuted`).
- Shuffle: `ShuffleOnLetter` / `ShuffleOffLetter`, each with its own color.
- Loop: three states — `LoopOnLetter` (always), `LoopOnceLetter` (once), `LoopOffLetter` (off).

**Theme section:** `Time`

---

## 7. Help Menu View

**Source:** `Components/HelpMenuComponent.cs`

Full-screen 4-column table replacing the main player when `playerView == "help"`.

```
╭─ Keybinds ──┬─ Description ─────────┬─ Keybinds ──┬─ Description (1/3) ─╮
│ Spacebar    │ Play/Pause             │ n           │ Next song             │
│ p           │ Previous song          │ q           │ Quit                  │
│ ...         │ ...                    │ ...         │ ...                   │
│             │                        │ PgDn/→      │ Next page             │
╰─────────────┴────────────────────────┴─────────────┴───────────────────────╯
```

**Details:**
- 10 items per page (25 total keybindings = 3 pages).
- Modifier keys coloured differently: `ModifierTextColor_1`, `_2`, `_3` for Shift/Ctrl/Alt.
- Actual key coloured with `ControlTextColor`.
- Paginated: PgDn/→ next, PgUp/← previous. Navigation hints added as table rows.
- Mini help bar is shown below the table (same as main view).

**Theme section:** `GeneralHelp`

---

## 8. Settings View

**Source:** `Components/SettingsComponent.cs`

3-column table: `Setting Name | Current Value | Change Key + Description`

```
╭─ Settings ──────────────────────────┬───────────┬─ Change Value (1/3) ──────────────╮
│ Forward seconds                     │ 5 sec     │ A  To Change                       │
│ Rewind seconds                      │ 5 sec     │ B  To Change                       │
│ ...                                 │ ...       │ ...                                │
│                                     │           │ PgDn/→ Next page                   │
╰─────────────────────────────────────┴───────────┴────────────────────────────────────╯
╭─ To Main Menu: Escape ─╮
```

**17 settings across 3 pages** (6 per page):
1. Forward seconds | 2. Rewind seconds | 3. Change Volume By | 4. Playlist Auto Save
5. Load Effects | 6. Toggle Media Buttons | 7. Toggle Visualizer | 8. Load Visualizer
9. Set Soundcloud Client ID | 10. Fetch Client ID | 11. Toggle Key Modifier Helpers
12. Toggle Skip Errors | 13. Toggle Playlist Position | 14. Skip RSS after time
15. Amount of time to skip RSS | 16. Toggle Quick Search | 17. Favorite Explainer

A separate smaller navigation table below shows "To Main Menu: [key]".

**Theme section:** `GeneralSettings`

---

## 9. Edit Keybindings View

**Source:** `TUI.cs:EditKeyBindings()`

2-column table: `Description | Current Keybind`

```
╭─ Description ─────────────────────────┬─ Keybind ─────────────────────╮
│ To main menu                           │ Escape                         │  ← highlighted row (selecting)
│ Play/Pause                             │ Spacebar                       │
│ ...                                    │ ...                            │
╰────────────────────────────────────────┴────────────────────────────────╯
Press Enter to edit highlighted keybind, move up and down with: PageUp, PageDown
Press Shift + Alt + Delete to reset keybinds
```

**Edit mode:** When a keybind is being edited, the text below changes to show the
key combination entered so far, with modifiers listed (e.g., `Shift + Ctrl + A`).

**Theme section:** `EditKeybinds`

---

## 10. Change Language View

**Source:** `TUI.cs:ChangeLanguage()`

1-column table listing available locale `.ini` files.

```
╭─ Language file ─────────────────────╮
│ /home/.../locales/en.ini             │  ← currently selected (CurrentLanguageColor)
│ /home/.../locales/fi.ini             │
│ /home/.../locales/pt-br.ini          │
╰──────────────────────────────────────╯
Enter to choose the language, move up and down with: PageUp, PageDown
```

**Theme section:** `LanguageChange`

---

## 11. Input Box (Modal)

**Source:** `Message.cs:Input()`

A two-table overlay that takes over the screen for text entry.

```
╭─ [title] ──────────────────────────────────────────────╮
│ ╭─ [input prompt] ──────────────────────────────────╮  │
│ │  _  (cursor here, line 5 col 5)                   │  │
│ ╰────────────────────────────────────────────────────╯  │
╰─────────────────────────────────────────────────────────╯
```

**Variants:**
- `oneChar = true` — single keypress, no line editor, no enter needed.
- `prefillText` — line pre-filled with existing text (for rename etc.).
- `setText[]` — history pre-loaded with suggestions.
- `options.EnableAutoComplete = true` — tab-completion from a list (via JRead library).
- Input is handled by JRead (custom readline library), not Spectre prompts.
- Cursor is manually positioned with `AnsiConsole.Cursor.SetPosition(5, 5 + newlines)`.

**Theme section:** `InputBox` (BorderStyle, InputBorderStyle, TitleColor, InputTextColor)

---

## 12. Data / Info Box (Modal)

**Source:** `Message.cs:Data()`

Same two-table layout as Input Box, but read-only. No input field — only displays data.

```
╭─ [title] ──────────────────────────────────────────────╮
│ ╭─ [message text] ───────────────────────────────────╮ │
│ │                                                     │ │
│ ╰─────────────────────────────────────────────────────╯ │
╰─────────────────────────────────────────────────────────╯
```

**Variants:**
- `isError = true` — uses `TitleColorIfError`, `InputTextColorIfError`, `InputBorderColorIfError`.
- `closeAfterMs > 0` — auto-closes, with optional keypress-to-dismiss-early.
- `readKey = false` — auto-closes without waiting for input.

Used for error messages, confirmations, status feedback.

**Theme section:** `InputBox` (same section, error keys for error variant)

---

## 13. Custom Menu Select (Modal)

**Source:** `Message.cs:CustomMenuSelect()`

A custom-implemented scrollable selection list (replaces Spectre's built-in `SelectionPrompt`
because ESC handling and right-aligned author display were needed).

```
╭─ [title] (Use arrows, Enter to select, ESC to cancel, PgUp/PgDn to scroll) ──╮
│  Page 1 of 3 | Items 1-10 of 25                                                │  (if >10 items)
│  Song Title                                              Author Name            │
│  > [b]Selected Song[/] <                                Author Name            │  ← highlighted
│  Song Title                                              Author Name            │
╰────────────────────────────────────────────────────────────────────────────────╯
╭─ Description ─────────────────────────────────────────────────────────────────╮  (optional)
│  [item description text]                                                        │
╰────────────────────────────────────────────────────────────────────────────────╯
```

**Keyboard:**
- `Up/Down` — move selection
- `PgUp/PgDn` — page jump
- `Ctrl+Up/Down` or `Home/End` — jump to start/end
- `Enter` — confirm, returns `DataURI` of selected item
- `Escape` — cancel, returns `null`

**Author alignment:** Title is left-aligned; author is right-aligned using space padding
calculated from visual width (handles CJK double-width characters).

Used for: search results (YouTube, SoundCloud, playlist search), song list picker.

**Theme section:** `InputBox`

---

## 14. Legacy Multi-Select (Spectre Native)

**Source:** `Message.cs:MultiSelect()`

Wraps Spectre.Console's `SelectionPrompt<string>`. Largely superseded by `CustomMenuSelect`.
Still present in codebase but not called from most flows.

- Prepends a `Cancel` option.
- No ESC handling (limitation of Spectre's built-in prompt).

---

## 15. CLI Help Tables (Non-Interactive)

**Source:** `TUI.cs:CliHelp()`, `TUI.cs:PlaylistHelp()`

Two separate 2-column tables printed to stdout at startup when `--help` is passed.
Not part of the interactive TUI — printed once and exits.

```
┌─ Commands ──────────────────────────────────────────┬─ Description ──────────────────┐
│ jammer <file> ...                                    │ Play song(s) from file(s)       │
│ ...                                                  │ ...                             │
└──────────────────────────────────────────────────────┴─────────────────────────────────┘
┌─ Playlist Commands ─────────────────────────────────┬─ Description ──────────────────┐
│ ...                                                  │ ...                             │
└──────────────────────────────────────────────────────┴─────────────────────────────────┘
```

Uses Spectre's default border style (square corners).

---

## 16. Top-of-Player Inline Message

**Source:** `TUI.cs:PrintToTopOfPlayer()`

Not a table. Uses a raw ANSI escape sequence `\x1b[2;3H` to overwrite the song path
area in the main table header with a transient message. Padded with spaces to clear
the old content.

Used for brief status messages (e.g., "Song added to playlist").

---

## Rendering Architecture Summary

| Concern | Current approach |
|---|---|
| Screen redraw | Full re-render: `Cursor.SetPosition(0,0)` + `Write(table)` every tick |
| Partial updates | Visualizer and time can be drawn at their positions independently |
| Layout engine | Manual: `LayoutConfig` / `LayoutCalculator` calculates row counts and widths from `consoleWidth`/`consoleHeight` |
| Widget nesting | Spectre `Table` rows contain other `Table` instances |
| Input capture | JRead (custom readline), `Console.ReadKey()`, SharpHook (global hotkeys) |
| Cursor | Manually shown/hidden around input prompts |
| Color/style | All via `Themes.sColor()` wrapping Spectre markup, `Themes.bStyle()` for borders |

---

## Theme Sections → UI Areas

| Theme section | UI areas it styles |
|---|---|
| `Playlist` | Main outer table, song path header, mini help bar |
| `GeneralPlaylist` | Default-view song list sub-table |
| `WholePlaylist` | All-songs-view song list sub-table |
| `GeneralHelp` | Help menu table |
| `GeneralSettings` | Settings table |
| `EditKeybinds` | Edit keybindings table |
| `LanguageChange` | Language picker table |
| `InputBox` | All modal dialogs (input, data, custom select) |
| `Time` | Progress bar row |
| `Visualizer` | Audio visualizer bar |
| `Rss` | RSS feed info in playlist sub-table |

---

## Known Rendering Pain Points

- **Full-screen re-render every tick** causes visible flicker when the terminal cannot
  draw fast enough, especially the visualizer + progress bar combination.
- **Visualizer outside the table** is drawn with a raw escape at a hardcoded offset
  (`consoleHeight - 5`). If the terminal is resized, it can misalign.
- **Magic index constants** in `LayoutConfig` are brittle — adding a row to any nested
  table requires updating the padding row count math manually.
- **`AnsiConsole.Clear()`** is called on page transitions in help/settings, causing
  a flash. The main view avoids it intentionally.
- **No event-driven redraw** — the main loop polls `Console.KeyAvailable` and redraws
  on a timer, even when nothing has changed.

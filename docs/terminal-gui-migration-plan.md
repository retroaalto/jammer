# Terminal.Gui Migration Plan

## Problem Statement

The current UI has two compounding structural problems that justify a full TUI framework migration:

### 1. Global keypress bleed

`Keyboard.cs:CheckKeyboardAsync()` is a single monolithic function.
The view-specific blocks at the top form an `if / else if / else if / else if` chain that covers
`editkeybindings`, `changelanguage`, `all`, and `help`. Then `settings` is a **separate `if`**
(not `else if`) at line 257. The global `else switch(Action)` at line 455 is paired only with
`if(settings)`.

This means:

- When `playerView == "help"`: the help block fires (lines 235–256), then `if(settings)` is
  `false`, so the global `else switch` also fires. Pressing **Space in the help view pauses the
  song**. Pressing **Up/Down adjusts the volume**.
- The same bleed affects `changelanguage`, `editkeybindings`, and `all`.
- `settings` avoids it only because of the accidental `if/else` pairing, not by design.

This is **not patched in the old code** — it is resolved structurally by Terminal.Gui's
focus model, which delivers key events to the focused view first and stops propagation when
the view consumes them.

### 2. PgUp/PgDn as list navigation

All list navigation (song list scroll, language picker, keybind list, help pages) uses
`PlaylistViewScrollup` / `PlaylistViewScrolldown`, which are bound to PgUp/PgDn by default.
Arrow keys (`Up`, `Down`) are reserved globally for volume. This is non-standard for a TUI —
users expect arrow keys to move a cursor in a list and are confused by PgUp/PgDn for
single-row movement.

Terminal.Gui solves both problems structurally: each `View` receives only keys that are not
consumed by a focused child, and `ListView` uses arrow keys natively.

---

## Scope of the Migration

**Old code is left entirely unchanged** until Phase 3 (removal). No patches, no refactors,
no incremental edits to `Keyboard.cs`, `TUI.cs`, or any existing file. All new work lives
in new files.

**Keep unchanged throughout:**
- BASS audio engine (`ManagedBass`) — untouched.
- Playback logic (`Play.cs`, `Effects.cs`, `Visual.cs` FFT) — untouched.
- Playlist / song data model (`Song`, `Playlists`, `Utils`) — untouched.
- Download / search backends (`Download.cs`, `Search.cs` logic) — untouched.
- Preferences and settings persistence — untouched.
- Locale / theming data — keep, adapt rendering to Terminal.Gui color scheme.
- Spectre.Console for `--help` and `--version` CLI output — keep, those are non-interactive.

**Replace (all at once in Phase 3, after new UI is stable):**
- `TUI.cs` + all `Components/` — replaced by Terminal.Gui views.
- `Keyboard.cs:CheckKeyboardAsync()` — replaced by Terminal.Gui key bindings per view.
- `Message.cs` interactive methods (`Input`, `Data`, `CustomMenuSelect`) — replaced by
  Terminal.Gui `Dialog` + `TextField`.
- `Layout/` classes — replaced by Terminal.Gui layout system.
- JRead readline library — replaced by Terminal.Gui `TextField`.

---

## Migration Phases

---

### Phase 1 — Terminal.Gui Scaffold + Proof of Concept

**Goal:** Add Terminal.Gui to the project, prove the integration works alongside the existing
code, and build one real view (the all-songs list) as the proof of concept. Old rendering
continues to run on the default path — nothing breaks.

#### 1.1 Add NuGet package

```bash
dotnet add Jammer.Core package Terminal.Gui
```

Current version: `2.x` (v2 is the active branch as of 2024).

#### 1.2 Application wrapper and launch path

```csharp
// Jammer.Core/src/TGuiApp.cs
public static class TGuiApp
{
    public static bool Enabled { get; private set; }

    public static void Init()
    {
        Application.Init();
        Enabled = true;
    }

    public static void Shutdown()
    {
        Application.Shutdown();
        Enabled = false;
    }

    public static void Run(Toplevel top) => Application.Run(top);
}
```

`--new-ui` is parsed in `Args.cs:CheckArgs()` alongside all other flags. It sets a static
bool (`Start.UseNewUI`) and **must be stripped from `Utils.Songs`** before song path
processing (same pattern as `-D` debug flag), otherwise it gets treated as a file path.

In `Start.StartUp()`, after BASS init and initial song load, the path branches:

```csharp
if (Start.UseNewUI)
{
    TGuiApp.Init();
    var top = JammerToplevel.Build();   // constructs the full layout
    visualizerThread.Start();            // still needed — writes to frame buffer
    TGuiApp.Run(top);                    // blocks until app exits
}
else
{
    loopThread.Start();
    visualizerThread.Start();
}
```

`Application.Run()` is blocking and replaces `Start.Loop()` entirely on the new-UI path.
The existing `EqualizerLoop()` thread continues to run — in the new UI it writes FFT data
to the visualizer frame buffer instead of setting `drawVisualizer = true`.

#### 1.3 Player state → UI updates

The current `Loop()` polls audio state and calls `TUI.*` draw methods directly.
In Terminal.Gui, UI updates from non-UI threads must go through `Application.Invoke()`.
A lightweight `System.Threading.Timer` (≈ 250 ms tick) calls `Application.Invoke()` to
refresh the player status bar label and progress bar:

```csharp
var uiTimer = new System.Threading.Timer(_ =>
{
    Application.Invoke(() =>
    {
        _playerStatusBar.UpdateProgress();
        Application.Refresh();
    });
}, null, 0, 250);
```

Song-change events from `Play.cs` also go through `Application.Invoke()` to update the
content area and player bar.

#### 1.4 Top-level layout

Terminal.Gui's `Application.Top` contains a permanent player bar and visualizer bar at the
bottom; the content area above them switches between views.

```
┌─ Application.Top ────────────────────────────────────────────────────────────┐
│  ┌─ ContentArea (fills all rows except status bars) ───────────────────────┐ │
│  │  [active view goes here — PlayerView, HelpView, etc.]                   │ │
│  └─────────────────────────────────────────────────────────────────────────┘ │
│  ┌─ VisualizerBar (1 row) ─────────────────────────────────────────────────┐ │
│  └─────────────────────────────────────────────────────────────────────────┘ │
│  ┌─ PlayerStatusBar (1 row) ───────────────────────────────────────────────┐ │
│  │  ❚❚  ⇌  ↻  0:05 |████████████████████     | 0:51   0%                  │ │
│  └─────────────────────────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────────────────────┘
```

The visualizer and player status bar are permanent fixtures — they never switch out.
Only the `ContentArea` changes between views. This removes the "empty row padding" hack
from the current layout and eliminates all magic index constants from `LayoutConfig`.

#### 1.5 Proof-of-concept: AllSongsView as `ListView`

```csharp
public class AllSongsWindow : Window
{
    private readonly ListView _list;

    public AllSongsWindow()
    {
        Title = "Playlist";
        _list = new ListView
        {
            X = 0, Y = 0,
            Width = Dim.Fill(),
            Height = Dim.Fill(),
        };
        _list.OpenSelectedItem += OnSongSelected;
        Add(_list);
    }

    public void Refresh()
    {
        _list.SetSource(Utils.Songs.Select(SongExtensions.Title).ToList());
        _list.SelectedItem = Utils.CurrentSongIndex;
    }

    private void OnSongSelected(object sender, ListViewItemEventArgs e)
    {
        Play.PlaySong(Utils.Songs, e.Item);
    }
}
```

Arrow keys move the cursor. Enter plays. No `PlaylistViewScrollup/down` mapping needed.
Volume keys do not fire because this `Window` does not have a binding for them —
they only live on the player status bar via `Application.KeyDown`.

---

### Phase 2 — View-by-View Migration

Build each remaining view as a new Terminal.Gui `Window` subclass. The old Spectre path
remains working via the `--new-ui` flag until all views are complete.

**Order (lowest risk → highest risk):**

#### 2.1 ChangeLanguageView → `ListView`

Simple list with single selection. Direct replacement. No complex state.
Arrow keys select, Enter confirms, Escape exits.

#### 2.2 HelpView → scrollable `TableView`

4-column `TableView`, scrollable via arrow keys. Pagination becomes internal
scroll — no explicit page concept needed since Terminal.Gui scrolls natively.
PgDn/PgUp can still be bound to explicit page jumps on top of scroll if desired.

#### 2.3 SettingsView → `TableView` + dialogs

3-column `TableView`. Action keys per row trigger `Dialog` boxes (replaces
`Message.Input` calls). Each `Dialog` has a `TextField` for value entry.

#### 2.4 EditKeybindingsView → `TableView`

2-column `TableView`. Enter on a row opens a capture `Dialog` that reads the
next keypress combination and writes it back.

#### 2.5 PlayerView (default) — `FrameView` + `Labels`

Three `Label` widgets for previous / current / next song. These are static
displays — no interaction needed beyond media keys on the status bar.

#### 2.6 AllSongsView polish

Favorite star prefix, current song highlight color, delete key binding.
(Core ListView done in Phase 1.)

#### 2.7 Input / Data modals → `Dialog`

`Message.Input()` → `Dialog` containing a `TextField`. Autocomplete maps to
Terminal.Gui's built-in autocomplete on `TextField`.

`Message.Data()` → `MessageBox.Query()` or a custom `Dialog` with a close button.

`Message.CustomMenuSelect()` → `Dialog` containing a `ListView`. ESC closes.
Author right-alignment: use `TableView` with two columns instead of space padding.

#### 2.8 RSS feed view

Currently handled inline in `PlaylistComponent`. Becomes a dedicated `Window`
subclass showing feed title, author, exit hint, and a `ListView` of episodes.

#### 2.9 Visualizer

Custom `View` subclass that overrides `OnDrawContent()`.
Draws the FFT bar character-by-character using `Driver.AddStr()`.
The BASS callback runs on a separate thread — a thread-safe frame buffer
(array of characters) is written by the audio thread and read by `OnDrawContent()`.
Positioned as the permanent second-to-last row of `Application.Top`.

---

### Phase 3 — Remove Old UI Code

Once all views are running stably under `--new-ui` and the flag is flipped to default-on:

1. Delete `Jammer.Core/src/TUI.cs` interactive methods (keep `CliHelp`, `PlaylistHelp`, `Version`).
2. Delete `Jammer.Core/src/Components/` (all replaced by Terminal.Gui views).
3. Delete `Jammer.Core/src/Layout/` (replaced by Terminal.Gui layout engine).
4. Delete `Jammer.Core/src/Message.cs` (replaced by Terminal.Gui dialogs).
5. Delete `Jammer.Core/src/Keyboard.cs` (replaced by Terminal.Gui key bindings per view).
6. Remove JRead submodule (`JRead/`) from the solution.
7. Remove SharpHook dependency only if global media button support is no longer needed;
   otherwise keep it — it operates independently of the TUI library.
8. Keep `Spectre.Console` as a dependency for CLI output (`--help`, `--version`).
9. Remove the `--new-ui` flag.

---

## Resizing Requirement

All Terminal.Gui views **must** respond correctly to terminal resize events.

- Use `Dim.Fill()` and `Pos.AnchorEnd()` for all width/height dimensions — never hardcode
  row or column counts.
- Never use magic index constants (e.g. `LayoutConfig.DEFAULT_VIEW_MAGIC_INDEX`) for
  heights or widths in any new view.
- The content area must expand or contract as the terminal window resizes without any
  manual resize handler needed — Terminal.Gui's layout engine handles this automatically
  when `Dim.Fill()` is used correctly.
- Verify resize behavior using the `tui-testing` skill by resizing the tmux pane after
  launching the app. Every view must be verified this way before merging.

---

## Key Design Decisions for Terminal.Gui

### Media key scope

Media keys (Space, Left/Right seek, Up/Down volume) are registered at the
`Application` level via `Application.KeyDown`, not on individual views.

```csharp
Application.KeyDown += (e) =>
{
    if (e.Key == Key.Space && !IsModalOpen()) { PauseSong(); e.Handled = true; }
    // etc.
};
```

When a `Dialog` or focused `ListView` is open, Terminal.Gui's event system
delivers key events to the focused view first. If the focused view consumes
the key (e.g., ListView consuming Up/Down), it does not bubble up to
`Application.KeyDown`. This is the correct behavior with no additional code.

### Arrow key conflict resolution

| Context | Up / Down |
|---|---|
| No dialog open, player is the focus | Volume (via Application.KeyDown fallback) |
| ListView focused (song list, language, etc.) | Move cursor (consumed by ListView) |
| Dialog with TextField focused | Do nothing (consumed by TextField) |
| TableView focused (help, settings) | Scroll rows |

### Theme mapping

`Themes.CurrentTheme` color strings map to Terminal.Gui `ColorScheme` objects.
Create one `ColorScheme` per major UI area (player, list, dialog, status bar)
and apply them to the corresponding `Window` / `View` instances.
The existing `Themes` INI format can be parsed into `ColorScheme` structs
without changing the theme files.

### Removal of "magic index" layout constants

`LayoutConfig.DEFAULT_VIEW_MAGIC_INDEX = 18` and friends exist to compensate
for Spectre's inability to fill remaining space. Terminal.Gui uses
`Dim.Fill()` and `Pos.AnchorEnd()` — no magic numbers needed.

---

## Risk and Rollback

Each phase is independently shippable with a fallback:

- **Phase 1–2**: Gated behind `--new-ui` flag. Old Spectre path stays working.
- **Phase 3**: Old code deletion only happens after Terminal.Gui path is stable
  and `--new-ui` is the default.

The `--new-ui` flag becomes the default once Phase 2 is complete, and is
removed as part of Phase 3.

---

## Design Decisions

1. **Mouse support** — Enabled. Terminal.Gui mouse support will be active for
   clicks in lists, dialogs, and scrollbars.

2. **Visualizer thread safety** — Thread-safe frame buffer confirmed. The BASS
   audio thread writes FFT bar characters into a `char[]` buffer;
   `View.OnDrawContent()` reads from it on the UI thread.

3. **SharpHook (global hotkeys)** — Kept. SharpHook stays in the project and
   continues to handle OS-level media key interception independently of the TUI
   library.

4. **JRead autocomplete** — Verify parity before removal. `TextField`
   autocomplete and history navigation must be confirmed equivalent to JRead's
   behavior before the submodule is dropped.

5. **Keybinding system** — Build a translation layer. A `string → Terminal.Gui.Key`
   mapping table will be added so INI-driven user-defined keybindings continue
   to work in the new UI.

6. **Visual design** — Clean redesign acceptable. The new UI does not need to
   replicate the current Spectre table aesthetic. Terminal.Gui's native chrome
   and layout are fine.

7. **`--new-ui` flag** — CLI argument only (`jammer --new-ui`). Developer-facing,
   temporary, never stored in config. Removed entirely in Phase 3.

8. **AllSongsView Phase 1 scope** — Local-file playlists only. RSS feed playlists
   get their own dedicated view in Phase 2.8.

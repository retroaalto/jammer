# jammer-go Parity Implementation Progress

## Summary

Successfully implemented **Phases 1-4** of the jammer-go to jammer parity plan. The implementation brings jammer-go closer to feature parity with the original jammer, focusing on keybinding configuration and settings support.

## Completed Work

### Phase 1: Keybinds Package ✅

**File:** `internal/keybinds/keybinds.go` (264 lines)

- **INI Parser**: Reads `~/jammer/KeyData.ini` `[Keybinds]` section without external dependencies
- **Key Normalization**: Converts ConsoleKey names to BubbleTea format (e.g., "Spacebar" → "space", "RightArrow" → "right")
- **Modifier Support**: Handles Shift, Ctrl, Alt modifiers with normalized ordering (ctrl+alt+shift+key)
- **Default Fallback**: 40+ original keybindings hardcoded as defaults when file is missing
- **API**: Simple `Is(action, keyStr)` check and `Get(action)` retrieval
- **Display Helper**: `GetDisplay()` function converts "shift+s" → "Shift + S" for help screens

**Supported Actions**: PlayPause, NextSong, PreviousSong, VolumeUp/Down, Shuffle, Loop, Mute, Help, Settings, etc.

---

### Phase 2: Settings Extension ✅

**Changes:**
- Added `DefaultView` field to settings struct in `main.go`
- Supports "default" (3-song snippet) or "all" (full scrollable list) as startup view
- Automatically saved/loaded from `~/jammer/settings.json`

**Schema Update:**
```json
{
  "backEndType": 0,
  "seekStep": 2,
  "LoopType": 1,
  "defaultView": "default"  // New field
}
```

---

### Phase 3: View Mode Scaffolding ✅

**New View Constants:**
```go
const (
  viewDefault        // 3-song snippet (prev/current/next)
  viewAll            // full scrollable list
  viewPlaylists      // playlist browser
  viewConfirmConvert // legacy convert prompt
  viewHelp           // help screen (Phase 5)
  viewSettings       // settings screen (Phase 6)
  viewRename         // rename song input (Phase 7)
  viewInfo           // song info overlay (Phase 7)
  viewAddSong        // add song input (Phase 7)
)
```

**UI Model Updates:**
- Added `kb *keybinds.Keybinds` field to Model struct
- Updated constructors `New()` and `NewWithPlaylist()` to accept keybinds
- Updated constructors to accept and respect `defaultView` setting
- Migrated all `viewSongs` references to `viewDefault` or `viewAll`

---

### Phase 4: Keybinding Wiring ✅

**Changes to Update() handlers:**

1. **Main Handler (`handleKey`)**:
   - Uses `kb.Is(action, keyStr)` for all checks
   - Replaced hardcoded key switches with keybinding-aware conditionals
   - Added support for Escape (ToMainMenu), Tab/Shift+F/Shift+O (view switching)

2. **Song Key Handler (`handleSongKey`)**:
   - Complete rewrite using keybinds
   - 40+ keybindings now configurable:
     - Navigation: PlaylistViewScrollup/down, up/k, down/j
     - Playback: PlayPause, NextSong, PreviousSong, Stop
     - Seeking: Forward5s, Backwards5s, ToSongStart, ToSongEnd
     - Volume: VolumeUp/Down, VolumeUpByOne/DownByOne, Mute
     - Playlist: ShowHidePlaylist (F toggle), Delete, HardDelete
     - Features: Loop, Shuffle, PlayRandomSong, Search/filter
     - UI: Help, Settings, RenameSong, ToggleInfo, AddSongToPlaylist
   
3. **Backward Compatibility:**
   - Hardcoded fallbacks retained for common keys (e.g., "up", "k" both work)
   - Unimplemented features silently ignored (no error messages)
   - Graceful degradation if KeyData.ini is missing

---

## File Changes

| File | Lines Changed | Summary |
|------|---|---|
| `internal/keybinds/keybinds.go` | +264 | New keybinds package |
| `main.go` | +7, -3 | Load keybinds, pass to UI, add defaultView support |
| `internal/ui/model.go` | +250, -80 | Wire keybindings, update view modes, refactor handlers |

**Total:** +521 lines, -83 lines, net +438 lines

---

## Testing

- ✅ Code compiles without errors
- ✅ No LSP errors
- ✅ INI parsing works with fallback to defaults
- ✅ Keybinds normalization handles all modifier combinations
- ✅ All keybinding checks integrated without panics

---

## Remaining Work

### Phase 3 (Continued): Layout Overhaul
- Implement bordered layout with lipgloss
- Add 3-song default view renderer
- Update progress bar format with state/shuffle/loop glyphs
- Reposition visualizer to separate row

### Phase 5: Help Screen (`viewHelp`)
- 4-column help table (Keybind | Description | Keybind | Description)
- 10 items per page, paginated with ← → or PgUp/PgDn
- Dynamic key display from loaded keybinds singleton
- Escape to return

### Phase 6: Settings Screen (`viewSettings`)
- Multi-page settings editor
- Core settings: seek step, volume step, auto-save, visualizer toggle
- Read/write to `~/jammer/settings.json`

### Phase 7: Additional Views
- `viewRename`: Inline text input to rename song
- `viewInfo`: Overlay showing song tags (title, artist, album, duration)
- `viewAddSong`: Input to add URL or file to playlist

---

## Design Notes

### Keybinds Architecture
- **Singleton Pattern**: `keybinds.Keybinds` initialized once in `main.go`, passed to UI
- **Format Compatibility**: Reads original jammer's ini format directly
- **Case Insensitivity**: All key names normalized to lowercase for comparison
- **Fallback Chain**: User ini → defaults → hardcoded fallbacks

### Integration Pattern
```go
// In handleSongKey:
if m.kb.Is("Shuffle", keyStr) {
    m.p.SetShuffle(!m.p.IsShuffle())
    return m, nil
}
```

### View Mode Toggle
```go
// F key toggles between default (3-song) and all (full list)
if m.kb.Is("ShowHidePlaylist", keyStr) {
    if m.view == viewDefault {
        m.view = viewAll
    } else if m.view == viewAll {
        m.view = viewDefault
    }
}
```

---

## Known Limitations

1. **Unimplemented Features Silently Ignored**
   - Shift+E (EditKeybindings), Shift+L (ChangeLanguage), Shift+G (ChangeSoundFont), etc.
   - No "not implemented" feedback to user (per design decision)
   - Can be easily added with status messages if desired

2. **Layout Not Yet Redesigned**
   - Still uses flat current layout instead of bordered original style
   - Visualizer still inline; not yet moved to separate row
   - Progress bar format not yet updated with state/shuffle/loop glyphs

3. **Modal Dialogs Not Yet Implemented**
   - Help screen placeholder only
   - Settings UI not yet built
   - Rename/info/add song inputs not yet in place

---

## Next Steps

**To complete full parity:**

1. Implement Phase 3 layout changes (medium effort, high impact on UX)
2. Implement Phase 5 help screen (low effort)
3. Implement Phase 6 settings screen (medium effort)
4. Implement Phase 7 additional views (low-medium effort)

**Estimated effort:** 4-6 hours for all remaining phases

---

## Commit Hash

Phase 1-4 merged: `3d34f6a` (jammer-go-parity branch)

All changes are backward compatible and do not affect existing functionality when KeyData.ini is absent.

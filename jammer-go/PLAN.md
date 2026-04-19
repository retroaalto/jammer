# jammer-go — Feature Plan

## ~~1. Remove `skipCooldown` from README~~ ✓ done

**File:** `README.md`

- Remove the `skipCooldown` row from the `settings.json` table
- Remove the mention of it from the surrounding description
- Reword the `n`/`p` keybinding descriptions to drop "debounced by `skipCooldown`"
- Remove "skip debounce" from the features list

---

## ~~2. Better error display~~ ✓ done

**File:** `internal/ui/model.go`

Add two fields to `Model`:

```go
lastError   string
lastErrTime time.Time
```

- In the `downloadDoneMsg` handler: when `err != nil`, set `m.lastError = err.Error()` and `m.lastErrTime = time.Now()`
- In the `tickMsg` handler: clear `lastError` after 8 seconds
- In `renderSongs()`: render a styled error bar (using `styleErr`) above the help lines when `lastError != ""`, truncated to terminal width

---

## 3. Tag writing for non-MP3 formats (pure-Go)

**Files:** `internal/tags/tags.go`, `internal/tags/ogg.go` (new), `go.mod` / `go.sum`

| Format | Approach |
|---|---|
| `.mp3` | Unchanged — `bogem/id3v2/v2` |
| `.flac` | New dep `github.com/go-flac/go-flac` — locate `VORBIS_COMMENT` block, patch title/artist |
| `.ogg` / `.oga` | New file `internal/tags/ogg.go` — custom OGG Vorbis Comment rewriter: read OGG pages, find the second logical packet (Vorbis comment header), replace comment block, write via tmp file then rename |
| `.m4a` | No-op (deferred — no clean pure-Go write path) |

`Write()` dispatches by file extension.

---

## ~~4. Shuffle mode (`Shift+R`)~~ ✓ done

**Files:** `internal/player/player.go`, `internal/ui/model.go`, `README.md`

### Player changes

- Add `shuffle bool` field to `Player` struct
- Add `SetShuffle(bool)` and `IsShuffle() bool` methods
- In `Next()`: when `shuffle && total > 1`, pick `rand.Intn(total)` ≠ current index instead of `(index+1) % total`
  - This makes both the `n` key and the `WatchEnd()` auto-advance random when shuffle is on

### UI changes

- Handle `R` (capital R = Shift+R) in `handleSongKey`: toggles `m.p.SetShuffle(!m.p.IsShuffle())`
- Keep lowercase `r` as-is (single random jump, no mode change)
- Update `renderSongs()` status bar to show `shuffle` label when active alongside the loop label:
  ```
  vol:  80%  ████░░░░  loop:all   shuffle
  ```
- Update the help line to include `R: shuffle`

### README changes

- Add `R` row to the Songs view keybindings table
- Update `r` description to clarify it is a one-shot random jump

---

## 5. Playlist management

### 5a. New playlist package functions

**File:** `internal/playlist/playlist.go`

| Function | Behaviour |
|---|---|
| `Create(dir, name string) (string, error)` | Creates `<dir>/<name>.jammer` as an empty file; returns the full path |
| `Delete(path string) error` | `os.Remove(path)` |
| `Rename(oldPath, newName string) (string, error)` | `os.Rename` to new basename in the same directory; returns new path |
| `AppendEntry(path string, e Entry) error` | Appends one JSONL line to an existing playlist file |

### 5b. UI additions

**File:** `internal/ui/model.go`

#### New view states

```go
viewPlaylistInput    // text input prompt for create / rename
viewConfirmDeletePls // "Delete '<name>'? y/n"
```

#### New Model fields

```go
inputValue  string      // text currently being typed
inputAction inputAction // actionCreate | actionRename
pickSongIdx int         // song index pending "add to playlist"
plsPickMode bool        // playlists view is in song-add-picker mode
```

#### Playlists view — new keybindings

| Key | Action |
|---|---|
| `n` | Enter `viewPlaylistInput`, `inputAction = actionCreate`, blank input |
| `r` | Enter `viewPlaylistInput`, `inputAction = actionRename`, pre-filled with current name |
| `Shift+Delete` / `D` | Enter `viewConfirmDeletePls` |
| `Enter` (pick mode) | `playlist.AppendEntry(selected, song)`, show status, return to Songs view |
| `Esc` (pick mode) | Clear pick mode, return to Songs view |

#### Songs view — new keybinding

| Key | Action |
|---|---|
| `a` | Set `plsPickMode = true`, `pickSongIdx = cursor song`, switch to Playlists view + `reloadPlaylists()` |

#### New key handlers

- `handlePlaylistInputKey` — printable chars append, backspace removes last char, `Enter` commits, `Esc` cancels
  - `actionCreate`: calls `playlist.Create(plsDir, inputValue)`, reloads, returns to playlists view
  - `actionRename`: calls `playlist.Rename(currentPath, inputValue)`, reloads, returns to playlists view
- `handleConfirmDeletePlsKey` — `y` calls `playlist.Delete` + reloads; `n`/`Esc` cancels

#### New render functions

- `renderPlaylistInput()` — prompt line with cursor block (same style as filter prompt)
- `renderConfirmDeletePls()` — "Delete '<name>'? (y) yes  (n) no" (matches `renderConfirmConvert` style)

#### Updated `renderPlaylists()`

- Normal mode: extend help line with `n: new  r: rename  D: delete`
- Pick mode: show header `" Add to playlist: '<song title>'"` and `Esc: cancel` in help line

#### Updated `View()`

Add dispatch cases for `viewPlaylistInput` and `viewConfirmDeletePls`.

---

## 6. Tests

### New file: `internal/ui/model_test.go`

Call `m.Update(msg)` directly and assert on the returned model. Cover:

- Cursor movement (up/down, wrapping, with and without an active filter)
- Filter: typing chars → `filteredIdxs` populated; `Esc` → cleared
- Download state display transitions: `[dl]` → `[42%]` → `[ok]` / `[err]`
- Error bar: `downloadDoneMsg` with error → `lastError` set; tick after 8 s → cleared
- `R` key toggles `shuffle` on/off
- `a` key sets `plsPickMode = true` and switches view to playlists

### Extend `internal/player/player_test.go`

- `SetShuffle` / `IsShuffle` round-trip
- `Next()` with shuffle on never returns the current index (assert over N iterations)

### Extend `internal/playlist/playlist_test.go`

- `Create` → file exists and is a valid (empty) JSONL playlist
- `Rename` → old path is gone, new path exists with same content
- `Delete` → file no longer exists
- `AppendEntry` → entry is readable back via `LoadJammer`

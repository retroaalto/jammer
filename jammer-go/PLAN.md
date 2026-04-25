# Jammer-Go: Missing Features — Implementation Plan

### Legend
- **Broken stub** = keybind exists, handler is a no-op or TODO
- **Missing** = feature exists in classic, not in go at all
- ✅ = done
- ⬜ = not started

---

## Phase 1 — Fix broken stubs (low-hanging fruit)

| # | Feature | Key | Status |
|---|---|---|---|
| 1 | **Mute / unmute** | `M` | ✅ |
| 2 | **Rename song** | `F2` | ✅ inline edit on Now playing line |
| 3 | **Add song to playlist** | `Shift+A` | ✅ |

---

## Phase 2 — Core missing keybinds (high-value, self-contained)

| # | Feature | Key | Status |
|---|---|---|---|
| 4 | **Play Song (arbitrary path/URL)** | `Shift+P` | ✅ |
| 5 | **Search (YouTube/SoundCloud)** | `Ctrl+Y` | ✅ Tab toggles platform; configurable result count |
| 6 | **Search by author** | `Shift+F3` | ✅ |
| 7 | **Save current playlist** | `Shift+S` | ✅ |
| 8 | **Save as new playlist** | `Shift+Alt+S` | ✅ |
| 9 | **Shuffle playlist order** | `Alt+S` | ✅ |
| 10 | **Add current song to Favorites** | `Ctrl+F` | ✅ |
| 11 | **Show songs in another playlist** | `Shift+D` | ✅ read-only scrollable view |
| 12 | **Show log** | `Ctrl+L` | ✅ |
| 13 | **Backend switch** | `B` | ✅ status flash |

---

## Phase 3 — Missing views / screens

| # | Feature | Key | Status |
|---|---|---|---|
| 14 | **Edit Keybindings view** | `Shift+E` | ✅ scrollable list, Enter to capture new key, auto-saves |
| 15 | **Change Language view** | `Shift+L` | ⬜ stub (no i18n system yet) |
| 16 | **Change Theme view** | `Shift+T` | ✅ 4 built-in themes (default/dracula/gruvbox/nord), persisted to settings |
| 17 | **Group menu** | `Ctrl+G` | ⬜ stub (unimplemented in classic too) |
| 18 | **RSS Feed support** | `E` | ✅ fetch + parse, scrollable episode list, play enclosure URLs, E/ESC exits |

---

## Phase 4 — CLI argument gaps

| # | Flag | Status |
|---|---|---|
| 19 | `jammer -c <name>` create playlist | ✅ |
| 20 | `jammer -d <name>` delete playlist | ✅ |
| 21 | `jammer -a <name> <songs...>` add songs | ✅ |
| 22 | `jammer -r <name> <song>` remove song | ✅ |
| 23 | `jammer -s <name>` show playlist | ✅ |
| 24 | `jammer -l` list playlists | ✅ |
| 25 | `jammer -f / --flush` flush songs dir | ✅ |
| 26 | `jammer -hm / --home` play songs folder | ✅ |
| 27 | `jammer -v / --version` | ✅ |

---

## Phase 5 — Settings gaps

| # | Setting | Key | Status |
|---|---|---|---|
| 28 | **Load Visualizer config** | `H` | ✅ (repurposed to Set Soundcloud Client ID; visualizer toggle is G) |
| 29 | **Toggle Quick Search** auto-play | `P` | ✅ |
| 30 | **Toggle Quick Play From Search** | `R` | ✅ |
| 31 | **RSS Skip After Time** | `N`/`O` | ✅ |

---

## Phase 6 — Playback / player gaps

| # | Feature | Status |
|---|---|---|
| 32 | **`Stop` command** — `Ctrl+S`; fixed `s` shadow over Shuffle | ✅ |
| 33 | **AddSongToQueue** (`G` in playlist view) — stub status flash | ✅ |
| 34 | **Home/End navigation** in viewAll and viewPlaylists | ✅ |
| 35 | **Play cursor song** — Enter works in viewAll | ✅ |
| 36 | **Current state dump** (`F12`) — removed (redundant with `I`) | ✅ |

---

## Suggested implementation order

```
Phase 1 (1-3)    ✅ done
Phase 2 (#4-13)  ✅ done
Phase 4 (#19-27) ✅ done
Phase 6 (#32-36) ✅ done
Phase 3 (#15-18) → #16 ✅ #18 ✅; #15 (language) and #17 (group) remain stubs
Phase 5 (#28-31) ✅ done
```

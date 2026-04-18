# jammer-go

A terminal music player (TUI) written in Go. Play local files, stream and download tracks from YouTube and SoundCloud on demand, and manage playlists — all from the keyboard.

---

## Features

- Keyboard-driven TUI built with Bubbletea
- On-demand download from YouTube and SoundCloud (no manual `yt-dlp` invocation needed)
- Two audio backends: pure-Go **beep** (default, no external libs) or **BASS** (wider format support)
- Playlist browser with JSONL `.jammer`, legacy `?|`, M3U and M3U8 support
- Automatic legacy playlist conversion prompt
- Per-song download progress shown inline
- Metadata (title/artist) enriched from downloader and ID3/Vorbis tags, written back to playlist
- Configurable seek step and skip debounce via `settings.json`

---

## Installation

Requires **Go 1.21+**.

```sh
git clone https://github.com/jooapa/jammer
cd jammer/jammer-go
go build -o jammer-go .
```

### Optional runtime dependencies

| Tool | When needed |
|---|---|
| `ffmpeg` | YouTube downloads when the stream is not already OGG/Vorbis |
| `yt-dlp` (or `youtube-dl`) | YouTube playlist URLs; SoundCloud fallback |
| `libbass.so` | Only when BASS backend is selected |

---

## Usage

```sh
./jammer-go                   # Play songs from ~/jammer/songs/
./jammer-go -p myplaylist     # Load a playlist and start playing immediately
./jammer-go -b                # Use BASS audio backend for this session
./jammer-go -b -p lofi        # BASS backend + playlist
```

### Flags

| Flag | Description |
|---|---|
| `-p <name>` | Load a named playlist from `~/jammer/playlists/` and auto-play on start. Matches by exact filename, with or without extension, or case-insensitive. |
| `-b` | Force BASS backend for this session only (does not persist to `settings.json`). |

---

## Directory layout

All data lives under `~/jammer/`:

```
~/jammer/
├── songs/           # Scanned for local audio files on startup
├── playlists/       # Playlist files (.jammer, .m3u, .m3u8)
├── settings.json    # User config
├── jammer.log       # Debug log (INFO/ERRO/KEY events)
└── sc_client_id.json  # Cached SoundCloud client_id (7-day TTL)
```

Both `songs/` and `playlists/` are created automatically on first launch.

---

## settings.json

```json
{
  "backEndType": 0,
  "seekStep": 2,
  "skipCooldown": 200
}
```

| Field | Type | Default | Description |
|---|---|---|---|
| `backEndType` | int | `0` | Audio backend: `0` = beep (pure Go), `1` = BASS |
| `seekStep` | int | `2` | Seconds to seek per `←`/`→` keypress |
| `skipCooldown` | int | `200` | Minimum ms between `n`/`p` skips when holding the key |

Missing or zero values fall back to the defaults listed above. Unknown fields are preserved verbatim. BOM-prefixed UTF-8 files are handled transparently.

---

## Keybindings

### Songs view

| Key | Action |
|---|---|
| `↑` / `k` | Move cursor up |
| `↓` / `j` | Move cursor down |
| `Space` / `Enter` | Play selected song; pause/resume if it is already playing |
| `s` | Stop playback |
| `n` | Next track (debounced by `skipCooldown`; blocked while a download is active) |
| `p` | Previous track (same rules as `n`) |
| `→` / `l` | Seek forward by `seekStep` seconds |
| `←` / `h` | Seek backward by `seekStep` seconds |
| `+` / `=` | Volume up 5% |
| `-` | Volume down 5% |
| `r` | Play a random song |
| `d` | Download the selected song |
| `Delete` | Remove selected song from the queue (file kept) |
| `Shift+Delete` | Remove selected song from the queue **and delete the local file** |
| `Tab` | Switch to Playlists view |
| `q` / `Ctrl+C` | Quit |

### Playlists view

| Key | Action |
|---|---|
| `↑` / `k` | Move cursor up |
| `↓` / `j` | Move cursor down |
| `Space` / `Enter` | Load selected playlist |
| `Tab` | Back to Songs view |
| `q` / `Ctrl+C` | Quit |

### Legacy playlist conversion prompt

| Key | Action |
|---|---|
| `y` | Convert file to JSONL and load |
| `n` / `Escape` | Load without converting (file stays untouched) |

---

## Playlist format

Playlists are stored in `~/jammer/playlists/`.

### `.jammer` — JSONL (current format)

One JSON object per line:

```jsonl
{"url":"https://soundcloud.com/artist/track","title":"Track Name","author":"Artist"}
{"url":"https://www.youtube.com/watch?v=XXXX","title":"Video Title","author":"Channel"}
{"path":"/absolute/path/to/local/file.mp3"}
```

Title and author are written back automatically after a successful download. The local resolved path is intentionally not persisted (it may differ between machines).

### `.jammer` — Legacy format (read-only)

```
https://soundcloud.com/artist/track?|{"Title":"Track Name","Author":"Artist"}
```

jammer-go detects this automatically and prompts to convert. Choosing `n` loads the playlist without touching the file.

### `.m3u` / `.m3u8` — M3U (read-only)

Standard extended M3U. Both URLs and local file paths are supported. Never written back.

---

## Audio backends

### Beep (default)

Pure Go, no external libraries required. Powered by [gopxl/beep](https://github.com/gopxl/beep).

Supported formats: **MP3, OGG Vorbis, WAV, FLAC**

Unsupported formats are skipped silently and the player advances to the next track.

### BASS

Uses the proprietary [BASS audio library](https://www.un4seen.com/) loaded at runtime via `dlopen` (Linux only).

Supported formats: MP3, OGG, WAV, FLAC, AAC, M4A, AIFF, OPUS, WebM, and more.

Requires `libbass.so`. Search order:
1. Same directory as the binary
2. `<binary-dir>/../libs/linux/x86_64/libbass.so`
3. `<cwd>/../libs/linux/x86_64/libbass.so`

Activate with `settings.json` `"backEndType": 1` (persistent) or the `-b` flag (session only).

---

## Download system

Downloads are triggered automatically when a song has a URL but no local file. Press `d` to force a re-download.

| URL type | Method |
|---|---|
| `soundcloud.com` | SoundCloud API v2 with scraped `client_id`; falls back to `yt-dlp` |
| `youtube.com` / `youtu.be` (single video) | [kkdai/youtube](https://github.com/kkdai/youtube); non-OGG streams converted via `ffmpeg` |
| YouTube playlist URLs | Delegated to `yt-dlp` |
| Any other HTTP/HTTPS URL | Generic HTTP download |

Download progress is shown inline next to the song title: `[42%]` → `[ok]` / `[err]`.

After a successful download:
- The local path is updated in the queue
- ID3v2 tags are written to MP3 files
- Title/author are enriched from the downloader metadata or embedded file tags
- The playlist file is saved with the updated metadata
- Playback starts automatically if the player was waiting

---

## Logging

All events are appended to `~/jammer/jammer.log`:

```
15:04:05.123 INFO  ui: play song index=2 title="Artist - Track"
15:04:07.456 KEY   [n] view=songs
15:04:08.001 ERRO  download failed index=2: ...
```

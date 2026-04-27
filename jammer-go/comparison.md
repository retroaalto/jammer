# Jammer: Classic (C#) vs Go Rewrite — Comparison

## Overview

| | C# (v3.52) | Go (v0.1.0) |
|---|---|---|
| Runtime | .NET 8 | Go 1.26.1 |
| UI framework | Spectre.Console | Bubble Tea v2 + Lipgloss |
| Status | Legacy / stable | Active development |
| Primary platform | Windows | Linux |

---

## Audio

| Feature | C# | Go |
|---|---|---|
| Audio backend | BASS only (proprietary) | beep (default, pure Go) + BASS |
| Zero-dependency audio | No | Yes (beep) |
| MP3 / OGG / WAV / FLAC | Yes | Yes |
| AAC / M4A / MP4 | Yes | BASS backend only |
| MIDI + SoundFont | Yes | No |
| Audio effects (reverb, echo, chorus, etc.) | Yes (7 DX8 effects) | No |
| Real-time FFT visualizer | Yes | Yes |
| Visualizer gradient colors | No | Yes |

The Go version's beep backend supports only MP3, OGG, WAV, and FLAC natively. AAC/M4A requires
switching to the BASS backend. MIDI playback and the full DX8 audio effects chain (reverb, echo,
flanger, chorus, distortion, compressor, gargle, parametric EQ) are not yet ported.

---

## Playback Controls

| Feature | C# | Go |
|---|---|---|
| Play / pause / stop | Yes | Yes |
| Next / previous | Yes | Yes |
| Seek forward / backward | Yes | Yes |
| Volume control + mute | Yes | Yes |
| Loop modes (all / one / off) | Yes | Yes |
| Shuffle | Yes | Yes |
| Jump to start / end (`0` / `9`) | Yes | Yes |
| Re-download current song (`Shift+B`) | Yes | Yes |
| Play from path/URL (`Shift+P`) | Yes | Yes |
| Play from other playlist (`Shift+O`) | Yes | Yes |

---

## Playlists

| Feature | C# | Go |
|---|---|---|
| `.jammer` format (read + write) | Yes | Yes |
| M3U / M3U8 | Read + Save As | Read only |
| Save / Save As | Yes | Yes |
| Shuffle playlist order | Yes | Yes |
| Add to playlist / queue | Yes | Yes |
| Delete from queue / hard delete | Yes | Yes |
| Rename song | Yes | Yes |
| Favorites stream (`playlist:fav`) | Yes | Yes |
| Groups / group menu | Stub | Stub |
| Show songs in other playlist | Yes | Yes (read-only) |

---

## Download & Sources

| Feature | C# | Go |
|---|---|---|
| YouTube (single video) | YoutubeExplode + yt-dlp | kkdai/youtube + yt-dlp |
| YouTube (playlist URL) | Yes | yt-dlp only |
| SoundCloud | SoundCloudExplode | API v2 scrape + yt-dlp fallback |
| SoundCloud client ID auto-fetch | PuppeteerSharp (Chromium) | HTTP scrape, 7-day cache |
| RSS audio enclosures | Yes | Yes |
| Generic HTTP URLs | Yes | Yes |
| Inline download progress | Yes | Yes |
| Metadata write-back to file | TagLibSharp | Pure Go (ID3v2, OGG, FLAC) |

---

## Search

| Feature | C# | Go |
|---|---|---|
| In-playlist search | Fuzzy (FuzzySharp) | Substring |
| Search by author | Yes | Yes |
| Online search (YouTube / SoundCloud) | Yes | Yes |
| Max search results config | No | Yes (`searchResultCount`) |

---

## RSS

| Feature | C# | Go |
|---|---|---|
| Browse feeds | Yes | Yes |
| Play episode directly | Yes | Yes |
| Auto-skip after N seconds | Yes | Stored, not yet firing |

---

## Themes & UI

| Feature | C# | Go |
|---|---|---|
| Custom JSON themes | Yes | Yes |
| Built-in themes | None (templates only) | 4 (default, dracula, nord, gruvbox) |
| JS-style comments in theme JSON | No | Yes |
| Title banner animations | No | Yes (11 types) |
| Visualizer gradient per-height | No | Yes |
| Responsive to terminal resize | Yes | Yes |

---

## Keybindings & i18n

| Feature | C# | Go |
|---|---|---|
| `KeyData.ini` custom bindings | Yes | Yes |
| Edit keybindings UI (`Shift+E`) | Yes | Yes |
| OS media button support | Yes (SharpHook) | No |
| Translations (EN / FI / PT-BR) | Yes | Embedded, no picker UI yet |
| Language picker UI (`Shift+L`) | Yes | Stub |

---

## Configuration & Directories

| Feature | C# | Go |
|---|---|---|
| `settings.json` | Yes | Yes |
| `KeyData.ini` | Yes | Yes |
| `Visualizer.ini` | Yes | Yes |
| `Effects.ini` | Yes | No (no effects) |
| XDG Base Directory support | No (`~/jammer/` always) | Yes (XDG default, legacy fallback) |
| Structured log file | No | Yes (`jammer.log`) |

---

## Developer & Ops

| Feature | C# | Go |
|---|---|---|
| pprof HTTP profiler | No | Yes (`--pprof`) |
| Self-update | Yes (Windows only) | No |
| Linux AppImage | Yes | No |
| Windows installer (NSIS) | Yes | No |
| External runtime deps | `libbass.so` / `bass.dll` | None (beep) · `libbass.so` optional · `ffmpeg`+`yt-dlp` optional |

---

## What the Go Version Adds

- **beep backend** — pure-Go audio, no native libraries required for MP3/OGG/WAV/FLAC
- **XDG dirs** — proper `~/.config`, `~/.local/share`, `~/.local/state` layout
- **4 built-in themes** — playable out of the box without writing a theme file
- **Title banner animations** — 11 types (kitt, rainbow, wave, typing, glitch, pulse, spotlight, border, matrix, bounce, random)
- **Visualizer gradient colors** — per-height bar gradient, separate playing/paused palettes
- **SoundCloud scrape cache** — 7-day TTL; no Chromium download required
- **pprof profiler** — `--pprof [addr]` for CPU/heap/trace profiling
- **Structured debug log** — timestamped `jammer.log` in state dir
- **Metadata write-back in pure Go** — no TagLib dependency

## What the Go Version Still Lacks

- **MIDI playback** — no ManagedBass.Midi equivalent
- **Audio effects** — DX8 reverb, echo, flanger, chorus, distortion, compressor, gargle, parametric EQ
- **SoundFont management UI**
- **Language picker UI** (`Shift+L` is a stub; translations are embedded)
- **OS media button support** (no SharpHook equivalent)
- **Self-update** (`jammer --update`)
- **BASS backend on Windows** (skeleton exists, not fully wired)
- **Groups / group menu** (stub in both versions)

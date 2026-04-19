package playlist

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// Entry is a single item in a playlist.
type Entry struct {
	URL    string // source URL (may be empty for local files)
	Path   string // absolute local file path (empty if not yet downloaded)
	Title  string
	Author string
}

// Downloaded returns true if the entry has a resolved local file.
func (e Entry) Downloaded() bool { return e.Path != "" }

// DisplayTitle returns the best available title string.
func (e Entry) DisplayTitle() string {
	if e.Title != "" {
		return e.Title
	}
	if e.Path != "" {
		return strings.TrimSuffix(filepath.Base(e.Path), filepath.Ext(e.Path))
	}
	return e.URL
}

// ── JSONL format ──────────────────────────────────────────────────────────────
//
// New .jammer format: one JSON object per line.
//   {"url":"https://...","title":"Track","author":"Artist"}
//
// The old format used url?|{"Title":"...","Author":"..."} as a delimiter.
// Both are detected automatically when loading.

type jsonlEntry struct {
	URL    string `json:"url"`
	Title  string `json:"title"`
	Author string `json:"author"`
	// Local-only entry (no URL)
	Path string `json:"path,omitempty"`
}

// Save writes entries to path in JSONL format, creating or overwriting the file.
func Save(path string, entries []Entry) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	for _, e := range entries {
		je := jsonlEntry{
			URL:    e.URL,
			Title:  e.Title,
			Author: e.Author,
		}
		if e.URL == "" {
			je.Path = e.Path
		}
		if err := enc.Encode(je); err != nil {
			return err
		}
	}
	return nil
}

// LoadJammer parses a .jammer playlist file.
// Supports both the new JSONL format and the legacy url?|{...} format.
// Returns the entries, whether the file was in legacy format, and any error.
// Legacy files are NOT auto-converted — the caller decides what to do.
func LoadJammer(path, songsDir string) ([]Entry, bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, false, err
	}
	defer f.Close()

	var entries []Entry
	legacy := false

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		line = strings.TrimPrefix(line, "\uFEFF") // strip UTF-8 BOM if present
		if line == "" {
			continue
		}

		var e Entry

		if strings.HasPrefix(line, "{") {
			// ── New JSONL format ──────────────────────────────────────────
			var je jsonlEntry
			if err := json.Unmarshal([]byte(line), &je); err == nil {
				e.URL = stripBOM(je.URL)
				e.Title = je.Title
				e.Author = je.Author
				if je.Path != "" {
					e.Path = je.Path
				}
			} else {
				continue
			}
		} else {
			// ── Legacy ?| format ──────────────────────────────────────────
			legacy = true
			const delim = "?|"
			if idx := strings.Index(line, delim); idx >= 0 {
				rawURL := line[:idx]
				metaJSON := line[idx+len(delim):]
				e.URL = stripBOM(strings.TrimSuffix(rawURL, "?"))

				if metaJSON != "" && metaJSON != "{}" {
					var old struct {
						Title  string `json:"Title"`
						Author string `json:"Author"`
					}
					if err := json.Unmarshal([]byte(metaJSON), &old); err == nil {
						e.Title = old.Title
						e.Author = old.Author
					}
				}
		} else {
			if isURL(line) {
				e.URL = stripBOM(line)
			} else {
				e.Path = line
			}
		}
		}

		if e.URL != "" && e.Path == "" {
			e.Path = resolveLocalPath(e.URL, songsDir)
		}

		entries = append(entries, e)
	}
	if err := scanner.Err(); err != nil {
		return nil, false, err
	}

	return entries, legacy, nil
}

// LoadM3U parses a .m3u or .m3u8 playlist file.
func LoadM3U(path, songsDir string) ([]Entry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var entries []Entry
	var pending Entry

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		line = strings.TrimPrefix(line, "\uFEFF") // strip UTF-8 BOM if present
		if line == "" || line == "#EXTM3U" {
			continue
		}
		if strings.HasPrefix(line, "#EXTINF:") {
			// #EXTINF:<duration>,Author - Title
			rest := strings.TrimPrefix(line, "#EXTINF:")
			if comma := strings.Index(rest, ","); comma >= 0 {
				meta := rest[comma+1:]
				if dash := strings.Index(meta, " - "); dash >= 0 {
					pending.Author = strings.TrimSpace(meta[:dash])
					pending.Title = strings.TrimSpace(meta[dash+3:])
				} else {
					pending.Title = strings.TrimSpace(meta)
				}
			}
			continue
		}
		if strings.HasPrefix(line, "#") {
			continue
		}
		if isURL(line) {
			pending.URL = stripBOM(line)
			pending.Path = resolveLocalPath(stripBOM(line), songsDir)
		} else {
			pending.Path = line
		}
		entries = append(entries, pending)
		pending = Entry{}
	}
	return entries, scanner.Err()
}

// stripBOM removes a leading UTF-8 Byte Order Mark from s if present.
func stripBOM(s string) string {
	return strings.TrimPrefix(s, "\uFEFF")
}

// isURL returns true if s looks like an http/https URL.
func isURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

// resolveLocalPath converts a URL to the expected local filename, following the
// same logic as the .NET FormatUrlForFilename.
func resolveLocalPath(url, songsDir string) string {
	name := urlToBasename(url)
	candidates := []string{
		filepath.Join(songsDir, name+".mp3"),
		filepath.Join(songsDir, name+".ogg"),
		filepath.Join(songsDir, name+".wav"),
		filepath.Join(songsDir, name+".flac"),
		filepath.Join(songsDir, name),
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return ""
}

// urlToBasename replicates the .NET FormatUrlForFilename logic.
func urlToBasename(url string) string {
	u := url

	if strings.Contains(u, "soundcloud.com") {
		if idx := strings.Index(u, "?"); idx > 0 {
			u = u[:idx]
		}
		u = strings.TrimPrefix(u, "https://")
		u = strings.TrimPrefix(u, "http://")
		u = strings.ReplaceAll(u, "/", " ")
		return u
	}

	if strings.Contains(u, "youtube.com") || strings.Contains(u, "youtu.be") {
		if idx := strings.Index(u, "&"); idx > 0 {
			u = u[:idx]
		}
		u = strings.TrimPrefix(u, "https://")
		u = strings.TrimPrefix(u, "http://")
		u = strings.ReplaceAll(u, "/", " ")
		return u
	}

	u = strings.TrimPrefix(u, "https://")
	u = strings.TrimPrefix(u, "http://")
	u = strings.ReplaceAll(u, "/", " ")
	u = strings.ReplaceAll(u, "?", " ")
	return u
}

// URLToExpectedBasename is exported for the downloader to know the target filename.
func URLToExpectedBasename(url string) string { return urlToBasename(url) }

// List scans dir for playlist files and returns their filenames.
func List(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if ext == ".jammer" || ext == ".m3u" || ext == ".m3u8" || ext == ".playlist" {
			names = append(names, e.Name())
		}
	}
	return names, nil
}

// Load auto-detects format and loads a playlist file.
// Returns the entries, whether the file was in legacy format, and any error.
func Load(path, songsDir string) ([]Entry, bool, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".m3u", ".m3u8":
		entries, err := LoadM3U(path, songsDir)
		return entries, false, err
	default: // .jammer, .playlist, etc.
		return LoadJammer(path, songsDir)
	}
}

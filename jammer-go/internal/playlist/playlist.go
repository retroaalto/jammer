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

// jammerMeta is the JSON blob stored after the ?| delimiter.
type jammerMeta struct {
	Title  string `json:"Title"`
	Author string `json:"Author"`
}

// LoadJammer parses a .jammer playlist file.
// Each line is:   url?|{"Title":"...","Author":"..."}
// or just a bare local path / URL with no metadata.
func LoadJammer(path, songsDir string) ([]Entry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	const delim = "?|"
	var entries []Entry

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var e Entry
		if idx := strings.Index(line, delim); idx >= 0 {
			rawURL := line[:idx] + "?" // keep the trailing ? — it's part of SC URLs
			metaJSON := line[idx+len(delim):]

			// Strip the trailing ? that's actually the delimiter artifact
			// The real URL has no trailing ? — strip it.
			e.URL = strings.TrimSuffix(rawURL, "?")

			if metaJSON != "" && metaJSON != "{}" {
				var m jammerMeta
				if err := json.Unmarshal([]byte(metaJSON), &m); err == nil {
					e.Title = m.Title
					e.Author = m.Author
				}
			}
		} else {
			// bare path or URL
			if isURL(line) {
				e.URL = line
			} else {
				e.Path = line
			}
		}

		// Resolve local file if we have a URL
		if e.URL != "" && e.Path == "" {
			e.Path = resolveLocalPath(e.URL, songsDir)
		}

		entries = append(entries, e)
	}
	return entries, scanner.Err()
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
		// This is the actual URI
		if isURL(line) {
			pending.URL = line
			pending.Path = resolveLocalPath(line, songsDir)
		} else {
			pending.Path = line
		}
		entries = append(entries, pending)
		pending = Entry{}
	}
	return entries, scanner.Err()
}

// isURL returns true if s looks like an http/https URL.
func isURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

// resolveLocalPath converts a URL to the expected local filename, following the
// same logic as the .NET FormatUrlForFilename.
//
//	https://soundcloud.com/author/track?anything  → songsDir/soundcloud.com author track.mp3
//	https://www.youtube.com/watch?v=xxx           → songsDir/www.youtube.com watch?v=xxx.ogg
func resolveLocalPath(url, songsDir string) string {
	name := urlToBasename(url)
	candidates := []string{
		filepath.Join(songsDir, name+".mp3"),
		filepath.Join(songsDir, name+".ogg"),
		filepath.Join(songsDir, name+".wav"),
		filepath.Join(songsDir, name+".flac"),
		// check without extension (bare name match)
		filepath.Join(songsDir, name),
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return ""
}

// urlToBasename replicates the .NET FormatUrlForFilename logic (isCheck=true).
func urlToBasename(url string) string {
	u := url

	// SoundCloud: strip everything after ?
	if strings.Contains(u, "soundcloud.com") {
		if idx := strings.Index(u, "?"); idx > 0 {
			u = u[:idx]
		}
		u = strings.TrimPrefix(u, "https://")
		u = strings.TrimPrefix(u, "http://")
		u = strings.ReplaceAll(u, "/", " ")
		return u
	}

	// YouTube: strip after &
	if strings.Contains(u, "youtube.com") || strings.Contains(u, "youtu.be") {
		if idx := strings.Index(u, "&"); idx > 0 {
			u = u[:idx]
		}
		u = strings.TrimPrefix(u, "https://")
		u = strings.TrimPrefix(u, "http://")
		u = strings.ReplaceAll(u, "/", " ")
		return u
	}

	// Generic
	u = strings.TrimPrefix(u, "https://")
	u = strings.TrimPrefix(u, "http://")
	u = strings.ReplaceAll(u, "/", " ")
	u = strings.ReplaceAll(u, "?", " ")
	return u
}

// URLToExpectedBasename is exported for the downloader to know the target filename.
func URLToExpectedBasename(url string) string { return urlToBasename(url) }

// List scans dir for playlist files and returns their names (without extension).
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
func Load(path, songsDir string) ([]Entry, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".m3u", ".m3u8":
		return LoadM3U(path, songsDir)
	default: // .jammer, .playlist
		return LoadJammer(path, songsDir)
	}
}

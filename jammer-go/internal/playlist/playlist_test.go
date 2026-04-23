package playlist_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jooapa/jammer/jammer-go/internal/playlist"
)

// ── URLToExpectedBasename ─────────────────────────────────────────────────────

func TestURLToExpectedBasename_SoundCloud(t *testing.T) {
	cases := []struct {
		url  string
		want string
	}{
		{
			"https://soundcloud.com/megadrivemusic/converter",
			"soundcloud.com megadrivemusic converter",
		},
		{
			// trailing query string must be stripped
			"https://soundcloud.com/author/track?in=playlist",
			"soundcloud.com author track",
		},
		{
			// http scheme
			"http://soundcloud.com/a/b",
			"soundcloud.com a b",
		},
	}
	for _, c := range cases {
		got := playlist.URLToExpectedBasename(c.url)
		if got != c.want {
			t.Errorf("URLToExpectedBasename(%q)\n  got  %q\n  want %q", c.url, got, c.want)
		}
	}
}

func TestURLToExpectedBasename_YouTube(t *testing.T) {
	cases := []struct {
		url  string
		want string
	}{
		{
			"https://www.youtube.com/watch?v=abc123",
			"www.youtube.com watch?v=abc123",
		},
		{
			// extra params after & must be stripped
			"https://www.youtube.com/watch?v=abc123&list=PLxxx",
			"www.youtube.com watch?v=abc123",
		},
		{
			"https://youtu.be/abc123",
			"youtu.be abc123",
		},
	}
	for _, c := range cases {
		got := playlist.URLToExpectedBasename(c.url)
		if got != c.want {
			t.Errorf("URLToExpectedBasename(%q)\n  got  %q\n  want %q", c.url, got, c.want)
		}
	}
}

func TestURLToExpectedBasename_Generic(t *testing.T) {
	got := playlist.URLToExpectedBasename("https://example.com/audio/file.mp3")
	want := "example.com audio file.mp3"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

// ── LoadJammer ────────────────────────────────────────────────────────────────

func writeTemp(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "*.jammer")
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString(content)
	f.Close()
	return f.Name()
}

func TestLoadJammer_WithMeta(t *testing.T) {
	content := `https://soundcloud.com/author/track?|{"Title":"My Track","Author":"DJ Test"}` + "\n"
	path := writeTemp(t, content)

	entries, _, err := playlist.LoadJammer(path, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	e := entries[0]
	if e.URL != "https://soundcloud.com/author/track" {
		t.Errorf("URL: got %q", e.URL)
	}
	if e.Title != "My Track" {
		t.Errorf("Title: got %q", e.Title)
	}
	if e.Author != "DJ Test" {
		t.Errorf("Author: got %q", e.Author)
	}
}

func TestLoadJammer_EmptyMeta(t *testing.T) {
	content := "https://soundcloud.com/author/track?|{}\n"
	path := writeTemp(t, content)
	entries, _, err := playlist.LoadJammer(path, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].URL != "https://soundcloud.com/author/track" {
		t.Errorf("URL: got %q", entries[0].URL)
	}
}

func TestLoadJammer_BareURL(t *testing.T) {
	content := "https://soundcloud.com/author/track\n"
	path := writeTemp(t, content)
	entries, _, err := playlist.LoadJammer(path, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].URL != "https://soundcloud.com/author/track" {
		t.Errorf("URL: got %q", entries[0].URL)
	}
}

func TestLoadJammer_SkipsBlankLines(t *testing.T) {
	content := "\n\nhttps://soundcloud.com/a/b?|{}\n\nhttps://soundcloud.com/c/d?|{}\n\n"
	path := writeTemp(t, content)
	entries, _, err := playlist.LoadJammer(path, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(entries))
	}
}

func TestLoadJammer_ResolvesLocalFile(t *testing.T) {
	songsDir := t.TempDir()
	// Create a fake song file that matches the expected basename
	basename := "soundcloud.com author track"
	songPath := filepath.Join(songsDir, basename+".mp3")
	os.WriteFile(songPath, []byte("fake"), 0o644)

	content := `https://soundcloud.com/author/track?|{"Title":"T","Author":"A"}` + "\n"
	path := writeTemp(t, content)

	entries, _, err := playlist.LoadJammer(path, songsDir)
	if err != nil {
		t.Fatal(err)
	}
	if entries[0].Path != songPath {
		t.Errorf("Path: got %q want %q", entries[0].Path, songPath)
	}
	if !entries[0].Downloaded() {
		t.Error("Downloaded() should be true")
	}
}

// ── LoadM3U ───────────────────────────────────────────────────────────────────

func TestLoadM3U_Basic(t *testing.T) {
	content := "#EXTM3U\n#EXTINF:180,Artist - Song Title\nhttps://soundcloud.com/a/b\n"
	f, err := os.CreateTemp(t.TempDir(), "*.m3u")
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString(content)
	f.Close()

	entries, err := playlist.LoadM3U(f.Name(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	e := entries[0]
	if e.Author != "Artist" {
		t.Errorf("Author: got %q", e.Author)
	}
	if e.Title != "Song Title" {
		t.Errorf("Title: got %q", e.Title)
	}
	if e.URL != "https://soundcloud.com/a/b" {
		t.Errorf("URL: got %q", e.URL)
	}
}

// ── List ──────────────────────────────────────────────────────────────────────

func TestList(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"a.jammer", "b.m3u", "c.m3u8", "d.txt", "e.mp3"} {
		os.WriteFile(filepath.Join(dir, name), nil, 0o644)
	}
	names, err := playlist.List(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 3 {
		t.Errorf("expected 3 playlist files, got %d: %v", len(names), names)
	}
}

// ── Classic format round-trip ──────────────────────────────────────────────────

func TestClassicRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jammer")

	original := []playlist.Entry{
		{URL: "https://soundcloud.com/a/b", Title: "Track One", Author: "Artist A"},
		{URL: "https://soundcloud.com/c/d", Title: "Track Two", Author: "Artist B"},
		{URL: "https://soundcloud.com/e/f", Title: "", Author: ""},
	}

	if err := playlist.Save(path, original); err != nil {
		t.Fatal(err)
	}

	loaded, jsonl, err := playlist.LoadJammer(path, dir)
	if err != nil {
		t.Fatal(err)
	}
	if jsonl {
		t.Error("expected jsonl=false for classic format file")
	}
	if len(loaded) != len(original) {
		t.Fatalf("want %d entries, got %d", len(original), len(loaded))
	}
	for i, e := range loaded {
		if e.URL != original[i].URL {
			t.Errorf("[%d] URL: want %q got %q", i, original[i].URL, e.URL)
		}
		if e.Title != original[i].Title {
			t.Errorf("[%d] Title: want %q got %q", i, original[i].Title, e.Title)
		}
		if e.Author != original[i].Author {
			t.Errorf("[%d] Author: want %q got %q", i, original[i].Author, e.Author)
		}
	}
}

func TestSaveJSONLRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jammer")

	original := []playlist.Entry{
		{URL: "https://soundcloud.com/a/b", Title: "Track One", Author: "Artist A"},
		{URL: "https://soundcloud.com/c/d", Title: "Track Two", Author: "Artist B"},
		{URL: "https://soundcloud.com/e/f", Title: "", Author: ""},
	}

	if err := playlist.SaveJSONL(path, original); err != nil {
		t.Fatal(err)
	}

	loaded, jsonl, err := playlist.LoadJammer(path, dir)
	if err != nil {
		t.Fatal(err)
	}
	if !jsonl {
		t.Error("expected jsonl=true for JSONL format file")
	}
	if len(loaded) != len(original) {
		t.Fatalf("want %d entries, got %d", len(original), len(loaded))
	}
	for i, e := range loaded {
		if e.URL != original[i].URL {
			t.Errorf("[%d] URL: want %q got %q", i, original[i].URL, e.URL)
		}
		if e.Title != original[i].Title {
			t.Errorf("[%d] Title: want %q got %q", i, original[i].Title, e.Title)
		}
		if e.Author != original[i].Author {
			t.Errorf("[%d] Author: want %q got %q", i, original[i].Author, e.Author)
		}
	}
}

func TestClassicCompat(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "classic.jammer")
	content := `https://soundcloud.com/author/track?|{"Title":"My Track","Author":"DJ Test"}` + "\n"
	os.WriteFile(path, []byte(content), 0o644)

	entries, jsonl, err := playlist.LoadJammer(path, dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("want 1 entry, got %d", len(entries))
	}
	if entries[0].Title != "My Track" {
		t.Errorf("Title: got %q", entries[0].Title)
	}
	if entries[0].Author != "DJ Test" {
		t.Errorf("Author: got %q", entries[0].Author)
	}
	if jsonl {
		t.Error("expected jsonl=false for classic ?| format")
	}
}

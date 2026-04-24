package player_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jooapa/jammer/jammer-go/internal/audio"
	"github.com/jooapa/jammer/jammer-go/internal/player"
	"github.com/jooapa/jammer/jammer-go/internal/playlist"
	"github.com/jooapa/jammer/jammer-go/internal/tags"
)

// minimalMP3 is a tiny valid MP3 file (ID3v2 header + one silent MPEG frame).
var minimalMP3 = []byte{
	0x49, 0x44, 0x33, 0x03, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0xff, 0xfb, 0x90, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
}

// helpers

func makePlayer(songs ...player.Song) *player.Player {
	p := player.NewHeadless(audio.NewNullBackend())
	p.SetSongs(songs)
	return p
}

func song(title string, downloaded bool) player.Song {
	path := ""
	if downloaded {
		path = "/fake/" + title + ".mp3"
	}
	return player.Song{Title: title, Path: path}
}

// ── Songs / UpdateSongPath ────────────────────────────────────────────────────

func TestSongs_ReturnsCopy(t *testing.T) {
	p := makePlayer(song("a", true), song("b", false))
	s := p.Songs()
	if len(s) != 2 {
		t.Fatalf("expected 2 songs, got %d", len(s))
	}
	// Mutating the returned slice must not affect internal state.
	s[0].Title = "mutated"
	if p.Songs()[0].Title == "mutated" {
		t.Error("Songs() returned a reference, not a copy")
	}
}

func TestUpdateSongPath(t *testing.T) {
	p := makePlayer(song("a", false))
	if p.Songs()[0].Downloaded() {
		t.Fatal("song should start as not downloaded")
	}
	p.UpdateSongPath(0, "/fake/a.mp3")
	if !p.Songs()[0].Downloaded() {
		t.Error("song should be downloaded after UpdateSongPath")
	}
}

func TestUpdateSongPath_OutOfBounds(t *testing.T) {
	p := makePlayer(song("a", true))
	// Should not panic
	p.UpdateSongPath(99, "/fake/x.mp3")
	p.UpdateSongPath(-1, "/fake/x.mp3")
}

// ── LoadPlaylist ──────────────────────────────────────────────────────────────

func TestLoadPlaylist(t *testing.T) {
	p := makePlayer(song("old", true))
	entries := []playlist.Entry{
		{URL: "https://soundcloud.com/a/b", Title: "New Song", Author: "Artist"},
		{Path: "/local/file.mp3", Title: "Local"},
	}
	p.LoadPlaylist(entries)
	songs := p.Songs()
	if len(songs) != 2 {
		t.Fatalf("expected 2 songs, got %d", len(songs))
	}
	if songs[0].Title != "New Song" {
		t.Errorf("songs[0].Title = %q", songs[0].Title)
	}
	if songs[1].Path != "/local/file.mp3" {
		t.Errorf("songs[1].Path = %q", songs[1].Path)
	}
}

func TestLoadPlaylist_ResetsIndex(t *testing.T) {
	p := makePlayer(song("a", true), song("b", true), song("c", true))
	// Simulate index being at 2
	p.SetIndexForTest(2)
	p.LoadPlaylist([]playlist.Entry{{Path: "/x.mp3", Title: "x"}})
	if p.Index() != 0 {
		t.Errorf("Index should reset to 0 after LoadPlaylist, got %d", p.Index())
	}
}

// ── Index / State ─────────────────────────────────────────────────────────────

func TestInitialState(t *testing.T) {
	p := makePlayer(song("a", true))
	if p.State() != player.StateStopped {
		t.Errorf("initial state should be Stopped")
	}
	if p.Index() != 0 {
		t.Errorf("initial index should be 0")
	}
}

// ── Volume ────────────────────────────────────────────────────────────────────

func TestVolume_Clamp(t *testing.T) {
	p := makePlayer()
	p.SetVolume(2.0)
	if p.Volume() != 1.0 {
		t.Errorf("volume above 1 should clamp to 1, got %f", p.Volume())
	}
	p.SetVolume(-0.5)
	if p.Volume() != 0.0 {
		t.Errorf("volume below 0 should clamp to 0, got %f", p.Volume())
	}
}

func TestVolume_NormalRange(t *testing.T) {
	p := makePlayer()
	p.SetVolume(0.75)
	if p.Volume() != 0.75 {
		t.Errorf("got %f", p.Volume())
	}
}

// ── DisplayTitle ──────────────────────────────────────────────────────────────

func TestDisplayTitle_TitleAndAuthor(t *testing.T) {
	s := player.Song{Title: "Track", Author: "Artist"}
	if s.DisplayTitle() != "Track" {
		t.Errorf("got %q", s.DisplayTitle())
	}
}

func TestDisplayTitle_TitleOnly(t *testing.T) {
	s := player.Song{Title: "Track"}
	if s.DisplayTitle() != "Track" {
		t.Errorf("got %q", s.DisplayTitle())
	}
}

func TestDisplayTitle_PathFallback(t *testing.T) {
	s := player.Song{Path: "/music/my song.mp3"}
	if s.DisplayTitle() != "my song" {
		t.Errorf("got %q", s.DisplayTitle())
	}
}

func TestDisplayTitle_URLFallback(t *testing.T) {
	s := player.Song{URL: "https://soundcloud.com/a/b"}
	if s.DisplayTitle() != "https://soundcloud.com/a/b" {
		t.Errorf("got %q", s.DisplayTitle())
	}
}

// ── LoadPlaylist enriches from embedded file tags ─────────────────────────────

// TestLoadPlaylist_EnrichesFromFileTags verifies that when a song is already
// downloaded (has a local path) but the playlist entry has no title/author,
// LoadPlaylist reads the embedded ID3 tags from the file and populates them.
// This covers the "song already downloaded" case where saveCurrentPlaylist
// will then persist the enriched metadata back to the .jammer file.
func TestLoadPlaylist_EnrichesFromFileTags(t *testing.T) {
	dir := t.TempDir()
	mp3Path := filepath.Join(dir, "track.mp3")

	if err := os.WriteFile(mp3Path, minimalMP3, 0644); err != nil {
		t.Fatal(err)
	}
	if err := tags.Write(mp3Path, "Tagged Title", "Tagged Artist"); err != nil {
		t.Fatalf("tags.Write: %v", err)
	}

	p := player.NewHeadless(audio.NewNullBackend())
	// Entry has a path but no title/author — simulates a playlist loaded from
	// disk where the .jammer file was saved before metadata was known.
	entries := []playlist.Entry{
		{URL: "https://soundcloud.com/a/b", Path: mp3Path},
	}
	p.LoadPlaylist(entries)

	songs := p.Songs()
	if songs[0].Title != "Tagged Title" {
		t.Errorf("Title = %q, want %q", songs[0].Title, "Tagged Title")
	}
	if songs[0].Author != "Tagged Artist" {
		t.Errorf("Author = %q, want %q", songs[0].Author, "Tagged Artist")
	}
}

// TestLoadPlaylist_DoesNotOverwriteExistingTitle verifies that if the playlist
// already has a title for a song, the embedded file tags are not read.
func TestLoadPlaylist_DoesNotOverwriteExistingTitle(t *testing.T) {
	dir := t.TempDir()
	mp3Path := filepath.Join(dir, "track.mp3")

	if err := os.WriteFile(mp3Path, minimalMP3, 0644); err != nil {
		t.Fatal(err)
	}
	if err := tags.Write(mp3Path, "File Title", "File Artist"); err != nil {
		t.Fatalf("tags.Write: %v", err)
	}

	p := player.NewHeadless(audio.NewNullBackend())
	entries := []playlist.Entry{
		{Path: mp3Path, Title: "Stored Title", Author: "Stored Artist"},
	}
	p.LoadPlaylist(entries)

	songs := p.Songs()
	if songs[0].Title != "Stored Title" {
		t.Errorf("Title = %q, want stored title %q", songs[0].Title, "Stored Title")
	}
}

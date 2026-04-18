package tags_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jooapa/jammer/jammer-go/internal/tags"
)

// minimalMP3 is a tiny but valid MP3 file (ID3v2 header + one silent MPEG frame).
// Enough for id3v2 to open and write tags, and for dhowden/tag to read them back.
var minimalMP3 = []byte{
	// ID3v2.3 header: "ID3" + version 2.3 + flags 0 + size 0
	0x49, 0x44, 0x33, 0x03, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00,
	// One silent MPEG1 Layer3 frame (128kbps, 44100Hz, stereo)
	0xff, 0xfb, 0x90, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
}

func TestWriteAndRead(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.mp3")
	if err := os.WriteFile(path, minimalMP3, 0644); err != nil {
		t.Fatal(err)
	}

	if err := tags.Write(path, "My Title", "My Artist"); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	info, err := tags.Read(path)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if info.Title != "My Title" {
		t.Errorf("Title = %q, want %q", info.Title, "My Title")
	}
	if info.Artist != "My Artist" {
		t.Errorf("Artist = %q, want %q", info.Artist, "My Artist")
	}
}

func TestWrite_NonMp3_IsSkipped(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.ogg")
	if err := os.WriteFile(path, []byte("fake ogg"), 0644); err != nil {
		t.Fatal(err)
	}
	// Should not error — ogg files are silently skipped.
	if err := tags.Write(path, "Title", "Artist"); err != nil {
		t.Errorf("Write on .ogg should be a no-op, got: %v", err)
	}
}

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

// minimalOGG is a minimal valid OGG Vorbis file containing three header pages
// (identification, comment, setup) with no audio frames — enough for the OGG
// tag writer to locate and patch the Vorbis Comment header.
var minimalOGG = []byte{
	// Page 0 — identification header
	79, 103, 103, 83, 0, 2, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0,
	234, 48, 134, 170, 1, 30, 1, 118, 111, 114, 98, 105, 115, 0, 0, 0, 0, 1,
	68, 172, 0, 0, 0, 0, 0, 0, 128, 181, 1, 0, 0, 0, 0, 0, 184, 1,
	// Page 1 — comment header (vendor "jammer-test", 0 comments)
	79, 103, 103, 83, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0,
	27, 207, 3, 228, 1, 27, 3, 118, 111, 114, 98, 105, 115, 11, 0, 0, 0, 106,
	97, 109, 109, 101, 114, 45, 116, 101, 115, 116, 0, 0, 0, 0, 1,
	// Page 2 — setup header (minimal stub)
	79, 103, 103, 83, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 2, 0, 0, 0,
	173, 71, 25, 19, 1, 8, 5, 118, 111, 114, 98, 105, 115, 1,
}

func TestWriteOGG(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.ogg")
	if err := os.WriteFile(path, minimalOGG, 0644); err != nil {
		t.Fatal(err)
	}

	if err := tags.Write(path, "OGG Title", "OGG Artist"); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Read back with dhowden/tag to verify the tags landed.
	info, err := tags.Read(path)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if info.Title != "OGG Title" {
		t.Errorf("Title = %q, want %q", info.Title, "OGG Title")
	}
	if info.Artist != "OGG Artist" {
		t.Errorf("Artist = %q, want %q", info.Artist, "OGG Artist")
	}
}

func TestWrite_UnknownExt_IsSkipped(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.wav")
	if err := os.WriteFile(path, []byte("fake wav data"), 0644); err != nil {
		t.Fatal(err)
	}
	// Unknown/unsupported extensions should be silently skipped.
	if err := tags.Write(path, "Title", "Artist"); err != nil {
		t.Errorf("Write on .wav should be a no-op, got: %v", err)
	}
}

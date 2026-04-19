// Package tags reads and writes ID3/Vorbis/MP4 metadata from audio files.
package tags

import (
	"os"
	"path/filepath"
	"strings"

	id3 "github.com/bogem/id3v2/v2"
	"github.com/dhowden/tag"
)

// Info holds the metadata fields we care about.
type Info struct {
	Title  string
	Artist string
	Album  string
	Year   int
	Genre  string
}

// Read opens path and returns embedded tag metadata.
// Returns an empty Info (not an error) if the file has no tags or an
// unsupported format — the caller should treat empty strings as "unknown".
func Read(path string) (Info, error) {
	f, err := os.Open(path)
	if err != nil {
		return Info{}, err
	}
	defer f.Close()

	m, err := tag.ReadFrom(f)
	if err != nil {
		// No tags or unrecognised format — not fatal.
		return Info{}, nil
	}

	return Info{
		Title:  m.Title(),
		Artist: m.Artist(),
		Album:  m.Album(),
		Year:   m.Year(),
		Genre:  m.Genre(),
	}, nil
}

// Write embeds title and artist metadata into an audio file at path.
// Supported formats: .mp3 (ID3v2), .ogg/.oga (Vorbis Comment), .flac (Vorbis Comment).
// Other formats are silently skipped (no error).
func Write(path, title, artist string) error {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".mp3":
		t, err := id3.Open(path, id3.Options{Parse: true})
		if err != nil {
			return err
		}
		defer t.Close()
		if title != "" {
			t.SetTitle(title)
		}
		if artist != "" {
			t.SetArtist(artist)
		}
		return t.Save()
	case ".ogg", ".oga":
		return writeOGG(path, title, artist)
	case ".flac":
		return writeFLAC(path, title, artist)
	default:
		return nil
	}
}

// Package tags reads ID3/Vorbis/MP4 metadata from audio files.
package tags

import (
	"os"

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

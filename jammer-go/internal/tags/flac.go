package tags

// FLAC Vorbis Comment tag writer using github.com/go-flac/go-flac +
// github.com/go-flac/flacvorbis.

import (
	flaclib "github.com/go-flac/go-flac"
	"github.com/go-flac/flacvorbis"
	"strings"
)

// writeFLAC patches the TITLE and ARTIST Vorbis comment fields in a FLAC file.
func writeFLAC(path, title, artist string) error {
	f, err := flaclib.ParseFile(path)
	if err != nil {
		return err
	}

	// Find and update existing VorbisComment block, or create one.
	found := false
	for i, block := range f.Meta {
		if block.Type != flaclib.VorbisComment {
			continue
		}
		vc, err := flacvorbis.ParseFromMetaDataBlock(*block)
		if err != nil {
			continue
		}
		// Remove existing TITLE / ARTIST entries.
		var keep []string
		for _, c := range vc.Comments {
			upper := strings.ToUpper(c)
			if strings.HasPrefix(upper, "TITLE=") || strings.HasPrefix(upper, "ARTIST=") {
				continue
			}
			keep = append(keep, c)
		}
		vc.Comments = keep
		if title != "" {
			_ = vc.Add(flacvorbis.FIELD_TITLE, title)
		}
		if artist != "" {
			_ = vc.Add(flacvorbis.FIELD_ARTIST, artist)
		}
		newBlock := vc.Marshal()
		f.Meta[i] = &newBlock
		found = true
		break
	}

	if !found {
		vc := flacvorbis.New()
		if title != "" {
			_ = vc.Add(flacvorbis.FIELD_TITLE, title)
		}
		if artist != "" {
			_ = vc.Add(flacvorbis.FIELD_ARTIST, artist)
		}
		newBlock := vc.Marshal()
		f.Meta = append(f.Meta, &newBlock)
	}

	return f.Save(path)
}

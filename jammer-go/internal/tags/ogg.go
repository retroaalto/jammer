package tags

// OGG Vorbis Comment rewriter.
//
// An OGG Vorbis file is a sequence of OGG pages.  The first two pages hold
// the two mandatory Vorbis header packets:
//   page 0 — identification header  (packet type 0x01)
//   page 1 — comment header         (packet type 0x03)
//
// We locate the comment header packet, replace the TITLE and ARTIST fields,
// serialise it back into the same page structure, and write the whole file
// to a tmp path then atomically rename it.

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// writeOGG patches the TITLE and ARTIST Vorbis comment fields in an OGG file.
func writeOGG(path, title, artist string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	patched, err := patchOGGComments(data, title, artist)
	if err != nil {
		return err
	}

	// Write via tmp + rename for atomicity.
	tmp := filepath.Join(filepath.Dir(path), filepath.Base(path)+".tmp")
	if err := os.WriteFile(tmp, patched, 0644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

// ── OGG page parsing ──────────────────────────────────────────────────────────

// oggPage is a parsed OGG page header + payload.
type oggPage struct {
	raw     []byte // the full original bytes of this page (header + segments)
	packets [][]byte
	// header fields we need for reconstruction
	headerTypeFlag byte
	granulePos     uint64
	serialNo       uint32
	seqNo          uint32
	// segment table as read
	segTable []byte
}

// parseOGGPages splits raw OGG data into pages.
func parseOGGPages(data []byte) ([]oggPage, error) {
	var pages []oggPage
	pos := 0
	for pos < len(data) {
		if pos+4 > len(data) {
			break
		}
		if !bytes.Equal(data[pos:pos+4], []byte("OggS")) {
			return nil, fmt.Errorf("ogg: missing capture pattern at offset %d", pos)
		}
		if pos+27 > len(data) {
			return nil, fmt.Errorf("ogg: truncated page header at offset %d", pos)
		}
		// OGG page header layout (27 bytes):
		//  0-3   "OggS"
		//  4     version (0)
		//  5     header_type_flag
		//  6-13  granule_position (int64 LE)
		//  14-17 bitstream_serial_number (uint32 LE)
		//  18-21 page_sequence_no (uint32 LE)
		//  22-25 CRC32 (uint32 LE)
		//  26    page_segments (uint8)
		htf := data[pos+5]
		granulePos := binary.LittleEndian.Uint64(data[pos+6 : pos+14])
		serialNo := binary.LittleEndian.Uint32(data[pos+14 : pos+18])
		seqNo := binary.LittleEndian.Uint32(data[pos+18 : pos+22])
		numSegs := int(data[pos+26])
		if pos+27+numSegs > len(data) {
			return nil, fmt.Errorf("ogg: truncated segment table at offset %d", pos)
		}
		segTable := data[pos+27 : pos+27+numSegs]

		// Compute total page body length.
		bodyLen := 0
		for _, s := range segTable {
			bodyLen += int(s)
		}
		pageEnd := pos + 27 + numSegs + bodyLen
		if pageEnd > len(data) {
			return nil, fmt.Errorf("ogg: page body exceeds file size at offset %d", pos)
		}

		raw := data[pos:pageEnd]
		body := data[pos+27+numSegs : pageEnd]

		// Collect packets from segments.
		var packets [][]byte
		var pkt []byte
		for _, s := range segTable {
			pkt = append(pkt, body[:s]...)
			body = body[s:]
			if s < 255 {
				packets = append(packets, pkt)
				pkt = nil
			}
		}
		// A continued packet at end (shouldn't happen for comment header page).
		if len(pkt) > 0 {
			packets = append(packets, pkt)
		}

		pages = append(pages, oggPage{
			raw:            raw,
			packets:        packets,
			headerTypeFlag: htf,
			granulePos:     granulePos,
			serialNo:       serialNo,
			seqNo:          seqNo,
			segTable:       append([]byte(nil), segTable...),
		})
		pos = pageEnd
	}
	return pages, nil
}

// patchOGGComments finds the Vorbis comment header packet, patches title/artist,
// serialises it back, and returns the full modified file bytes.
func patchOGGComments(data []byte, title, artist string) ([]byte, error) {
	pages, err := parseOGGPages(data)
	if err != nil {
		return nil, err
	}

	commentPageIdx := -1
	commentPktIdx := -1

	for pi, pg := range pages {
		for pki, pkt := range pg.packets {
			if len(pkt) > 7 &&
				pkt[0] == 0x03 && // comment header type
				bytes.Equal(pkt[1:7], []byte("vorbis")) {
				commentPageIdx = pi
				commentPktIdx = pki
				break
			}
		}
		if commentPageIdx >= 0 {
			break
		}
	}
	if commentPageIdx < 0 {
		return nil, fmt.Errorf("ogg: vorbis comment header not found")
	}

	// Parse existing comment block and patch it.
	pkt := pages[commentPageIdx].packets[commentPktIdx]
	newPkt, err := patchVorbisCommentPacket(pkt, title, artist)
	if err != nil {
		return nil, err
	}

	// Rebuild the page with the patched packet.
	pages[commentPageIdx].packets[commentPktIdx] = newPkt
	newPageBytes, err := rebuildOGGPage(pages[commentPageIdx])
	if err != nil {
		return nil, err
	}
	pages[commentPageIdx].raw = newPageBytes

	// Reassemble file.
	var buf bytes.Buffer
	for _, pg := range pages {
		buf.Write(pg.raw)
	}
	return buf.Bytes(), nil
}

// patchVorbisCommentPacket replaces TITLE and ARTIST in a raw Vorbis comment packet.
// Layout: [0x03]["vorbis"][vendor_len LE32][vendor][num_comments LE32][comments...][framing_bit]
func patchVorbisCommentPacket(pkt []byte, title, artist string) ([]byte, error) {
	if len(pkt) < 7 {
		return nil, fmt.Errorf("ogg: comment packet too short")
	}
	r := bytes.NewReader(pkt[7:]) // skip packet type + "vorbis"

	readU32 := func() (uint32, error) {
		var v uint32
		return v, binary.Read(r, binary.LittleEndian, &v)
	}

	vendorLen, err := readU32()
	if err != nil {
		return nil, fmt.Errorf("ogg: comment vendor len: %w", err)
	}
	vendorBytes := make([]byte, vendorLen)
	if _, err := r.Read(vendorBytes); err != nil {
		return nil, fmt.Errorf("ogg: comment vendor: %w", err)
	}

	numComments, err := readU32()
	if err != nil {
		return nil, fmt.Errorf("ogg: num comments: %w", err)
	}
	var comments []string
	for i := uint32(0); i < numComments; i++ {
		clen, err := readU32()
		if err != nil {
			return nil, fmt.Errorf("ogg: comment %d len: %w", i, err)
		}
		cbytes := make([]byte, clen)
		if _, err := r.Read(cbytes); err != nil {
			return nil, fmt.Errorf("ogg: comment %d: %w", i, err)
		}
		comments = append(comments, string(cbytes))
	}

	// Remove existing TITLE / ARTIST entries, then add new ones.
	var filtered []string
	for _, c := range comments {
		upper := strings.ToUpper(c)
		if strings.HasPrefix(upper, "TITLE=") || strings.HasPrefix(upper, "ARTIST=") {
			continue
		}
		filtered = append(filtered, c)
	}
	if title != "" {
		filtered = append(filtered, "TITLE="+title)
	}
	if artist != "" {
		filtered = append(filtered, "ARTIST="+artist)
	}

	// Serialise.
	var buf bytes.Buffer
	buf.Write([]byte{0x03})
	buf.Write([]byte("vorbis"))
	writeU32 := func(v uint32) {
		b := make([]byte, 4)
		binary.LittleEndian.PutUint32(b, v)
		buf.Write(b)
	}
	writeU32(uint32(len(vendorBytes)))
	buf.Write(vendorBytes)
	writeU32(uint32(len(filtered)))
	for _, c := range filtered {
		writeU32(uint32(len(c)))
		buf.WriteString(c)
	}
	buf.WriteByte(0x01) // framing bit

	return buf.Bytes(), nil
}

// rebuildOGGPage serialises an oggPage (with updated packets) back to bytes,
// recomputing the segment table and CRC.
func rebuildOGGPage(pg oggPage) ([]byte, error) {
	// Build body and segment table from packets.
	var body bytes.Buffer
	var segTable []byte
	for _, pkt := range pg.packets {
		remaining := len(pkt)
		if remaining == 0 {
			segTable = append(segTable, 0)
		}
		for remaining > 0 {
			seg := remaining
			if seg > 255 {
				seg = 255
			}
			segTable = append(segTable, byte(seg))
			body.Write(pkt[len(pkt)-remaining : len(pkt)-remaining+seg])
			remaining -= seg
		}
		// Terminate packet with a <255 segment if the last segment was exactly 255.
		if len(pkt)%255 == 0 && len(pkt) > 0 {
			segTable = append(segTable, 0)
		}
	}
	numSegs := len(segTable)

	var hdr bytes.Buffer
	hdr.Write([]byte("OggS"))
	hdr.WriteByte(0) // version
	hdr.WriteByte(pg.headerTypeFlag)
	b8 := make([]byte, 8)
	binary.LittleEndian.PutUint64(b8, pg.granulePos)
	hdr.Write(b8)
	b4 := make([]byte, 4)
	binary.LittleEndian.PutUint32(b4, pg.serialNo)
	hdr.Write(b4)
	binary.LittleEndian.PutUint32(b4, pg.seqNo)
	hdr.Write(b4)
	hdr.Write([]byte{0, 0, 0, 0}) // CRC placeholder
	hdr.WriteByte(byte(numSegs))
	hdr.Write(segTable)

	page := append(hdr.Bytes(), body.Bytes()...)

	// Compute and write CRC.
	crc := oggCRC32(page)
	binary.LittleEndian.PutUint32(page[22:26], crc)

	return page, nil
}

// oggCRC32 computes the OGG CRC32 checksum (with CRC bytes zeroed).
// Polynomial: 0x04c11db7 (no reflection, no final XOR).
var oggCRCTable [256]uint32

func init() {
	for i := 0; i < 256; i++ {
		crc := uint32(i) << 24
		for j := 0; j < 8; j++ {
			if crc&0x80000000 != 0 {
				crc = (crc << 1) ^ 0x04c11db7
			} else {
				crc <<= 1
			}
		}
		oggCRCTable[i] = crc
	}
}

func oggCRC32(data []byte) uint32 {
	var crc uint32
	for _, b := range data {
		crc = (crc << 8) ^ oggCRCTable[((crc>>24)^uint32(b))&0xff]
	}
	return crc
}

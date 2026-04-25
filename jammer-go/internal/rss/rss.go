// Package rss fetches and parses RSS/podcast feeds, mirroring the behaviour of
// the classic Jammer Rss.cs implementation.
package rss

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Feed contains channel-level metadata and the list of episodes.
type Feed struct {
	Title       string
	Author      string
	Link        string
	Description string
	Items       []Item
}

// Item represents a single podcast/RSS episode.
type Item struct {
	Title       string
	URL         string // resolved playable media URL (enclosure / media:content / link)
	Description string
	PubDate     string
	Author      string
}

// ── XML unmarshalling structures ──────────────────────────────────────────────

type xmlRSS struct {
	Channel xmlChannel `xml:"channel"`
}

type xmlChannel struct {
	Title         string    `xml:"title"`
	Author        string    `xml:"author"`
	ManagingEditor string   `xml:"managingEditor"`
	Link          string    `xml:"link"`
	Description   string    `xml:"description"`
	Items         []xmlItem `xml:"item"`
}

type xmlItem struct {
	Title        string           `xml:"title"`
	Link         string           `xml:"link"`
	GUID         string           `xml:"guid"`
	Description  string           `xml:"description"`
	PubDate      string           `xml:"pubDate"`
	Author       string           `xml:"author"`  // standard <author> or itunes:author (prefix stripped)
	DCCreator    string           `xml:"creator"` // dc:creator (prefix stripped → creator)
	Enclosure    *xmlEnclosure    `xml:"enclosure"`
	MediaContent *xmlMediaContent `xml:"content"` // media:content (prefix stripped → content)
}

type xmlEnclosure struct {
	URL string `xml:"url,attr"`
}

type xmlMediaContent struct {
	URL string `xml:"url,attr"`
}

// ── Public API ────────────────────────────────────────────────────────────────

var httpClient = &http.Client{Timeout: 15 * time.Second}

// Fetch downloads and parses the RSS feed at url.  It never returns nil; on
// error it returns a Feed with the error message in Description and an empty
// Items slice.
func Fetch(url string) (*Feed, error) {
	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", url, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", url, err)
	}
	return parse(body)
}

func parse(data []byte) (*Feed, error) {
	// encoding/xml doesn't handle namespace prefixes automatically; strip them
	// from the raw bytes so dc:creator, media:content, itunes:* all parse as
	// their local names.
	src := stripNSPrefixes(string(data))

	var raw xmlRSS
	if err := xml.Unmarshal([]byte(src), &raw); err != nil {
		return nil, fmt.Errorf("parse rss: %w", err)
	}
	ch := raw.Channel

	feed := &Feed{
		Title:       orDefault(ch.Title, "Unknown Title"),
		Author:      orDefault(ch.Author, ch.ManagingEditor, "Unknown Author"),
		Link:        orDefault(ch.Link, ""),
		Description: orDefault(ch.Description, ""),
	}

	for _, xi := range ch.Items {
		// Resolve media URL: enclosure > media:content > link > guid
		mediaURL := ""
		if xi.Enclosure != nil && xi.Enclosure.URL != "" {
			mediaURL = xi.Enclosure.URL
		} else if xi.MediaContent != nil && xi.MediaContent.URL != "" {
			mediaURL = xi.MediaContent.URL
		} else if xi.Link != "" {
			mediaURL = xi.Link
		} else {
			mediaURL = xi.GUID
		}

		author := orDefault(xi.Author, xi.DCCreator, feed.Author)

		feed.Items = append(feed.Items, Item{
			Title:       orDefault(xi.Title, "Unknown Title"),
			URL:         mediaURL,
			Description: xi.Description,
			PubDate:     xi.PubDate,
			Author:      author,
		})
	}
	return feed, nil
}

// IsURL returns true for strings that look like http(s) URLs — used by the UI
// to decide whether to attempt RSS parsing when a "song" is loaded.
func IsURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func orDefault(candidates ...string) string {
	for _, c := range candidates {
		if c != "" {
			return c
		}
	}
	return ""
}

// stripNSPrefixes removes namespace prefixes from XML tag names so
// encoding/xml can match them by local name.  For example:
//   <dc:creator>  →  <creator>
//   </itunes:author>  →  </author>
//
// Processing instructions (<?...?>), comments (<!--...-->), and CDATA
// sections (<![CDATA[...]]>) are passed through unchanged.
func stripNSPrefixes(src string) string {
	out := &strings.Builder{}
	out.Grow(len(src))
	for i := 0; i < len(src); {
		if src[i] != '<' {
			out.WriteByte(src[i])
			i++
			continue
		}
		// Check what follows '<'
		if i+1 < len(src) {
			next := src[i+1]
			// Processing instruction or declaration: <?...?>
			if next == '?' {
				end := strings.Index(src[i:], "?>")
				if end < 0 {
					out.WriteString(src[i:])
					return out.String()
				}
				out.WriteString(src[i : i+end+2])
				i += end + 2
				continue
			}
			// Comment: <!--...-->
			if next == '!' && strings.HasPrefix(src[i:], "<!--") {
				end := strings.Index(src[i:], "-->")
				if end < 0 {
					out.WriteString(src[i:])
					return out.String()
				}
				out.WriteString(src[i : i+end+3])
				i += end + 3
				continue
			}
			// CDATA: <![CDATA[...]]>
			if next == '!' && strings.HasPrefix(src[i:], "<![CDATA[") {
				end := strings.Index(src[i:], "]]>")
				if end < 0 {
					out.WriteString(src[i:])
					return out.String()
				}
				out.WriteString(src[i : i+end+3])
				i += end + 3
				continue
			}
		}
		// Regular element tag: strip optional namespace prefix from tag name.
		out.WriteByte('<')
		i++
		// optional '/' for closing tags
		if i < len(src) && src[i] == '/' {
			out.WriteByte('/')
			i++
		}
		// scan tag name up to first whitespace, '>', or '/'
		nameStart := i
		for i < len(src) && src[i] != '>' && src[i] != ' ' && src[i] != '\t' &&
			src[i] != '\n' && src[i] != '\r' && src[i] != '/' {
			i++
		}
		name := src[nameStart:i]
		if colon := strings.IndexByte(name, ':'); colon >= 0 {
			name = name[colon+1:]
		}
		out.WriteString(name)
	}
	return out.String()
}

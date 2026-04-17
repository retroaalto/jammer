// Package scid fetches and caches the SoundCloud API client_id.
//
// SoundCloud embeds the client_id in one of their JS bundles served from
// the homepage. We find a bundle URL from the page HTML, fetch it, and
// extract the client_id with a regex — no headless browser required.
package scid

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// cacheFile returns the path to the on-disk client_id cache.
func cacheFile() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "jammer", "sc_client_id.json")
}

type cachedID struct {
	ClientID  string    `json:"client_id"`
	FetchedAt time.Time `json:"fetched_at"`
}

const cacheTTL = 7 * 24 * time.Hour // re-fetch after one week

// Get returns a valid SoundCloud client_id, using a disk cache when possible.
func Get(ctx context.Context) (string, error) {
	if id, ok := loadCache(); ok {
		return id, nil
	}
	id, err := fetch(ctx)
	if err != nil {
		return "", err
	}
	saveCache(id)
	return id, nil
}

// Invalidate removes the cached client_id so the next call to Get fetches a fresh one.
func Invalidate() {
	os.Remove(cacheFile())
}

// ── internal ──────────────────────────────────────────────────────────────────

var (
	// matches <script ... src="https://a-v2.sndcdn.com/assets/..."></script>
	scriptRe = regexp.MustCompile(`https://a-v2\.sndcdn\.com/assets/[^"']+\.js`)
	// matches client_id:"XXXX" or client_id="XXXX" or client_id\x3a"XXXX"
	clientIDRe = regexp.MustCompile(`client_id[=:]["']?([a-zA-Z0-9_-]{20,})`)
)

func fetch(ctx context.Context) (string, error) {
	// 1. Fetch soundcloud.com homepage
	body, err := httpGet(ctx, "https://soundcloud.com/")
	if err != nil {
		return "", fmt.Errorf("scid: fetch homepage: %w", err)
	}

	// 2. Find JS bundle URLs
	matches := scriptRe.FindAllString(body, -1)
	if len(matches) == 0 {
		return "", fmt.Errorf("scid: no JS bundle URLs found on soundcloud.com homepage")
	}

	// 3. Try each bundle until we find client_id (check the last few — the app
	//    bundle is usually near the end of the page).
	for i := len(matches) - 1; i >= 0 && i >= len(matches)-5; i-- {
		jsURL := matches[i]
		jsBody, err := httpGet(ctx, jsURL)
		if err != nil {
			continue
		}
		if id := extractClientID(jsBody); id != "" {
			return id, nil
		}
	}

	// 4. Also check the homepage HTML itself (sometimes inlined)
	if id := extractClientID(body); id != "" {
		return id, nil
	}

	return "", fmt.Errorf("scid: client_id not found in any of the %d JS bundles checked", len(matches))
}

func extractClientID(s string) string {
	// Unescape \x3a (:) which minifiers sometimes emit
	s = strings.ReplaceAll(s, `\x3a`, ":")
	m := clientIDRe.FindStringSubmatch(s)
	if len(m) > 1 {
		return m[1]
	}
	return ""
}

func httpGet(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 Chrome/120 Safari/537.36")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("HTTP %d for %s", resp.StatusCode, url)
	}
	b, err := io.ReadAll(resp.Body)
	return string(b), err
}

func loadCache() (string, bool) {
	data, err := os.ReadFile(cacheFile())
	if err != nil {
		return "", false
	}
	var c cachedID
	if err := json.Unmarshal(data, &c); err != nil {
		return "", false
	}
	if time.Since(c.FetchedAt) > cacheTTL || c.ClientID == "" {
		return "", false
	}
	return c.ClientID, true
}

func saveCache(id string) {
	path := cacheFile()
	os.MkdirAll(filepath.Dir(path), 0o755)
	data, _ := json.Marshal(cachedID{ClientID: id, FetchedAt: time.Now()})
	os.WriteFile(path, data, 0o644)
}

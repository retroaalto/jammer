package downloader

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	jlog "github.com/jooapa/jammer/jammer-go/internal/log"
	"github.com/jooapa/jammer/jammer-go/internal/playlist"
	"github.com/jooapa/jammer/jammer-go/internal/scid"
	yt "github.com/kkdai/youtube/v2"
)

// Progress is sent during a download to report completion in [0.0, 1.0].
type Progress struct {
	Done    bool
	Err     error
	Frac    float64 // 0..1
	Message string
}

// Meta holds track metadata returned alongside the downloaded file path.
type Meta struct {
	Title  string
	Artist string
}

// Download downloads the song at url into songsDir and returns the local path
// and any track metadata known at download time (title, artist).
// progress receives updates; it is closed when the download finishes.
func Download(ctx context.Context, rawURL, songsDir string, progress chan<- Progress) (string, Meta, error) {
	defer close(progress)
	jlog.Infof("downloader: start url=%q", rawURL)

	if isSoundCloud(rawURL) {
		return downloadSoundCloud(ctx, rawURL, songsDir, progress)
	}
	if isYouTubePlaylist(rawURL) {
		return downloadViaYtdlp(ctx, rawURL, songsDir, progress)
	}
	if isYouTube(rawURL) {
		return downloadYouTube(ctx, rawURL, songsDir, progress)
	}
	path, err := downloadHTTP(ctx, rawURL, songsDir, progress)
	return path, Meta{}, err
}

// ── SoundCloud ────────────────────────────────────────────────────────────────

// SC API v2 types (minimal)
type scTrack struct {
	Media struct {
		Transcodings []struct {
			URL    string `json:"url"`
			Format struct {
				Protocol string `json:"protocol"`
				MimeType string `json:"mime_type"`
			} `json:"format"`
			Quality string `json:"quality"`
		} `json:"transcodings"`
	} `json:"media"`
	Title string `json:"title"`
	User  struct {
		Username string `json:"username"`
	} `json:"user"`
}

type scStreamResponse struct {
	URL string `json:"url"`
}

func downloadSoundCloud(ctx context.Context, trackURL, songsDir string, progress chan<- Progress) (string, Meta, error) {
	send := func(frac float64, msg string) {
		select {
		case progress <- Progress{Frac: frac, Message: msg}:
		default:
		}
	}

	send(0, "fetching SC client_id...")
	clientID, err := scid.Get(ctx)
	if err != nil {
		jlog.Errorf("sc: client_id fetch failed: %v — falling back to yt-dlp", err)
		send(0.05, "client_id unavailable, trying yt-dlp...")
		return downloadViaYtdlp(ctx, trackURL, songsDir, progress)
	}
	jlog.Infof("sc: got client_id (len=%d)", len(clientID))

	send(0.1, "resolving track...")
	track, err := scResolveTrack(ctx, trackURL, clientID)
	if err != nil {
		jlog.Errorf("sc: resolve failed: %v — invalidating client_id and retrying", err)
		scid.Invalidate()
		send(0.1, "retrying with fresh client_id...")
		clientID, err = scid.Get(ctx)
		if err != nil {
			send(0.1, "client_id unavailable, trying yt-dlp...")
			return downloadViaYtdlp(ctx, trackURL, songsDir, progress)
		}
		track, err = scResolveTrack(ctx, trackURL, clientID)
		if err != nil {
			send(0.1, "SC API failed, trying yt-dlp...")
			return downloadViaYtdlp(ctx, trackURL, songsDir, progress)
		}
	}

	meta := Meta{Title: track.Title, Artist: track.User.Username}

	// Pick best transcoding: prefer progressive mp3
	var chosen *struct {
		URL    string
		Format struct {
			Protocol string `json:"protocol"`
			MimeType string `json:"mime_type"`
		}
		Quality string
	}
	for i := range track.Media.Transcodings {
		t := &track.Media.Transcodings[i]
		if t.Format.Protocol == "progressive" {
			if chosen == nil || strings.Contains(t.Format.MimeType, "mpeg") {
				chosen = &struct {
					URL    string
					Format struct {
						Protocol string `json:"protocol"`
						MimeType string `json:"mime_type"`
					}
					Quality string
				}{URL: t.URL, Format: t.Format, Quality: t.Quality}
			}
		}
	}
	if chosen == nil && len(track.Media.Transcodings) > 0 {
		t := &track.Media.Transcodings[0]
		chosen = &struct {
			URL    string
			Format struct {
				Protocol string `json:"protocol"`
				MimeType string `json:"mime_type"`
			}
			Quality string
		}{URL: t.URL, Format: t.Format, Quality: t.Quality}
	}
	if chosen == nil {
		send(0.2, "no transcodings found, trying yt-dlp...")
		return downloadViaYtdlp(ctx, trackURL, songsDir, progress)
	}

	send(0.2, "fetching stream URL...")
	streamURL, err := scGetStreamURL(ctx, chosen.URL, clientID)
	if err != nil {
		jlog.Errorf("sc: stream URL fetch failed: %v — falling back to yt-dlp", err)
		send(0.2, "stream URL fetch failed, trying yt-dlp...")
		return downloadViaYtdlp(ctx, trackURL, songsDir, progress)
	}
	jlog.Infof("sc: stream URL obtained")

	// Determine output extension
	ext := ".mp3"
	if strings.Contains(chosen.Format.MimeType, "ogg") || strings.Contains(chosen.Format.MimeType, "opus") {
		ext = ".ogg"
	}

	basename := playlist.URLToExpectedBasename(trackURL)
	tmpPath := filepath.Join(songsDir, basename+ext+".tmp")
	finalPath := filepath.Join(songsDir, basename+ext)

	send(0.25, "downloading...")
	req, err := http.NewRequestWithContext(ctx, "GET", streamURL, nil)
	if err != nil {
		return "", meta, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", meta, fmt.Errorf("sc download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", meta, fmt.Errorf("sc download: HTTP %d", resp.StatusCode)
	}

	tmpF, err := os.Create(tmpPath)
	if err != nil {
		return "", meta, err
	}

	total := resp.ContentLength
	written := int64(0)
	buf := make([]byte, 32*1024)
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := tmpF.Write(buf[:n]); werr != nil {
				tmpF.Close()
				os.Remove(tmpPath)
				return "", meta, werr
			}
			written += int64(n)
			if total > 0 {
				send(0.25+0.7*float64(written)/float64(total), "downloading...")
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			tmpF.Close()
			os.Remove(tmpPath)
			return "", meta, rerr
		}
	}
	tmpF.Close()

	if err := os.Rename(tmpPath, finalPath); err != nil {
		os.Remove(tmpPath)
		return "", meta, err
	}
	send(1.0, "done")
	return finalPath, meta, nil
}

func scResolveTrack(ctx context.Context, trackURL, clientID string) (*scTrack, error) {
	apiURL := "https://api-v2.soundcloud.com/resolve?url=" + url.QueryEscape(trackURL) + "&client_id=" + clientID
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("SC resolve: HTTP %d", resp.StatusCode)
	}
	var track scTrack
	if err := json.NewDecoder(resp.Body).Decode(&track); err != nil {
		return nil, err
	}
	return &track, nil
}

func scGetStreamURL(ctx context.Context, transcodingURL, clientID string) (string, error) {
	fullURL := transcodingURL + "?client_id=" + clientID
	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("SC stream URL: HTTP %d", resp.StatusCode)
	}
	var sr scStreamResponse
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return "", err
	}
	if sr.URL == "" {
		return "", fmt.Errorf("SC stream URL: empty response")
	}
	return sr.URL, nil
}

// ── YouTube ───────────────────────────────────────────────────────────────────

func downloadYouTube(ctx context.Context, url, songsDir string, progress chan<- Progress) (string, Meta, error) {
	send := func(frac float64, msg string) {
		select {
		case progress <- Progress{Frac: frac, Message: msg}:
		default:
		}
	}

	send(0, "fetching video info...")
	client := yt.Client{}
	video, err := client.GetVideoContext(ctx, url)
	if err != nil {
		return "", Meta{}, fmt.Errorf("youtube: %w", err)
	}

	meta := Meta{Title: video.Title, Artist: video.Author}

	// Pick best audio-only format (prefer opus/webm, fallback to mp4 audio)
	formats := video.Formats.WithAudioChannels()
	formats.Sort()
	var chosen *yt.Format
	for i := range formats {
		f := &formats[i]
		if f.AudioChannels > 0 && (strings.Contains(f.MimeType, "audio") || f.Width == 0) {
			chosen = f
			break
		}
	}
	if chosen == nil && len(formats) > 0 {
		chosen = &formats[0]
	}
	if chosen == nil {
		return "", meta, fmt.Errorf("youtube: no suitable format found")
	}

	send(0.05, "starting download...")
	stream, _, err := client.GetStreamContext(ctx, video, chosen)
	if err != nil {
		return "", meta, fmt.Errorf("youtube stream: %w", err)
	}
	defer stream.Close()

	// Determine extension from mime type
	ext := ".ogg"
	if strings.Contains(chosen.MimeType, "mp4") {
		ext = ".m4a"
	} else if strings.Contains(chosen.MimeType, "webm") {
		ext = ".webm"
	}

	basename := playlist.URLToExpectedBasename(url)
	tmpPath := filepath.Join(songsDir, basename+ext+".tmp")
	finalPath := filepath.Join(songsDir, basename+ext)

	tmpF, err := os.Create(tmpPath)
	if err != nil {
		return "", meta, err
	}

	totalBytes := chosen.ContentLength

	// Track bytes written atomically so a ticker goroutine can report progress
	// independently of the chunk-boundary bursts from the YouTube library.
	var written atomic.Int64

	// Ticker: send progress updates every 200ms while download is running.
	stopTicker := make(chan struct{})
	tickerDone := make(chan struct{})
	go func() {
		defer close(tickerDone)
		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()
		pulse := 0.05
		for {
			select {
			case <-ticker.C:
				w := written.Load()
				if totalBytes > 0 {
					send(0.05+0.8*float64(w)/float64(totalBytes), "downloading...")
				} else {
					pulse += 0.01
					if pulse > 0.80 {
						pulse = 0.80
					}
					send(pulse, "downloading...")
				}
			case <-stopTicker:
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	buf := make([]byte, 32*1024)
	var copyErr error
	for {
		n, rerr := stream.Read(buf)
		if n > 0 {
			if _, werr := tmpF.Write(buf[:n]); werr != nil {
				copyErr = werr
				break
			}
			written.Add(int64(n))
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			copyErr = rerr
			break
		}
	}

	close(stopTicker)
	<-tickerDone
	tmpF.Close()

	if copyErr != nil {
		os.Remove(tmpPath)
		return "", meta, copyErr
	}

	// Convert to ogg with ffmpeg if needed
	if ext != ".ogg" {
		send(0.85, "converting to ogg...")
		oggPath := filepath.Join(songsDir, basename+".ogg")
		cmd := exec.CommandContext(ctx, "ffmpeg", "-y", "-i", tmpPath, "-vn",
			"-c:a", "libvorbis", "-q:a", "5", oggPath)
		if out, err := cmd.CombinedOutput(); err != nil {
			os.Remove(tmpPath)
			return "", meta, fmt.Errorf("ffmpeg: %w\n%s", err, out)
		}
		os.Remove(tmpPath)
		send(1.0, "done")
		return oggPath, meta, nil
	}

	if err := os.Rename(tmpPath, finalPath); err != nil {
		return "", meta, err
	}
	send(1.0, "done")
	return finalPath, meta, nil
}

// ── SoundCloud / yt-dlp fallback ──────────────────────────────────────────────

func downloadViaYtdlp(ctx context.Context, url, songsDir string, progress chan<- Progress) (string, Meta, error) {
	send := func(frac float64, msg string) {
		select {
		case progress <- Progress{Frac: frac, Message: msg}:
		default:
		}
	}

	// Check yt-dlp is available
	ytdlpBin, err := findYtdlp()
	if err != nil {
		return "", Meta{}, err
	}

	basename := playlist.URLToExpectedBasename(url)
	outputTemplate := filepath.Join(songsDir, basename+".%(ext)s")

	send(0, "starting yt-dlp...")
	cmd := exec.CommandContext(ctx, ytdlpBin,
		"--no-playlist",
		"--extract-audio",
		"--audio-format", "mp3",
		"--audio-quality", "0",
		"--add-metadata",
		"--output", outputTemplate,
		url,
	)
	var stderrBuf bytes.Buffer
	cmd.Stdout = io.Discard
	cmd.Stderr = &stderrBuf

	// We can't easily hook into yt-dlp's progress inline, so just report indeterminate.
	done := make(chan error, 1)
	go func() { done <- cmd.Run() }()

	// Pulse progress while waiting
	frac := 0.0
	for {
		select {
		case err := <-done:
			if err != nil {
				if stderrBuf.Len() > 0 {
					jlog.Errorf("yt-dlp stderr: %s", stderrBuf.String())
				}
				return "", Meta{}, fmt.Errorf("yt-dlp: %w", err)
			}
			// Find the file yt-dlp wrote (basename + .mp3)
			path := filepath.Join(songsDir, basename+".mp3")
			if _, serr := os.Stat(path); serr == nil {
				send(1.0, "done")
				return path, Meta{}, nil
			}
			// Fallback: search for any file matching basename.*
			matches, _ := filepath.Glob(filepath.Join(songsDir, basename+".*"))
			for _, m := range matches {
				if !strings.HasSuffix(m, ".tmp") {
					send(1.0, "done")
					return m, Meta{}, nil
				}
			}
			return "", Meta{}, fmt.Errorf("yt-dlp: output file not found for %s", basename)
		case <-ctx.Done():
			return "", Meta{}, ctx.Err()
		default:
			frac += 0.02
			if frac > 0.95 {
				frac = 0.95
			}
			send(frac, "downloading via yt-dlp...")
		}
	}
}

func findYtdlp() (string, error) {
	for _, name := range []string{"yt-dlp", "yt-dlp.exe", "youtube-dl"} {
		if p, err := exec.LookPath(name); err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("yt-dlp not found in PATH — install it with: pip install yt-dlp")
}

// ── Generic HTTP download ─────────────────────────────────────────────────────

func downloadHTTP(ctx context.Context, url, songsDir string, progress chan<- Progress) (string, error) {
	send := func(frac float64, msg string) {
		select {
		case progress <- Progress{Frac: frac, Message: msg}:
		default:
		}
	}

	send(0, "connecting...")
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	basename := playlist.URLToExpectedBasename(url)
	// Try to get extension from content-type
	ext := extFromContentType(resp.Header.Get("Content-Type"))
	if ext == "" {
		ext = filepath.Ext(url)
	}
	if ext == "" {
		ext = ".mp3"
	}

	path := filepath.Join(songsDir, basename+ext)
	f, err := os.Create(path)
	if err != nil {
		return "", err
	}

	total := resp.ContentLength
	written := int64(0)
	buf := make([]byte, 32*1024)
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := f.Write(buf[:n]); werr != nil {
				f.Close()
				os.Remove(path)
				return "", werr
			}
			written += int64(n)
			if total > 0 {
				send(float64(written)/float64(total), "downloading...")
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			f.Close()
			os.Remove(path)
			return "", rerr
		}
	}
	f.Close()
	send(1.0, "done")
	return path, nil
}

func extFromContentType(ct string) string {
	switch {
	case strings.Contains(ct, "mpeg"):
		return ".mp3"
	case strings.Contains(ct, "ogg"):
		return ".ogg"
	case strings.Contains(ct, "wav"):
		return ".wav"
	case strings.Contains(ct, "flac"):
		return ".flac"
	}
	return ""
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func IsYouTube(url string) bool {
	return strings.Contains(url, "youtube.com") || strings.Contains(url, "youtu.be")
}

func IsYouTubePlaylist(url string) bool {
	return IsYouTube(url) && strings.Contains(url, "list=")
}

func IsSoundCloud(url string) bool {
	return strings.Contains(url, "soundcloud.com")
}

func isYouTube(url string) bool         { return IsYouTube(url) }
func isYouTubePlaylist(url string) bool { return IsYouTubePlaylist(url) }
func isSoundCloud(url string) bool      { return IsSoundCloud(url) }

// ── Search ────────────────────────────────────────────────────────────────────

// SearchResult holds metadata for one search hit from yt-dlp.
type SearchResult struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Artist string `json:"uploader"`
	URL    string `json:"webpage_url"`
}

// Search queries YouTube or SoundCloud via yt-dlp and returns metadata hits.
// platform should be "youtube" or "soundcloud".
// limit caps the number of results (default 10, max 20).
func Search(ctx context.Context, query, platform string, limit int) ([]SearchResult, error) {
	bin, err := findYtdlp()
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 10
	}
	if limit > 20 {
		limit = 20
	}

	prefix := "ytsearch"
	if platform == "soundcloud" {
		prefix = "scsearch"
	}
	searchArg := fmt.Sprintf("%s%d:%s", prefix, limit, query)

	cmd := exec.CommandContext(ctx, bin,
		"--flat-playlist",
		"--dump-json",
		"--no-check-formats",
		"--playlist-items", fmt.Sprintf("1:%d", limit),
		"--no-download",
		searchArg,
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("yt-dlp search: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("yt-dlp search: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("yt-dlp search: %w", err)
	}

	var results []SearchResult
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var r SearchResult
		if err := json.Unmarshal(line, &r); err != nil {
			jlog.Errorf("yt-dlp search: bad json line: %v", err)
			continue
		}
		results = append(results, r)
	}

	// Drain stderr so yt-dlp doesn't block.
	go io.Copy(io.Discard, stderr)

	if err := cmd.Wait(); err != nil {
		return results, fmt.Errorf("yt-dlp search failed: %w", err)
	}
	return results, nil
}

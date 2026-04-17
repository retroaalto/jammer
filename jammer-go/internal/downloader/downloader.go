package downloader

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jooapa/jammer/jammer-go/internal/playlist"
	yt "github.com/kkdai/youtube/v2"
)

// Progress is sent during a download to report completion in [0.0, 1.0].
type Progress struct {
	Done    bool
	Err     error
	Frac    float64 // 0..1
	Message string
}

// Download downloads the song at url into songsDir and returns the local path.
// progress receives updates; it is closed when the download finishes.
func Download(ctx context.Context, url, songsDir string, progress chan<- Progress) (string, error) {
	defer close(progress)

	if isSoundCloud(url) || isYouTubePlaylist(url) {
		return downloadViaYtdlp(ctx, url, songsDir, progress)
	}
	if isYouTube(url) {
		return downloadYouTube(ctx, url, songsDir, progress)
	}
	return downloadHTTP(ctx, url, songsDir, progress)
}

// ── YouTube ───────────────────────────────────────────────────────────────────

func downloadYouTube(ctx context.Context, url, songsDir string, progress chan<- Progress) (string, error) {
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
		return "", fmt.Errorf("youtube: %w", err)
	}

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
		return "", fmt.Errorf("youtube: no suitable format found")
	}

	send(0.05, "starting download...")
	stream, _, err := client.GetStreamContext(ctx, video, chosen)
	if err != nil {
		return "", fmt.Errorf("youtube stream: %w", err)
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
		return "", err
	}

	totalBytes := chosen.ContentLength
	written := int64(0)
	buf := make([]byte, 32*1024)
	for {
		n, rerr := stream.Read(buf)
		if n > 0 {
			if _, werr := tmpF.Write(buf[:n]); werr != nil {
				tmpF.Close()
				os.Remove(tmpPath)
				return "", werr
			}
			written += int64(n)
			if totalBytes > 0 {
				send(0.05+0.8*float64(written)/float64(totalBytes), "downloading...")
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			tmpF.Close()
			os.Remove(tmpPath)
			return "", rerr
		}
	}
	tmpF.Close()

	// Convert to ogg with ffmpeg if needed
	if ext != ".ogg" {
		send(0.85, "converting to ogg...")
		oggPath := filepath.Join(songsDir, basename+".ogg")
		cmd := exec.CommandContext(ctx, "ffmpeg", "-y", "-i", tmpPath, "-vn",
			"-c:a", "libvorbis", "-q:a", "5", oggPath)
		if out, err := cmd.CombinedOutput(); err != nil {
			os.Remove(tmpPath)
			return "", fmt.Errorf("ffmpeg: %w\n%s", err, out)
		}
		os.Remove(tmpPath)
		send(1.0, "done")
		return oggPath, nil
	}

	if err := os.Rename(tmpPath, finalPath); err != nil {
		return "", err
	}
	send(1.0, "done")
	return finalPath, nil
}

// ── SoundCloud / yt-dlp fallback ──────────────────────────────────────────────

func downloadViaYtdlp(ctx context.Context, url, songsDir string, progress chan<- Progress) (string, error) {
	send := func(frac float64, msg string) {
		select {
		case progress <- Progress{Frac: frac, Message: msg}:
		default:
		}
	}

	// Check yt-dlp is available
	ytdlpBin, err := findYtdlp()
	if err != nil {
		return "", err
	}

	basename := playlist.URLToExpectedBasename(url)
	outputTemplate := filepath.Join(songsDir, basename+".%(ext)s")

	send(0, "starting yt-dlp...")
	cmd := exec.CommandContext(ctx, ytdlpBin,
		"--no-playlist",
		"--extract-audio",
		"--audio-format", "mp3",
		"--audio-quality", "0",
		"--output", outputTemplate,
		url,
	)
	cmd.Stdout = os.Stdout // yt-dlp prints progress to stdout
	cmd.Stderr = os.Stderr

	// We can't easily hook into yt-dlp's progress inline, so just report indeterminate.
	done := make(chan error, 1)
	go func() { done <- cmd.Run() }()

	// Pulse progress while waiting
	frac := 0.0
	for {
		select {
		case err := <-done:
			if err != nil {
				return "", fmt.Errorf("yt-dlp: %w", err)
			}
			// Find the file yt-dlp wrote (basename + .mp3)
			path := filepath.Join(songsDir, basename+".mp3")
			if _, serr := os.Stat(path); serr == nil {
				send(1.0, "done")
				return path, nil
			}
			// Fallback: search for any file matching basename.*
			matches, _ := filepath.Glob(filepath.Join(songsDir, basename+".*"))
			for _, m := range matches {
				if !strings.HasSuffix(m, ".tmp") {
					send(1.0, "done")
					return m, nil
				}
			}
			return "", fmt.Errorf("yt-dlp: output file not found for %s", basename)
		case <-ctx.Done():
			return "", ctx.Err()
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

func isYouTube(url string) bool {
	return strings.Contains(url, "youtube.com") || strings.Contains(url, "youtu.be")
}

func isYouTubePlaylist(url string) bool {
	return isYouTube(url) && strings.Contains(url, "list=")
}

func isSoundCloud(url string) bool {
	return strings.Contains(url, "soundcloud.com")
}

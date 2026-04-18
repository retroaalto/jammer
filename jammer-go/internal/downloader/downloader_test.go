package downloader_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/jooapa/jammer/jammer-go/internal/downloader"
)

// ── URL classification ────────────────────────────────────────────────────────

func TestIsYouTube(t *testing.T) {
	cases := []struct {
		url  string
		want bool
	}{
		{"https://www.youtube.com/watch?v=abc", true},
		{"https://youtu.be/abc", true},
		{"https://soundcloud.com/a/b", false},
		{"https://example.com/file.mp3", false},
	}
	for _, c := range cases {
		got := downloader.IsYouTube(c.url)
		if got != c.want {
			t.Errorf("IsYouTube(%q) = %v, want %v", c.url, got, c.want)
		}
	}
}

func TestIsSoundCloud(t *testing.T) {
	cases := []struct {
		url  string
		want bool
	}{
		{"https://soundcloud.com/artist/track", true},
		{"https://www.youtube.com/watch?v=abc", false},
		{"https://example.com/file.mp3", false},
	}
	for _, c := range cases {
		got := downloader.IsSoundCloud(c.url)
		if got != c.want {
			t.Errorf("IsSoundCloud(%q) = %v, want %v", c.url, got, c.want)
		}
	}
}

func TestIsYouTubePlaylist(t *testing.T) {
	cases := []struct {
		url  string
		want bool
	}{
		{"https://www.youtube.com/watch?v=abc&list=PL123", true},
		{"https://www.youtube.com/watch?v=abc", false},
		{"https://soundcloud.com/a/b", false},
	}
	for _, c := range cases {
		got := downloader.IsYouTubePlaylist(c.url)
		if got != c.want {
			t.Errorf("IsYouTubePlaylist(%q) = %v, want %v", c.url, got, c.want)
		}
	}
}

// ── HTTP download ─────────────────────────────────────────────────────────────

func TestDownloadHTTP_MP3(t *testing.T) {
	fakeAudio := []byte("ID3fake-audio-content")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "audio/mpeg")
		w.Write(fakeAudio)
	}))
	defer srv.Close()

	songsDir := t.TempDir()
	progressCh := make(chan downloader.Progress, 32)

	path, _, err := downloader.Download(context.Background(), srv.URL+"/audio/test-track", songsDir, progressCh)
	if err != nil {
		t.Fatalf("Download failed: %v", err)
	}

	if filepath.Ext(path) != ".mp3" {
		t.Errorf("expected .mp3 extension, got %q", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("output file not found: %v", err)
	}
	if string(data) != string(fakeAudio) {
		t.Errorf("file content mismatch")
	}
}

func TestDownloadHTTP_ReportsProgress(t *testing.T) {
	fakeAudio := make([]byte, 64*1024) // 64 KB so progress ticks fire

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "audio/mpeg")
		w.Header().Set("Content-Length", "65536")
		w.Write(fakeAudio)
	}))
	defer srv.Close()

	songsDir := t.TempDir()
	progressCh := make(chan downloader.Progress, 128)

	_, _, err := downloader.Download(context.Background(), srv.URL+"/audio/progress-test", songsDir, progressCh)
	if err != nil {
		t.Fatalf("Download failed: %v", err)
	}

	// Drain and verify we got at least a final frac=1 progress
	var final downloader.Progress
	for p := range progressCh {
		final = p
	}
	if final.Frac != 1.0 {
		t.Errorf("last progress frac = %f, want 1.0", final.Frac)
	}
}

func TestDownloadHTTP_ContextCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Never respond fully
		w.Header().Set("Content-Type", "audio/mpeg")
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	progressCh := make(chan downloader.Progress, 8)
	_, _, err := downloader.Download(ctx, srv.URL+"/audio/slow", t.TempDir(), progressCh)
	if err == nil {
		t.Error("expected error on cancelled context, got nil")
	}
}

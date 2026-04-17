package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/jooapa/jammer/jammer-go/internal/audio"
	jlog "github.com/jooapa/jammer/jammer-go/internal/log"
	"github.com/jooapa/jammer/jammer-go/internal/player"
	"github.com/jooapa/jammer/jammer-go/internal/playlist"
	"github.com/jooapa/jammer/jammer-go/internal/ui"
)

func main() {
	if err := jlog.Init(); err != nil {
		fmt.Fprintln(os.Stderr, "warning: could not open log file:", err)
	}
	defer jlog.Close()
	jlog.Info("jammer-go starting up")

	exeDir, err := execDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "cannot determine executable dir:", err)
		os.Exit(1)
	}

	libPath := findLib(exeDir, "libbass.so")
	if libPath == "" {
		jlog.Error("libbass.so not found")
		fmt.Fprintln(os.Stderr, "libbass.so not found — set LD_LIBRARY_PATH or place the lib next to the binary")
		os.Exit(1)
	}
	jlog.Infof("loading BASS from %s", libPath)

	if err := audio.Load(libPath); err != nil {
		jlog.Errorf("audio.Load: %v", err)
		fmt.Fprintln(os.Stderr, "audio load error:", err)
		os.Exit(1)
	}
	if err := audio.Init(); err != nil {
		jlog.Errorf("audio.Init: %v", err)
		fmt.Fprintln(os.Stderr, "audio init error:", err)
		os.Exit(1)
	}
	defer audio.Free()

	songsDir := jammerDir("songs")
	plsDir := jammerDir("playlists")

	// Ensure directories exist.
	for _, d := range []string{songsDir, plsDir} {
		if err := os.MkdirAll(d, 0755); err != nil {
			fmt.Fprintln(os.Stderr, "cannot create directory:", d, err)
			os.Exit(1)
		}
	}

	p := player.New()

	// Parse flags.
	playlistFlag := ""
	for i, arg := range os.Args[1:] {
		if arg == "-p" && i+1 < len(os.Args[1:]) {
			playlistFlag = os.Args[i+2]
		}
	}

	if playlistFlag != "" {
		// Resolve playlist file: try exact name, then with .jammer extension.
		resolved := resolvePlaylist(plsDir, playlistFlag)
		if resolved == "" {
			fmt.Fprintf(os.Stderr, "playlist %q not found in %s\n", playlistFlag, plsDir)
			os.Exit(1)
		}
		jlog.Infof("-p: loading playlist %q", resolved)
		entries, err := playlist.Load(filepath.Join(plsDir, resolved), songsDir)
		if err != nil {
			fmt.Fprintln(os.Stderr, "failed to load playlist:", err)
			os.Exit(1)
		}
		p.LoadPlaylist(entries)
		jlog.Infof("-p: loaded %d songs", len(p.Songs()))
		// Start playing the first song immediately.
		if len(p.Songs()) > 0 {
			if err := p.PlayIndex(0); err != nil {
				jlog.Errorf("-p: PlayIndex(0) failed: %v", err)
			}
		}
	} else {
		jlog.Infof("loading songs from %s", songsDir)
		if err := p.LoadDir(songsDir); err != nil {
			jlog.Errorf("LoadDir: %v", err)
			fmt.Fprintln(os.Stderr, "failed to load songs:", err)
			os.Exit(1)
		}
		jlog.Infof("loaded %d songs", len(p.Songs()))
	}

	go p.WatchEnd()

	m := ui.New(p, songsDir, plsDir)
	prog := tea.NewProgram(m)
	if _, err := prog.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "TUI error:", err)
		os.Exit(1)
	}
}

func execDir() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Dir(exe), nil
}

func findLib(exeDir, name string) string {
	candidates := []string{
		filepath.Join(exeDir, name),
	}
	if runtime.GOOS == "linux" {
		candidates = append(candidates,
			filepath.Join(exeDir, "..", "libs", "linux", "x86_64", name),
			repoLibPath(name),
		)
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			abs, _ := filepath.Abs(c)
			return abs
		}
	}
	return ""
}

func repoLibPath(name string) string {
	cwd, _ := os.Getwd()
	return filepath.Join(cwd, "..", "libs", "linux", "x86_64", name)
}

// resolvePlaylist finds a playlist filename in plsDir matching name.
// Tries: exact match, then name+".jammer", then case-insensitive prefix.
func resolvePlaylist(plsDir, name string) string {
	entries, err := os.ReadDir(plsDir)
	if err != nil {
		return ""
	}
	lower := strings.ToLower(name)
	// Pass 1: exact filename or exact name + .jammer
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		n := e.Name()
		if n == name || n == name+".jammer" || n == name+".m3u" || n == name+".m3u8" {
			return n
		}
	}
	// Pass 2: case-insensitive
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		n := e.Name()
		base := strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(n, ".jammer"), ".m3u8"), ".m3u")
		if strings.ToLower(base) == lower || strings.ToLower(n) == lower {
			return n
		}
	}
	return ""
}

func jammerDir(sub string) string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "jammer", sub)
}

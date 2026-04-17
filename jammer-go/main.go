package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	tea "charm.land/bubbletea/v2"
	"github.com/jooapa/jammer/jammer-go/internal/audio"
	"github.com/jooapa/jammer/jammer-go/internal/player"
	"github.com/jooapa/jammer/jammer-go/internal/ui"
)

func main() {
	// Determine paths relative to the executable (or repo root during dev).
	exeDir, err := execDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "cannot determine executable dir:", err)
		os.Exit(1)
	}

	// Load libbass.so — check next to binary first, then repo libs/
	libPath := findLib(exeDir, "libbass.so")
	if libPath == "" {
		fmt.Fprintln(os.Stderr, "libbass.so not found — set LD_LIBRARY_PATH or place the lib next to the binary")
		os.Exit(1)
	}

	if err := audio.Load(libPath); err != nil {
		fmt.Fprintln(os.Stderr, "audio load error:", err)
		os.Exit(1)
	}
	if err := audio.Init(); err != nil {
		fmt.Fprintln(os.Stderr, "audio init error:", err)
		os.Exit(1)
	}
	defer audio.Free()

	// Songs directory: ~/.config/Jammer/songs  (same as .NET app)
	songsDir := songsDirectory()
	if _, err := os.Stat(songsDir); os.IsNotExist(err) {
		fmt.Fprintln(os.Stderr, "songs directory not found:", songsDir)
		os.Exit(1)
	}

	p := player.New()
	if err := p.LoadDir(songsDir); err != nil {
		fmt.Fprintln(os.Stderr, "failed to load songs:", err)
		os.Exit(1)
	}
	if len(p.Songs()) == 0 {
		fmt.Fprintln(os.Stderr, "no supported audio files found in", songsDir)
		os.Exit(1)
	}

	// Auto-advance when track ends.
	go p.WatchEnd()

	m := ui.New(p)
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
	// During development, locate repo libs/ relative to this file.
	if runtime.GOOS == "linux" {
		// Walk up from jammer-go/ to repo root.
		candidates = append(candidates,
			filepath.Join(exeDir, "..", "libs", "linux", "x86_64", name),
			// When run with `go run` or `go build` from jammer-go/
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

// repoLibPath resolves libs/linux/x86_64/<name> relative to the source file.
func repoLibPath(name string) string {
	// __FILE__ equivalent: use a known relative path from the module root.
	// The module is at jammer-go/, repo root is one level up.
	cwd, _ := os.Getwd()
	return filepath.Join(cwd, "..", "libs", "linux", "x86_64", name)
}

func songsDirectory() string {
	home, _ := os.UserHomeDir()
	// Match the .NET app's songs location.
	return filepath.Join(home, "jammer", "songs")
}

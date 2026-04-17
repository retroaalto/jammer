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
	exeDir, err := execDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "cannot determine executable dir:", err)
		os.Exit(1)
	}

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
	if err := p.LoadDir(songsDir); err != nil {
		fmt.Fprintln(os.Stderr, "failed to load songs:", err)
		os.Exit(1)
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

func jammerDir(sub string) string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "jammer", sub)
}

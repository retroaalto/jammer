package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/jooapa/jammer/jammer-go/internal/audio"
	"github.com/jooapa/jammer/jammer-go/internal/keybinds"
	jlog "github.com/jooapa/jammer/jammer-go/internal/log"
	"github.com/jooapa/jammer/jammer-go/internal/player"
	"github.com/jooapa/jammer/jammer-go/internal/ui"
)

// settings mirrors the fields of settings.json that main.go cares about.
// Unknown fields are preserved via the raw map when saving.
type settings struct {
	BackEndType               int     `json:"backEndType"`
	SeekStep                  int     `json:"seekStep"`
	LoopType                  int     `json:"LoopType"`
	DefaultView               string  `json:"defaultView"`
	ForwardSeconds            int     `json:"forwardSeconds"`
	RewindSeconds             int     `json:"rewindSeconds"`
	ChangeVolumeBy            float64 `json:"changeVolumeBy"`
	IsAutoSave                bool    `json:"isAutoSave"`
	IsMediaButtons            bool    `json:"isMediaButtons"`
	IsVisualizer              bool    `json:"isVisualizer"`
	ClientID                  string  `json:"clientID"`
	ModifierKeyHelper         bool    `json:"modifierKeyHelper"`
	IsIgnoreErrors            bool    `json:"isIgnoreErrors"`
	ShowPlaylistPosition      bool    `json:"showPlaylistPosition"`
	RssSkipAfterTime          bool    `json:"rssSkipAfterTime"`
	RssSkipAfterTimeValue     int     `json:"rssSkipAfterTimeValue"`
	EnableQuickSearch         bool    `json:"EnableQuickSearch"`
	FavoriteExplainer         bool    `json:"favoriteExplainer"`
	EnableQuickPlayFromSearch bool    `json:"EnableQuickPlayFromSearch"`
	ShowTitle                 bool    `json:"showTitle"`
	TitleText                 string  `json:"titleText"`
	TitleAnimationSpeed       int     `json:"titleAnimationSpeed"`
	TitleAnimationInterval    int     `json:"titleAnimationInterval"`
}

const defaultSeekStep = 2

func stripBOM(data []byte) []byte {
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		return data[3:]
	}
	return data
}

func loadSettings(path string) settings {
	var s settings
	data, err := os.ReadFile(path)
	if err != nil {
		return s
	}
	_ = json.Unmarshal(stripBOM(data), &s)
	return s
}

func saveBackend(path string, backendType int) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var raw map[string]any
	if err := json.Unmarshal(stripBOM(data), &raw); err != nil {
		return
	}
	raw["backEndType"] = backendType
	out, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(path, out, 0o644)
}

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

	settingsPath := filepath.Join(jammerDir(""), "settings.json")
	cfg := loadSettings(settingsPath)

	// Parse flags: -p <playlist>  -b (use BASS backend)
	playlistFlag := ""
	bassFlag := false
	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-p":
			if i+1 < len(args) {
				i++
				playlistFlag = args[i]
			}
		case "-b":
			bassFlag = true
		}
	}

	// Determine backend: -b flag overrides settings.json for this session only.
	useBass := cfg.BackEndType == 1 || bassFlag

	var backend audio.Backend
	if useBass {
		libPath := findLib(exeDir, "libbass.so")
		if libPath == "" {
			fmt.Fprintln(os.Stderr, "BASS backend requested but libbass.so not found — set LD_LIBRARY_PATH or place the lib next to the binary")
			os.Exit(1)
		}
		jlog.Infof("loading BASS from %s", libPath)
		b, err := audio.LoadBass(libPath)
		if err != nil {
			jlog.Errorf("audio.LoadBass: %v", err)
			fmt.Fprintln(os.Stderr, "audio load error:", err)
			os.Exit(1)
		}
		backend = b
		jlog.Info("audio backend: BASS")
	} else {
		backend = audio.NewBeepBackend()
		jlog.Info("audio backend: beep (default)")
	}

	if err := backend.Init(); err != nil {
		jlog.Errorf("backend.Init: %v", err)
		fmt.Fprintln(os.Stderr, "audio init error:", err)
		os.Exit(1)
	}
	defer backend.Free()

	songsDir := jammerDir("songs")
	plsDir := jammerDir("playlists")

	// Ensure directories exist.
	for _, d := range []string{songsDir, plsDir} {
		if err := os.MkdirAll(d, 0755); err != nil {
			fmt.Fprintln(os.Stderr, "cannot create directory:", d, err)
			os.Exit(1)
		}
	}

	p := player.New(backend)
	p.SetLoopMode(player.LoopMode(cfg.LoopType))

	resolvedPlaylist := ""
	if playlistFlag != "" {
		resolved := resolvePlaylist(plsDir, playlistFlag)
		if resolved == "" {
			fmt.Fprintf(os.Stderr, "playlist %q not found in %s\n", playlistFlag, plsDir)
			os.Exit(1)
		}
		resolvedPlaylist = filepath.Join(plsDir, resolved)
		jlog.Infof("-p: will load playlist %q via UI", resolvedPlaylist)
	}

	if playlistFlag == "" {
		jlog.Infof("loading songs from %s", songsDir)
		if err := p.LoadDir(songsDir); err != nil {
			jlog.Errorf("LoadDir: %v", err)
			fmt.Fprintln(os.Stderr, "failed to load songs:", err)
			os.Exit(1)
		}
		jlog.Infof("loaded %d songs", len(p.Songs()))
	}

	go p.WatchEnd()

	seekStep := cfg.SeekStep
	if seekStep <= 0 {
		seekStep = defaultSeekStep
	}

	// Load keybindings from KeyData.ini
	kb := keybinds.New()

	// Build Prefs from settings for the UI.
	prefs := ui.Prefs{
		SettingsPath:              settingsPath,
		ForwardSeconds:            cfg.ForwardSeconds,
		RewindSeconds:             cfg.RewindSeconds,
		ChangeVolumeBy:            cfg.ChangeVolumeBy,
		IsAutoSave:                cfg.IsAutoSave,
		IsMediaButtons:            cfg.IsMediaButtons,
		IsVisualizer:              cfg.IsVisualizer,
		ClientID:                  cfg.ClientID,
		ModifierKeyHelper:         cfg.ModifierKeyHelper,
		IsIgnoreErrors:            cfg.IsIgnoreErrors,
		ShowPlaylistPosition:      cfg.ShowPlaylistPosition,
		RssSkipAfterTime:          cfg.RssSkipAfterTime,
		RssSkipAfterTimeValue:     cfg.RssSkipAfterTimeValue,
		EnableQuickSearch:         cfg.EnableQuickSearch,
		FavoriteExplainer:         cfg.FavoriteExplainer,
		EnableQuickPlayFromSearch: cfg.EnableQuickPlayFromSearch,
		ShowTitle:                 cfg.ShowTitle,
		TitleText:                 cfg.TitleText,
		TitleAnimationSpeed:       cfg.TitleAnimationSpeed,
		TitleAnimationInterval:    cfg.TitleAnimationInterval,
	}

	var m ui.Model
	if resolvedPlaylist != "" {
		m = ui.NewWithPlaylist(p, songsDir, plsDir, resolvedPlaylist, seekStep, cfg.DefaultView, kb, prefs)
	} else {
		m = ui.New(p, songsDir, plsDir, seekStep, cfg.DefaultView, kb, prefs)
	}
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
	if sub == "" {
		return filepath.Join(home, "jammer")
	}
	return filepath.Join(home, "jammer", sub)
}

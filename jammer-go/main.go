package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/jooapa/jammer/jammer-go/internal/audio"
	"github.com/jooapa/jammer/jammer-go/internal/dirs"
	"github.com/jooapa/jammer/jammer-go/internal/keybinds"
	"github.com/jooapa/jammer/jammer-go/internal/locale"
	jlog "github.com/jooapa/jammer/jammer-go/internal/log"
	"github.com/jooapa/jammer/jammer-go/internal/player"
	"github.com/jooapa/jammer/jammer-go/internal/playlist"
	"github.com/jooapa/jammer/jammer-go/internal/theme"
	"github.com/jooapa/jammer/jammer-go/internal/ui"
)

const version = "0.1.0"

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
	SearchResultCount         int     `json:"searchResultCount"`
	ShowTitle                 bool    `json:"showTitle"`
	TitleText                 string  `json:"titleText"`
	TitleAnimation            string  `json:"titleAnimation"`
	TitleAnimationSpeed       int     `json:"titleAnimationSpeed"`
	TitleAnimationInterval    int     `json:"titleAnimationInterval"`
	Theme                     string  `json:"theme"`
	Language                  string  `json:"language"`
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
		s.ShowTitle = true
		return s
	}
	data = stripBOM(data)
	_ = json.Unmarshal(data, &s)
	// Default showTitle to true when the key is missing from settings.json.
	if !s.ShowTitle {
		var raw map[string]any
		if err := json.Unmarshal(data, &raw); err == nil {
			if _, ok := raw["showTitle"]; !ok {
				s.ShowTitle = true
			}
		}
	}
	if s.TitleAnimation == "" {
		s.TitleAnimation = "kitt"
	}
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
	// Handle --xdg before anything that touches dirs (including jlog.Init).
	for _, arg := range os.Args[1:] {
		if arg == "--xdg" {
			dirs.ForceXDG()
			break
		}
	}

	if err := jlog.Init(); err != nil {
		fmt.Fprintln(os.Stderr, "warning: could not open log file:", err)
	}
	defer jlog.Close()
	jlog.Info("jammer-go starting up")
	if dirs.IsLegacy() {
		jlog.Infof("config location: legacy (%s)", dirs.Config())
	} else {
		jlog.Infof("config location: XDG (data=%s config=%s state=%s cache=%s)",
			dirs.Data(), dirs.Config(), dirs.State(), dirs.Cache())
	}

	exeDir, err := execDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "cannot determine executable dir:", err)
		os.Exit(1)
	}

	settingsPath := filepath.Join(dirs.Config(), "settings.json")
	cfg := loadSettings(settingsPath)

	// Load locale (falls back to English if language is empty or unknown).
	if err := locale.Load(cfg.Language); err != nil {
		jlog.Infof("locale.Load(%q): %v — falling back to English", cfg.Language, err)
	}
	jlog.Infof("locale: %s", locale.CurrentLang())

	// ── CLI-only commands (no TUI, exit after running) ──────────────────────
	args := make([]string, 0, len(os.Args)-1)
	for _, a := range os.Args[1:] {
		if a != "--xdg" {
			args = append(args, a)
		}
	}
	plsDir0 := filepath.Join(dirs.Data(), "playlists")
	songsDir0 := filepath.Join(dirs.Data(), "songs")
	for _, d := range []string{plsDir0, songsDir0} {
		_ = os.MkdirAll(d, 0755)
	}

	if len(args) > 0 {
		switch args[0] {
		case "-v", "--version":
			fmt.Println("jammer-go", version)
			os.Exit(0)

		case "-l":
			// List all playlists.
			names, err := playlist.List(plsDir0)
			if err != nil {
				fmt.Fprintln(os.Stderr, "error listing playlists:", err)
				os.Exit(1)
			}
			for _, n := range names {
				fmt.Println(n)
			}
			os.Exit(0)

		case "-c":
			// Create an empty playlist.
			if len(args) < 2 {
				fmt.Fprintln(os.Stderr, "usage: jammer -c <name>")
				os.Exit(1)
			}
			name := args[1]
			path := playlistPath(plsDir0, name)
			if _, err := os.Stat(path); err == nil {
				fmt.Fprintf(os.Stderr, "playlist %q already exists\n", name)
				os.Exit(1)
			}
			if err := playlist.Save(path, nil); err != nil {
				fmt.Fprintln(os.Stderr, "error creating playlist:", err)
				os.Exit(1)
			}
			fmt.Printf("created playlist %q\n", name)
			os.Exit(0)

		case "-d":
			// Delete a playlist.
			if len(args) < 2 {
				fmt.Fprintln(os.Stderr, "usage: jammer -d <name>")
				os.Exit(1)
			}
			resolved := resolvePlaylist(plsDir0, args[1])
			if resolved == "" {
				fmt.Fprintf(os.Stderr, "playlist %q not found\n", args[1])
				os.Exit(1)
			}
			if err := os.Remove(filepath.Join(plsDir0, resolved)); err != nil {
				fmt.Fprintln(os.Stderr, "error deleting playlist:", err)
				os.Exit(1)
			}
			fmt.Printf("deleted playlist %q\n", resolved)
			os.Exit(0)

		case "-a":
			// Add songs to a playlist.
			if len(args) < 3 {
				fmt.Fprintln(os.Stderr, "usage: jammer -a <name> <song> [song...]")
				os.Exit(1)
			}
			resolved := resolvePlaylist(plsDir0, args[1])
			if resolved == "" {
				fmt.Fprintf(os.Stderr, "playlist %q not found\n", args[1])
				os.Exit(1)
			}
			path := filepath.Join(plsDir0, resolved)
			entries, _, err := playlist.Load(path, songsDir0)
			if err != nil {
				fmt.Fprintln(os.Stderr, "error loading playlist:", err)
				os.Exit(1)
			}
			for _, song := range args[2:] {
				entries = append(entries, playlist.Entry{URL: song})
			}
			if err := playlist.Save(path, entries); err != nil {
				fmt.Fprintln(os.Stderr, "error saving playlist:", err)
				os.Exit(1)
			}
			fmt.Printf("added %d song(s) to %q\n", len(args)-2, resolved)
			os.Exit(0)

		case "-r":
			// Remove a song from a playlist.
			if len(args) < 3 {
				fmt.Fprintln(os.Stderr, "usage: jammer -r <name> <song>")
				os.Exit(1)
			}
			resolved := resolvePlaylist(plsDir0, args[1])
			if resolved == "" {
				fmt.Fprintf(os.Stderr, "playlist %q not found\n", args[1])
				os.Exit(1)
			}
			path := filepath.Join(plsDir0, resolved)
			entries, _, err := playlist.Load(path, songsDir0)
			if err != nil {
				fmt.Fprintln(os.Stderr, "error loading playlist:", err)
				os.Exit(1)
			}
			target := args[2]
			newEntries := entries[:0]
			removed := 0
			for _, e := range entries {
				if e.URL == target || e.Path == target || e.Title == target {
					removed++
				} else {
					newEntries = append(newEntries, e)
				}
			}
			if removed == 0 {
				fmt.Fprintf(os.Stderr, "song %q not found in playlist\n", target)
				os.Exit(1)
			}
			if err := playlist.Save(path, newEntries); err != nil {
				fmt.Fprintln(os.Stderr, "error saving playlist:", err)
				os.Exit(1)
			}
			fmt.Printf("removed %d song(s) from %q\n", removed, resolved)
			os.Exit(0)

		case "-s":
			// Show songs in a playlist.
			if len(args) < 2 {
				fmt.Fprintln(os.Stderr, "usage: jammer -s <name>")
				os.Exit(1)
			}
			resolved := resolvePlaylist(plsDir0, args[1])
			if resolved == "" {
				fmt.Fprintf(os.Stderr, "playlist %q not found\n", args[1])
				os.Exit(1)
			}
			entries, _, err := playlist.Load(filepath.Join(plsDir0, resolved), songsDir0)
			if err != nil {
				fmt.Fprintln(os.Stderr, "error loading playlist:", err)
				os.Exit(1)
			}
			fmt.Printf("playlist: %s (%d songs)\n", resolved, len(entries))
			for i, e := range entries {
				fmt.Printf("  %d. %s\n", i+1, e.DisplayTitle())
			}
			os.Exit(0)

		case "-f", "--flush":
			fmt.Fprintln(os.Stderr, "warning: -f/--flush is deprecated and has no effect")
			os.Exit(0)
		}
	}

	// Parse flags: -p <playlist>  -b (use BASS backend)  -hm/--home (songs folder)  --pprof [addr]
	playlistFlag := ""
	bassFlag := false
	pprofAddr := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-p":
			if i+1 < len(args) {
				i++
				playlistFlag = args[i]
			}
		case "-b":
			bassFlag = true
		case "-hm", "--home":
			// Explicitly play songs folder (default behaviour).
			playlistFlag = ""
		case "--pprof":
			// Optional address argument; default to localhost:6060.
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				i++
				pprofAddr = args[i]
			} else {
				pprofAddr = "127.0.0.1:6060"
			}
		}
	}

	// Start pprof HTTP server if requested.
	if pprofAddr != "" {
		go func() {
			jlog.Infof("pprof server listening on http://%s/debug/pprof/", pprofAddr)
			if err := http.ListenAndServe(pprofAddr, nil); err != nil {
				jlog.Errorf("pprof server: %v", err)
			}
		}()
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

	songsDir := filepath.Join(dirs.Data(), "songs")
	plsDir := filepath.Join(dirs.Data(), "playlists")

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

	// Tell the theme package where to find custom .json theme files.
	theme.SetThemesDir(filepath.Join(dirs.Data(), "themes"))

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
		SearchResultCount:         cfg.SearchResultCount,
		ShowTitle:                 cfg.ShowTitle,
		TitleText:                 cfg.TitleText,
		TitleAnimationRaw:         cfg.TitleAnimation,
		TitleAnimation:            ui.ResolveAnimation(cfg.TitleAnimation),
		TitleAnimationSpeed:       cfg.TitleAnimationSpeed,
		TitleAnimationInterval:    cfg.TitleAnimationInterval,
		Theme:                     cfg.Theme,
		Language:                  cfg.Language,
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

// playlistPath returns the full path for a new playlist with the given name,
// appending ".jammer" if no extension is present.
func playlistPath(plsDir, name string) string {
	if filepath.Ext(name) == "" {
		name += ".jammer"
	}
	return filepath.Join(plsDir, name)
}

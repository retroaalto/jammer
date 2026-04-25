package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/lipgloss"
	"github.com/jooapa/jammer/jammer-go/internal/dirs"
	"github.com/jooapa/jammer/jammer-go/internal/downloader"
	"github.com/jooapa/jammer/jammer-go/internal/keybinds"
	jlog "github.com/jooapa/jammer/jammer-go/internal/log"
	"github.com/jooapa/jammer/jammer-go/internal/player"
	"github.com/jooapa/jammer/jammer-go/internal/playlist"
	"github.com/jooapa/jammer/jammer-go/internal/rss"
	"github.com/jooapa/jammer/jammer-go/internal/tags"
	"github.com/jooapa/jammer/jammer-go/internal/theme"
)

// ── Messages ──────────────────────────────────────────────────────────────────

type tickMsg time.Time
type vizTickMsg time.Time
type titleTickMsg time.Time

type downloadProgressMsg struct {
	index      int
	prog       downloader.Progress
	progressCh <-chan downloader.Progress
	doneCh     <-chan downloadDoneMsg
}

type downloadDoneMsg struct {
	index int
	path  string
	meta  downloader.Meta
	err   error
}

type searchDoneMsg struct {
	results []downloader.SearchResult
	err     error
}

type rssFetchDoneMsg struct {
	feed *rss.Feed
	err  error
}

// ── Views ─────────────────────────────────────────────────────────────────────

type viewKind int

const (
	viewDefault       viewKind = iota // 3-song snippet (prev/current/next)
	viewAll                           // full scrollable list
	viewPlaylists                     // playlist browser
	viewHelp                          // help screen
	viewSettings                      // settings screen
	viewSettingsInput                 // text input for a settings value
	viewRename                        // rename song input
	viewInfo                          // song info overlay
	viewAddSong                       // add song input
	viewPlaySong                      // play arbitrary path/URL input (Phase 2 #4)
	viewSaveAs                        // save playlist as new name (Phase 2 #8)
	viewLog                           // scrollable log viewer (Phase 2 #12)
	viewShowSongs                     // read-only song list from another playlist (Phase 2 #11)
	viewSearchQuery                   // online search query input (Phase 2 #5)
	viewSearchResults                 // online search results list (Phase 2 #5)
	viewEditKeybinds                  // edit keybindings view (Phase 3 #14)
	viewChangeTheme                   // theme picker view (Phase 3 #16)
	viewRssFeed                       // RSS feed episode list (Phase 3 #18)
)

// ── Download state per song ───────────────────────────────────────────────────

type dlState struct {
	active  bool
	frac    float64
	message string
	err     error
}

// ── Styles ────────────────────────────────────────────────────────────────────

// activePalette holds the currently applied theme palette so render functions
// can access border colors, glyphs, and per-section colors.
var activePalette theme.Palette

var (
	styleTitle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("212")).
			Bold(true)

	styleSelected = lipgloss.NewStyle().
			Foreground(lipgloss.Color("212")).
			Background(lipgloss.Color("57")).
			Bold(true)

	styleNormal = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	stylePlaying = lipgloss.NewStyle().
			Foreground(lipgloss.Color("46")).
			Bold(true)

	styleHelp = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	styleBar = lipgloss.NewStyle().
			Foreground(lipgloss.Color("63"))

	styleBarFill = lipgloss.NewStyle().
			Foreground(lipgloss.Color("212"))

	styleVolume = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214"))

	styleNotDL = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	styleDLing = lipgloss.NewStyle().
			Foreground(lipgloss.Color("220"))

	styleErr = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	styleTabActive = lipgloss.NewStyle().
			Foreground(lipgloss.Color("212")).
			Bold(true).
			Underline(true)

	styleTabInactive = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	// ── Help screen ──────────────────────────────────────────────────────
	styleHelpHeader  = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)
	styleHelpControl = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	styleHelpDesc    = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))

	// ── Settings screen ───────────────────────────────────────────────────
	styleSettingsHeader = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)
	styleSettingsName   = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	styleSettingsValue  = lipgloss.NewStyle().Foreground(lipgloss.Color("46"))
	styleSettingsHint   = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	// ── Edit Keybinds screen ──────────────────────────────────────────────
	styleKeybindsHeader  = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)
	styleKeybindsDesc    = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	styleKeybindsControl = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	styleKeybindsCurrent = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	styleKeybindsEntered = lipgloss.NewStyle().Foreground(lipgloss.Color("46"))

	// ── RSS feed ──────────────────────────────────────────────────────────
	styleRssTitle  = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)
	styleRssNormal = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))

	// ── Input / modal ──────────────────────────────────────────────────────
	styleInputTitle    = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)
	styleInputText     = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	styleInputTitleErr = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	styleInputTextErr  = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
)

// applyTheme rebuilds all global style variables from the given palette.
func applyTheme(p theme.Palette) {
	activePalette = p

	// WholePlaylist / song list colors
	styleTitle = lipgloss.NewStyle().Foreground(p.PlaylistTitle).Bold(true)
	styleSelected = lipgloss.NewStyle().Foreground(p.WholePlaylistChoosing).Background(p.PlaylistSelectedBg).Bold(true)
	styleNormal = lipgloss.NewStyle().Foreground(p.WholePlaylistNormal)
	stylePlaying = lipgloss.NewStyle().Foreground(p.WholePlaylistCurrent).Bold(true)
	styleHelp = lipgloss.NewStyle().Foreground(p.PlaylistHelp)
	styleBar = lipgloss.NewStyle().Foreground(p.TimebarColor)
	styleBarFill = lipgloss.NewStyle().Foreground(p.TimebarFill)
	styleVolume = lipgloss.NewStyle().Foreground(p.VolumeColor)
	styleNotDL = lipgloss.NewStyle().Foreground(p.PlaylistNotDL)
	styleDLing = lipgloss.NewStyle().Foreground(p.PlaylistDownloading)
	styleErr = lipgloss.NewStyle().Foreground(p.PlaylistError)
	styleTabActive = lipgloss.NewStyle().Foreground(p.TabActive).Bold(true).Underline(true)
	styleTabInactive = lipgloss.NewStyle().Foreground(p.TabInactive)

	// Help screen
	styleHelpHeader = lipgloss.NewStyle().Foreground(p.HelpHeader).Bold(true)
	styleHelpControl = lipgloss.NewStyle().Foreground(p.HelpControl)
	styleHelpDesc = lipgloss.NewStyle().Foreground(p.HelpDescription)

	// Settings screen
	styleSettingsHeader = lipgloss.NewStyle().Foreground(p.SettingsHeader).Bold(true)
	styleSettingsName = lipgloss.NewStyle().Foreground(p.SettingsName)
	styleSettingsValue = lipgloss.NewStyle().Foreground(p.SettingsValue)
	styleSettingsHint = lipgloss.NewStyle().Foreground(p.SettingsChangeHint)

	// Edit Keybinds screen
	styleKeybindsHeader = lipgloss.NewStyle().Foreground(p.KeybindsHeader).Bold(true)
	styleKeybindsDesc = lipgloss.NewStyle().Foreground(p.KeybindsDescription)
	styleKeybindsControl = lipgloss.NewStyle().Foreground(p.KeybindsControl)
	styleKeybindsCurrent = lipgloss.NewStyle().Foreground(p.KeybindsCurrentKey)
	styleKeybindsEntered = lipgloss.NewStyle().Foreground(p.KeybindsEnteredKey)

	// RSS feed
	styleRssTitle = lipgloss.NewStyle().Foreground(p.RssTitle).Bold(true)
	styleRssNormal = lipgloss.NewStyle().Foreground(p.RssDescription)

	// Input / modals
	styleInputTitle = lipgloss.NewStyle().Foreground(p.InputTitle).Bold(true)
	styleInputText = lipgloss.NewStyle().Foreground(p.InputText)
	styleInputTitleErr = lipgloss.NewStyle().Foreground(p.InputTitleError).Bold(true)
	styleInputTextErr = lipgloss.NewStyle().Foreground(p.InputTextError)

	rebuildVizStyles()
}

// vizPlayingStyle / vizPausedStyle are updated by rebuildVizStyles() whenever
// the theme changes. They replace the old hardcoded gradient palette.
var vizPlayingStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
var vizPausedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

// rebuildVizStyles updates the visualizer styles from the active palette.
func rebuildVizStyles() {
	vizPlayingStyle = lipgloss.NewStyle().Foreground(activePalette.VizPlayingColor)
	vizPausedStyle = lipgloss.NewStyle().Foreground(activePalette.VizPausedColor)
}

// ── Visualizer config ─────────────────────────────────────────────────────────

// VizConfig holds the settings read from Visualizer.ini.
// All fields have built-in defaults that are used when the file is absent.
type VizConfig struct {
	RefreshTime          int     // tick interval in ms
	MinFrequency         float64 // Hz — left edge of bar display
	MaxFrequency         float64 // Hz — right edge of bar display
	FrequencyMultiplier  float64 // linear multiplier inside log10 step
	LogarithmicMultiplier float64 // power exponent applied to raw FFT values
	PausingEffect        bool    // decay bars when paused
}

// defaultVizConfig returns the built-in defaults (matches Visualizer.ini defaults).
func defaultVizConfig() VizConfig {
	return VizConfig{
		RefreshTime:          35,
		MinFrequency:         50,
		MaxFrequency:         17000,
		FrequencyMultiplier:  900000000,
		LogarithmicMultiplier: 4,
		PausingEffect:        true,
	}
}

// loadVizConfig reads Visualizer.ini from path and returns a VizConfig.
// Any missing or unparseable key falls back to the built-in default.
// If the file does not exist the full default config is returned silently.
func loadVizConfig(path string) VizConfig {
	cfg := defaultVizConfig()
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg
	}
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "[") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		// Strip inline comments.
		if idx := strings.Index(val, ";"); idx >= 0 {
			val = strings.TrimSpace(val[:idx])
		}
		switch key {
		case "RefreshTime":
			if v, err := strconv.Atoi(val); err == nil && v > 0 {
				cfg.RefreshTime = v
			}
		case "MinFrequency":
			if v, err := strconv.ParseFloat(val, 64); err == nil && v > 0 {
				cfg.MinFrequency = v
			}
		case "MaxFrequency":
			if v, err := strconv.ParseFloat(val, 64); err == nil && v > 0 {
				cfg.MaxFrequency = v
			}
		case "FrequencyMultiplier":
			if v, err := strconv.ParseFloat(val, 64); err == nil && v > 0 {
				cfg.FrequencyMultiplier = v
			}
		case "LogarithmicMultiplier":
			if v, err := strconv.ParseFloat(val, 64); err == nil && v > 0 {
				cfg.LogarithmicMultiplier = v
			}
		case "PausingEffect":
			cfg.PausingEffect = strings.ToLower(val) == "true"
		}
	}
	jlog.Infof("vizconfig: loaded path=%q minFreq=%.0f maxFreq=%.0f logMult=%.2f freqMult=%.0f pausingEffect=%v",
		path, cfg.MinFrequency, cfg.MaxFrequency, cfg.LogarithmicMultiplier, cfg.FrequencyMultiplier, cfg.PausingEffect)
	return cfg
}

// ── Model ─────────────────────────────────────────────────────────────────────

// Prefs holds all user-configurable settings mirrored from settings.json.
type Prefs struct {
	SettingsPath              string
	ForwardSeconds            int
	RewindSeconds             int
	ChangeVolumeBy            float64
	IsAutoSave                bool
	IsMediaButtons            bool
	IsVisualizer              bool
	ClientID                  string
	ModifierKeyHelper         bool
	IsIgnoreErrors            bool
	ShowPlaylistPosition      bool
	RssSkipAfterTime          bool
	RssSkipAfterTimeValue     int
	EnableQuickSearch         bool
	FavoriteExplainer         bool
	EnableQuickPlayFromSearch bool
	SearchResultCount         int
	ShowTitle                 bool
	TitleText                 string
	TitleAnimationSpeed       int // ms per scanner step (default 80)
	TitleAnimationInterval    int // ms pause at each end before reversing (default 1000)
	Theme                     string
}

type Model struct {
	// core
	p        *player.Player
	songsDir string
	plsDir   string
	kb       *keybinds.Keybinds // loaded keybindings

	// config
	seekStep int   // seconds per seek keypress
	autoPlay bool  // play index 0 on Init (set when launched with -p)
	prefs    Prefs // user settings

	// view
	view            viewKind
	helpPageNum     int // current page in help screen (0-indexed)
	settingsCursor  int // current settings item cursor (0-indexed)
	settingsPageNum int // current page in settings screen (0-indexed)
	width           int
	height          int

	// songs view
	songs       []player.Song
	scursor     int
	soffset     int
	playing     int
	prevPlaying int // track changes detected in tickMsg
	pos, dur    float64
	dlStates    map[int]*dlState
	plsFile     string // absolute path of currently loaded playlist (empty = songs dir)

	// filter
	filter         string // current filter text (empty = no filter)
	filterByAuthor bool   // when true, filter matches Author instead of title
	filtering      bool   // true while the user is typing a filter
	filteredIdxs   []int  // indices into songs that match the filter (nil = no filter active)

	// visualizer
	vizBars    []float64 // current smoothed bar heights (0.0–1.0)
	vizTargets []float64 // FFT target heights bars animate toward
	vizRunning bool      // true while the 100ms viz tick is scheduled
	vizCfg     VizConfig // settings loaded from Visualizer.ini

	// playlists view
	playlists []string // filenames (basename only) in plsDir
	pcursor   int
	poffset   int

	// error display
	lastError   string    // most recent download error message (empty = none)
	lastErrTime time.Time // when lastError was set; cleared after 8 s by tickMsg

	// title animation
	titleTick       int  // kept for backwards compat, unused now
	titleScanPos    int  // index of bright spot in title runes
	titleScanDir    int  // +1 or -1
	titlePauseTicks int  // remaining pause ticks at each end
	titleRunning    bool // whether the title tick loop is active

	// modal inputs (rename, add song, play song, save as, etc.)
	modalInput          string // current text in modal dialogs
	modalCursor         int    // rune cursor position within modalInput
	modalIdx            int    // index for rename/info view (which song)
	settingsInputIdx    int    // which settings item is being edited (0-indexed)
	settingsInputPrompt string // prompt label shown above the input

	// Phase 2: log view
	logLines  []string // lines loaded from jammer.log
	logOffset int      // scroll offset in viewLog

	// Phase 2: show songs from another playlist
	showSongsEntries  []playlist.Entry // songs loaded for viewShowSongs
	showSongsCursor   int
	showSongsOffset   int
	showSongsName     string // playlist name being shown
	showSongsPickMode bool   // when true, playlist picker is in "show songs" mode (not load-and-play)

	// Phase 2: online search (Ctrl+Y)
	searchPlatform string
	searchQuery    string
	searchResults  []downloader.SearchResult
	searchCursor   int
	searchOffset   int
	searchLoading  bool
	searchErr      string

	// Phase 3: edit keybindings
	kbEditCursor   int
	kbEditOffset   int
	kbEditInput    bool   // true when capturing a new key for the selected action
	kbEditAction   string // action currently being edited
	kbEditCaptured string // detected key name (shown for confirmation)
	kbEditKeys     []string // sorted list of action names for the editor

	// Phase 3: theme picker
	themeCursor int // cursor position in theme list

	// Phase 3: RSS feed view
	rssFeed        *rss.Feed  // currently loaded feed (nil = none)
	rssCursor      int        // episode list cursor
	rssOffset      int        // episode list scroll offset
	rssLoading     bool       // true while fetching
	rssErr         string     // last fetch error
	rssOriginSongs []player.Song // songs to restore when exiting RSS view
	rssOriginFile  string        // playlist file to restore

	// Phase 2: status flash message (shown in progress bar area briefly)
	statusMsg     string
	statusMsgTime time.Time
}

func New(p *player.Player, songsDir, plsDir string, seekStep int, defaultView string, kb *keybinds.Keybinds, prefs Prefs) Model {
	return NewWithPlaylist(p, songsDir, plsDir, "", seekStep, defaultView, kb, prefs)
}

func NewWithPlaylist(p *player.Player, songsDir, plsDir, plsFile string, seekStep int, defaultView string, kb *keybinds.Keybinds, prefs Prefs) Model {
	if seekStep <= 0 {
		seekStep = 2
	}
	// Apply theme before first render.
	applyTheme(theme.Get(prefs.Theme))
	// Determine initial view based on defaultView setting
	initialView := viewDefault
	if defaultView == "all" {
		initialView = viewAll
	}

	m := Model{
		p:            p,
		songsDir:     songsDir,
		plsDir:       plsDir,
		kb:           kb,
		seekStep:     seekStep,
		prefs:        prefs,
		songs:        p.Songs(),
		playing:      p.Index(),
		dlStates:     make(map[int]*dlState),
		view:         initialView,
		titleScanDir: 1,
		vizCfg:       loadVizConfig(filepath.Join(dirs.Data(), "Visualizer.ini")),
	}
	m.reloadPlaylists()
	if plsFile != "" {
		// Use the UI's loadPlaylist so the legacy-convert dialog fires if needed.
		m.loadPlaylist(filepath.Base(plsFile))
		m.autoPlay = true
	}
	return m
}

func (m *Model) reloadPlaylists() {
	names, _ := playlist.List(m.plsDir)
	m.playlists = names
}

func tick() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m *Model) vizTick() tea.Cmd {
	ms := m.vizCfg.RefreshTime
	if ms <= 0 {
		ms = 100
	}
	return tea.Tick(time.Duration(ms)*time.Millisecond, func(t time.Time) tea.Msg {
		return vizTickMsg(t)
	})
}

func titleTickCmd(speedMs int) tea.Cmd {
	if speedMs <= 0 {
		speedMs = 80
	}
	return tea.Tick(time.Duration(speedMs)*time.Millisecond, func(t time.Time) tea.Msg {
		return titleTickMsg(t)
	})
}

func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{tick(), m.startViz(0)}
	if m.prefs.ShowTitle {
		cmds = append(cmds, titleTickCmd(m.prefs.TitleAnimationSpeed))
	}
	if m.autoPlay && len(m.songs) > 0 {
		cmds = append(cmds, func() tea.Msg {
			if err := m.p.PlayIndex(0); err != nil {
				jlog.Errorf("auto-play on start: %v", err)
			}
			return nil
		}, m.downloadIfNeeded(0))
	}
	return tea.Batch(cmds...)
}

// ── Update ────────────────────────────────────────────────────────────────────

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tickMsg:
		m.titleTick++
		pos, dur := m.p.Progress()
		m.pos = pos
		m.dur = dur
		newIdx := m.p.Index()
		m.songs = m.p.Songs()
		if newIdx != m.prevPlaying {
			// Track changed (auto-advance from WatchEnd or similar).
			jlog.Infof("tick: track changed %d → %d", m.prevPlaying, newIdx)
			m.prevPlaying = newIdx
			m.playing = newIdx
			// Keep the cursor and scroll in sync so the playing song is always visible.
			m.scursor = newIdx
			m.clampSongScroll()
		}
		m.playing = newIdx
		// Clear stale error message after 8 seconds.
		if m.lastError != "" && time.Since(m.lastErrTime) >= 8*time.Second {
			m.lastError = ""
		}
		// Always check whether the current song needs downloading — this is the
		// sole trigger for n/p navigation so rapid key-holds don't pile up downloads.
		return m, tea.Batch(tick(), m.downloadIfNeeded(newIdx))

	case downloadProgressMsg:
		ds := m.getOrCreateDlState(msg.index)
		ds.active = !msg.prog.Done
		ds.frac = msg.prog.Frac
		ds.message = msg.prog.Message
		if msg.prog.Done {
			ds.active = false
		}
		// Schedule the next read from the same channels.
		return m, readNextDownloadEvent(msg.index, msg.progressCh, msg.doneCh)

	case downloadDoneMsg:
		ds := m.getOrCreateDlState(msg.index)
		ds.active = false
		if msg.err != nil {
			ds.err = msg.err
			ds.frac = 0
			jlog.Errorf("download failed index=%d: %v", msg.index, msg.err)
			m.lastError = msg.err.Error()
			m.lastErrTime = time.Now()
			// Skip to the next song if this was the currently playing track.
			if msg.index == m.playing {
				jlog.Infof("download failed for playing track — skipping to next index=%d", msg.index)
				if err := m.p.Next(); err != nil {
					jlog.Errorf("auto-skip after failed download: %v", err)
				}
				m.playing = m.p.Index()
				m.prevPlaying = m.playing
				m.scursor = m.playing
				m.clampSongScroll()
				return m, m.downloadIfNeeded(m.playing)
			}
		} else {
			ds.frac = 1
			ds.err = nil
			jlog.Infof("download done index=%d path=%q", msg.index, msg.path)
			m.p.UpdateSongPath(msg.index, msg.path)

			// Use metadata from the downloader (title/artist known at download time).
			// Fall back to reading embedded ID3/Vorbis tags from the file.
			title, artist := msg.meta.Title, msg.meta.Artist
			if title == "" {
				if info, err := tags.Read(msg.path); err == nil && info.Title != "" {
					title, artist = info.Title, info.Artist
				}
			}
			if title != "" || artist != "" {
				m.p.UpdateSongTags(msg.index, title, artist)
				jlog.Infof("download tags index=%d title=%q artist=%q", msg.index, title, artist)
				if err := tags.Write(msg.path, title, artist); err != nil {
					jlog.Errorf("download tags write failed index=%d: %v", msg.index, err)
				}
			}

			m.songs = m.p.Songs()

			// Persist enriched metadata back to the playlist file.
			m.saveCurrentPlaylist()

		// Auto-play if this is the currently selected track and player is stopped.
		if msg.index == m.playing && m.p.State() == player.StateStopped {
			jlog.Infof("auto-play after download index=%d", msg.index)
			if err := m.p.PlayIndex(msg.index); err != nil {
				jlog.Errorf("auto-play failed index=%d: %v", msg.index, err)
			}
			return m, m.startViz(0)
		}
	}

	case searchDoneMsg:
		m.searchLoading = false
		if msg.err != nil {
			m.searchErr = msg.err.Error()
			jlog.Errorf("search failed: %v", msg.err)
			return m, nil
		}
		m.searchResults = msg.results
		m.searchCursor = 0
		m.searchOffset = 0
		m.searchErr = ""
		m.view = viewSearchResults
		return m, nil

	case rssFetchDoneMsg:
		m.rssLoading = false
		if msg.err != nil {
			m.rssErr = msg.err.Error()
			m.rssLoading = false
			jlog.Errorf("rss fetch failed: %v", msg.err)
			m.setStatus("RSS error: " + msg.err.Error())
			m.view = viewDefault
			return m, nil
		}
		m.rssFeed = msg.feed
		m.rssCursor = 0
		m.rssOffset = 0
		m.rssErr = ""
		m.view = viewRssFeed
		return m, nil

	case vizTickMsg:
		nBars := m.vizNBars()
		if m.p.State() == player.StatePlaying || (m.vizRunning && nBars > 0 && len(m.vizBars) > 0) {
			m.stepViz(nBars)
			return m, m.vizTick()
		}
		// Player stopped and no bars to animate — let the tick lapse.
		m.vizRunning = false

	case titleTickMsg:
		if m.prefs.ShowTitle {
			titleRunes := []rune(m.titleString())
			n := len(titleRunes)
			if n > 0 {
				intervalTicks := m.prefs.TitleAnimationInterval
				if intervalTicks <= 0 {
					intervalTicks = 1000
				}
				speedMs := m.prefs.TitleAnimationSpeed
				if speedMs <= 0 {
					speedMs = 80
				}
				pauseNeeded := intervalTicks / speedMs
				if m.titlePauseTicks > 0 {
					m.titlePauseTicks--
				} else {
					m.titleScanPos += m.titleScanDir
					if m.titleScanPos >= n {
						// Right end: reverse immediately, no pause
						m.titleScanPos = n - 1
						m.titleScanDir = -1
					} else if m.titleScanPos < 0 {
						// Left end: pause before starting next sweep
						m.titleScanPos = 0
						m.titleScanDir = 1
						m.titlePauseTicks = pauseNeeded
					}
				}
			}
			return m, titleTickCmd(m.prefs.TitleAnimationSpeed)
		}

	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

// startViz initialises the viz state and fires the first vizTick if not already running.
func (m *Model) startViz(nBars int) tea.Cmd {
	if nBars < 1 {
		nBars = m.vizNBars()
	}
	if nBars < 1 {
		nBars = 20
	}
	if len(m.vizBars) != nBars {
		m.vizBars = make([]float64, nBars)
		m.vizTargets = make([]float64, nBars)
	}
	if m.vizRunning {
		return nil
	}
	m.vizRunning = true
	return m.vizTick()
}

// vizNBars returns the number of visualizer bars for the current terminal width.
// Returns 0 if the terminal is too narrow to show a viz.
func (m Model) vizNBars() int {
	if m.width <= 30 {
		return 0
	}
	// Match the inner content width of the outer box (│ …content… │)
	// = total width - 4 (left border + margin + margin + right border)
	n := m.width - 4
	if n < 20 {
		n = 20
	}
	return n
}

func (m *Model) stepViz(nBars int) {
	fft := m.p.FFTData() // 256 bins, or nil
	isPlaying := m.p.State() == player.StatePlaying
	if fft == nil || nBars < 1 {
		// No data: decay bars toward zero.
		for i := range m.vizBars {
			m.vizBars[i] *= 0.7
		}
		return
	}

	// When paused: decay bars if PausingEffect=true, freeze them if false.
	if !isPlaying {
		if m.vizCfg.PausingEffect {
			for i := range m.vizBars {
				m.vizBars[i] *= 0.95
			}
		}
		return
	}

	// Ensure slices are the right size.
	if len(m.vizBars) != nBars {
		m.vizBars = make([]float64, nBars)
		m.vizTargets = make([]float64, nBars)
	}

	// Map FFT bins into nBars groups using a logarithmic frequency scale.
	// 512-point FFT at 44100 Hz → bin width ≈ 86 Hz.
	const binWidth = 44100.0 / 512.0
	fLow := m.vizCfg.MinFrequency
	fHigh := m.vizCfg.MaxFrequency
	logRatio := math.Log(fHigh / fLow)

	for i := 0; i < nBars; i++ {
		loFreq := fLow * math.Exp(logRatio*float64(i)/float64(nBars))
		hiFreq := fLow * math.Exp(logRatio*float64(i+1)/float64(nBars))
		loBin := int(loFreq / binWidth)
		hiBin := int(hiFreq/binWidth) + 1
		if loBin < 1 {
			loBin = 1
		}
		if hiBin > len(fft) {
			hiBin = len(fft)
		}
		if loBin >= hiBin {
			hiBin = loBin + 1
		}
		if hiBin > len(fft) {
			break
		}
		var sum float32
		for _, v := range fft[loBin:hiBin] {
			sum += v
		}
		avg := float64(sum) / float64(hiBin-loBin)
		// Apply power exponent then log10 scale — mirrors the classic formula:
		//   pow(fftValue, logMult) → log10(1 + value * freqMult)
		avg = math.Pow(avg, m.vizCfg.LogarithmicMultiplier)
		avg = math.Log10(1 + avg*m.vizCfg.FrequencyMultiplier)
		if avg > 1 {
			avg = 1
		}
		m.vizTargets[i] = avg
	}

	// Smooth bars toward targets: fast attack, moderate decay.
	for i := range m.vizBars {
		delta := m.vizTargets[i] - m.vizBars[i]
		if delta > 0 {
			m.vizBars[i] += delta * 0.5
		} else {
			m.vizBars[i] += delta * 0.45
		}
	}
}

func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	viewName := "songs"
	if m.view == viewPlaylists {
		viewName = "playlists"
	}
	jlog.Key(msg.String(), viewName)

	keyStr := msg.String()

	// Help screen navigation
	if m.view == viewHelp {
		return m.handleHelpKey(msg)
	}

	// Settings screen navigation
	if m.view == viewSettings {
		return m.handleSettingsKey(msg)
	}
	if m.view == viewSettingsInput {
		return m.handleSettingsInputKey(msg)
	}

	// Phase 7: Modal views
	if m.view == viewRename {
		return m.handleRenameKey(msg)
	}
	if m.view == viewInfo {
		return m.handleInfoKey(msg)
	}
	if m.view == viewAddSong {
		return m.handleAddSongKey(msg)
	}
	if m.view == viewPlaySong {
		return m.handlePlaySongKey(msg)
	}
	if m.view == viewSaveAs {
		return m.handleSaveAsKey(msg)
	}
	if m.view == viewLog {
		return m.handleLogKey(msg)
	}
	if m.view == viewShowSongs {
		return m.handleShowSongsKey(msg)
	}
	if m.view == viewSearchQuery {
		return m.handleSearchQueryKey(msg)
	}
	if m.view == viewSearchResults {
		return m.handleSearchResultsKey(msg)
	}
	if m.view == viewEditKeybinds {
		return m.handleEditKeybindsKey(msg)
	}
	if m.view == viewChangeTheme {
		return m.handleChangeThemeKey(msg)
	}
	if m.view == viewRssFeed {
		return m.handleRssFeedKey(msg)
	}

	// Quit
	if m.kb.Is("Quit", keyStr) {
		m.p.Stop()
		return m, tea.Quit
	}

	// ToMainMenu (Escape) - clears filter if active; otherwise returns to viewDefault
	if m.kb.Is("ToMainMenu", keyStr) {
		if m.filtering || m.filter != "" {
			// Only clear the filter; stay in the current view.
			m.filtering = false
			m.filter = ""
			m.filteredIdxs = nil
			m.filterByAuthor = false
			m.scursor = m.playing
			m.soffset = 0
			m.clampSongScroll()
			return m, nil
		}
		m.view = viewDefault
		m.showSongsPickMode = false
		return m, nil
	}

	// View switching (Tab, Shift+F, Shift+O)
	if m.kb.Is("CommandHelpScreen", keyStr) || m.kb.Is("ListAllPlaylists", keyStr) || m.kb.Is("PlayOtherPlaylist", keyStr) {
		if m.view == viewDefault || m.view == viewAll {
			m.view = viewPlaylists
			m.reloadPlaylists()
		} else {
			m.view = viewDefault
		}
		return m, nil
	}

	// Default handler routing
	if m.view == viewPlaylists {
		return m.handlePlaylistKey(msg)
	}
	return m.handleSongKey(msg)
}

func (m Model) handleSongKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// While filter input is active, route most keys into the filter.
	if m.filtering {
		return m.handleFilterKey(msg)
	}

	keyStr := msg.String()

	// Navigation
	if m.kb.Is("PlaylistViewScrollup", keyStr) || keyStr == "up" || keyStr == "k" {
		if m.scursor > 0 {
			m.scursor--
			m.clampSongScroll()
		}
		return m, nil
	}
	if m.kb.Is("PlaylistViewScrolldown", keyStr) || keyStr == "down" || keyStr == "j" {
		if m.scursor < m.filterLen()-1 {
			m.scursor++
			m.clampSongScroll()
		}
		return m, nil
	}
	if keyStr == "home" {
		m.scursor = 0
		m.clampSongScroll()
		return m, nil
	}
	if keyStr == "end" {
		if n := m.filterLen(); n > 0 {
			m.scursor = n - 1
			m.clampSongScroll()
		}
		return m, nil
	}

	// Play/Pause
	if m.kb.Is("PlayPause", keyStr) || keyStr == "space" || keyStr == "enter" {
		if m.scursor >= m.filterLen() {
			return m, nil
		}
		_, realIdx := m.filterSong(m.scursor)
		if realIdx == m.playing && m.p.State() != player.StateStopped {
			jlog.Infof("ui: pause song index=%d", realIdx)
			if err := m.p.Pause(); err != nil {
				jlog.Errorf("ui: pause failed: %v", err)
			}
			// Pause() toggles: if we just unpaused, restart the viz tick.
			if m.p.State() == player.StatePlaying {
				return m, m.startViz(0)
			}
		} else {
			jlog.Infof("ui: play song index=%d title=%q", realIdx, m.songs[realIdx].DisplayTitle())
			if err := m.p.PlayIndex(realIdx); err != nil {
				jlog.Errorf("ui: PlayIndex failed index=%d: %v", realIdx, err)
			}
			m.playing = realIdx
			m.prevPlaying = realIdx
			// Download if the song isn't local yet.
			return m, tea.Batch(m.downloadIfNeeded(realIdx), m.startViz(0))
		}
		return m, nil
	}

	// Stop
	if m.kb.Is("Stop", keyStr) {
		jlog.Infof("ui: stop")
		m.p.Stop()
		return m, nil
	}

	// Next Song
	if m.kb.Is("NextSong", keyStr) {
		if ds := m.dlStates[m.playing]; ds != nil && ds.active {
			jlog.Infof("ui: next blocked — download active for index=%d", m.playing)
			return m, nil
		}
		if err := m.p.Next(); err != nil {
			jlog.Errorf("ui: next failed: %v", err)
		}
		m.playing = m.p.Index()
		m.prevPlaying = m.playing
		m.scursor = m.playing
		m.clampSongScroll()
		jlog.Infof("ui: next → index=%d", m.playing)
		return m, tea.Batch(m.downloadIfNeeded(m.playing), m.startViz(0))
	}

	// Previous Song
	if m.kb.Is("PreviousSong", keyStr) {
		if ds := m.dlStates[m.playing]; ds != nil && ds.active {
			jlog.Infof("ui: prev blocked — download active for index=%d", m.playing)
			return m, nil
		}
		if err := m.p.Prev(); err != nil {
			jlog.Errorf("ui: prev failed: %v", err)
		}
		m.playing = m.p.Index()
		m.prevPlaying = m.playing
		m.scursor = m.playing
		m.clampSongScroll()
		jlog.Infof("ui: prev → index=%d", m.playing)
		return m, tea.Batch(m.downloadIfNeeded(m.playing), m.startViz(0))
	}

	// Seek forward
	if m.kb.Is("Forward5s", keyStr) || keyStr == "right" {
		fwd := m.prefs.ForwardSeconds
		if fwd <= 0 {
			fwd = m.seekStep
		}
		m.p.SeekForward(float64(fwd))
		jlog.Infof("ui: seek +%ds", fwd)
		return m, nil
	}

	// Seek backward
	if m.kb.Is("Backwards5s", keyStr) || keyStr == "left" {
		bwd := m.prefs.RewindSeconds
		if bwd <= 0 {
			bwd = m.seekStep
		}
		m.p.SeekBackward(float64(bwd))
		jlog.Infof("ui: seek -%ds", bwd)
		return m, nil
	}

	// Volume up
	if m.kb.Is("VolumeUp", keyStr) || keyStr == "+" || keyStr == "=" {
		m.p.SetVolume(m.p.Volume() + 0.05)
		jlog.Infof("ui: volume up → %.0f%%", float64(m.p.Volume())*100)
		return m, nil
	}

	// Volume down
	if m.kb.Is("VolumeDown", keyStr) || keyStr == "-" {
		m.p.SetVolume(m.p.Volume() - 0.05)
		jlog.Infof("ui: volume down → %.0f%%", float64(m.p.Volume())*100)
		return m, nil
	}

	// Volume +1%
	if m.kb.Is("VolumeUpByOne", keyStr) {
		m.p.SetVolume(m.p.Volume() + 0.01)
		jlog.Infof("ui: volume +1%% → %.0f%%", float64(m.p.Volume())*100)
		return m, nil
	}

	// Volume -1%
	if m.kb.Is("VolumeDownByOne", keyStr) {
		m.p.SetVolume(m.p.Volume() - 0.01)
		jlog.Infof("ui: volume -1%% → %.0f%%", float64(m.p.Volume())*100)
		return m, nil
	}

	// Mute
	if m.kb.Is("Mute", keyStr) {
		m.p.Mute()
		if m.p.IsMuted() {
			jlog.Info("ui: muted")
		} else {
			jlog.Infof("ui: unmuted → %.0f%%", float64(m.p.Volume())*100)
		}
		return m, nil
	}

	// Loop
	if m.kb.Is("Loop", keyStr) {
		next := (m.p.GetLoopMode() + 1) % 3
		m.p.SetLoopMode(next)
		jlog.Infof("ui: loop mode → %d", next)
		return m, nil
	}

	// Random song
	if m.kb.Is("PlayRandomSong", keyStr) {
		if len(m.songs) == 0 {
			return m, nil
		}
		idx := rand.Intn(len(m.songs))
		jlog.Infof("ui: random song index=%d title=%q", idx, m.songs[idx].DisplayTitle())
		if err := m.p.PlayIndex(idx); err != nil {
			jlog.Errorf("ui: random play failed: %v", err)
		}
		m.playing = idx
		m.prevPlaying = idx
		m.scursor = idx
		m.clampSongScroll()
		return m, tea.Batch(m.downloadIfNeeded(idx), m.startViz(0))
	}

	// Shuffle
	if m.kb.Is("Shuffle", keyStr) {
		m.p.SetShuffle(!m.p.IsShuffle())
		jlog.Infof("ui: shuffle → %v", m.p.IsShuffle())
		return m, nil
	}

	// Download
	if m.kb.Is("RedownloadCurrentSong", keyStr) || keyStr == "d" {
		_, realIdx := m.filterSong(m.scursor)
		jlog.Infof("ui: download requested index=%d url=%q", realIdx, m.songs[realIdx].URL)
		return m, m.startDownload(realIdx)
	}

	// Show/hide playlist (toggle between default and all views)
	if m.kb.Is("ShowHidePlaylist", keyStr) {
		if m.view == viewDefault {
			m.view = viewAll
		} else if m.view == viewAll {
			m.view = viewDefault
		}
		return m, nil
	}

	// Help
	if m.kb.Is("Help", keyStr) {
		m.view = viewHelp
		return m, nil
	}

	// Settings
	if m.kb.Is("Settings", keyStr) {
		m.view = viewSettings
		return m, nil
	}

	// Search/filter
	if m.kb.Is("Search", keyStr) {
		m.view = viewSearchQuery
		m.searchPlatform = "youtube"
		m.searchQuery = ""
		m.searchResults = nil
		m.searchCursor = 0
		m.searchOffset = 0
		m.searchLoading = false
		m.searchErr = ""
		return m, nil
	}
	if m.kb.Is("SearchInPlaylist", keyStr) || keyStr == "/" {
		m.filtering = true
		m.filter = ""
		m.filteredIdxs = nil
		m.scursor = 0
		m.soffset = 0
		return m, nil
	}

	// Clear filter
	if keyStr == "esc" {
		if m.filter != "" || m.filteredIdxs != nil {
			m.filter = ""
			m.filteredIdxs = nil
			m.scursor = m.playing
			m.soffset = 0
			m.clampSongScroll()
		}
		return m, nil
	}

	// Delete from playlist
	if m.kb.Is("DeleteCurrentSong", keyStr) {
		_, realIdx := m.filterSong(m.scursor)
		return m.removeSong(realIdx, false), nil
	}

	// Hard delete (remove from disk)
	if m.kb.Is("HardDeleteCurrentSong", keyStr) {
		_, realIdx := m.filterSong(m.scursor)
		return m.removeSong(realIdx, true), nil
	}

	// Go to song start
	if m.kb.Is("ToSongStart", keyStr) {
		m.p.SeekBackward(m.pos) // Seek back to beginning
		jlog.Infof("ui: seek to start")
		return m, nil
	}

	// Go to song end
	if m.kb.Is("ToSongEnd", keyStr) {
		m.p.SeekForward(m.dur - m.pos) // Seek to end
		jlog.Infof("ui: seek to end")
		return m, nil
	}

	// Toggle info
	if m.kb.Is("ToggleInfo", keyStr) {
		m.modalIdx = m.playing
		m.view = viewInfo
		return m, nil
	}

	// Rename song
	if m.kb.Is("RenameSong", keyStr) {
		m.modalIdx = m.playing
		if m.modalIdx >= 0 && m.modalIdx < len(m.songs) {
			s := m.songs[m.modalIdx]
			if s.Author != "" {
				m.modalInput = s.Author + " - " + s.Title
			} else {
				m.modalInput = s.Title
			}
		} else {
			m.modalInput = ""
		}
		m.modalCursor = len([]rune(m.modalInput))
		m.view = viewRename
		return m, nil
	}

	// Add song to playlist
	if m.kb.Is("AddSongToPlaylist", keyStr) {
		m.view = viewAddSong
		return m, nil
	}

	// #7 Save current playlist (Shift+S)
	if m.kb.Is("SaveCurrentPlaylist", keyStr) {
		m.saveCurrentPlaylist()
		m.setStatus("Playlist saved")
		jlog.Info("ui: playlist saved")
		return m, nil
	}

	// #8 Save as new playlist (Shift+Alt+S)
	if m.kb.Is("SaveAsPlaylist", keyStr) {
		m.modalInput = ""
		m.modalCursor = 0
		m.view = viewSaveAs
		return m, nil
	}

	// #9 Shuffle playlist order (Alt+S)
	if m.kb.Is("ShufflePlaylist", keyStr) {
		songs := m.p.Songs()
		rand.Shuffle(len(songs), func(i, j int) { songs[i], songs[j] = songs[j], songs[i] })
		m.p.SetSongs(songs)
		m.songs = m.p.Songs()
		m.playing = m.p.Index()
		m.saveCurrentPlaylist()
		m.setStatus("Playlist shuffled")
		jlog.Info("ui: playlist shuffled")
		return m, nil
	}

	// #10 Add current song to Favorites (Ctrl+F)
	if m.kb.Is("AddCurrentSongToFavorites", keyStr) {
		if m.playing >= 0 && m.playing < len(m.songs) {
			msg := m.toggleFavorite(m.playing)
			m.setStatus(msg)
		}
		return m, nil
	}

	// #11 Show songs in another playlist (Shift+D)
	if m.kb.Is("ShowSongsInPlaylists", keyStr) {
		m.reloadPlaylists()
		m.pcursor = 0
		m.poffset = 0
		m.showSongsPickMode = true
		m.view = viewPlaylists
		return m, nil
	}

	// #12 Show log (Ctrl+L)
	if m.kb.Is("ShowLog", keyStr) {
		m.loadLogLines()
		visibleLines := m.height - 6
		if visibleLines < 1 {
			visibleLines = 1
		}
		m.logOffset = len(m.logLines) - visibleLines
		if m.logOffset < 0 {
			m.logOffset = 0
		}
		m.view = viewLog
		return m, nil
	}

	// #13 Backend switch (B)
	if m.kb.Is("BackEndChange", keyStr) {
		m.setStatus("Backend change requires restart — edit settings.json")
		jlog.Info("ui: backend change requested (requires restart)")
		return m, nil
	}

	// #14 Edit Keybindings (Shift+E)
	if m.kb.Is("EditKeybindings", keyStr) {
		m.kbEditCursor = 0
		m.kbEditOffset = 0
		m.kbEditInput = false
		m.kbEditAction = ""
		m.kbEditKeys = nil
		for k := range m.kb.GetAll() {
			m.kbEditKeys = append(m.kbEditKeys, k)
		}
		sort.Strings(m.kbEditKeys)
		m.view = viewEditKeybinds
		return m, nil
	}

	// Stubs for other Phase 3 features
	if m.kb.Is("ChangeLanguage", keyStr) {
		m.setStatus("Change language: not yet implemented")
		return m, nil
	}
	if m.kb.Is("ChangeTheme", keyStr) {
		names := theme.Names()
		m.themeCursor = 0
		for i, n := range names {
			if n == m.prefs.Theme {
				m.themeCursor = i
				break
			}
		}
		m.view = viewChangeTheme
		return m, nil
	}
	if m.kb.Is("GroupMenu", keyStr) {
		m.setStatus("Group menu: not yet implemented")
		return m, nil
	}
	if m.kb.Is("AddToGroup", keyStr) {
		m.setStatus("Add to group: not yet implemented")
		return m, nil
	}
	if m.kb.Is("ExitRssFeed", keyStr) {
		// Determine which song's URL to use: cursor song if available, else playing
		idx := m.playing
		if m.scursor >= 0 && m.scursor < m.filterLen() {
			_, idx = m.filterSong(m.scursor)
		}
		var feedURL string
		if idx >= 0 && idx < len(m.songs) {
			feedURL = m.songs[idx].URL
		}
		if feedURL == "" || !rss.IsURL(feedURL) {
			m.setStatus("No RSS URL for selected song")
			return m, nil
		}
		// Save current state so we can restore it on exit.
		m.rssOriginSongs = m.p.Songs()
		m.rssOriginFile = m.plsFile
		m.rssLoading = true
		m.rssErr = ""
		m.rssFeed = nil
		m.setStatus("Loading RSS feed…")
		return m, fetchRssCmd(feedURL)
	}

	// #4 Play Song modal (Shift+P)
	if m.kb.Is("PlaySong", keyStr) {
		m.modalInput = ""
		m.modalCursor = 0
		m.view = viewPlaySong
		return m, nil
	}

	// #6 Search by author (Shift+F3)
	if m.kb.Is("SearchByAuthor", keyStr) {
		m.filtering = true
		m.filterByAuthor = true
		m.filter = ""
		m.filteredIdxs = nil
		m.scursor = 0
		m.soffset = 0
		return m, nil
	}

	return m, nil
}

func (m Model) handleFilterKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.filtering = false
		// Play the currently selected (filtered) song immediately.
		if m.scursor < m.filterLen() {
			_, realIdx := m.filterSong(m.scursor)
			if realIdx == m.playing && m.p.State() != player.StateStopped {
				if err := m.p.Pause(); err != nil {
					jlog.Errorf("ui: pause failed: %v", err)
				}
				if m.p.State() == player.StatePlaying {
					return m, m.startViz(0)
				}
			} else {
				jlog.Infof("ui: filter enter play index=%d", realIdx)
				if err := m.p.PlayIndex(realIdx); err != nil {
					jlog.Errorf("ui: PlayIndex failed index=%d: %v", realIdx, err)
				}
				m.playing = realIdx
				m.prevPlaying = realIdx
				return m, tea.Batch(m.downloadIfNeeded(realIdx), m.startViz(0))
			}
		}
	case "esc":
		m.filtering = false
		m.filter = ""
		m.filteredIdxs = nil
		m.filterByAuthor = false
		m.scursor = m.playing
		m.soffset = 0
		m.clampSongScroll()
	case "ctrl+n":
		if m.scursor < m.filterLen()-1 {
			m.scursor++
			m.clampSongScroll()
		}
	case "ctrl+p":
		if m.scursor > 0 {
			m.scursor--
			m.clampSongScroll()
		}
	case "backspace":
		if len(m.filter) > 0 {
			runes := []rune(m.filter)
			m.filter = string(runes[:len(runes)-1])
			m.rebuildFilter()
			m.scursor = 0
			m.soffset = 0
		}
	default:
		// Accept printable single characters.
		r := []rune(msg.String())
		if len(r) == 1 && r[0] >= 32 {
			m.filter += string(r)
			m.rebuildFilter()
			m.scursor = 0
			m.soffset = 0
		}
	}
	return m, nil
}

func (m Model) handlePlaylistKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.pcursor > 0 {
			m.pcursor--
			m.clampPLScroll()
		}
	case "down", "j":
		if m.pcursor < len(m.playlists)-1 {
			m.pcursor++
			m.clampPLScroll()
		}
	case "home":
		m.pcursor = 0
		m.clampPLScroll()
	case "end":
		if n := len(m.playlists); n > 0 {
			m.pcursor = n - 1
			m.clampPLScroll()
		}
	case "space", "enter":
		if m.pcursor >= 0 && m.pcursor < len(m.playlists) {
			if m.showSongsPickMode {
				// Load playlist into showSongsEntries for read-only display.
				plsName := m.playlists[m.pcursor]
				plsPath := filepath.Join(m.plsDir, plsName)
				entries, _, err := playlist.Load(plsPath, m.songsDir)
				if err != nil {
					jlog.Errorf("ui: show songs load failed: %v", err)
					m.showSongsPickMode = false
					m.view = viewDefault
				} else {
					m.showSongsEntries = entries
					m.showSongsName = strings.TrimSuffix(plsName, filepath.Ext(plsName))
					m.showSongsCursor = 0
					m.showSongsOffset = 0
					m.showSongsPickMode = false
					m.view = viewShowSongs
				}
			} else {
				jlog.Infof("ui: loading playlist %q", m.playlists[m.pcursor])
				m.loadPlaylist(m.playlists[m.pcursor])
				m.view = viewDefault
				m.scursor = 0
				m.soffset = 0
			}
		}
	}
	return m, nil
}

func (m *Model) loadPlaylist(filename string) {
	path := filepath.Join(m.plsDir, filename)
	entries, _, err := playlist.Load(path, m.songsDir)
	if err != nil {
		return
	}

	m.applyPlaylist(path, entries)
}

func (m *Model) applyPlaylist(path string, entries []playlist.Entry) {
	m.p.LoadPlaylist(entries)
	m.songs = m.p.Songs()
	m.dlStates = make(map[int]*dlState)
	m.plsFile = path
	m.filter = ""
	m.filteredIdxs = nil
	m.filtering = false

	// Write back enriched metadata (tags filled in for downloaded songs).
	m.saveCurrentPlaylist()
}

// removeSong removes the song at index from the playlist.
// If deleteFile is true the downloaded file is also removed from disk.
func (m Model) removeSong(index int, deleteFile bool) Model {
	if index < 0 || index >= len(m.songs) {
		return m
	}
	title := m.songs[index].DisplayTitle()
	removedPath := m.p.RemoveSong(index)

	if deleteFile && removedPath != "" {
		if err := os.Remove(removedPath); err != nil {
			jlog.Errorf("ui: delete file failed path=%q: %v", removedPath, err)
		} else {
			jlog.Infof("ui: deleted file path=%q", removedPath)
		}
	}

	m.songs = m.p.Songs()
	delete(m.dlStates, index)

	// Adjust cursor.
	if m.scursor >= len(m.songs) && m.scursor > 0 {
		m.scursor = len(m.songs) - 1
	}
	m.playing = m.p.Index()
	m.prevPlaying = m.playing
	m.clampSongScroll()

	jlog.Infof("ui: removed song %q deleteFile=%v", title, deleteFile)
	m.saveCurrentPlaylist()
	return m
}

// saveCurrentPlaylist persists the current song list (with enriched metadata)
// back to the playlist file. No-op if no playlist is loaded.
func (m *Model) saveCurrentPlaylist() {
	if m.plsFile == "" {
		return
	}
	songs := m.p.Songs()
	entries := make([]playlist.Entry, len(songs))
	for i, s := range songs {
		entries[i] = playlist.Entry{
			URL:    s.URL,
			Title:  s.Title,
			Author: s.Author,
		}
		// Don't persist the resolved local path — it may differ between machines.
	}
	if err := playlist.Save(m.plsFile, entries); err != nil {
		jlog.Errorf("saveCurrentPlaylist: %v", err)
	} else {
		jlog.Infof("saveCurrentPlaylist: wrote %d entries to %q", len(entries), m.plsFile)
	}
}

// setStatus shows a flash message in the UI for ~3 seconds.
func (m *Model) setStatus(msg string) {
	m.statusMsg = msg
	m.statusMsgTime = time.Now()
}

// toggleFavorite adds or removes the song at idx from favorites.jammer.
// Returns a status string describing what happened.
func (m *Model) toggleFavorite(idx int) string {
	if idx < 0 || idx >= len(m.songs) {
		return "No song selected"
	}
	song := m.songs[idx]
	favPath := filepath.Join(filepath.Dir(m.songsDir), "favorites.jammer")

	// Load existing favorites (ignore missing file).
	existing, _, _ := playlist.Load(favPath, m.songsDir)

	// Check if already present (by URL or Path).
	key := song.URL
	if key == "" {
		key = song.Path
	}
	for i, e := range existing {
		ek := e.URL
		if ek == "" {
			ek = e.Path
		}
		if ek == key {
			// Remove it.
			existing = append(existing[:i], existing[i+1:]...)
			if err := playlist.Save(favPath, existing); err != nil {
				jlog.Errorf("toggleFavorite remove: %v", err)
				return "Error saving favorites"
			}
			jlog.Infof("ui: removed from favorites: %q", key)
			return "Removed from Favorites"
		}
	}

	// Add it.
	entry := playlist.Entry{
		Title:  song.Title,
		Author: song.Author,
		URL:    song.URL,
		Path:   song.Path,
	}
	existing = append(existing, entry)
	if err := playlist.Save(favPath, existing); err != nil {
		jlog.Errorf("toggleFavorite add: %v", err)
		return "Error saving favorites"
	}
	jlog.Infof("ui: added to favorites: %q", key)
	return "Added to Favorites"
}

// loadLogLines reads the jammer log file into m.logLines.
func (m *Model) loadLogLines() {
	logPath := filepath.Join(dirs.State(), "jammer.log")
	data, err := os.ReadFile(logPath)
	if err != nil {
		m.logLines = []string{"Log file not found: " + logPath}
		return
	}
	m.logLines = strings.Split(strings.TrimRight(string(data), "\n"), "\n")
}

// ── On-demand download ────────────────────────────────────────────────────────

// downloadIfNeeded starts a download for song i if it has a URL and is not yet
// local. Returns nil if nothing needs to be done.
func (m Model) downloadIfNeeded(i int) tea.Cmd {
	if i < 0 || i >= len(m.songs) {
		return nil
	}
	s := m.songs[i]
	if s.Downloaded() || s.URL == "" {
		return nil
	}
	if ds := m.dlStates[i]; ds != nil && (ds.active || ds.frac >= 1 || ds.err != nil) {
		return nil
	}
	jlog.Infof("auto-download: triggering index=%d url=%q", i, s.URL)
	return m.startDownload(i)
}

// startDownload kicks off a download for song at index i.
func (m Model) startDownload(i int) tea.Cmd {
	if i < 0 || i >= len(m.songs) {
		return nil
	}
	song := m.songs[i]
	if song.URL == "" {
		return nil
	}
	ds := m.getOrCreateDlState(i)
	if ds.active {
		return nil // already running
	}
	ds.active = true
	ds.frac = 0
	ds.err = nil

	progressCh := make(chan downloader.Progress, 32)
	doneCh := make(chan downloadDoneMsg, 1)

	// Goroutine: run the download; result goes to doneCh.
	go func() {
		jlog.Infof("download start index=%d url=%q", i, song.URL)
		path, meta, err := downloader.Download(context.Background(), song.URL, m.songsDir, progressCh)
		doneCh <- downloadDoneMsg{index: i, path: path, meta: meta, err: err}
	}()

	// Return a streaming Cmd that reads one event (progress or done) and
	// returns it as a tea.Msg. Update will schedule the next read.
	return readNextDownloadEvent(i, progressCh, doneCh)
}

// readNextDownloadEvent returns a Cmd that blocks until either a progress
// update or the final done message arrives, then surfaces it as a tea.Msg.
func readNextDownloadEvent(i int, progressCh <-chan downloader.Progress, doneCh <-chan downloadDoneMsg) tea.Cmd {
	return func() tea.Msg {
		select {
		case p, ok := <-progressCh:
			if !ok {
				// channel closed — wait for done
				return <-doneCh
			}
			if p.Done {
				return <-doneCh
			}
			return downloadProgressMsg{index: i, prog: p, progressCh: progressCh, doneCh: doneCh}
		case d := <-doneCh:
			return d
		}
	}
}

// performSearchCmd runs an online search via yt-dlp and returns a searchDoneMsg.
func performSearchCmd(platform, query string, limit int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		results, err := downloader.Search(ctx, query, platform, limit)
		return searchDoneMsg{results: results, err: err}
	}
}

func (m Model) getOrCreateDlState(i int) *dlState {
	if m.dlStates[i] == nil {
		m.dlStates[i] = &dlState{}
	}
	return m.dlStates[i]
}

// ── Filter helpers ────────────────────────────────────────────────────────────

// rebuildFilter recomputes filteredIdxs from the current filter string.
func (m *Model) rebuildFilter() {
	if m.filter == "" {
		m.filteredIdxs = nil
		return
	}
	lower := strings.ToLower(m.filter)
	m.filteredIdxs = m.filteredIdxs[:0]
	for i, s := range m.songs {
		var haystack string
		if m.filterByAuthor {
			haystack = strings.ToLower(s.Author)
		} else {
			haystack = strings.ToLower(s.DisplayTitle())
		}
		if strings.Contains(haystack, lower) {
			m.filteredIdxs = append(m.filteredIdxs, i)
		}
	}
}

// filterLen returns the number of visible songs (filtered or total).
func (m Model) filterLen() int {
	if m.filteredIdxs != nil {
		return len(m.filteredIdxs)
	}
	return len(m.songs)
}

// filterSong returns the song at visible position i (filtered or direct).
func (m Model) filterSong(i int) (player.Song, int) {
	if m.filteredIdxs != nil {
		idx := m.filteredIdxs[i]
		return m.songs[idx], idx
	}
	return m.songs[i], i
}

// ── Scroll helpers ────────────────────────────────────────────────────────────

func (m *Model) clampSongScroll() {
	lh := m.songListHeight()
	if m.scursor < m.soffset {
		m.soffset = m.scursor
	}
	if m.scursor >= m.soffset+lh {
		m.soffset = m.scursor - lh + 1
	}
}

func (m *Model) clampPLScroll() {
	lh := m.plListHeight()
	if m.pcursor < m.poffset {
		m.poffset = m.pcursor
	}
	if m.pcursor >= m.poffset+lh {
		m.poffset = m.pcursor - lh + 1
	}
}

func (m Model) songListHeight() int {
	// Different overhead for different views.
	// renderSongsDefault: top 3-song box (border + padding) + help bar (3) + visualizer (1) + progress bar (3) ≈ 14 lines
	// renderSongsAll: same but with full list box overhead is higher. The comment in renderSongsAll
	//   says: outer(3+1) + inner border(2) + header+sep+2 instr(4) + help bar(3) + viz(1) + prog(3) = 17
	reserved := 14
	if m.view == viewAll {
		reserved = 17
	}
	if m.filter != "" || m.filtering {
		reserved++ // filter prompt line
	}
	h := m.height - reserved
	if h < 4 {
		h = 4
	}
	return h
}

func (m Model) plListHeight() int {
	reserved := 6
	h := m.height - reserved
	if h < 4 {
		h = 4
	}
	return h
}

// ── View ──────────────────────────────────────────────────────────────────────

// currentSongPath returns the path/URL of the currently playing song,
// falling back to a friendly message if nothing is playing.
func (m Model) currentSongPath() string {
	if len(m.songs) == 0 {
		return "No song is playing"
	}
	if m.playing >= 0 && m.playing < len(m.songs) {
		s := m.songs[m.playing]
		if s.Path != "" {
			return s.Path
		}
		if s.URL != "" {
			return s.URL
		}
		return s.DisplayTitle()
	}
	return "No song is playing"
}

func (m Model) View() tea.View {
	if m.width == 0 {
		v := tea.NewView("loading...")
		v.AltScreen = true
		return v
	}
	var b strings.Builder

	// Render based on current view
	switch m.view {
	case viewHelp:
		b.WriteString(m.renderHelp())
	case viewSettings:
		b.WriteString(m.renderSettings())
	case viewSettingsInput:
		b.WriteString(m.renderSettingsInput())
	case viewRename:
		b.WriteString(m.renderOuterBox(m.currentSongPath(), m.renderSongsDefault()))
	case viewInfo:
		b.WriteString(m.renderInfo())
	case viewAddSong:
		b.WriteString(m.renderAddSong())
	case viewPlaySong:
		b.WriteString(m.renderPlaySong())
	case viewSaveAs:
		b.WriteString(m.renderSaveAs())
	case viewLog:
		b.WriteString(m.renderLog())
	case viewShowSongs:
		b.WriteString(m.renderShowSongs())
	case viewSearchQuery:
		b.WriteString(m.renderSearchQuery())
	case viewSearchResults:
		b.WriteString(m.renderSearchResults())
	case viewEditKeybinds:
		b.WriteString(m.renderEditKeybinds())
	case viewChangeTheme:
		b.WriteString(m.renderChangeTheme())
	case viewRssFeed:
		b.WriteString(m.renderRssFeed())
	case viewPlaylists:
		// Playlists view: show song path in outer box header
		b.WriteString(m.renderOuterBox(m.currentSongPath(), m.renderPlaylists()))
	default:
		// viewDefault / viewAll: full outer box with song path header
		var inner string
		if m.view == viewAll {
			inner = m.renderSongsAll()
		} else {
			inner = m.renderSongsDefault()
		}
		b.WriteString(m.renderOuterBox(m.currentSongPath(), inner))
	}

	v := tea.NewView(b.String())
	v.AltScreen = true
	return v
}

// jammerColors is the palette cycled through for the J a m m e r title animation.
var jammerColors = []lipgloss.Color{"#ff79c6", "#bd93f9", "#8be9fd", "#50fa7b", "#ffb86c", "#ff5555", "#f1fa8c"}

// titleString returns the effective title text (settings override or default).
func (m Model) titleString() string {
	if m.prefs.TitleText != "" {
		return m.prefs.TitleText
	}
	return "Jammer - light-weight CLI music player"
}

// renderJammerTitle renders the title with a K.I.T.T.-style scanner:
// one bright red spot bouncing left/right, trailing tail, rest dim.
func (m Model) renderJammerTitle() string {
	runes := []rune(m.titleString())
	n := len(runes)
	if n == 0 {
		return ""
	}

	bright := lipgloss.NewStyle().Foreground(lipgloss.Color("#ff4444")).Bold(true)
	tail1 := lipgloss.NewStyle().Foreground(lipgloss.Color("#aa1111"))
	tail2 := lipgloss.NewStyle().Foreground(lipgloss.Color("#661111"))
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	pos := m.titleScanPos
	dir := m.titleScanDir
	paused := m.titlePauseTicks > 0

	var s strings.Builder
	for i, ch := range runes {
		c := string(ch)
		isTail1 := (dir > 0 && i == pos-1) || (dir < 0 && i == pos+1)
		isTail2 := (dir > 0 && i == pos-2) || (dir < 0 && i == pos+2)
		switch {
		case paused:
			s.WriteString(dim.Render(c))
		case i == pos:
			s.WriteString(bright.Render(c))
		case isTail1:
			s.WriteString(tail1.Render(c))
		case isTail2:
			s.WriteString(tail2.Render(c))
		default:
			s.WriteString(dim.Render(c))
		}
	}
	return s.String()
}

// renderOuterBox wraps inner content in a rounded border box.
// When ShowTitle is true the header row shows the animated "J a m m e r" title;
// when false the header and separator are omitted entirely.
func (m Model) renderOuterBox(_, inner string) string {
	w := m.width
	if w < 4 {
		w = 4
	}
	innerW := w - 2 // width inside the left/right border chars

	// Top border: ╭───╮
	top := "╭" + strings.Repeat("─", innerW) + "╮"

	// Bottom border: ╰───╯
	bottom := "╰" + strings.Repeat("─", innerW) + "╯"

	// Wrap each line of inner content with │ │ borders
	lines := strings.Split(strings.TrimRight(inner, "\n"), "\n")
	var body strings.Builder
	for _, line := range lines {
		lw := lipgloss.Width(line)
		padding := innerW - lw - 2 // -2 for 1-space margin on each side
		if padding < 0 {
			padding = 0
		}
		body.WriteString("│ " + line + strings.Repeat(" ", padding) + " │\n")
	}

	headerRows := 0
	if m.prefs.ShowTitle {
		headerRows = 2 // header line + separator
	}
	usedRows := 2 + headerRows + len(lines) // top + (header+sep) + lines + bottom counted separately
	for usedRows < m.height-1 {
		body.WriteString("│" + strings.Repeat(" ", innerW) + "│\n")
		usedRows++
	}

	if !m.prefs.ShowTitle {
		return top + "\n" + body.String() + bottom + "\n"
	}

	// Header line: │ J a m m e r          │
	title := m.renderJammerTitle()
	titleW := lipgloss.Width(title)
	headerLine := "│ " + title + strings.Repeat(" ", innerW-2-titleW) + " │"
	sep := "├" + strings.Repeat("─", innerW) + "┤"

	return top + "\n" + headerLine + "\n" + sep + "\n" + body.String() + bottom + "\n"
}

// ── Songs view ────────────────────────────────────────────────────────────────

// songBoxWidth returns the Width param for inner lipgloss boxes.
// Inner boxes need 1-space margin on each side within the outer box.
// Outer box inner area = m.width - 2. With 1-char margin each side:
// inner box total rendered width = m.width - 4.
// lipgloss adds 2 chars for border → Width = m.width - 6.
func (m Model) songBoxWidth() int {
	w := m.width - 6
	if w < 10 {
		w = 10
	}
	return w
}

// songBoxTextWidth is the actual text area inside inner boxes.
// Width - Padding(0,1)×2 = songBoxWidth() - 2.
func (m Model) songBoxTextWidth() int {
	return m.songBoxWidth() - 2
}

func dlSuffix(m Model, idx int) string {
	song := m.songs[idx]
	if ds, ok := m.dlStates[idx]; ok && ds != nil {
		switch {
		case ds.active:
			pct := int(ds.frac * 100)
			return styleDLing.Render(fmt.Sprintf(" [%d%%]", pct))
		case ds.err != nil:
			return styleErr.Render(" [err]")
		case ds.frac >= 1:
			return stylePlaying.Render(" [ok]")
		}
	} else if !song.Downloaded() {
		return styleNotDL.Render(" [dl]")
	}
	return ""
}

// formatSongLine formats a song line with the author right-aligned.
// boxW is the usable width inside the inner box (after border+padding overhead).
func formatSongLine(label, title, author string, boxW int) string {
	labelPart := fmt.Sprintf("%-11s : ", label) // "Now playing : " = 14 chars
	// Available chars for title: boxW - labelPart - author - 1 (space before author)
	titleMax := boxW - len([]rune(labelPart))
	if author != "" {
		titleMax -= len([]rune(author)) + 1
	}
	if titleMax < 4 {
		titleMax = 4
	}
	titleTrunc := truncate(title, titleMax)
	if author == "" {
		return labelPart + titleTrunc
	}
	// Pad title to right-align author
	padLen := boxW - len([]rune(labelPart)) - len([]rune(titleTrunc)) - len([]rune(author))
	if padLen < 1 {
		padLen = 1
	}
	return labelPart + titleTrunc + strings.Repeat(" ", padLen) + author
}

func (m Model) renderSongsDefault() string {
	var b strings.Builder

	boxW := m.songBoxWidth()      // Width param for lipgloss
	textW := m.songBoxTextWidth() // actual text area = boxW - 2

	// ── 3-song inner box ──────────────────────────────────────────────────────
	boxBorderColor := activePalette.PlaylistBorderColor
	if boxBorderColor == "" {
		boxBorderColor = lipgloss.Color("61")
	}
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(boxBorderColor).
		Padding(0, 1).
		Width(boxW)

	if len(m.songs) == 0 {
		boxContent := "playlist .\n" +
			strings.Repeat("─", textW) + "\n" +
			styleHelp.Render("  No songs loaded")
		b.WriteString(boxStyle.Render(boxContent))
		b.WriteString("\n")
	} else {
		prevIdx := m.playing - 1
		if prevIdx < 0 {
			prevIdx = len(m.songs) - 1
		}
		currIdx := m.playing
		nextIdx := m.playing + 1
		if nextIdx >= len(m.songs) {
			nextIdx = 0
		}

		plsName := "."
		if m.plsFile != "" {
			plsName = strings.TrimSuffix(filepath.Base(m.plsFile), filepath.Ext(m.plsFile))
		}

		// playlist header + separator
		header := styleTitle.Render(plsName)
		sep := strings.Repeat("─", textW)

		prevSong := m.songs[prevIdx]
		currSong := m.songs[currIdx]
		nextSong := m.songs[nextIdx]

		prevSfx := dlSuffix(m, prevIdx)
		currSfx := dlSuffix(m, currIdx)
		nextSfx := dlSuffix(m, nextIdx)

		prevSfxW := lipgloss.Width(prevSfx)
		currSfxW := lipgloss.Width(currSfx)
		nextSfxW := lipgloss.Width(nextSfx)

		prevLine := styleHelp.Render(formatSongLine("Previous", prevSong.DisplayTitle(), prevSong.Author, textW-prevSfxW)) + prevSfx
		var currLine string
		if m.view == viewRename {
			// Inline rename: show input field on current line, no author on right
			cursor := styleBarFill.Render("█")
			label := "Now playing : "
			inputW := textW - len([]rune(label))
			if inputW < 4 {
				inputW = 4
			}
			runes := []rune(m.modalInput)
			cur := m.modalCursor
			if cur > len(runes) {
				cur = len(runes)
			}
			// Scroll the visible window so the cursor is always in view.
			// Reserve 1 column for the cursor block itself.
			visW := inputW - 1
			if visW < 1 {
				visW = 1
			}
			start := 0
			if cur >= visW {
				start = cur - visW + 1
			}
			end := start + visW
			if end > len(runes) {
				end = len(runes)
			}
			before := string(runes[start:cur])
			after := ""
			if cur < end {
				after = string(runes[cur:end])
			}
			currLine = stylePlaying.Render(label+before) + cursor + stylePlaying.Render(after)
		} else {
			currLine = stylePlaying.Render(formatSongLine("Now playing", currSong.DisplayTitle(), currSong.Author, textW-currSfxW)) + currSfx
		}
		nextLine := styleHelp.Render(formatSongLine("Next", nextSong.DisplayTitle(), nextSong.Author, textW-nextSfxW)) + nextSfx

		boxContent := header + "\n" + sep + "\n" +
			prevLine + "\n" +
			currLine + "\n" +
			nextLine

		b.WriteString(boxStyle.Render(boxContent))
		b.WriteString("\n")
	}

	// ── Filter prompt (shown when search is active) ───────────────────────────
	if m.filtering || m.filter != "" {
		label := "Search:"
		if m.filterByAuthor {
			label = "Author:"
		}
		cursor := styleBarFill.Render("█")
		prompt := styleHelp.Render(fmt.Sprintf("  %s %s", label, m.filter)) + cursor
		b.WriteString(prompt + "\n")
	}

	// ── Mini help bar (auto-sized, left-aligned, lowercase keybinds) ──────────
	helpKey, _ := m.kb.Get("Help")
	settingsKey, _ := m.kb.Get("Settings")
	playlistKey, _ := m.kb.Get("ShowHidePlaylist")
	helpText := fmt.Sprintf("%s for help | %s for settings | %s for playlist",
		helpKey, settingsKey, playlistKey)
	helpBarBorderColor := activePalette.GeneralPlaylistBorderColor
	if helpBarBorderColor == "" {
		helpBarBorderColor = lipgloss.Color("61")
	}
	helpBar := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(helpBarBorderColor).
		Padding(0, 1).
		Render(helpText)
	b.WriteString(helpBar + "\n")

	// ── Visualizer row ────────────────────────────────────────────────────────
	b.WriteString(m.renderVisualizer() + "\n")

	// ── Progress/time bar ─────────────────────────────────────────────────────
	b.WriteString(boxStyle.Render(m.renderProgressBar()))
	b.WriteString("\n")

	// ── Status flash message ──────────────────────────────────────────────────
	if m.statusMsg != "" && time.Since(m.statusMsgTime) < 3*time.Second {
		b.WriteString(styleHelp.Render("  "+m.statusMsg) + "\n")
	}

	return b.String()
}

func (m Model) renderSongsAll() string {
	var b strings.Builder

	boxW := m.songBoxWidth()
	textW := m.songBoxTextWidth()

	allBoxBorderColor := activePalette.WholePlaylistBorderColor
	if allBoxBorderColor == "" {
		allBoxBorderColor = lipgloss.Color("61")
	}
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(allBoxBorderColor).
		Padding(0, 1).
		Width(boxW)

	// ── All-songs inner box ───────────────────────────────────────────────────
	// Calculate how many song rows fit inside the box.
	// Overhead: outer(3+1) + inner border(2) + header+sep+2 instructions(4) +
	//           help bar(3) + visualizer(1) + progress bar(3) = 17
	lh := m.height - 17
	if lh < 2 {
		lh = 2
	}

	plsName := "."
	if m.plsFile != "" {
		plsName = strings.TrimSuffix(filepath.Base(m.plsFile), filepath.Ext(m.plsFile))
	}

	// keybinds for instructions
	scrollUpKey, _ := m.kb.Get("PlaylistViewScrollup")
	scrollDownKey, _ := m.kb.Get("PlaylistViewScrolldown")
	chooseKey, _ := m.kb.Get("Choose")
	deleteKey, _ := m.kb.Get("DeleteCurrentSong")

	header := styleTitle.Render(plsName)
	sep := strings.Repeat("─", textW)
	instr1 := styleHelp.Render(fmt.Sprintf("Move with %s, %s", scrollUpKey, scrollDownKey))
	instr2 := styleHelp.Render(fmt.Sprintf("Play with %s. Delete with %s.", chooseKey, deleteKey))

	var songLines strings.Builder
	total := m.filterLen()
	end := m.soffset + lh
	if end > total {
		end = total
	}
	for vi := m.soffset; vi < end; vi++ {
		song, realIdx := m.filterSong(vi)
		// number + title truncated to text width minus "N. " prefix
		numPrefix := fmt.Sprintf("%d. ", realIdx+1)
		title := truncate(song.DisplayTitle(), textW-len([]rune(numPrefix)))
		line := numPrefix + title

		// download status suffix
		suffix := ""
		if ds, ok := m.dlStates[realIdx]; ok && ds != nil {
			switch {
			case ds.active:
				pct := int(ds.frac * 100)
				suffix = styleDLing.Render(fmt.Sprintf(" [%d%%]", pct))
			case ds.err != nil:
				suffix = styleErr.Render(" [err]")
			case ds.frac >= 1:
				suffix = stylePlaying.Render(" [ok]")
			}
		} else if !song.Downloaded() {
			suffix = styleNotDL.Render(" [dl]")
		}

		var rendered string
		switch {
		case !song.Downloaded() && (m.dlStates[realIdx] == nil || !m.dlStates[realIdx].active):
			if vi == m.scursor {
				rendered = styleSelected.Render(line)
			} else {
				rendered = styleNotDL.Render(line)
			}
		case vi == m.scursor && realIdx == m.playing:
			rendered = styleSelected.Render(line)
		case vi == m.scursor:
			rendered = styleSelected.Render(line)
		case realIdx == m.playing:
			rendered = stylePlaying.Render(line)
		default:
			rendered = styleNormal.Render(line)
		}
		songLines.WriteString(rendered + suffix + "\n")
	}
	// scroll indicator if list is longer than visible area
	if total > lh {
		songLines.WriteString(styleHelp.Render(fmt.Sprintf("%d-%d / %d", m.soffset+1, end, total)) + "\n")
	}

	boxContent := header + "\n" + sep + "\n" + instr1 + "\n" + instr2 + "\n" + strings.TrimRight(songLines.String(), "\n")
	b.WriteString(boxStyle.Render(boxContent))
	b.WriteString("\n")

	// ── Mini help bar ─────────────────────────────────────────────────────────
	// ── Filter prompt (shown when search is active) ───────────────────────────
	if m.filtering || m.filter != "" {
		label := "Search:"
		if m.filterByAuthor {
			label = "Author:"
		}
		cursor := styleBarFill.Render("█")
		prompt := styleHelp.Render(fmt.Sprintf("  %s %s", label, m.filter)) + cursor
		b.WriteString(prompt + "\n")
	}

	helpKey, _ := m.kb.Get("Help")
	settingsKey, _ := m.kb.Get("Settings")
	playlistKey, _ := m.kb.Get("ShowHidePlaylist")
	helpText := fmt.Sprintf("%s for help | %s for settings | %s for playlist",
		helpKey, settingsKey, playlistKey)
	allHelpBarBorderColor := activePalette.WholePlaylistBorderColor
	if allHelpBarBorderColor == "" {
		allHelpBarBorderColor = lipgloss.Color("61")
	}
	helpBar := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(allHelpBarBorderColor).
		Padding(0, 1).
		Render(helpText)
	b.WriteString(helpBar + "\n")

	// ── Visualizer row ────────────────────────────────────────────────────────
	b.WriteString(m.renderVisualizer() + "\n")

	// ── Progress/time bar ─────────────────────────────────────────────────────
	b.WriteString(boxStyle.Render(m.renderProgressBar()))
	b.WriteString("\n")

	// ── Status flash message ──────────────────────────────────────────────────
	if m.statusMsg != "" && time.Since(m.statusMsgTime) < 3*time.Second {
		b.WriteString(styleHelp.Render("  "+m.statusMsg) + "\n")
	}

	return b.String()
}

// renderHelp returns the help screen with paginated keybindings.
// Layout matches classic jammer: 4-column table (Controls│Desc│ModControls│Desc)
// wrapped inside the standard outer box.
func (m Model) renderHelp() string {
	allBindings := m.getHelpBindings()

	// 10 rows per page; each row uses 2 flat items (left + right 2 cols).
	rowsPerPage := 10
	itemsPerPage := rowsPerPage * 2
	totalPages := (len(allBindings) + itemsPerPage - 1) / itemsPerPage
	if totalPages == 0 {
		totalPages = 1
	}

	page := m.helpPageNum
	if page >= totalPages {
		page = totalPages - 1
	}
	if page < 0 {
		page = 0
	}

	start := page * itemsPerPage
	end := start + itemsPerPage
	if end > len(allBindings) {
		end = len(allBindings)
	}
	pageItems := allBindings[start:end]

	// Column widths (each includes 1 leading space)
	// innerW = m.width - 4 (outer box margins); minus 3 separators
	innerW := m.width - 4
	if innerW < 40 {
		innerW = 40
	}
	c1W := 21 // space + 20 key text
	c2W := 26 // space + 25 desc text
	c3W := 21 // space + 20 key text
	c4W := innerW - 3 - c1W - c2W - c3W
	if c4W < 15 {
		c4W = 15
	}

	helpCell := func(text string, width int) string {
		text = " " + text
		runes := []rune(text)
		if len(runes) > width {
			runes = runes[:width]
		}
		return string(runes) + strings.Repeat(" ", width-len(runes))
	}
	row := func(c1, c2, c3, c4 string) string {
		return helpCell(c1, c1W) + "│" + helpCell(c2, c2W) + "│" + helpCell(c3, c3W) + "│" + helpCell(c4, c4W)
	}
	sepRow := strings.Repeat("─", c1W) + "┼" + strings.Repeat("─", c2W) + "┼" + strings.Repeat("─", c3W) + "┼" + strings.Repeat("─", c4W)

	var inner strings.Builder

	// Header row
	inner.WriteString(styleHelpHeader.Render(row("Controls", "Description", "ModControls", fmt.Sprintf("Description (%d/%d)", page+1, totalPages))) + "\n")
	inner.WriteString(styleHelpControl.Render(sepRow) + "\n")

	// Item rows (pairs)
	mid := (len(pageItems) + 1) / 2
	for i := 0; i < mid; i++ {
		left := pageItems[i]
		leftKey := keybinds.GetDisplay(left.key)
		leftDesc := left.desc

		var rightKey, rightDesc string
		if i+mid < len(pageItems) {
			right := pageItems[i+mid]
			rightKey = keybinds.GetDisplay(right.key)
			rightDesc = right.desc
		}
		inner.WriteString(styleHelpDesc.Render(row(leftKey, leftDesc, rightKey, rightDesc)) + "\n")
	}

	// Navigation hints
	inner.WriteString("\n")
	var navParts []string
	if page > 0 {
		navParts = append(navParts, "← prev page")
	}
	if page < totalPages-1 {
		navParts = append(navParts, "→ next page")
	}
	navParts = append(navParts, "ESC: back")
	inner.WriteString(styleHelpControl.Render(strings.Join(navParts, "  ")))

	return m.renderOuterBox(m.currentSongPath(), inner.String())
}

// renderSettings renders the settings screen.
// Layout matches classic jammer: 3-column table (Settings│Value│Change Value (page/total))
// 6 items per page, 3 pages total, letter keys A-R to change/toggle.
func (m Model) renderSettings() string {
	boolStr := func(b bool) string {
		if b {
			return "True"
		}
		return "False"
	}
	fwdSec := fmt.Sprintf("%d sec", m.prefs.ForwardSeconds)
	if m.prefs.ForwardSeconds == 0 {
		fwdSec = "5 sec"
	}
	rwdSec := fmt.Sprintf("%d sec", m.prefs.RewindSeconds)
	if m.prefs.RewindSeconds == 0 {
		rwdSec = "5 sec"
	}
	volBy := fmt.Sprintf("%d %%", int(m.prefs.ChangeVolumeBy*100))
	if m.prefs.ChangeVolumeBy == 0 {
		volBy = "5 %"
	}
	rssVal := fmt.Sprintf("%d", m.prefs.RssSkipAfterTimeValue)
	if m.prefs.RssSkipAfterTimeValue == 0 {
		rssVal = "5"
	}

	type settingItem struct {
		name       string
		value      string
		changeHint string
	}
	allItems := []settingItem{
		// Page 1 (A-F)
		{"Forward seconds", fwdSec, "A To Change"},
		{"Rewind seconds", rwdSec, "B To Change"},
		{"Change Volume By", volBy, "C To Change"},
		{"Playlist Auto Save", boolStr(m.prefs.IsAutoSave), "D To Toggle"},
		{"Load Effects", "", "E To Load Effects settings"},
		{"Toggle Media Buttons", boolStr(m.prefs.IsMediaButtons), "F To Toggle"},
		// Page 2 (G-L)
		{"Toggle Visualizer", boolStr(m.prefs.IsVisualizer), "G To Toggle Visualizer"},
		{"Load Visualizer", "", "H To Load Visualizer settings"},
		{"Set Soundcloud Client ID", "", "I To Set Soundcloud Client ID"},
		{"Fetch Client ID", "", "J To Fetch and set Soundcloud Client ID"},
		{"Toggle Key Modifier Helpers", boolStr(m.prefs.ModifierKeyHelper), "K To Toggle Key Helpers (restart required)"},
		{"Toggle Skip Errors", boolStr(m.prefs.IsIgnoreErrors), "L To Toggle Skip Errors"},
		// Page 3 (M-R)
		{"Toggle Playlist Position", boolStr(m.prefs.ShowPlaylistPosition), "M To Toggle Playlist Position"},
		{"Skip Rss after some time", boolStr(m.prefs.RssSkipAfterTime), "N To Toggle Skip Rss after some time"},
		{"Amount of time to skip Rss", rssVal, "O To Set Amount of time to skip Rss"},
		{"Toggle Quick Search", boolStr(m.prefs.EnableQuickSearch), "P To Toggle (will autoplay search result if exact match)"},
		{"Favorite Explainer", boolStr(m.prefs.FavoriteExplainer), "Q To Toggle (show explainer when favoriting a song)"},
		{"Toggle Quick Play From Search", boolStr(m.prefs.EnableQuickPlayFromSearch), "R To Toggle (automatically play the first search result when searching)"},
		// Page 4 (S)
		{"Search Result Count", fmt.Sprintf("%d", m.prefs.SearchResultCount), "S To Change (number of online search results, default 10)"},
	}

	const itemsPerPage = 6
	totalPages := (len(allItems) + itemsPerPage - 1) / itemsPerPage
	page := m.settingsPageNum
	if page < 0 {
		page = 0
	}
	if page >= totalPages {
		page = totalPages - 1
	}
	start := page * itemsPerPage
	end := start + itemsPerPage
	if end > len(allItems) {
		end = len(allItems)
	}
	pageItems := allItems[start:end]

	// Column widths
	innerW := m.width - 4
	if innerW < 60 {
		innerW = 60
	}
	c1W := 31
	c2W := 9
	c3W := innerW - 2 - c1W - c2W
	if c3W < 15 {
		c3W = 15
	}

	cell := func(text string, width int) string {
		text = " " + text
		runes := []rune(text)
		if len(runes) > width {
			runes = runes[:width]
		}
		return string(runes) + strings.Repeat(" ", width-len(runes))
	}
	row := func(c1, c2, c3 string) string {
		return cell(c1, c1W) + "│" + cell(c2, c2W) + "│" + cell(c3, c3W)
	}
	sepRow := strings.Repeat("─", c1W) + "┼" + strings.Repeat("─", c2W) + "┼" + strings.Repeat("─", c3W)

	pageLabel := fmt.Sprintf("%d/%d", page+1, totalPages)

	var inner strings.Builder
	inner.WriteString(styleSettingsHeader.Render(row("Settings", "Value", "Change Value ("+pageLabel+")")) + "\n")
	inner.WriteString(styleSettingsHint.Render(sepRow) + "\n")

	for i, s := range pageItems {
		globalIdx := start + i
		r := row(s.name, s.value, s.changeHint)
		if globalIdx == m.settingsCursor {
			inner.WriteString(styleSelected.Render(r) + "\n")
		} else {
			inner.WriteString(styleSettingsName.Render(r) + "\n")
		}
	}

	// Fill remaining rows so the table always has itemsPerPage rows
	for i := len(pageItems); i < itemsPerPage; i++ {
		inner.WriteString(styleSettingsName.Render(row("", "", "")) + "\n")
	}

	inner.WriteString(styleSettingsHint.Render(sepRow) + "\n")

	// Navigation hints: left-aligned "← prev page" and right-aligned "next page →"
	var navLeft, navRight string
	if page > 0 {
		navLeft = "PgUp/← Prev page"
	}
	if page < totalPages-1 {
		navRight = "PgDn/→ Next page"
	}
	navWidth := c1W + 1 + c2W + 1 + c3W
	navLine := navLeft + strings.Repeat(" ", navWidth-len([]rune(navLeft))-len([]rune(navRight))) + navRight
	inner.WriteString(styleSettingsHint.Render(navLine) + "\n")

	// Escape box below the table
	inner.WriteString("\n")
	inner.WriteString(styleSettingsHint.Render("╭──────────────────────╮") + "\n")
	inner.WriteString(styleSettingsHint.Render("│ To Main Menu: Escape │") + "\n")
	inner.WriteString(styleSettingsHint.Render("╰──────────────────────╯"))

	return m.renderOuterBox(m.currentSongPath(), inner.String())
}

// getHelpBindings returns a sorted list of all keybindings for the help screen
func (m Model) getHelpBindings() []struct {
	action string
	key    string
	desc   string
} {
	allBindings := m.kb.GetAll()

	// Map of action → description
	descriptions := map[string]string{
		"PlayPause":              "Play/Pause",
		"NextSong":               "Next song",
		"PreviousSong":           "Previous song",
		"Quit":                   "Quit",
		"Help":                   "Show help",
		"Settings":               "Settings",
		"ToMainMenu":             "Main menu",
		"Forward5s":              "Seek forward",
		"Backwards5s":            "Seek backward",
		"VolumeUp":               "Volume up",
		"VolumeDown":             "Volume down",
		"VolumeUpByOne":          "Volume +1%",
		"VolumeDownByOne":        "Volume -1%",
		"Mute":                   "Toggle mute",
		"Shuffle":                "Shuffle",
		"PlayRandomSong":         "Random song",
		"Loop":                   "Loop mode",
		"ShowHidePlaylist":       "Toggle playlist view",
		"ListAllPlaylists":       "All playlists",
		"PlayOtherPlaylist":      "Other playlist",
		"DeleteCurrentSong":      "Delete song",
		"HardDeleteCurrentSong":  "Delete from disk",
		"RedownloadCurrentSong":  "Redownload",
		"Search":                 "Search",
		"SearchInPlaylist":       "Search in playlist",
		"ToSongStart":            "Go to start",
		"ToSongEnd":              "Go to end",
		"ToggleInfo":             "Toggle info",
		"RenameSong":             "Rename song",
		"AddSongToPlaylist":      "Add song",
		"PlaylistViewScrollup":   "Scroll up",
		"PlaylistViewScrolldown": "Scroll down",
		"CommandHelpScreen":      "Keyboard help",
	}

	var result []struct {
		action string
		key    string
		desc   string
	}

	// Build list, sorted by action name
	for action, key := range allBindings {
		desc, exists := descriptions[action]
		if !exists {
			desc = action
		}
		result = append(result, struct {
			action string
			key    string
			desc   string
		}{action, key, desc})
	}

	// Sort by description
	sort.Slice(result, func(i, j int) bool {
		return result[i].desc < result[j].desc
	})

	return result
}

// handleHelpKey handles key input on the help screen
func (m Model) handleHelpKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	keyStr := msg.String()

	if keyStr == "esc" || m.kb.Is("ToMainMenu", keyStr) {
		m.view = viewDefault
		m.helpPageNum = 0
		return m, nil
	}

	if keyStr == "left" || m.kb.Is("PlaylistViewScrollup", keyStr) {
		if m.helpPageNum > 0 {
			m.helpPageNum--
		}
		return m, nil
	}

	if keyStr == "right" || m.kb.Is("PlaylistViewScrolldown", keyStr) {
		m.helpPageNum++
		return m, nil
	}

	if keyStr == "q" || m.kb.Is("Quit", keyStr) {
		m.p.Stop()
		return m, tea.Quit
	}

	return m, nil
}

// handleSettingsKey handles key input on the settings screen
func (m Model) handleSettingsKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	keyStr := msg.String()

	if keyStr == "esc" || m.kb.Is("ToMainMenu", keyStr) {
		m.view = viewDefault
		m.settingsCursor = 0
		m.settingsPageNum = 0
		return m, nil
	}

	if keyStr == "q" || m.kb.Is("Quit", keyStr) {
		m.p.Stop()
		return m, tea.Quit
	}

	const itemsPerPage = 6

	// Page navigation
	if keyStr == "pgdown" || keyStr == "right" {
		m.settingsPageNum++
		if m.settingsPageNum > 2 {
			m.settingsPageNum = 2
		}
		m.settingsCursor = m.settingsPageNum * itemsPerPage
		return m, nil
	}
	if keyStr == "pgup" || keyStr == "left" {
		m.settingsPageNum--
		if m.settingsPageNum < 0 {
			m.settingsPageNum = 0
		}
		m.settingsCursor = m.settingsPageNum * itemsPerPage
		return m, nil
	}

	// Cursor navigation within page
	if keyStr == "up" {
		if m.settingsCursor > m.settingsPageNum*itemsPerPage {
			m.settingsCursor--
		}
		return m, nil
	}
	if keyStr == "down" {
		pageEnd := m.settingsPageNum*itemsPerPage + itemsPerPage - 1
		if pageEnd > 18 {
			pageEnd = 18
		}
		if m.settingsCursor < pageEnd {
			m.settingsCursor++
		}
		return m, nil
	}

	// Letter keys A-R to change/toggle corresponding setting
	// A-F = page 1 (indices 0-5), G-L = page 2 (6-11), M-R = page 3 (12-17)
	letterKeys := map[string]int{
		"a": 0, "b": 1, "c": 2, "d": 3, "e": 4, "f": 5,
		"g": 6, "h": 7, "i": 8, "j": 9, "k": 10, "l": 11,
		"m": 12, "n": 13, "o": 14, "p": 15, "q": 16, "r": 17,
		"s": 18,
	}
	if idx, ok := letterKeys[keyStr]; ok {
		m.settingsCursor = idx
		m.settingsPageNum = idx / itemsPerPage
		m = m.applySettingAction(idx)
		return m, nil
	}

	// Enter applies action on currently focused item
	if keyStr == "enter" {
		m = m.applySettingAction(m.settingsCursor)
		return m, nil
	}

	return m, nil
}

// applySettingAction toggles or changes the setting at index idx.
// For numeric/string settings it opens the text-input overlay instead of
// toggling immediately.
func (m Model) applySettingAction(idx int) Model {
	switch idx {
	// Numeric/string inputs — open the input overlay
	case 0: // Forward seconds
		m.settingsInputIdx = idx
		m.settingsInputPrompt = "Enter Forward seconds (number):"
		m.modalInput = fmt.Sprintf("%d", m.prefs.ForwardSeconds)
		m.view = viewSettingsInput
		return m
	case 1: // Rewind seconds
		m.settingsInputIdx = idx
		m.settingsInputPrompt = "Enter Rewind seconds (number):"
		m.modalInput = fmt.Sprintf("%d", m.prefs.RewindSeconds)
		m.view = viewSettingsInput
		return m
	case 2: // Change Volume By
		m.settingsInputIdx = idx
		m.settingsInputPrompt = "Enter Volume change % (number):"
		m.modalInput = fmt.Sprintf("%d", int(m.prefs.ChangeVolumeBy*100))
		m.view = viewSettingsInput
		return m
	case 7: // Load Visualizer — re-read Visualizer.ini
		m.vizCfg = loadVizConfig(filepath.Join(dirs.Data(), "Visualizer.ini"))
		m.setStatus("Visualizer config reloaded")
		return m
	case 8: // Set Soundcloud Client ID
		m.settingsInputIdx = idx
		m.settingsInputPrompt = "Enter Soundcloud Client ID:"
		m.modalInput = m.prefs.ClientID
		m.view = viewSettingsInput
		return m
	case 14: // Amount of time to skip Rss
		m.settingsInputIdx = idx
		m.settingsInputPrompt = "Enter amount of time to skip Rss (number):"
		m.modalInput = fmt.Sprintf("%d", m.prefs.RssSkipAfterTimeValue)
		m.view = viewSettingsInput
		return m

	// Toggles
	case 3: // Playlist Auto Save
		m.prefs.IsAutoSave = !m.prefs.IsAutoSave
	case 5: // Toggle Media Buttons
		m.prefs.IsMediaButtons = !m.prefs.IsMediaButtons
	case 6: // Toggle Visualizer
		m.prefs.IsVisualizer = !m.prefs.IsVisualizer
	case 10: // Toggle Key Modifier Helpers
		m.prefs.ModifierKeyHelper = !m.prefs.ModifierKeyHelper
	case 11: // Toggle Skip Errors
		m.prefs.IsIgnoreErrors = !m.prefs.IsIgnoreErrors
	case 12: // Toggle Playlist Position
		m.prefs.ShowPlaylistPosition = !m.prefs.ShowPlaylistPosition
	case 13: // Skip Rss after some time
		m.prefs.RssSkipAfterTime = !m.prefs.RssSkipAfterTime
	case 15: // Toggle Quick Search
		m.prefs.EnableQuickSearch = !m.prefs.EnableQuickSearch
	case 16: // Favorite Explainer
		m.prefs.FavoriteExplainer = !m.prefs.FavoriteExplainer
	case 17: // Toggle Quick Play From Search
		m.prefs.EnableQuickPlayFromSearch = !m.prefs.EnableQuickPlayFromSearch
	case 18: // Search Result Count
		m.settingsInputIdx = idx
		m.settingsInputPrompt = "Enter Search Result Count (number, default 10, max 20):"
		m.modalInput = fmt.Sprintf("%d", m.prefs.SearchResultCount)
		m.view = viewSettingsInput
		return m
	}
	if m.prefs.SettingsPath != "" {
		saveSettings(m.prefs)
	}
	return m
}

// handleRenameKey handles key input on the rename dialog
func (m Model) handleRenameKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	keyStr := msg.String()
	runes := []rune(m.modalInput)
	cur := m.modalCursor

	if keyStr == "esc" {
		m.view = viewDefault
		m.modalInput = ""
		m.modalCursor = 0
		m.modalIdx = -1
		return m, nil
	}

	if keyStr == "enter" {
		input := strings.TrimSpace(m.modalInput)
		if input != "" && m.modalIdx >= 0 && m.modalIdx < len(m.songs) {
			// Parse "Author - Title" or plain title.
			title := input
			author := ""
			if idx := strings.Index(input, " - "); idx > 0 {
				author = strings.TrimSpace(input[:idx])
				title = strings.TrimSpace(input[idx+3:])
			}
			// Write tags to disk if the file exists.
			if path := m.songs[m.modalIdx].Path; path != "" {
				if err := tags.Write(path, title, author); err != nil {
					jlog.Errorf("ui: rename tags.Write failed: %v", err)
				}
			}
			// Update in-memory player state.
			m.p.UpdateSongTags(m.modalIdx, title, author)
			// Refresh local snapshot.
			m.songs = m.p.Songs()
			// Persist to playlist file.
			m.saveCurrentPlaylist()
			jlog.Infof("ui: renamed song index=%d to %q / %q", m.modalIdx, title, author)
		}
		// Empty input = cancelled (keep original name)
		m.view = viewDefault
		m.modalInput = ""
		m.modalCursor = 0
		m.modalIdx = -1
		return m, nil
	}

	// Cursor movement
	switch keyStr {
	case "left":
		if cur > 0 {
			m.modalCursor--
		}
		return m, nil
	case "right":
		if cur < len(runes) {
			m.modalCursor++
		}
		return m, nil
	case "home", "ctrl+a":
		m.modalCursor = 0
		return m, nil
	case "end", "ctrl+e":
		m.modalCursor = len(runes)
		return m, nil
	}

	// Backspace: delete rune before cursor
	if keyStr == "backspace" && cur > 0 {
		m.modalInput = string(runes[:cur-1]) + string(runes[cur:])
		m.modalCursor--
		return m, nil
	}

	// Delete: delete rune at cursor
	if keyStr == "delete" && cur < len(runes) {
		m.modalInput = string(runes[:cur]) + string(runes[cur+1:])
		return m, nil
	}

	// Character input: insert at cursor
	ch := ""
	if keyStr == "space" {
		ch = " "
	} else if len(keyStr) == 1 && keyStr[0] >= 32 && keyStr[0] <= 126 {
		ch = keyStr
	}
	if ch != "" {
		m.modalInput = string(runes[:cur]) + ch + string(runes[cur:])
		m.modalCursor++
		return m, nil
	}

	return m, nil
}

// handleInfoKey handles key input on the info dialog
func (m Model) handleInfoKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	keyStr := msg.String()

	if keyStr == "esc" || keyStr == "q" {
		m.view = viewDefault
		m.modalIdx = -1
		return m, nil
	}

	return m, nil
}

// handleAddSongKey handles key input on the add song dialog
func (m Model) handleAddSongKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	keyStr := msg.String()

	if keyStr == "esc" {
		m.view = viewDefault
		m.modalInput = ""
		return m, nil
	}

	if keyStr == "enter" {
		input := strings.TrimSpace(m.modalInput)
		if input != "" {
			song := player.Song{}
			// Determine whether it's a URL or a local path.
			if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {
				song.URL = input
			} else {
				// Treat as local file path.
				song.Path = input
				if info, err := tags.Read(input); err == nil && info.Title != "" {
					song.Title = info.Title
					song.Author = info.Artist
				} else {
					song.Title = strings.TrimSuffix(filepath.Base(input), filepath.Ext(input))
				}
			}
			m.p.AddSong(song)
			m.songs = m.p.Songs()
			m.saveCurrentPlaylist()
			jlog.Infof("ui: added song %q to queue (total=%d)", input, len(m.songs))
		}
		m.view = viewDefault
		m.modalInput = ""
		return m, nil
	}

	// Character input - allow URLs and paths
	if keyStr == "space" {
		m.modalInput += " "
		return m, nil
	}
	if len(keyStr) == 1 && (keyStr[0] >= 32 && keyStr[0] <= 126) {
		m.modalInput += keyStr
		return m, nil
	}

	// Backspace
	if keyStr == "backspace" && len(m.modalInput) > 0 {
		m.modalInput = m.modalInput[:len(m.modalInput)-1]
		return m, nil
	}

	return m, nil
}

// ── Phase 7: Modal Views ──────────────────────────────────────────────────────

// renderInfo renders the song information overlay
func (m Model) renderInfo() string {
	var b strings.Builder

	b.WriteString(styleInputTitle.Render("  Song Information") + "\n")
	b.WriteString(strings.Repeat("─", m.width-2) + "\n")

	if m.modalIdx >= 0 && m.modalIdx < len(m.songs) {
		song := m.songs[m.modalIdx]
		b.WriteString(styleInputText.Render(fmt.Sprintf("  Title:    %s", song.DisplayTitle())) + "\n")
		b.WriteString(styleInputText.Render(fmt.Sprintf("  Author:   %s", song.Author)) + "\n")
		b.WriteString(styleInputText.Render(fmt.Sprintf("  URL:      %s", song.URL)) + "\n")
		b.WriteString(styleInputText.Render(fmt.Sprintf("  File:     %s", filepath.Base(song.Path))) + "\n")
		b.WriteString(styleInputText.Render(fmt.Sprintf("  Downloaded: %v", song.Downloaded())) + "\n")
	} else {
		b.WriteString(styleInputText.Render("  No song selected") + "\n")
	}

	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", m.width-2) + "\n")
	b.WriteString(styleInputText.Render("  ESC: close") + "\n")

	return b.String()
}

// renderAddSong renders the add song input dialog
func (m Model) renderAddSong() string {
	var b strings.Builder

	b.WriteString(styleInputTitle.Render("  Add Song") + "\n")
	b.WriteString(strings.Repeat("─", m.width-2) + "\n")
	b.WriteString(styleInputText.Render("  Enter URL or file path to add to playlist") + "\n\n")

	// Input line
	cursor := styleBarFill.Render("█")
	b.WriteString(styleInputText.Render("  ") + m.modalInput + cursor + "\n")

	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", m.width-2) + "\n")
	b.WriteString(styleInputText.Render("  Enter: add  ESC: cancel") + "\n")

	return b.String()
}

// ── Visualizer ────────────────────────────────────────────────────────────────

// ── Phase 2: Play Song modal (#4) ─────────────────────────────────────────────

func (m Model) renderPlaySong() string {
	var b strings.Builder
	b.WriteString(styleInputTitle.Render("  Play Song") + "\n")
	b.WriteString(strings.Repeat("─", m.width-2) + "\n")
	b.WriteString(styleInputText.Render("  Enter a file path or URL to insert and play immediately") + "\n\n")
	cursor := styleBarFill.Render("█")
	b.WriteString(styleInputText.Render("  ") + m.modalInput + cursor + "\n")
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", m.width-2) + "\n")
	b.WriteString(styleInputText.Render("  Enter: play  ESC: cancel") + "\n")
	return b.String()
}

func (m Model) handlePlaySongKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	keyStr := msg.String()
	if keyStr == "esc" {
		m.view = viewDefault
		m.modalInput = ""
		return m, nil
	}
	if keyStr == "enter" {
		input := strings.TrimSpace(m.modalInput)
		m.view = viewDefault
		m.modalInput = ""
		if input == "" {
			return m, nil
		}
		song := player.Song{}
		if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {
			song.URL = input
			song.Title = input
		} else {
			song.Path = input
			if info, err := tags.Read(input); err == nil && info.Title != "" {
				song.Title = info.Title
				song.Author = info.Artist
			} else {
				song.Title = strings.TrimSuffix(filepath.Base(input), filepath.Ext(input))
			}
		}
		// Insert after current playing position.
		insertAt := m.playing + 1
		songs := m.p.Songs()
		newSongs := make([]player.Song, 0, len(songs)+1)
		newSongs = append(newSongs, songs[:insertAt]...)
		newSongs = append(newSongs, song)
		newSongs = append(newSongs, songs[insertAt:]...)
		m.p.SetSongs(newSongs)
		m.songs = m.p.Songs()
		m.saveCurrentPlaylist()
		if err := m.p.PlayIndex(insertAt); err != nil {
			jlog.Errorf("ui: PlaySong PlayIndex failed: %v", err)
		}
		m.playing = insertAt
		m.prevPlaying = insertAt
		return m, tea.Batch(m.downloadIfNeeded(insertAt), m.startViz(0))
	}
	if keyStr == "space" {
		m.modalInput += " "
		return m, nil
	}
	if len(keyStr) == 1 && keyStr[0] >= 32 && keyStr[0] <= 126 {
		m.modalInput += keyStr
		return m, nil
	}
	if keyStr == "backspace" && len(m.modalInput) > 0 {
		m.modalInput = m.modalInput[:len(m.modalInput)-1]
		return m, nil
	}
	return m, nil
}

// ── Phase 2: Save As modal (#8) ───────────────────────────────────────────────

func (m Model) renderSaveAs() string {
	var b strings.Builder
	b.WriteString(styleInputTitle.Render("  Save Playlist As") + "\n")
	b.WriteString(strings.Repeat("─", m.width-2) + "\n")
	b.WriteString(styleInputText.Render("  Enter a name for the new playlist (without extension)") + "\n\n")
	cursor := styleBarFill.Render("█")
	b.WriteString(styleInputText.Render("  ") + m.modalInput + cursor + "\n")
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", m.width-2) + "\n")
	b.WriteString(styleInputText.Render("  Enter: save  ESC: cancel") + "\n")
	return b.String()
}

func (m Model) handleSaveAsKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	keyStr := msg.String()
	if keyStr == "esc" {
		m.view = viewDefault
		m.modalInput = ""
		return m, nil
	}
	if keyStr == "enter" {
		name := strings.TrimSpace(m.modalInput)
		m.view = viewDefault
		m.modalInput = ""
		if name == "" {
			return m, nil
		}
		path := filepath.Join(m.plsDir, name+".jammer")
		songs := m.p.Songs()
		entries := make([]playlist.Entry, len(songs))
		for i, s := range songs {
			entries[i] = playlist.Entry{Path: s.Path, URL: s.URL, Title: s.Title, Author: s.Author}
		}
		if err := playlist.Save(path, entries); err != nil {
			jlog.Errorf("ui: save-as failed: %v", err)
			m.setStatus("Save failed: " + err.Error())
		} else {
			m.setStatus("Saved as " + name)
		}
		return m, nil
	}
	if keyStr == "space" {
		m.modalInput += " "
		return m, nil
	}
	if len(keyStr) == 1 && keyStr[0] >= 32 && keyStr[0] <= 126 {
		m.modalInput += keyStr
		return m, nil
	}
	if keyStr == "backspace" && len(m.modalInput) > 0 {
		m.modalInput = m.modalInput[:len(m.modalInput)-1]
		return m, nil
	}
	return m, nil
}

// ── Phase 2: Log view (#12) ───────────────────────────────────────────────────

func (m Model) renderLog() string {
	var b strings.Builder
	b.WriteString(styleTitle.Render("  Jammer Log") + "\n")
	b.WriteString(strings.Repeat("─", m.width-2) + "\n")

	visibleLines := m.height - 6
	if visibleLines < 1 {
		visibleLines = 1
	}
	lines := m.logLines
	end := m.logOffset + visibleLines
	if end > len(lines) {
		end = len(lines)
	}
	start := m.logOffset
	if start > len(lines) {
		start = len(lines)
	}
	for _, line := range lines[start:end] {
		b.WriteString(styleHelp.Render("  "+truncate(line, m.width-4)) + "\n")
	}
	// Pad remaining lines.
	for i := end - start; i < visibleLines; i++ {
		b.WriteString("\n")
	}

	b.WriteString(strings.Repeat("─", m.width-2) + "\n")
	b.WriteString(styleHelp.Render(fmt.Sprintf("  ↑/↓ to scroll  ESC: close  (%d/%d lines)", m.logOffset, len(lines))) + "\n")
	return b.String()
}

func (m Model) handleLogKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	keyStr := msg.String()
	visibleLines := m.height - 6
	if visibleLines < 1 {
		visibleLines = 1
	}
	switch keyStr {
	case "esc", "ctrl+l":
		m.view = viewDefault
	case "up", "k":
		if m.logOffset > 0 {
			m.logOffset--
		}
	case "down", "j":
		if m.logOffset+visibleLines < len(m.logLines) {
			m.logOffset++
		}
	case "home", "g":
		m.logOffset = 0
	case "end", "G":
		max := len(m.logLines) - visibleLines
		if max < 0 {
			max = 0
		}
		m.logOffset = max
	}
	return m, nil
}

// ── Phase 2: Show Songs from another playlist (#11) ───────────────────────────

func (m Model) renderShowSongs() string {
	var b strings.Builder
	title := "Songs"
	if m.showSongsName != "" {
		title = m.showSongsName
	}
	b.WriteString(styleTitle.Render("  "+title) + "\n")
	b.WriteString(strings.Repeat("─", m.width-2) + "\n")

	visibleLines := m.height - 6
	if visibleLines < 1 {
		visibleLines = 1
	}
	entries := m.showSongsEntries
	end := m.showSongsOffset + visibleLines
	if end > len(entries) {
		end = len(entries)
	}
	start := m.showSongsOffset
	if start > len(entries) {
		start = len(entries)
	}
	for i, e := range entries[start:end] {
		realIdx := start + i
		displayTitle := e.Title
		if displayTitle == "" {
			displayTitle = strings.TrimSuffix(filepath.Base(e.Path), filepath.Ext(e.Path))
		}
		line := fmt.Sprintf("  %s", truncate(displayTitle, m.width-4))
		if realIdx == m.showSongsCursor {
			b.WriteString(styleSelected.Render(line))
		} else {
			b.WriteString(styleNormal.Render(line))
		}
		b.WriteString("\n")
	}
	for i := end - start; i < visibleLines; i++ {
		b.WriteString("\n")
	}

	b.WriteString(strings.Repeat("─", m.width-2) + "\n")
	b.WriteString(styleHelp.Render("  ↑/↓ to scroll  ESC: close") + "\n")
	return b.String()
}

func (m Model) handleShowSongsKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	keyStr := msg.String()
	visibleLines := m.height - 6
	if visibleLines < 1 {
		visibleLines = 1
	}
	switch keyStr {
	case "esc":
		m.view = viewDefault
		m.showSongsEntries = nil
		m.showSongsName = ""
		m.showSongsCursor = 0
		m.showSongsOffset = 0
	case "up", "k":
		if m.showSongsCursor > 0 {
			m.showSongsCursor--
			if m.showSongsCursor < m.showSongsOffset {
				m.showSongsOffset = m.showSongsCursor
			}
		}
	case "down", "j":
		if m.showSongsCursor < len(m.showSongsEntries)-1 {
			m.showSongsCursor++
			if m.showSongsCursor >= m.showSongsOffset+visibleLines {
				m.showSongsOffset = m.showSongsCursor - visibleLines + 1
			}
		}
	case "home", "g":
		m.showSongsCursor = 0
		m.showSongsOffset = 0
	case "end", "G":
		m.showSongsCursor = len(m.showSongsEntries) - 1
		if m.showSongsCursor < 0 {
			m.showSongsCursor = 0
		}
		m.showSongsOffset = m.showSongsCursor - visibleLines + 1
		if m.showSongsOffset < 0 {
			m.showSongsOffset = 0
		}
	}
	return m, nil
}

// ── Phase 2: Online Search (#5) ───────────────────────────────────────────────

func (m Model) renderSearchQuery() string {
	var b strings.Builder
	platform := "YouTube"
	if m.searchPlatform == "soundcloud" {
		platform = "SoundCloud"
	}
	b.WriteString(styleTitle.Render(fmt.Sprintf("  Search (%s)", platform)) + "\n")
	b.WriteString(strings.Repeat("─", m.width-2) + "\n")
	b.WriteString(styleHelp.Render("  Enter query:") + "\n\n")
	cursor := styleBarFill.Render("█")
	if m.searchLoading {
		b.WriteString(styleHelp.Render("  ") + m.searchQuery + cursor + "\n")
		b.WriteString(styleHelp.Render("\n  Searching...") + "\n")
	} else if m.searchErr != "" {
		b.WriteString(styleHelp.Render("  ") + m.searchQuery + cursor + "\n")
		b.WriteString(styleHelp.Render(fmt.Sprintf("\n  Error: %s", truncate(m.searchErr, m.width-10))) + "\n")
	} else {
		b.WriteString(styleHelp.Render("  ") + m.searchQuery + cursor + "\n")
		b.WriteString("\n")
	}
	b.WriteString(strings.Repeat("─", m.width-2) + "\n")
	b.WriteString(styleHelp.Render("  Tab: switch platform  Enter: search  ESC: cancel") + "\n")
	return b.String()
}

func (m Model) handleSearchQueryKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	keyStr := msg.String()
	if keyStr == "esc" {
		m.view = viewDefault
		m.searchQuery = ""
		m.searchResults = nil
		m.searchErr = ""
		m.searchLoading = false
		return m, nil
	}
	if keyStr == "tab" {
		if m.searchPlatform == "youtube" {
			m.searchPlatform = "soundcloud"
		} else {
			m.searchPlatform = "youtube"
		}
		return m, nil
	}
	if keyStr == "enter" {
		query := strings.TrimSpace(m.searchQuery)
		if query == "" {
			return m, nil
		}
		m.searchLoading = true
		m.searchErr = ""
		limit := m.prefs.SearchResultCount
		if limit <= 0 {
			limit = 10
		}
		return m, performSearchCmd(m.searchPlatform, query, limit)
	}
	if keyStr == "space" {
		m.searchQuery += " "
		return m, nil
	}
	if len(keyStr) == 1 && keyStr[0] >= 32 && keyStr[0] <= 126 {
		m.searchQuery += keyStr
		return m, nil
	}
	if keyStr == "backspace" && len(m.searchQuery) > 0 {
		m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
		return m, nil
	}
	return m, nil
}

func (m Model) renderSearchResults() string {
	var b strings.Builder
	platform := "YouTube"
	if m.searchPlatform == "soundcloud" {
		platform = "SoundCloud"
	}
	b.WriteString(styleTitle.Render(fmt.Sprintf("  Results (%s): %q", platform, truncate(m.searchQuery, m.width-30))) + "\n")
	b.WriteString(strings.Repeat("─", m.width-2) + "\n")

	visibleLines := m.height - 6
	if visibleLines < 1 {
		visibleLines = 1
	}
	results := m.searchResults
	end := m.searchOffset + visibleLines
	if end > len(results) {
		end = len(results)
	}
	start := m.searchOffset
	if start > len(results) {
		start = len(results)
	}
	for i, r := range results[start:end] {
		realIdx := start + i
		display := fmt.Sprintf("  %s — %s", truncate(r.Artist, (m.width-6)/2), truncate(r.Title, (m.width-6)/2))
		if realIdx == m.searchCursor {
			b.WriteString(styleSelected.Render(display))
		} else {
			b.WriteString(styleNormal.Render(display))
		}
		b.WriteString("\n")
	}
	for i := end - start; i < visibleLines; i++ {
		b.WriteString("\n")
	}

	b.WriteString(strings.Repeat("─", m.width-2) + "\n")
	b.WriteString(styleHelp.Render(fmt.Sprintf("  ↑/↓: navigate  Enter: add & play  ESC: back  (%d/%d)", m.searchCursor+1, len(results))) + "\n")
	return b.String()
}

func (m Model) handleSearchResultsKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	keyStr := msg.String()
	if keyStr == "esc" {
		m.view = viewSearchQuery
		m.searchResults = nil
		m.searchCursor = 0
		m.searchOffset = 0
		return m, nil
	}
	visibleLines := m.height - 6
	if visibleLines < 1 {
		visibleLines = 1
	}
	switch keyStr {
	case "up", "k":
		if m.searchCursor > 0 {
			m.searchCursor--
			if m.searchCursor < m.searchOffset {
				m.searchOffset = m.searchCursor
			}
		}
	case "down", "j":
		if m.searchCursor < len(m.searchResults)-1 {
			m.searchCursor++
			if m.searchCursor >= m.searchOffset+visibleLines {
				m.searchOffset = m.searchCursor - visibleLines + 1
			}
		}
	case "home", "g":
		m.searchCursor = 0
		m.searchOffset = 0
	case "end", "G":
		m.searchCursor = len(m.searchResults) - 1
		if m.searchCursor < 0 {
			m.searchCursor = 0
		}
		m.searchOffset = m.searchCursor - visibleLines + 1
		if m.searchOffset < 0 {
			m.searchOffset = 0
		}
	case "enter":
		if m.searchCursor < 0 || m.searchCursor >= len(m.searchResults) {
			return m, nil
		}
		r := m.searchResults[m.searchCursor]
		song := player.Song{
			URL:    r.URL,
			Title:  r.Title,
			Author: r.Artist,
		}
		// Insert after current playing position.
		insertAt := m.playing + 1
		songs := m.p.Songs()
		newSongs := make([]player.Song, 0, len(songs)+1)
		newSongs = append(newSongs, songs[:insertAt]...)
		newSongs = append(newSongs, song)
		newSongs = append(newSongs, songs[insertAt:]...)
		m.p.SetSongs(newSongs)
		m.songs = m.p.Songs()
		m.saveCurrentPlaylist()
		if err := m.p.PlayIndex(insertAt); err != nil {
			jlog.Errorf("ui: search result PlayIndex failed: %v", err)
		}
		m.playing = insertAt
		m.prevPlaying = insertAt
		m.view = viewDefault
		m.searchQuery = ""
		m.searchResults = nil
		m.searchCursor = 0
		m.searchOffset = 0
		return m, tea.Batch(m.downloadIfNeeded(insertAt), m.startViz(0))
	}
	return m, nil
}

// ── Phase 3: Edit Keybindings (#14) ───────────────────────────────────────────

func (m Model) renderEditKeybinds() string {
	var b strings.Builder
	if m.kbEditInput {
		b.WriteString(styleKeybindsHeader.Render("  Edit Keybinding") + "\n")
		b.WriteString(strings.Repeat("─", m.width-2) + "\n")
		b.WriteString(styleKeybindsDesc.Render(fmt.Sprintf("  Action: %s", m.kbEditAction)) + "\n")
		if m.kbEditCaptured != "" {
			b.WriteString(styleKeybindsCurrent.Render(fmt.Sprintf("  Detected: %s", keybinds.GetDisplay(m.kbEditCaptured))) + "\n")
			b.WriteString(styleKeybindsControl.Render("  Press Enter to confirm or ESC to cancel") + "\n")
		} else {
			b.WriteString(styleKeybindsDesc.Render("  Press the new key combination...") + "\n")
			b.WriteString(styleKeybindsDesc.Render("  (type the key name + Enter if direct capture fails)") + "\n")
		}
		b.WriteString(strings.Repeat("─", m.width-2) + "\n")
		b.WriteString(styleKeybindsControl.Render("  ESC: cancel") + "\n")
		return b.String()
	}

	b.WriteString(styleKeybindsHeader.Render("  Edit Keybindings") + "\n")
	b.WriteString(strings.Repeat("─", m.width-2) + "\n")

	visibleLines := m.height - 6
	if visibleLines < 1 {
		visibleLines = 1
	}
	keys := m.kbEditKeys
	end := m.kbEditOffset + visibleLines
	if end > len(keys) {
		end = len(keys)
	}
	start := m.kbEditOffset
	if start > len(keys) {
		start = len(keys)
	}
	for i, action := range keys[start:end] {
		realIdx := start + i
		keyStr, _ := m.kb.Get(action)
		displayKey := keybinds.GetDisplay(keyStr)
		line := fmt.Sprintf("  %-28s %s", action, displayKey)
		if realIdx == m.kbEditCursor {
			b.WriteString(styleSelected.Render(truncate(line, m.width-2)))
		} else {
			b.WriteString(styleKeybindsDesc.Render(truncate(line, m.width-2)))
		}
		b.WriteString("\n")
	}
	for i := end - start; i < visibleLines; i++ {
		b.WriteString("\n")
	}

	b.WriteString(strings.Repeat("─", m.width-2) + "\n")
	b.WriteString(styleKeybindsControl.Render(fmt.Sprintf("  ↑/↓: navigate  Enter: edit  ESC: close  (%d/%d)", m.kbEditCursor+1, len(keys))) + "\n")
	return b.String()
}

func (m Model) handleEditKeybindsKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	keyStr := msg.String()

	if m.kbEditInput {
		jlog.Infof("kb capture: raw=%q action=%s captured=%q", keyStr, m.kbEditAction, m.kbEditCaptured)
		if keyStr == "esc" {
			m.kbEditInput = false
			m.kbEditAction = ""
			m.kbEditCaptured = ""
			return m, nil
		}
		// Ignore modifier-only keys
		if keyStr == "ctrl" || keyStr == "alt" || keyStr == "shift" {
			return m, nil
		}
		// Enter confirms the currently captured/typed key.
		if keyStr == "enter" {
			if m.kbEditCaptured == "" {
				return m, nil
			}
			m.kb.Set(m.kbEditAction, m.kbEditCaptured)
			if err := m.kb.Save(); err != nil {
				jlog.Errorf("save keybindings failed: %v", err)
				m.setStatus(fmt.Sprintf("Save failed: %v", err))
			} else {
				m.setStatus(fmt.Sprintf("%s → %s", m.kbEditAction, keybinds.GetDisplay(m.kbEditCaptured)))
				jlog.Infof("keybinding updated: %s = %s", m.kbEditAction, m.kbEditCaptured)
			}
			m.kbEditInput = false
			m.kbEditAction = ""
			m.kbEditCaptured = ""
			return m, nil
		}
		// Backspace deletes last accumulated character.
		if keyStr == "backspace" && m.kbEditCaptured != "" {
			m.kbEditCaptured = m.kbEditCaptured[:len(m.kbEditCaptured)-1]
			return m, nil
		}
		// Single printable characters are accumulated so the user can type a key
		// name (e.g. "f12", "ctrl+y") when direct capture fails.
		isSinglePrintable := len(keyStr) == 1 && keyStr[0] >= 32 && keyStr[0] <= 126
		if isSinglePrintable {
			m.kbEditCaptured += keyStr
			return m, nil
		}
		// Everything else (special keys like f12, ctrl+y, space, etc.) is
		// captured immediately, overwriting any typed text.
		m.kbEditCaptured = keyStr
		return m, nil
	}

	if keyStr == "esc" {
		m.view = viewDefault
		m.kbEditKeys = nil
		m.kbEditCursor = 0
		m.kbEditOffset = 0
		return m, nil
	}

	visibleLines := m.height - 6
	if visibleLines < 1 {
		visibleLines = 1
	}
	switch keyStr {
	case "up", "k":
		if m.kbEditCursor > 0 {
			m.kbEditCursor--
			if m.kbEditCursor < m.kbEditOffset {
				m.kbEditOffset = m.kbEditCursor
			}
		}
	case "down", "j":
		if m.kbEditCursor < len(m.kbEditKeys)-1 {
			m.kbEditCursor++
			if m.kbEditCursor >= m.kbEditOffset+visibleLines {
				m.kbEditOffset = m.kbEditCursor - visibleLines + 1
			}
		}
	case "home", "g":
		m.kbEditCursor = 0
		m.kbEditOffset = 0
	case "end", "G":
		m.kbEditCursor = len(m.kbEditKeys) - 1
		if m.kbEditCursor < 0 {
			m.kbEditCursor = 0
		}
		m.kbEditOffset = m.kbEditCursor - visibleLines + 1
		if m.kbEditOffset < 0 {
			m.kbEditOffset = 0
		}
	case "enter":
		if m.kbEditCursor >= 0 && m.kbEditCursor < len(m.kbEditKeys) {
			m.kbEditAction = m.kbEditKeys[m.kbEditCursor]
			m.kbEditInput = true
		}
	}
	return m, nil
}

// lerpVizColor interpolates a color along a list of gradient stops.
// t is in [0,1] where 0 = first stop and 1 = last stop.
// Each stop is a lipgloss.Color (hex "#RRGGBB" or terminal index).
// Adjacent stops are linearly interpolated in RGB space.
func lerpVizColor(stops []lipgloss.Color, t float64) lipgloss.Color {
	if len(stops) == 0 {
		return ""
	}
	if len(stops) == 1 {
		return stops[0]
	}
	if t <= 0 {
		return stops[0]
	}
	if t >= 1 {
		return stops[len(stops)-1]
	}
	seg := t * float64(len(stops)-1)
	lo := int(seg)
	hi := lo + 1
	if hi >= len(stops) {
		return stops[len(stops)-1]
	}
	frac := seg - float64(lo)

	parse := func(c lipgloss.Color) (r, g, b float64) {
		s := string(c)
		if len(s) == 7 && s[0] == '#' {
			var rv, gv, bv uint8
			fmt.Sscanf(s[1:3], "%02x", &rv)
			fmt.Sscanf(s[3:5], "%02x", &gv)
			fmt.Sscanf(s[5:7], "%02x", &bv)
			return float64(rv), float64(gv), float64(bv)
		}
		// Terminal index — can't interpolate, just return the nearest stop.
		return -1, -1, -1
	}

	r0, g0, b0 := parse(stops[lo])
	r1, g1, b1 := parse(stops[hi])
	if r0 < 0 || r1 < 0 {
		// Non-hex stop: snap to nearest
		if frac < 0.5 {
			return stops[lo]
		}
		return stops[hi]
	}

	r := r0 + (r1-r0)*frac
	g := g0 + (g1-g0)*frac
	b := b0 + (b1-b0)*frac
	return lipgloss.Color(fmt.Sprintf("#%02X%02X%02X", int(r), int(g), int(b)))
}

// renderVisualizer renders FFT visualization bars
func (m Model) renderVisualizer() string {
	if len(m.vizBars) == 0 {
		return strings.Repeat("▁", 20)
	}

	// Scale viz bars to fill available width.
	avail := m.width - 4
	if avail < len(m.vizBars) {
		avail = len(m.vizBars)
	}
	barWidth := avail / len(m.vizBars)
	if barWidth < 1 {
		barWidth = 1
	}

	// Use themed unicode map if available, fallback to built-in default.
	barChars := activePalette.VizUnicodeMap
	if len(barChars) == 0 {
		barChars = []string{" ", "▁", "▂", "▃", "▄", "▅", "▆", "▇", "█"}
	}

	isPlaying := m.p.State() == player.StatePlaying

	// Choose gradient or flat style per bar.
	gradient := activePalette.VizGradientPaused
	flatStyle := vizPausedStyle
	if isPlaying {
		gradient = activePalette.VizGradient
		flatStyle = vizPlayingStyle
	}

	var bars []string
	for _, h := range m.vizBars {
		idx := int(h * float64(len(barChars)-1))
		if idx >= len(barChars) {
			idx = len(barChars) - 1
		}
		ch := strings.Repeat(barChars[idx], barWidth)

		if len(gradient) >= 2 {
			col := lerpVizColor(gradient, h)
			bars = append(bars, lipgloss.NewStyle().Foreground(col).Render(ch))
		} else {
			bars = append(bars, flatStyle.Render(ch))
		}
	}
	return strings.Join(bars, "")
}

// renderProgressBar returns formatted progress bar with state/shuffle/loop indicators.
// Matches the classic jammer format: state shuffle loop elapsed |bar| total  vol%
func (m Model) renderProgressBar() string {
	p := activePalette

	// State glyph from palette (fallback to classic defaults).
	playingLetter := p.PlayingLetter
	if playingLetter == "" {
		playingLetter = "❚❚"
	}
	pausedLetter := p.PausedLetter
	if pausedLetter == "" {
		pausedLetter = "▶ "
	}
	stoppedLetter := p.StoppedLetter
	if stoppedLetter == "" {
		stoppedLetter = "■ "
	}

	state := stoppedLetter
	switch m.p.State() {
	case player.StatePlaying:
		state = playingLetter
	case player.StatePaused:
		state = pausedLetter
	case player.StateStopped:
		state = stoppedLetter
	}

	// Shuffle glyph
	shuffleLetter := p.ShuffleOnLetter
	if shuffleLetter == "" {
		shuffleLetter = "⇌ "
	}
	shuffleOffLetter := p.ShuffleOffLetter
	if shuffleOffLetter == "" {
		shuffleOffLetter = shuffleLetter
	}
	shuffleColor := p.ShuffleOnColor
	shuffleOffColor := p.ShuffleOffColor
	shuffleStr := ""
	if m.p.IsShuffle() {
		if shuffleColor != "" {
			shuffleStr = lipgloss.NewStyle().Foreground(shuffleColor).Render(shuffleLetter)
		} else {
			shuffleStr = shuffleLetter
		}
	} else {
		if shuffleOffColor != "" {
			shuffleStr = lipgloss.NewStyle().Foreground(shuffleOffColor).Render(shuffleOffLetter)
		} else {
			shuffleStr = shuffleOffLetter
		}
	}

	// Loop glyph — three modes: all (↻), off (↻ same glyph), once (1)
	loopAllLetter := p.LoopOnLetter
	if loopAllLetter == "" {
		loopAllLetter = " ⟳  "
	}
	loopOffLetter := p.LoopOffLetter
	if loopOffLetter == "" {
		loopOffLetter = " ↻  "
	}
	loopOnceLetter := p.LoopOnceLetter
	if loopOnceLetter == "" {
		loopOnceLetter = " 1  "
	}
	loopAllColor := p.LoopOnColor
	loopOffColor := p.LoopOffColor
	loopOnceColor := p.LoopOnceColor

	renderLoopGlyph := func(letter string, color lipgloss.Color) string {
		if color != "" {
			return lipgloss.NewStyle().Foreground(color).Render(letter)
		}
		return letter
	}

	loopGlyph := ""
	switch m.p.GetLoopMode() {
	case player.LoopAll:
		loopGlyph = renderLoopGlyph(loopAllLetter, loopAllColor)
	case player.LoopOff:
		loopGlyph = renderLoopGlyph(loopOffLetter, loopOffColor)
	case player.LoopOne:
		loopGlyph = renderLoopGlyph(loopOnceLetter, loopOnceColor)
	}
	elapsed := fmtTime(m.pos)
	total := fmtTime(m.dur)

	// Inner box text width: Width(boxW) + Padding(0,1) → inner text = boxW - 2
	// Format: state(2) + shuffle(2) + loop(4) + elapsed(~5) + " |"(2) + bar(N) + "| "(2) + total(~5) + "   "(3) + vol(4)
	// Fixed ≈ 2+2+4+5+2+2+5+3+4 = 29 chars → use textW - 29
	textW := m.songBoxTextWidth()
	barLen := textW - 29
	if barLen < 6 {
		barLen = 6
	}

	filled := 0
	if m.dur > 0 {
		filled = int((m.pos / m.dur) * float64(barLen))
	}
	if filled > barLen {
		filled = barLen
	}

	// Use themed timebar fill letter (fallback to █)
	fillChar := p.TimebarLetter
	if fillChar == "" {
		fillChar = "█"
	}
	bar := strings.Repeat(fillChar, filled) + strings.Repeat(" ", barLen-filled)
	barStyled := styleBar.Render("|") + styleBarFill.Render(bar) + styleBar.Render("|")

	vol := int(math.Round(float64(m.p.Volume()) * 100))
	volStr := fmt.Sprintf("%3d%%", vol)
	if m.p.IsMuted() {
		volStr = "MUTE"
		volStr = styleVolume.Copy().Foreground(p.VolumeMutedColor).Render(volStr)
	} else {
		volStr = styleVolume.Render(volStr)
	}

	timeStr := styleBar.Render(elapsed) + " " + barStyled + " " + styleBar.Render(total)
	return fmt.Sprintf("%s%s%s %s  %s", state, shuffleStr, loopGlyph, timeStr, volStr)
}

// ── Playlists view ────────────────────────────────────────────────────────────

func (m Model) renderPlaylists() string {
	var b strings.Builder

	if len(m.playlists) == 0 {
		b.WriteString(styleNotDL.Render("  No playlists found in "+m.plsDir) + "\n")
	} else {
		lh := m.plListHeight()
		end := m.poffset + lh
		if end > len(m.playlists) {
			end = len(m.playlists)
		}
		for i := m.poffset; i < end; i++ {
			name := strings.TrimSuffix(m.playlists[i], filepath.Ext(m.playlists[i]))
			name = truncate(name, m.width-4)
			line := "  " + name
			if i == m.pcursor {
				b.WriteString(styleSelected.Render(line))
			} else {
				b.WriteString(styleNormal.Render(line))
			}
			b.WriteByte('\n')
		}
		if len(m.playlists) > lh {
			b.WriteString(styleHelp.Render(fmt.Sprintf("  %d-%d / %d", m.poffset+1, end, len(m.playlists))))
			b.WriteByte('\n')
		}
	}

	b.WriteByte('\n')
	if m.showSongsPickMode {
		b.WriteString(styleHelp.Render(" enter/space: show songs  ↑/↓: navigate  esc: back  q: quit") + "\n")
	} else {
		b.WriteString(styleHelp.Render(" space: load playlist  ↑/↓: navigate  tab: back to songs  q: quit") + "\n")
	}
	return b.String()
}

// ── Progress bars ─────────────────────────────────────────────────────────────

// vizString renders the current viz bars as a string (empty when not playing).
func (m Model) vizString(nBars int) string {
	if m.p.State() != player.StatePlaying || len(m.vizBars) == 0 {
		return ""
	}
	const blocks = " ▁▂▃▄▅▆▇█"
	runes := []rune(blocks)
	var sb strings.Builder
	bars := m.vizBars
	if len(bars) > nBars {
		bars = bars[:nBars]
	}
	for _, h := range bars {
		idx := int(h * float64(len(runes)-1))
		if idx < 0 {
			idx = 0
		}
		if idx >= len(runes) {
			idx = len(runes) - 1
		}
		sb.WriteRune(runes[idx])
	}
	return styleBarFill.Render(sb.String())
}

func (m Model) progressBar() string {
	nBars := 0
	if m.p.State() == player.StatePlaying {
		nBars = m.vizNBars()
	}

	vizReserve := 0
	if nBars > 0 {
		vizReserve = nBars + 2
	}

	barW := m.width - 20 - vizReserve
	if barW < 10 {
		barW = 10
	}
	ratio := 0.0
	if m.dur > 0 {
		ratio = m.pos / m.dur
	}
	filled := int(math.Round(ratio * float64(barW)))
	if filled > barW {
		filled = barW
	}
	bar := styleBarFill.Render(strings.Repeat("━", filled)) +
		styleBar.Render(strings.Repeat("─", barW-filled))

	line := fmt.Sprintf("%s %s %s", fmtTime(m.pos), bar, fmtTime(m.dur))
	if nBars > 0 {
		line += "  " + m.vizString(nBars)
	}
	return line
}

func (m Model) volumeBar() string {
	barW := 10
	filled := int(math.Round(float64(m.p.Volume()) * float64(barW)))
	return styleBarFill.Render(strings.Repeat("█", filled)) +
		styleBar.Render(strings.Repeat("░", barW-filled))
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func fmtTime(s float64) string {
	if s < 0 {
		s = 0
	}
	m := int(s) / 60
	sec := int(s) % 60
	return fmt.Sprintf("%d:%02d", m, sec)
}

func truncate(s string, max int) string {
	if max <= 3 {
		return s
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max-3]) + "..."
}

// songsDir and plsDir are wired from main.go via an exported helper so
// the model doesn't need to import os itself for the path check.
func DefaultPlaylistsDir() string {
	return filepath.Join(dirs.Data(), "playlists")
}

// renderSettingsInput renders the text-input overlay for changing a settings value.
func (m Model) renderSettingsInput() string {
	var inner strings.Builder
	inner.WriteString(styleTitle.Render(" "+m.settingsInputPrompt) + "\n")
	inner.WriteString(styleHelp.Render(strings.Repeat("─", m.width-6)) + "\n")
	inner.WriteString("\n")
	cursor := styleBarFill.Render("█")
	inner.WriteString(styleHelp.Render("  ") + m.modalInput + cursor + "\n")
	inner.WriteString("\n")
	inner.WriteString(styleHelp.Render(strings.Repeat("─", m.width-6)) + "\n")
	inner.WriteString(styleHelp.Render("  Enter: confirm  ESC: cancel") + "\n")
	return m.renderOuterBox(m.currentSongPath(), inner.String())
}

// handleSettingsInputKey handles key input on the settings text-input overlay.
func (m Model) handleSettingsInputKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	keyStr := msg.String()

	if keyStr == "esc" {
		m.view = viewSettings
		m.modalInput = ""
		return m, nil
	}

	if keyStr == "enter" {
		m = m.commitSettingsInput()
		m.view = viewSettings
		m.modalInput = ""
		return m, nil
	}

	if keyStr == "backspace" && len([]rune(m.modalInput)) > 0 {
		runes := []rune(m.modalInput)
		m.modalInput = string(runes[:len(runes)-1])
		return m, nil
	}

	// Accept printable ASCII
	if len(keyStr) == 1 && keyStr[0] >= 32 && keyStr[0] <= 126 {
		m.modalInput += keyStr
		return m, nil
	}

	return m, nil
}

// commitSettingsInput parses modalInput and writes it into the correct Prefs field.
func (m Model) commitSettingsInput() Model {
	input := strings.TrimSpace(m.modalInput)
	switch m.settingsInputIdx {
	case 0: // Forward seconds
		if v, err := strconv.Atoi(input); err == nil && v > 0 {
			m.prefs.ForwardSeconds = v
		}
	case 1: // Rewind seconds
		if v, err := strconv.Atoi(input); err == nil && v > 0 {
			m.prefs.RewindSeconds = v
		}
	case 2: // Change Volume By (entered as integer percent)
		if v, err := strconv.Atoi(input); err == nil && v > 0 {
			m.prefs.ChangeVolumeBy = float64(v) / 100
		}
	case 8: // Soundcloud Client ID
		m.prefs.ClientID = input
	case 14: // Amount of time to skip Rss
		if v, err := strconv.Atoi(input); err == nil && v > 0 {
			m.prefs.RssSkipAfterTimeValue = v
		}
	case 18: // Search Result Count
		if v, err := strconv.Atoi(input); err == nil && v > 0 {
			if v > 20 {
				v = 20
			}
			m.prefs.SearchResultCount = v
		}
	}
	if m.prefs.SettingsPath != "" {
		saveSettings(m.prefs)
	}
	return m
}

// saveSettings writes the changed Prefs back to settings.json while preserving
// all unknown fields (same raw-map approach as saveBackend in main.go).
func saveSettings(p Prefs) {
	data, err := os.ReadFile(p.SettingsPath)
	if err != nil {
		return
	}
	// Strip UTF-8 BOM if present
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		data = data[3:]
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return
	}
	raw["forwardSeconds"] = p.ForwardSeconds
	raw["rewindSeconds"] = p.RewindSeconds
	raw["changeVolumeBy"] = p.ChangeVolumeBy
	raw["isAutoSave"] = p.IsAutoSave
	raw["isMediaButtons"] = p.IsMediaButtons
	raw["isVisualizer"] = p.IsVisualizer
	raw["clientID"] = p.ClientID
	raw["modifierKeyHelper"] = p.ModifierKeyHelper
	raw["isIgnoreErrors"] = p.IsIgnoreErrors
	raw["showPlaylistPosition"] = p.ShowPlaylistPosition
	raw["rssSkipAfterTime"] = p.RssSkipAfterTime
	raw["rssSkipAfterTimeValue"] = p.RssSkipAfterTimeValue
	raw["EnableQuickSearch"] = p.EnableQuickSearch
	raw["favoriteExplainer"] = p.FavoriteExplainer
	raw["EnableQuickPlayFromSearch"] = p.EnableQuickPlayFromSearch
	raw["searchResultCount"] = p.SearchResultCount
	raw["theme"] = p.Theme
	out, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(p.SettingsPath, out, 0o644)
}

// ── RSS feed view (Phase 3 #18) ───────────────────────────────────────────────

func fetchRssCmd(url string) tea.Cmd {
	return func() tea.Msg {
		feed, err := rss.Fetch(url)
		return rssFetchDoneMsg{feed: feed, err: err}
	}
}

func (m Model) renderRssFeed() string {
	var b strings.Builder
	if m.rssLoading {
		b.WriteString(styleRssTitle.Render("  RSS Feed") + "\n")
		b.WriteString(styleRssNormal.Render("  Fetching…") + "\n")
		return b.String()
	}
	if m.rssErr != "" {
		b.WriteString(styleInputTitleErr.Render("  RSS Feed — Error") + "\n")
		b.WriteString(styleErr.Render("  "+m.rssErr) + "\n")
		b.WriteString(styleRssNormal.Render("  E/ESC: back") + "\n")
		return b.String()
	}
	if m.rssFeed == nil {
		return ""
	}
	feed := m.rssFeed
	header := feed.Title
	if feed.Author != "" && feed.Author != "Unknown Author" {
		header += " — " + feed.Author
	}
	b.WriteString(styleRssTitle.Render("  "+header) + "\n")
	b.WriteString(strings.Repeat("─", m.width-2) + "\n")

	visibleRows := m.height - 5
	if visibleRows < 1 {
		visibleRows = 1
	}
	for i := m.rssOffset; i < len(feed.Items) && i < m.rssOffset+visibleRows; i++ {
		item := feed.Items[i]
		label := fmt.Sprintf("%d. %s", i+1, item.Title)
		if item.Author != "" && item.Author != feed.Author && item.Author != "Unknown Author" {
			label += " [" + item.Author + "]"
		}
		if i == m.rssCursor {
			b.WriteString(styleSelected.Render("> "+label) + "\n")
		} else {
			b.WriteString(styleRssNormal.Render("  "+label) + "\n")
		}
	}
	b.WriteString(strings.Repeat("─", m.width-2) + "\n")
	b.WriteString(styleRssNormal.Render(fmt.Sprintf("  ↑/↓: navigate  Enter: play  E/ESC: exit feed  (%d/%d)", m.rssCursor+1, len(feed.Items))) + "\n")
	return b.String()
}

func (m Model) handleRssFeedKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.rssFeed == nil {
		m.view = viewDefault
		return m, nil
	}
	items := m.rssFeed.Items
	keyStr := msg.String()

	switch {
	case keyStr == "esc" || m.kb.Is("ExitRssFeed", keyStr):
		// Restore original songs.
		if m.rssOriginSongs != nil {
			m.p.SetSongs(m.rssOriginSongs)
			m.songs = m.p.Songs()
			m.plsFile = m.rssOriginFile
			m.rssOriginSongs = nil
		}
		m.rssFeed = nil
		m.view = viewDefault
	case keyStr == "up" || keyStr == "k":
		if m.rssCursor > 0 {
			m.rssCursor--
			if m.rssCursor < m.rssOffset {
				m.rssOffset = m.rssCursor
			}
		}
	case keyStr == "down" || keyStr == "j":
		if m.rssCursor < len(items)-1 {
			m.rssCursor++
			visibleRows := m.height - 5
			if visibleRows < 1 {
				visibleRows = 1
			}
			if m.rssCursor >= m.rssOffset+visibleRows {
				m.rssOffset = m.rssCursor - visibleRows + 1
			}
		}
	case keyStr == "home":
		m.rssCursor = 0
		m.rssOffset = 0
	case keyStr == "end":
		if len(items) > 0 {
			m.rssCursor = len(items) - 1
			visibleRows := m.height - 5
			if visibleRows < 1 {
				visibleRows = 1
			}
			if m.rssCursor >= visibleRows {
				m.rssOffset = m.rssCursor - visibleRows + 1
			}
		}
	case keyStr == "enter" || keyStr == " ":
		if m.rssCursor < 0 || m.rssCursor >= len(items) {
			break
		}
		item := items[m.rssCursor]
		if item.URL == "" {
			m.setStatus("No playable URL for this episode")
			break
		}
		// Load the episode as a single-song playlist in the player.
		// Clear plsFile so saves don't overwrite the origin playlist file.
		entries := []playlist.Entry{{URL: item.URL, Title: item.Title, Author: item.Author}}
		m.p.LoadPlaylist(entries)
		m.songs = m.p.Songs()
		m.plsFile = ""
		m.playing = 0
		m.scursor = 0
		m.soffset = 0
		m.dlStates = make(map[int]*dlState)
		return m, tea.Batch(m.downloadIfNeeded(0), m.startViz(0))
	}
	return m, nil
}

func (m Model) renderChangeTheme() string {
	var b strings.Builder
	names := theme.Names()
	b.WriteString(styleTitle.Render("  Change Theme") + "\n")
	b.WriteString(strings.Repeat("─", m.width-2) + "\n")
	for i, name := range names {
		label := "  " + name
		if i == m.themeCursor {
			label = styleSelected.Render("> " + name)
		} else if name == m.prefs.Theme {
			label = stylePlaying.Render("  " + name + "  ✓")
		} else {
			label = styleNormal.Render(label)
		}
		b.WriteString(label + "\n")
	}
	b.WriteString(strings.Repeat("─", m.width-2) + "\n")
	b.WriteString(styleHelp.Render("  ↑/↓: navigate  Enter: apply  ESC: cancel") + "\n")
	return b.String()
}

func (m Model) handleChangeThemeKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	names := theme.Names()
	switch msg.String() {
	case "esc":
		m.view = viewDefault
	case "up", "k":
		if m.themeCursor > 0 {
			m.themeCursor--
		}
	case "down", "j":
		if m.themeCursor < len(names)-1 {
			m.themeCursor++
		}
	case "enter", " ":
		selected := names[m.themeCursor]
		m.prefs.Theme = selected
		applyTheme(theme.Get(selected))
		if m.prefs.SettingsPath != "" {
			saveSettings(m.prefs)
		}
		m.view = viewDefault
		m.setStatus("Theme: " + selected)
	}
	return m, nil
}

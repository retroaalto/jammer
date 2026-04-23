package ui

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/lipgloss"
	"github.com/jooapa/jammer/jammer-go/internal/downloader"
	"github.com/jooapa/jammer/jammer-go/internal/keybinds"
	jlog "github.com/jooapa/jammer/jammer-go/internal/log"
	"github.com/jooapa/jammer/jammer-go/internal/player"
	"github.com/jooapa/jammer/jammer-go/internal/playlist"
	"github.com/jooapa/jammer/jammer-go/internal/tags"
)

// ── Messages ──────────────────────────────────────────────────────────────────

type tickMsg time.Time
type vizTickMsg time.Time

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

// ── Views ─────────────────────────────────────────────────────────────────────

type viewKind int

const (
	viewDefault        viewKind = iota // 3-song snippet (prev/current/next)
	viewAll                            // full scrollable list
	viewPlaylists                      // playlist browser
	viewConfirmConvert                 // prompt to convert legacy playlist
	viewHelp                           // help screen (Phase 5)
	viewSettings                       // settings screen (Phase 6)
	viewRename                         // rename song input (Phase 7)
	viewInfo                           // song info overlay (Phase 7)
	viewAddSong                        // add song input (Phase 7)
)

// ── Download state per song ───────────────────────────────────────────────────

type dlState struct {
	active  bool
	frac    float64
	message string
	err     error
}

// ── Styles ────────────────────────────────────────────────────────────────────

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
)

// ── Model ─────────────────────────────────────────────────────────────────────

type Model struct {
	// core
	p        *player.Player
	songsDir string
	plsDir   string
	kb       *keybinds.Keybinds // loaded keybindings

	// config
	seekStep int  // seconds per seek keypress
	autoPlay bool // play index 0 on Init (set when launched with -p)

	// view
	view           viewKind
	helpPageNum    int // current page in help screen (0-indexed)
	settingsCursor int // current settings item cursor (0-indexed)
	width          int
	height         int

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
	filter       string // current filter text (empty = no filter)
	filtering    bool   // true while the user is typing a filter
	filteredIdxs []int  // indices into songs that match the filter (nil = no filter active)

	// visualizer
	vizBars    []float64 // current smoothed bar heights (0.0–1.0)
	vizTargets []float64 // FFT target heights bars animate toward
	vizRunning bool      // true while the 100ms viz tick is scheduled

	// playlists view
	playlists []string // filenames (basename only) in plsDir
	pcursor   int
	poffset   int

	// legacy convert prompt
	convertFile    string           // path of legacy playlist pending conversion
	convertEntries []playlist.Entry // parsed entries from the legacy file

	// error display
	lastError   string    // most recent download error message (empty = none)
	lastErrTime time.Time // when lastError was set; cleared after 8 s by tickMsg

	// Phase 7: modal inputs
	modalInput string // current text in modal dialogs (rename, add song, etc.)
	modalIdx   int    // index for rename/info view (which song)
}

func New(p *player.Player, songsDir, plsDir string, seekStep int, defaultView string, kb *keybinds.Keybinds) Model {
	return NewWithPlaylist(p, songsDir, plsDir, "", seekStep, defaultView, kb)
}

func NewWithPlaylist(p *player.Player, songsDir, plsDir, plsFile string, seekStep int, defaultView string, kb *keybinds.Keybinds) Model {
	if seekStep <= 0 {
		seekStep = 2
	}
	// Determine initial view based on defaultView setting
	initialView := viewDefault
	if defaultView == "all" {
		initialView = viewAll
	}

	m := Model{
		p:        p,
		songsDir: songsDir,
		plsDir:   plsDir,
		kb:       kb,
		seekStep: seekStep,
		songs:    p.Songs(),
		playing:  p.Index(),
		dlStates: make(map[int]*dlState),
		view:     initialView,
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

func vizTick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return vizTickMsg(t)
	})
}

func (m Model) Init() tea.Cmd {
	if m.autoPlay && len(m.songs) > 0 {
		return tea.Batch(tick(), func() tea.Msg {
			if err := m.p.PlayIndex(0); err != nil {
				jlog.Errorf("auto-play on start: %v", err)
			}
			return nil
		}, m.downloadIfNeeded(0), m.startViz(0))
	}
	return tick()
}

// ── Update ────────────────────────────────────────────────────────────────────

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tickMsg:
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

	case vizTickMsg:
		if m.p.State() == player.StatePlaying {
			nBars := m.vizNBars()
			m.stepViz(nBars)
			return m, vizTick()
		}
		// Player stopped/paused — let the tick lapse; will be restarted on next play.
		m.vizRunning = false

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
	return vizTick()
}

// vizNBars returns the number of visualizer bars for the current terminal width.
// Returns 0 if the terminal is too narrow to show a viz.
func (m Model) vizNBars() int {
	if m.width <= 50 {
		return 0
	}
	n := (m.width - 40) / 3
	if n > 20 {
		n = 20
	}
	if n < 4 {
		return 0
	}
	return n
}

func (m *Model) stepViz(nBars int) {
	fft := m.p.FFTData() // 256 bins, or nil
	if fft == nil || nBars < 1 {
		// No data: decay bars toward zero.
		for i := range m.vizBars {
			m.vizBars[i] *= 0.7
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
	// Log scale spans 80 Hz – 16000 Hz so bars cover roughly equal octaves,
	// matching how human hearing perceives pitch.
	const (
		binWidth = 44100.0 / 512.0
		fLow     = 80.0
		fHigh    = 16000.0
	)
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
		avg = math.Sqrt(avg) * 3
		// Rising gain: compensate for natural high-frequency roll-off.
		// Ramps from 1× at the left edge to 5× at the right edge.
		gain := 1.0 + 4.0*(float64(i)/float64(nBars-1))
		avg *= gain
		if avg > 1 {
			avg = 1
		}
		m.vizTargets[i] = avg
	}

	// Smooth bars toward targets: fast attack (0.6), slower decay (0.3).
	for i := range m.vizBars {
		delta := m.vizTargets[i] - m.vizBars[i]
		if delta > 0 {
			m.vizBars[i] += delta * 0.6
		} else {
			m.vizBars[i] += delta * 0.3
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

	// Quit
	if m.kb.Is("Quit", keyStr) {
		m.p.Stop()
		return m, tea.Quit
	}

	// ToMainMenu (Escape) - only if not in confirm convert
	if m.kb.Is("ToMainMenu", keyStr) && m.view != viewConfirmConvert {
		if m.view == viewDefault || m.view == viewAll {
			// If not in songs view, return to default view
			if m.view != viewDefault {
				m.view = viewDefault
			}
		}
		return m, nil
	}

	// View switching (Tab, Shift+F, Shift+O)
	if m.kb.Is("CommandHelpScreen", keyStr) || m.kb.Is("ListAllPlaylists", keyStr) || m.kb.Is("PlayOtherPlaylist", keyStr) {
		if m.view == viewConfirmConvert {
			return m, nil // ignore during confirm prompt
		}
		if m.view == viewDefault || m.view == viewAll {
			m.view = viewPlaylists
			m.reloadPlaylists()
		} else {
			m.view = viewDefault
		}
		return m, nil
	}

	// Default handler routing
	if m.view == viewConfirmConvert {
		return m.handleConfirmConvertKey(msg)
	}
	if m.view == viewPlaylists {
		return m.handlePlaylistKey(msg)
	}
	return m.handleSongKey(msg)
}

func (m Model) handleConfirmConvertKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		// Convert: overwrite the legacy file with JSONL format.
		_ = playlist.Save(m.convertFile, m.convertEntries)
		jlog.Infof("convert: saved JSONL to %q", m.convertFile)
		fallthrough
	case "n", "N", "esc":
		// Load entries but don't set plsFile — legacy file stays untouched
		// and no metadata updates will be written back to it.
		m.applyPlaylist("", m.convertEntries)
		m.convertFile = ""
		m.convertEntries = nil
		m.view = viewDefault
	}
	return m, nil
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
	if keyStr == "s" {
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
	if m.kb.Is("Forward5s", keyStr) || keyStr == "right" || keyStr == "l" {
		m.p.SeekForward(float64(m.seekStep))
		jlog.Infof("ui: seek +%ds", m.seekStep)
		return m, nil
	}

	// Seek backward
	if m.kb.Is("Backwards5s", keyStr) || keyStr == "left" || keyStr == "h" {
		m.p.SeekBackward(float64(m.seekStep))
		jlog.Infof("ui: seek -%ds", m.seekStep)
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
		// TODO: implement mute in player if not already there
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
	if m.kb.Is("Search", keyStr) || m.kb.Is("SearchInPlaylist", keyStr) || keyStr == "/" {
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
	if m.kb.Is("ToggleInfo", keyStr) || m.kb.Is("CurrentState", keyStr) {
		m.view = viewInfo
		return m, nil
	}

	// Rename song
	if m.kb.Is("RenameSong", keyStr) {
		m.view = viewRename
		return m, nil
	}

	// Add song to playlist
	if m.kb.Is("AddSongToPlaylist", keyStr) {
		m.view = viewAddSong
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
	case "space", "enter":
		if m.pcursor >= 0 && m.pcursor < len(m.playlists) {
			jlog.Infof("ui: loading playlist %q", m.playlists[m.pcursor])
			m.loadPlaylist(m.playlists[m.pcursor])
			m.view = viewDefault
			m.scursor = 0
			m.soffset = 0
		}
	}
	return m, nil
}

func (m *Model) loadPlaylist(filename string) {
	path := filepath.Join(m.plsDir, filename)
	entries, legacy, err := playlist.Load(path, m.songsDir)
	if err != nil {
		return
	}

	if legacy {
		// Don't load yet — ask the user first.
		m.convertFile = path
		m.convertEntries = entries
		m.view = viewConfirmConvert
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
		if strings.Contains(strings.ToLower(s.DisplayTitle()), lower) {
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
	reserved := 14
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

func (m Model) View() tea.View {
	if m.width == 0 {
		v := tea.NewView("loading...")
		v.AltScreen = true
		return v
	}
	var b strings.Builder

	// ── Tabs ──────────────────────────────────────────────────────────────────
	songs := styleTabInactive.Render("Songs")
	pls := styleTabInactive.Render("Playlists")
	if m.view == viewDefault || m.view == viewAll {
		songs = styleTabActive.Render("Songs")
	} else {
		pls = styleTabActive.Render("Playlists")
	}
	b.WriteString(styleTitle.Render("  jammer") + "  " + songs + "  " + pls + "\n\n")

	// Render based on current view
	if m.view == viewConfirmConvert {
		b.WriteString(m.renderConfirmConvert())
	} else if m.view == viewPlaylists {
		b.WriteString(m.renderPlaylists())
	} else if m.view == viewHelp {
		b.WriteString(m.renderHelp())
	} else if m.view == viewSettings {
		b.WriteString(m.renderSettings())
	} else if m.view == viewDefault {
		b.WriteString(m.renderSongsDefault())
	} else if m.view == viewAll {
		b.WriteString(m.renderSongsAll())
	} else if m.view == viewRename {
		b.WriteString(m.renderRename())
	} else if m.view == viewInfo {
		b.WriteString(m.renderInfo())
	} else if m.view == viewAddSong {
		b.WriteString(m.renderAddSong())
	}

	v := tea.NewView(b.String())
	v.AltScreen = true
	return v
}

// ── Songs view ────────────────────────────────────────────────────────────────

func (m Model) renderSongsDefault() string {
	var b strings.Builder

	// Render 3-song snippet (prev / current / next)
	if len(m.songs) == 0 {
		b.WriteString(styleHelp.Render("  No songs loaded\n"))
	} else {
		// Build the 3-song display
		prevIdx := m.playing - 1
		if prevIdx < 0 {
			prevIdx = len(m.songs) - 1 // wrap around
		}
		currIdx := m.playing
		nextIdx := m.playing + 1
		if nextIdx >= len(m.songs) {
			nextIdx = 0 // wrap around
		}

		// Draw bordered box with 3-song snippet
		b.WriteString(lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("61")).
			Padding(0, 1).
			Width(m.width - 8).
			Render(fmt.Sprintf("playlist %s", filepath.Base(m.plsFile))))
		b.WriteString("\n")

		// Previous song
		prevTitle := truncate(m.songs[prevIdx].DisplayTitle(), m.width-20)
		b.WriteString(styleHelp.Render(fmt.Sprintf(" previous : %s\n", prevTitle)))

		// Current song (highlighted)
		currTitle := truncate(m.songs[currIdx].DisplayTitle(), m.width-20)
		b.WriteString(stylePlaying.Render(fmt.Sprintf(" current  : %s\n", currTitle)))

		// Next song
		nextTitle := truncate(m.songs[nextIdx].DisplayTitle(), m.width-20)
		b.WriteString(styleHelp.Render(fmt.Sprintf(" next     : %s\n", nextTitle)))
	}

	// Spacer rows to fill screen
	lines := 3 + 10 // header + song box + spacer + progress + help
	for i := lines; i < m.height-10; i++ {
		b.WriteString("\n")
	}

	// Mini help bar
	b.WriteString(lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("61")).
		Padding(0, 1).
		Width(m.width - 8).
		Render("[H] for help | [C] for settings | [F] for playlist"))
	b.WriteString("\n")

	// Visualizer row
	b.WriteString(" ")
	b.WriteString(m.renderVisualizer())
	b.WriteString("\n")

	// Progress/time bar
	b.WriteString(lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("61")).
		Padding(0, 1).
		Width(m.width - 8).
		Render(m.renderProgressBar()))
	b.WriteString("\n")

	return b.String()
}

func (m Model) renderSongsAll() string {
	var b strings.Builder

	// Filter prompt (shown above the list when filtering or a filter is active).
	if m.filtering {
		cursor := styleBarFill.Render("█")
		b.WriteString(styleHelp.Render(" / ") + m.filter + cursor + "\n")
	} else if m.filter != "" {
		b.WriteString(styleHelp.Render(fmt.Sprintf(" / %s  (esc to clear)", m.filter)) + "\n")
	}

	total := m.filterLen()
	lh := m.songListHeight()
	end := m.soffset + lh
	if end > total {
		end = total
	}

	for vi := m.soffset; vi < end; vi++ {
		song, realIdx := m.filterSong(vi)
		title := truncate(song.DisplayTitle(), m.width-12)

		// Left status prefix
		prefix := "  "
		if realIdx == m.playing {
			switch m.p.State() {
			case player.StatePlaying:
				prefix = "> "
			case player.StatePaused:
				prefix = "| "
			default:
				prefix = "  "
			}
		}

		// Right download status
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

		line := prefix + title

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
		b.WriteString(rendered + suffix + "\n")
	}

	if total > lh {
		b.WriteString(styleHelp.Render(fmt.Sprintf("  %d-%d / %d", m.soffset+1, end, total)))
		b.WriteByte('\n')
	}
	b.WriteByte('\n')

	// Now playing
	nowTitle := "—"
	if m.playing >= 0 && m.playing < len(m.songs) {
		nowTitle = truncate(m.songs[m.playing].DisplayTitle(), m.width-12)
	}
	icon := " "
	switch m.p.State() {
	case player.StatePlaying:
		icon = "▶"
	case player.StatePaused:
		icon = "⏸"
	case player.StateStopped:
		icon = "■"
	}
	b.WriteString(styleTitle.Render(fmt.Sprintf(" %s  %s", icon, nowTitle)) + "\n")
	b.WriteString(" " + m.progressBar() + "\n")
	vol := int(math.Round(float64(m.p.Volume()) * 100))
	loopLabels := [3]string{"loop:all", "loop:off", "loop:one"}
	loopLabel := loopLabels[int(m.p.GetLoopMode())%3]
	shuffleLabel := ""
	if m.p.IsShuffle() {
		shuffleLabel = "   shuffle"
	}
	b.WriteString(styleVolume.Render(fmt.Sprintf(" vol: %3d%%  %s   %s%s", vol, m.volumeBar(), loopLabel, shuffleLabel)) + "\n\n")

	if m.lastError != "" {
		errMsg := truncate("! "+m.lastError, m.width-3)
		b.WriteString(styleErr.Render(" "+errMsg) + "\n")
	}
	b.WriteString(styleHelp.Render(" space/enter: play/pause  s: stop  n: next  p: prev  r: random  R: shuffle  d: download") + "\n")
	b.WriteString(styleHelp.Render(fmt.Sprintf(" /: filter  ←/→: seek %ds  +/-: vol  L: loop  del: remove  S+del: +file  tab: playlists  q: quit", m.seekStep)) + "\n")
	return b.String()
}

// renderHelp returns the help screen with paginated keybindings
func (m Model) renderHelp() string {
	var b strings.Builder

	// All keybindings to display
	allBindings := m.getHelpBindings()

	// Calculate pages (10 keybindings per page, 2 columns = 20 per page)
	itemsPerPage := 20
	totalPages := (len(allBindings) + itemsPerPage - 1) / itemsPerPage

	if totalPages == 0 {
		totalPages = 1
	}

	// Clamp page number
	page := m.helpPageNum
	if page >= totalPages {
		page = totalPages - 1
	}
	if page < 0 {
		page = 0
	}

	// Get items for this page
	start := page * itemsPerPage
	end := start + itemsPerPage
	if end > len(allBindings) {
		end = len(allBindings)
	}
	pageItems := allBindings[start:end]

	// Render header
	pageIndicator := fmt.Sprintf("(%d/%d)", page+1, totalPages)
	b.WriteString(styleTitle.Render(fmt.Sprintf("  Keybindings %s\n", pageIndicator)))
	b.WriteString(strings.Repeat("─", m.width-2) + "\n")

	// Render 2-column table
	// Split items into left and right columns
	mid := (len(pageItems) + 1) / 2

	for i := 0; i < mid; i++ {
		// Left column
		leftItem := pageItems[i]
		leftKey := keybinds.GetDisplay(leftItem.key)
		leftDesc := truncate(leftItem.desc, 25)
		left := fmt.Sprintf("  %-30s %-25s", leftKey, leftDesc)

		// Right column
		var right string
		if i+mid < len(pageItems) {
			rightItem := pageItems[i+mid]
			rightKey := keybinds.GetDisplay(rightItem.key)
			rightDesc := truncate(rightItem.desc, 25)
			right = fmt.Sprintf("  %-30s %-25s", rightKey, rightDesc)
		}

		b.WriteString(styleHelp.Render(left + right + "\n"))
	}

	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", m.width-2) + "\n")
	b.WriteString(styleHelp.Render("  ←/→: prev/next page  ESC: back  q: quit\n"))

	return b.String()
}

// renderSettings renders the settings screen (Phase 6)
func (m Model) renderSettings() string {
	var b strings.Builder

	// Render header
	b.WriteString(styleTitle.Render("  Settings\n"))
	b.WriteString(strings.Repeat("─", m.width-2) + "\n")

	// Settings list
	settings := []struct {
		name  string
		value string
	}{
		{"Seek step", "5s"},
		{"Volume step", "5%"},
		{"Auto-save", "on"},
		{"Visualizer", "on"},
		{"Default view", "3-song"},
	}

	// Clamp cursor
	if m.settingsCursor >= len(settings) {
		m.settingsCursor = len(settings) - 1
	}
	if m.settingsCursor < 0 {
		m.settingsCursor = 0
	}

	for i, s := range settings {
		line := fmt.Sprintf("  %-20s: %s", s.name, s.value)
		if i == m.settingsCursor {
			b.WriteString(styleSelected.Render(line) + "\n")
		} else {
			b.WriteString(styleHelp.Render(line) + "\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", m.width-2) + "\n")
	b.WriteString(styleHelp.Render("  ↑/↓: navigate  ESC: back  q: quit\n"))

	return b.String()
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
		"CurrentState":           "Current state",
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
		return m, nil
	}

	if keyStr == "q" || m.kb.Is("Quit", keyStr) {
		m.p.Stop()
		return m, tea.Quit
	}

	// Navigation with up/down
	if keyStr == "up" || m.kb.Is("PlaylistViewScrollup", keyStr) {
		if m.settingsCursor > 0 {
			m.settingsCursor--
		}
		return m, nil
	}

	if keyStr == "down" || m.kb.Is("PlaylistViewScrolldown", keyStr) {
		// 5 settings items
		if m.settingsCursor < 4 {
			m.settingsCursor++
		}
		return m, nil
	}

	// Phase 6 placeholder: settings values can be adjusted with left/right later
	return m, nil
}

// handleRenameKey handles key input on the rename dialog
func (m Model) handleRenameKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	keyStr := msg.String()

	if keyStr == "esc" {
		m.view = viewDefault
		m.modalInput = ""
		m.modalIdx = -1
		return m, nil
	}

	if keyStr == "enter" {
		// TODO: Actually rename the song
		// For now, just close the dialog
		m.view = viewDefault
		m.modalInput = ""
		m.modalIdx = -1
		return m, nil
	}

	// Character input
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
		// TODO: Actually add the song to the playlist
		// For now, just close the dialog
		m.view = viewDefault
		m.modalInput = ""
		return m, nil
	}

	// Character input - allow URLs and paths
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

// renderRename renders the rename song dialog
func (m Model) renderRename() string {
	var b strings.Builder

	b.WriteString(styleTitle.Render("  Rename Song\n"))
	b.WriteString(strings.Repeat("─", m.width-2) + "\n")

	if m.modalIdx >= 0 && m.modalIdx < len(m.songs) {
		currentName := m.songs[m.modalIdx].DisplayTitle()
		b.WriteString(styleHelp.Render(fmt.Sprintf("  Current: %s\n\n", currentName)))
	}

	// Input line
	cursor := styleBarFill.Render("█")
	b.WriteString(styleHelp.Render("  New name: ") + m.modalInput + cursor + "\n")

	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", m.width-2) + "\n")
	b.WriteString(styleHelp.Render("  Enter: confirm  ESC: cancel\n"))

	return b.String()
}

// renderInfo renders the song information overlay
func (m Model) renderInfo() string {
	var b strings.Builder

	b.WriteString(styleTitle.Render("  Song Information\n"))
	b.WriteString(strings.Repeat("─", m.width-2) + "\n")

	if m.modalIdx >= 0 && m.modalIdx < len(m.songs) {
		song := m.songs[m.modalIdx]
		b.WriteString(styleHelp.Render(fmt.Sprintf("  Title:    %s\n", song.DisplayTitle())))
		b.WriteString(styleHelp.Render(fmt.Sprintf("  Author:   %s\n", song.Author)))
		b.WriteString(styleHelp.Render(fmt.Sprintf("  URL:      %s\n", song.URL)))
		b.WriteString(styleHelp.Render(fmt.Sprintf("  File:     %s\n", filepath.Base(song.Path))))
		b.WriteString(styleHelp.Render(fmt.Sprintf("  Downloaded: %v\n", song.Downloaded())))
	} else {
		b.WriteString(styleHelp.Render("  No song selected\n"))
	}

	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", m.width-2) + "\n")
	b.WriteString(styleHelp.Render("  ESC: close\n"))

	return b.String()
}

// renderAddSong renders the add song input dialog
func (m Model) renderAddSong() string {
	var b strings.Builder

	b.WriteString(styleTitle.Render("  Add Song\n"))
	b.WriteString(strings.Repeat("─", m.width-2) + "\n")
	b.WriteString(styleHelp.Render("  Enter URL or file path to add to playlist\n\n"))

	// Input line
	cursor := styleBarFill.Render("█")
	b.WriteString(styleHelp.Render("  ") + m.modalInput + cursor + "\n")

	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", m.width-2) + "\n")
	b.WriteString(styleHelp.Render("  Enter: add  ESC: cancel\n"))

	return b.String()
}

// ── Visualizer ────────────────────────────────────────────────────────────────

// renderVisualizer renders FFT visualization bars
func (m Model) renderVisualizer() string {
	if len(m.vizBars) == 0 {
		return ""
	}

	// Scale viz bars to fit available width (rough estimation)
	barWidth := (m.width - 10) / len(m.vizBars)
	if barWidth < 1 {
		barWidth = 1
	}

	var bars []string
	barChars := []rune{' ', '▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

	for _, h := range m.vizBars {
		idx := int(h * float64(len(barChars)-1))
		if idx >= len(barChars) {
			idx = len(barChars) - 1
		}
		bars = append(bars, string(barChars[idx]))
	}
	return styleBarFill.Render(strings.Join(bars, ""))
}

// renderProgressBar returns formatted progress bar with state/shuffle/loop indicators
func (m Model) renderProgressBar() string {
	// State glyph
	state := " "
	switch m.p.State() {
	case player.StatePlaying:
		state = "▶"
	case player.StatePaused:
		state = "❚❚"
	case player.StateStopped:
		state = "■"
	}

	// Shuffle glyph
	shuffle := " "
	if m.p.IsShuffle() {
		shuffle = "⇌"
	}

	// Loop glyph
	loopGlyph := " "
	switch m.p.GetLoopMode() {
	case 0: // all
		loopGlyph = "↻"
	case 1: // off
		loopGlyph = " "
	case 2: // one
		loopGlyph = "↺"
	}

	elapsed := fmtTime(m.pos)
	total := fmtTime(m.dur)
	barLen := m.width - 30
	if barLen < 10 {
		barLen = 10
	}

	filled := 0
	if m.dur > 0 {
		filled = int((m.pos / m.dur) * float64(barLen))
	}
	if filled > barLen {
		filled = barLen
	}

	bar := strings.Repeat("━", filled) + strings.Repeat("─", barLen-filled)
	vol := int(math.Round(float64(m.p.Volume()) * 100))

	return fmt.Sprintf("%s %s %s  %s |%s| %s  %3d%%", state, shuffle, loopGlyph, elapsed, bar, total, vol)
}

// ── Playlists view ────────────────────────────────────────────────────────────

func (m Model) renderConfirmConvert() string {
	var b strings.Builder
	name := filepath.Base(m.convertFile)
	b.WriteString(styleHelp.Render("  Legacy playlist format detected: "+name) + "\n\n")
	b.WriteString("  Convert to new JSONL format? " +
		stylePlaying.Render("y") + " yes  " +
		styleNotDL.Render("n") + " no\n")
	return b.String()
}

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
	b.WriteString(styleHelp.Render(" space: load playlist  ↑/↓: navigate  tab: back to songs  q: quit") + "\n")
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
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "jammer", "playlists")
}

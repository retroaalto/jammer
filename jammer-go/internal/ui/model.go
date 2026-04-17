package ui

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/lipgloss"
	"github.com/jooapa/jammer/jammer-go/internal/downloader"
	jlog "github.com/jooapa/jammer/jammer-go/internal/log"
	"github.com/jooapa/jammer/jammer-go/internal/player"
	"github.com/jooapa/jammer/jammer-go/internal/playlist"
	"github.com/jooapa/jammer/jammer-go/internal/tags"
)

// ── Messages ──────────────────────────────────────────────────────────────────

type tickMsg time.Time

type downloadProgressMsg struct {
	index      int
	prog       downloader.Progress
	progressCh <-chan downloader.Progress
	doneCh     <-chan downloadDoneMsg
}

type downloadDoneMsg struct {
	index int
	path  string
	err   error
}

// ── Views ─────────────────────────────────────────────────────────────────────

type viewKind int

const (
	viewSongs     viewKind = iota // song list
	viewPlaylists                 // playlist browser
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

	// view
	view   viewKind
	width  int
	height int

	// songs view
	songs       []player.Song
	scursor     int
	soffset     int
	playing     int
	prevPlaying int // track changes detected in tickMsg
	pos, dur    float64
	dlStates    map[int]*dlState
	plsFile     string // absolute path of currently loaded playlist (empty = songs dir)

	// playlists view
	playlists []string // filenames (basename only) in plsDir
	pcursor   int
	poffset   int
}

func New(p *player.Player, songsDir, plsDir string) Model {
	m := Model{
		p:        p,
		songsDir: songsDir,
		plsDir:   plsDir,
		songs:    p.Songs(),
		playing:  p.Index(),
		dlStates: make(map[int]*dlState),
	}
	m.reloadPlaylists()
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

func (m Model) Init() tea.Cmd {
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
			return m, tea.Batch(tick(), m.downloadIfNeeded(newIdx))
		}
		m.playing = newIdx
		return m, tick()

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
		} else {
			ds.frac = 1
			ds.err = nil
			jlog.Infof("download done index=%d path=%q", msg.index, msg.path)
			m.p.UpdateSongPath(msg.index, msg.path)

			// Read ID3 tags from the freshly downloaded file and push them
			// into the player so the UI shows proper title/artist immediately.
			if info, err := tags.Read(msg.path); err == nil && info.Title != "" {
				m.p.UpdateSongTags(msg.index, info.Title, info.Artist)
				jlog.Infof("download tags index=%d title=%q artist=%q", msg.index, info.Title, info.Artist)
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
			}
		}

	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	viewName := "songs"
	if m.view == viewPlaylists {
		viewName = "playlists"
	}
	jlog.Key(msg.String(), viewName)

	switch msg.String() {
	case "q", "ctrl+c":
		m.p.Stop()
		return m, tea.Quit

	case "tab":
		if m.view == viewSongs {
			m.view = viewPlaylists
			m.reloadPlaylists()
		} else {
			m.view = viewSongs
		}

	default:
		if m.view == viewPlaylists {
			return m.handlePlaylistKey(msg)
		}
		return m.handleSongKey(msg)
	}
	return m, nil
}

func (m Model) handleSongKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.scursor > 0 {
			m.scursor--
			m.clampSongScroll()
		}
	case "down", "j":
		if m.scursor < len(m.songs)-1 {
			m.scursor++
			m.clampSongScroll()
		}
	case "enter":
		if m.scursor == m.playing && m.p.State() != player.StateStopped {
			jlog.Infof("ui: pause song index=%d", m.scursor)
			if err := m.p.Pause(); err != nil {
				jlog.Errorf("ui: pause failed: %v", err)
			}
		} else {
			jlog.Infof("ui: play song index=%d title=%q", m.scursor, m.songs[m.scursor].DisplayTitle())
			if err := m.p.PlayIndex(m.scursor); err != nil {
				jlog.Errorf("ui: PlayIndex failed index=%d: %v", m.scursor, err)
			}
			m.playing = m.scursor
			m.prevPlaying = m.scursor
			// Download if the song isn't local yet.
			return m, m.downloadIfNeeded(m.scursor)
		}
	case " ":
		jlog.Infof("ui: toggle pause index=%d", m.playing)
		if err := m.p.Pause(); err != nil {
			jlog.Errorf("ui: pause toggle failed: %v", err)
		}
	case "s":
		jlog.Infof("ui: stop")
		m.p.Stop()
	case "n":
		if err := m.p.Next(); err != nil {
			jlog.Errorf("ui: next failed: %v", err)
		}
		m.playing = m.p.Index()
		m.prevPlaying = m.playing
		m.scursor = m.playing
		m.clampSongScroll()
		jlog.Infof("ui: next → index=%d", m.playing)
		return m, m.downloadIfNeeded(m.playing)
	case "b":
		if err := m.p.Prev(); err != nil {
			jlog.Errorf("ui: prev failed: %v", err)
		}
		m.playing = m.p.Index()
		m.prevPlaying = m.playing
		m.scursor = m.playing
		m.clampSongScroll()
		jlog.Infof("ui: prev → index=%d", m.playing)
		return m, m.downloadIfNeeded(m.playing)
	case "right", "l":
		m.p.SeekForward(10)
		jlog.Info("ui: seek +10s")
	case "left", "h":
		m.p.SeekBackward(10)
		jlog.Info("ui: seek -10s")
	case "+", "=":
		m.p.SetVolume(m.p.Volume() + 0.05)
		jlog.Infof("ui: volume up → %.0f%%", float64(m.p.Volume())*100)
	case "-":
		m.p.SetVolume(m.p.Volume() - 0.05)
		jlog.Infof("ui: volume down → %.0f%%", float64(m.p.Volume())*100)
	case "d":
		jlog.Infof("ui: download requested index=%d url=%q", m.scursor, m.songs[m.scursor].URL)
		return m, m.startDownload(m.scursor)
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
	case "enter", " ":
		if m.pcursor >= 0 && m.pcursor < len(m.playlists) {
			jlog.Infof("ui: loading playlist %q", m.playlists[m.pcursor])
			m.loadPlaylist(m.playlists[m.pcursor])
			m.view = viewSongs
			m.scursor = 0
			m.soffset = 0
		}
	}
	return m, nil
}

func (m *Model) loadPlaylist(filename string) {
	path := filepath.Join(m.plsDir, filename)
	entries, err := playlist.Load(path, m.songsDir)
	if err != nil {
		return
	}
	m.p.LoadPlaylist(entries)
	m.songs = m.p.Songs()
	m.dlStates = make(map[int]*dlState)
	m.plsFile = path

	// Write back enriched metadata (tags filled in for downloaded songs).
	m.saveCurrentPlaylist()
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
	if ds := m.dlStates[i]; ds != nil && (ds.active || ds.frac >= 1) {
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
	if song.URL == "" || song.Downloaded() {
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
		path, err := downloader.Download(context.Background(), song.URL, m.songsDir, progressCh)
		doneCh <- downloadDoneMsg{index: i, path: path, err: err}
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
	if m.view == viewSongs {
		songs = styleTabActive.Render("Songs")
	} else {
		pls = styleTabActive.Render("Playlists")
	}
	b.WriteString(styleTitle.Render("  jammer") + "  " + songs + "  " + pls + "\n\n")

	if m.view == viewPlaylists {
		b.WriteString(m.renderPlaylists())
	} else {
		b.WriteString(m.renderSongs())
	}

	v := tea.NewView(b.String())
	v.AltScreen = true
	return v
}

// ── Songs view ────────────────────────────────────────────────────────────────

func (m Model) renderSongs() string {
	var b strings.Builder

	lh := m.songListHeight()
	end := m.soffset + lh
	if end > len(m.songs) {
		end = len(m.songs)
	}

	for i := m.soffset; i < end; i++ {
		song := m.songs[i]
		title := truncate(song.DisplayTitle(), m.width-12)

		// Left status prefix
		prefix := "  "
		if i == m.playing {
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
		if ds, ok := m.dlStates[i]; ok && ds != nil {
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
		case !song.Downloaded() && (m.dlStates[i] == nil || !m.dlStates[i].active):
			if i == m.scursor {
				rendered = styleSelected.Render(line)
			} else {
				rendered = styleNotDL.Render(line)
			}
		case i == m.scursor && i == m.playing:
			rendered = styleSelected.Render(line)
		case i == m.scursor:
			rendered = styleSelected.Render(line)
		case i == m.playing:
			rendered = stylePlaying.Render(line)
		default:
			rendered = styleNormal.Render(line)
		}
		b.WriteString(rendered + suffix + "\n")
	}

	if len(m.songs) > lh {
		b.WriteString(styleHelp.Render(fmt.Sprintf("  %d-%d / %d", m.soffset+1, end, len(m.songs))))
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
	b.WriteString(styleVolume.Render(fmt.Sprintf(" vol: %3d%%  %s", vol, m.volumeBar())) + "\n\n")

	b.WriteString(styleHelp.Render(" enter: play  space/p: pause  s: stop  n: next  b: prev  d: download") + "\n")
	b.WriteString(styleHelp.Render(" ←/→: seek 10s  +/-: volume  tab: playlists  q: quit") + "\n")
	return b.String()
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
	b.WriteString(styleHelp.Render(" enter: load playlist  ↑/↓: navigate  tab: back to songs  q: quit") + "\n")
	return b.String()
}

// ── Progress bars ─────────────────────────────────────────────────────────────

func (m Model) progressBar() string {
	barW := m.width - 20
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
	return fmt.Sprintf("%s %s %s", fmtTime(m.pos), bar, fmtTime(m.dur))
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

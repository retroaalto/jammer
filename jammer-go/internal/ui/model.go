package ui

import (
	"fmt"
	"math"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/lipgloss"
	"github.com/jooapa/jammer/jammer-go/internal/player"
)

// tickMsg is sent on a timer tick to refresh progress.
type tickMsg time.Time

// trackChangedMsg is sent when the track changes.
type trackChangedMsg int

// ── Styles ────────────────────────────────────────────────────────────────────

var (
	styleBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63")).
			Padding(0, 1)

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
)

// ── Model ─────────────────────────────────────────────────────────────────────

// Model is the Bubble Tea model for the jammer TUI.
type Model struct {
	p       *player.Player
	songs   []player.Song
	cursor  int // list cursor position
	offset  int // scroll offset
	height  int
	width   int
	pos     float64
	dur     float64
	playing int // currently playing index
}

// New returns a new TUI Model.
func New(p *player.Player) Model {
	songs := p.Songs()
	return Model{
		p:      p,
		songs:  songs,
		cursor: 0,
	}
}

func tick() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) Init() tea.Cmd {
	return tick()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tickMsg:
		pos, dur := m.p.Progress()
		m.pos = pos
		m.dur = dur
		m.playing = m.p.Index()
		m.songs = m.p.Songs()
		return m, tick()

	case trackChangedMsg:
		m.playing = int(msg)
		m.cursor = m.playing
		m.clampScroll()

	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.p.Stop()
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				m.clampScroll()
			}

		case "down", "j":
			if m.cursor < len(m.songs)-1 {
				m.cursor++
				m.clampScroll()
			}

		case "enter", " ":
			if m.cursor == m.playing && m.p.State() != player.StateStopped {
				_ = m.p.Pause()
			} else {
				_ = m.p.PlayIndex(m.cursor)
				m.playing = m.cursor
			}

		case "p":
			_ = m.p.Pause()

		case "s":
			m.p.Stop()

		case "n":
			_ = m.p.Next()
			m.playing = m.p.Index()
			m.cursor = m.playing
			m.clampScroll()

		case "b":
			_ = m.p.Prev()
			m.playing = m.p.Index()
			m.cursor = m.playing
			m.clampScroll()

		case "right", "l":
			m.p.SeekForward(10)

		case "left", "h":
			m.p.SeekBackward(10)

		case "+", "=":
			vol := m.p.Volume() + 0.05
			m.p.SetVolume(vol)

		case "-":
			vol := m.p.Volume() - 0.05
			m.p.SetVolume(vol)
		}
	}

	return m, nil
}

func (m *Model) clampScroll() {
	listH := m.listHeight()
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+listH {
		m.offset = m.cursor - listH + 1
	}
}

func (m Model) listHeight() int {
	// reserve: title(1) + gap(1) + nowplaying(2) + progress(1) + volume(1) + help(2) + borders(2)
	reserved := 12
	h := m.height - reserved
	if h < 4 {
		h = 4
	}
	return h
}

func (m Model) View() tea.View {
	if m.width == 0 {
		v := tea.NewView("loading...")
		v.AltScreen = true
		return v
	}

	var b strings.Builder

	// ── Header ────────────────────────────────────────────────────────────────
	b.WriteString(styleTitle.Render("  jammer") + "\n\n")

	// ── Song list ─────────────────────────────────────────────────────────────
	listH := m.listHeight()
	end := m.offset + listH
	if end > len(m.songs) {
		end = len(m.songs)
	}

	for i := m.offset; i < end; i++ {
		title := truncate(m.songs[i].Title, m.width-8)

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

		line := prefix + title

		switch {
		case i == m.cursor && i == m.playing:
			b.WriteString(styleSelected.Render(line))
		case i == m.cursor:
			b.WriteString(styleSelected.Render(line))
		case i == m.playing:
			b.WriteString(stylePlaying.Render(line))
		default:
			b.WriteString(styleNormal.Render(line))
		}
		b.WriteByte('\n')
	}

	// scroll indicator
	if len(m.songs) > listH {
		b.WriteString(styleHelp.Render(fmt.Sprintf("  %d-%d / %d", m.offset+1, end, len(m.songs))))
		b.WriteByte('\n')
	}
	b.WriteByte('\n')

	// ── Now playing ───────────────────────────────────────────────────────────
	nowTitle := "—"
	if m.playing >= 0 && m.playing < len(m.songs) {
		nowTitle = truncate(m.songs[m.playing].Title, m.width-12)
	}
	stateIcon := " "
	switch m.p.State() {
	case player.StatePlaying:
		stateIcon = "▶"
	case player.StatePaused:
		stateIcon = "⏸"
	case player.StateStopped:
		stateIcon = "■"
	}
	b.WriteString(styleTitle.Render(fmt.Sprintf(" %s  %s", stateIcon, nowTitle)) + "\n")

	// ── Progress bar ──────────────────────────────────────────────────────────
	b.WriteString(" " + m.progressBar() + "\n")

	// ── Volume ────────────────────────────────────────────────────────────────
	vol := int(math.Round(float64(m.p.Volume()) * 100))
	b.WriteString(styleVolume.Render(fmt.Sprintf(" vol: %3d%%  %s", vol, m.volumeBar())) + "\n\n")

	// ── Help ──────────────────────────────────────────────────────────────────
	b.WriteString(styleHelp.Render(" enter/space: play  p: pause  s: stop  n: next  b: prev") + "\n")
	b.WriteString(styleHelp.Render(" ←/→: seek 10s  +/-: volume  q: quit") + "\n")

	v := tea.NewView(b.String())
	v.AltScreen = true
	return v
}

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
	return fmt.Sprintf("%s %s %s",
		fmtTime(m.pos),
		bar,
		fmtTime(m.dur),
	)
}

func (m Model) volumeBar() string {
	barW := 10
	filled := int(math.Round(float64(m.p.Volume()) * float64(barW)))
	return styleBarFill.Render(strings.Repeat("█", filled)) +
		styleBar.Render(strings.Repeat("░", barW-filled))
}

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

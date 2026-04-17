package player

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/jooapa/jammer/jammer-go/internal/audio"
	"github.com/jooapa/jammer/jammer-go/internal/playlist"
)

var supportedExts = map[string]bool{
	".mp3": true, ".ogg": true, ".wav": true, ".flac": true,
	".aac": true, ".m4a": true, ".opus": true, ".mp4": true,
	".aiff": true, ".aif": true,
}

// Song holds metadata and playback info for a track.
type Song struct {
	Path   string // absolute local file path (empty if not downloaded)
	URL    string // source URL (empty for local-only files)
	Title  string
	Author string
}

// Downloaded reports whether the song has a local file ready to play.
func (s Song) Downloaded() bool { return s.Path != "" }

// DisplayTitle returns the best available title.
func (s Song) DisplayTitle() string {
	if s.Title != "" && s.Author != "" {
		return s.Author + " - " + s.Title
	}
	if s.Title != "" {
		return s.Title
	}
	if s.Path != "" {
		return strings.TrimSuffix(filepath.Base(s.Path), filepath.Ext(s.Path))
	}
	return s.URL
}

// State is the playback state.
type State int

const (
	StateStopped State = iota
	StatePlaying
	StatePaused
)

// Player manages audio playback and the song queue.
type Player struct {
	mu     sync.Mutex
	songs  []Song
	index  int
	state  State
	stream audio.Stream
	volume float32

	OnTrackChange func(index int)
	OnStop        func()
}

// New creates a new Player. BASS must already be loaded and initialised.
func New() *Player {
	return &Player{volume: 0.8}
}

// LoadDir scans a directory for audio files and loads them as the queue.
func (p *Player) LoadDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	var songs []Song
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if !supportedExts[ext] {
			continue
		}
		name := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
		songs = append(songs, Song{
			Path:  filepath.Join(dir, e.Name()),
			Title: name,
		})
	}
	p.mu.Lock()
	p.songs = songs
	p.index = 0
	p.mu.Unlock()
	return nil
}

// LoadPlaylist replaces the queue with entries from a playlist.
func (p *Player) LoadPlaylist(entries []playlist.Entry) {
	songs := make([]Song, len(entries))
	for i, e := range entries {
		songs[i] = Song{
			Path:   e.Path,
			URL:    e.URL,
			Title:  e.Title,
			Author: e.Author,
		}
	}
	p.mu.Lock()
	p.songs = songs
	p.index = 0
	// stop current stream if any
	if p.stream != 0 {
		p.stream.Stop()
		p.stream.Free()
		p.stream = 0
		p.state = StateStopped
	}
	p.mu.Unlock()
}

// UpdateSongPath refreshes the local path of a song after it has been downloaded.
func (p *Player) UpdateSongPath(index int, path string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if index >= 0 && index < len(p.songs) {
		p.songs[index].Path = path
	}
}

// Songs returns a snapshot of the song list.
func (p *Player) Songs() []Song {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]Song, len(p.songs))
	copy(out, p.songs)
	return out
}

// Index returns the current track index.
func (p *Player) Index() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.index
}

// State returns the current playback state.
func (p *Player) State() State {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.state
}

// Play starts or resumes playback.
func (p *Player) Play() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.state == StatePaused && p.stream != 0 {
		if err := p.stream.Play(false); err != nil {
			return err
		}
		p.state = StatePlaying
		return nil
	}
	return p.openAndPlay(p.index)
}

// PlayIndex plays a specific song by index.
func (p *Player) PlayIndex(i int) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if i < 0 || i >= len(p.songs) {
		return nil
	}
	p.index = i
	return p.openAndPlay(i)
}

// openAndPlay opens and starts playback. Must be called with lock held.
func (p *Player) openAndPlay(i int) error {
	if p.stream != 0 {
		p.stream.Stop()
		p.stream.Free()
		p.stream = 0
	}
	if len(p.songs) == 0 {
		return nil
	}
	if !p.songs[i].Downloaded() {
		p.state = StateStopped
		return nil // song not available locally yet
	}
	s, err := audio.OpenFile(p.songs[i].Path)
	if err != nil {
		return err
	}
	s.SetVolume(p.volume)
	if err := s.Play(false); err != nil {
		s.Free()
		return err
	}
	p.stream = s
	p.state = StatePlaying
	if p.OnTrackChange != nil {
		idx := i
		go p.OnTrackChange(idx)
	}
	return nil
}

// Pause toggles pause/resume.
func (p *Player) Pause() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.stream == 0 {
		return nil
	}
	if p.state == StatePlaying {
		if err := p.stream.Pause(); err != nil {
			return err
		}
		p.state = StatePaused
	} else if p.state == StatePaused {
		if err := p.stream.Play(false); err != nil {
			return err
		}
		p.state = StatePlaying
	}
	return nil
}

// Stop stops playback.
func (p *Player) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.stream != 0 {
		p.stream.Stop()
		p.stream.Free()
		p.stream = 0
	}
	p.state = StateStopped
	if p.OnStop != nil {
		go p.OnStop()
	}
}

// Next advances to the next downloaded track, wrapping around.
func (p *Player) Next() error {
	p.mu.Lock()
	total := len(p.songs)
	start := (p.index + 1) % total
	p.mu.Unlock()

	// Skip songs that are not downloaded.
	for i := 0; i < total; i++ {
		idx := (start + i) % total
		p.mu.Lock()
		avail := p.songs[idx].Downloaded()
		p.mu.Unlock()
		if avail {
			return p.PlayIndex(idx)
		}
	}
	return nil
}

// Prev goes to the previous downloaded track.
func (p *Player) Prev() error {
	p.mu.Lock()
	total := len(p.songs)
	start := p.index - 1
	if start < 0 {
		start = total - 1
	}
	p.mu.Unlock()

	for i := 0; i < total; i++ {
		idx := (start - i + total) % total
		p.mu.Lock()
		avail := p.songs[idx].Downloaded()
		p.mu.Unlock()
		if avail {
			return p.PlayIndex(idx)
		}
	}
	return nil
}

// SeekForward seeks forward by d seconds.
func (p *Player) SeekForward(d float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.stream == 0 {
		return
	}
	pos := p.stream.Position() + d
	dur := p.stream.Duration()
	if pos > dur {
		pos = dur - 0.5
	}
	p.stream.Seek(pos)
}

// SeekBackward seeks backward by d seconds.
func (p *Player) SeekBackward(d float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.stream == 0 {
		return
	}
	pos := p.stream.Position() - d
	if pos < 0 {
		pos = 0
	}
	p.stream.Seek(pos)
}

// SetVolume sets volume in [0.0, 1.0].
func (p *Player) SetVolume(v float32) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if v < 0 {
		v = 0
	}
	if v > 1 {
		v = 1
	}
	p.volume = v
	if p.stream != 0 {
		p.stream.SetVolume(v)
	}
}

// Volume returns current volume.
func (p *Player) Volume() float32 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.volume
}

// Progress returns (position, duration) in seconds.
func (p *Player) Progress() (float64, float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.stream == 0 {
		return 0, 0
	}
	return p.stream.Position(), p.stream.Duration()
}

// WatchEnd polls for track end and auto-advances. Call in a goroutine.
func (p *Player) WatchEnd() {
	for {
		time.Sleep(500 * time.Millisecond)
		p.mu.Lock()
		if p.state == StatePlaying && p.stream != 0 {
			if p.stream.IsActive() == audio.ActiveStopped {
				p.mu.Unlock()
				_ = p.Next()
				continue
			}
		}
		p.mu.Unlock()
	}
}

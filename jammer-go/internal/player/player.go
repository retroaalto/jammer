package player

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/jooapa/jammer/jammer-go/internal/audio"
)

var supportedExts = map[string]bool{
	".mp3": true, ".ogg": true, ".wav": true, ".flac": true,
	".aac": true, ".m4a": true, ".opus": true, ".mp4": true,
	".aiff": true, ".aif": true,
}

// Song holds metadata about a track.
type Song struct {
	Path  string
	Title string // display name (filename without ext)
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
	mu            sync.Mutex
	songs         []Song
	index         int
	state         State
	stream        audio.Stream
	volume        float32
	OnTrackChange func(index int)
	OnStop        func()
}

// New creates a new Player (BASS must already be loaded and initialised).
func New() *Player {
	return &Player{volume: 0.8}
}

// LoadDir scans a directory and populates the song list.
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

// Songs returns a copy of the song list.
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

// must be called with lock held.
func (p *Player) openAndPlay(i int) error {
	if p.stream != 0 {
		p.stream.Stop()
		p.stream.Free()
		p.stream = 0
	}
	if len(p.songs) == 0 {
		return nil
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
	idx := i
	if p.OnTrackChange != nil {
		go p.OnTrackChange(idx)
	}
	return nil
}

// Pause toggles pause/play.
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

// Next advances to the next track.
func (p *Player) Next() error {
	p.mu.Lock()
	next := (p.index + 1) % len(p.songs)
	p.mu.Unlock()
	return p.PlayIndex(next)
}

// Prev goes back to the previous track.
func (p *Player) Prev() error {
	p.mu.Lock()
	prev := p.index - 1
	if prev < 0 {
		prev = len(p.songs) - 1
	}
	p.mu.Unlock()
	return p.PlayIndex(prev)
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
			active := p.stream.IsActive()
			if active == audio.ActiveStopped {
				p.mu.Unlock()
				_ = p.Next()
				continue
			}
		}
		p.mu.Unlock()
	}
}

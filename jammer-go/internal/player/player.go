package player

import (
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/jooapa/jammer/jammer-go/internal/audio"
	jlog "github.com/jooapa/jammer/jammer-go/internal/log"
	"github.com/jooapa/jammer/jammer-go/internal/playlist"
	"github.com/jooapa/jammer/jammer-go/internal/tags"
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

// LoopMode controls what happens when a track ends.
type LoopMode int

const (
	LoopAll LoopMode = iota // wrap around to the first track after the last (default)
	LoopOff                 // play through the queue once, then stop
	LoopOne                 // repeat the current track indefinitely
)

// Player manages audio playback and the song queue.
type Player struct {
	mu       sync.Mutex
	songs    []Song
	index    int
	state    State
	stream   audio.Stream
	backend  audio.Backend
	volume   float32
	loopMode LoopMode
	shuffle  bool

	OnTrackChange func(index int)
	OnStop        func()
}

// New creates a new Player using the given audio backend.
// The backend must already be initialised (Init called) before passing it here.
func New(backend audio.Backend) *Player {
	return &Player{volume: 0.8, backend: backend}
}

// NewHeadless creates a Player with the given backend — safe for tests using audio.NewNullBackend().
func NewHeadless(backend audio.Backend) *Player {
	return &Player{volume: 0.8, backend: backend}
}

// SetSongs replaces the song list without touching audio — for tests.
func (p *Player) SetSongs(songs []Song) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.songs = make([]Song, len(songs))
	copy(p.songs, songs)
	p.index = 0
}

// SetIndexForTest moves the internal index — for tests only.
func (p *Player) SetIndexForTest(i int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.index = i
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
		path := filepath.Join(dir, e.Name())
		s := Song{Path: path}

		// Try to read embedded tags; fall back to filename as title.
		if info, err := tags.Read(path); err == nil && info.Title != "" {
			s.Title = info.Title
			s.Author = info.Artist
		} else {
			s.Title = strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
		}

		songs = append(songs, s)
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
		// Enrich from embedded tags for songs already on disk.
		if songs[i].Path != "" && (songs[i].Title == "" || songs[i].Author == "") {
			if info, err := tags.Read(songs[i].Path); err == nil && info.Title != "" {
				if songs[i].Title == "" {
					songs[i].Title = info.Title
				}
				if songs[i].Author == "" {
					songs[i].Author = info.Artist
				}
			}
		}
	}
	p.mu.Lock()
	p.songs = songs
	p.index = 0
	if p.stream != nil {
		p.stream.Stop()
		p.stream.Free()
		p.stream = nil
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

// UpdateSongTags sets the Title and Artist of a song by index.
func (p *Player) UpdateSongTags(index int, title, artist string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if index >= 0 && index < len(p.songs) {
		if title != "" {
			p.songs[index].Title = title
		}
		if artist != "" {
			p.songs[index].Author = artist
		}
	}
}

// RemoveSong removes the song at the given index from the queue.
// If it is the currently playing track, playback is stopped first.
// The index is adjusted so it stays in bounds after removal.
// Returns the path of the removed song (empty string if not downloaded).
func (p *Player) RemoveSong(index int) string {
	p.mu.Lock()
	defer p.mu.Unlock()
	if index < 0 || index >= len(p.songs) {
		return ""
	}
	removedPath := p.songs[index].Path

	// Stop playback if we're removing the active track.
	if index == p.index && p.stream != nil {
		p.stream.Stop()
		p.stream.Free()
		p.stream = nil
		p.state = StateStopped
	}

	// Splice out the song.
	p.songs = append(p.songs[:index], p.songs[index+1:]...)

	// Keep p.index in bounds.
	if p.index >= len(p.songs) && p.index > 0 {
		p.index = len(p.songs) - 1
	}

	return removedPath
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
	if p.state == StatePaused && p.stream != nil {
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
	if p.stream != nil {
		p.stream.Stop()
		p.stream.Free()
		p.stream = nil
	}
	if len(p.songs) == 0 {
		jlog.Info("openAndPlay: no songs in queue")
		return nil
	}
	if !p.songs[i].Downloaded() {
		jlog.Infof("openAndPlay: index=%d not downloaded yet, stopping", i)
		p.state = StateStopped
		return nil // song not available locally yet
	}
	jlog.Infof("openAndPlay: opening index=%d path=%q", i, p.songs[i].Path)
	s, err := p.backend.OpenFile(p.songs[i].Path)
	if err != nil {
		if err == audio.ErrUnsupportedFormat {
			jlog.Infof("openAndPlay: unsupported format index=%d path=%q — skipping", i, p.songs[i].Path)
			p.state = StateStopped
			// Unlock so Next() can re-acquire.
			p.mu.Unlock()
			_ = p.Next()
			p.mu.Lock()
			return nil
		}
		jlog.Errorf("openAndPlay: OpenFile failed index=%d path=%q: %v", i, p.songs[i].Path, err)
		return err
	}

	// Enrich metadata from embedded tags if not already set from playlist.
	if p.songs[i].Title == "" || p.songs[i].Author == "" {
		if info, terr := tags.Read(p.songs[i].Path); terr == nil && info.Title != "" {
			if p.songs[i].Title == "" {
				p.songs[i].Title = info.Title
			}
			if p.songs[i].Author == "" {
				p.songs[i].Author = info.Artist
			}
			jlog.Infof("openAndPlay: read tags index=%d title=%q artist=%q", i, info.Title, info.Artist)
		}
	}
	s.SetVolume(p.volume)
	if err := s.Play(false); err != nil {
		jlog.Errorf("openAndPlay: Play failed index=%d: %v", i, err)
		s.Free()
		return err
	}
	jlog.Infof("openAndPlay: playing index=%d title=%q", i, p.songs[i].DisplayTitle())
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
	if p.stream == nil {
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
	if p.stream != nil {
		p.stream.Stop()
		p.stream.Free()
		p.stream = nil
	}
	p.state = StateStopped
	if p.OnStop != nil {
		go p.OnStop()
	}
}

// Next advances to the next track and plays it if it is available locally.
// When shuffle is enabled the next track is chosen at random (never the same index).
// The index always moves regardless of download status.
func (p *Player) Next() error {
	p.mu.Lock()
	total := len(p.songs)
	if total == 0 {
		p.mu.Unlock()
		return nil
	}
	var next int
	if p.shuffle && total > 1 {
		next = rand.Intn(total - 1)
		if next >= p.index {
			next++
		}
	} else {
		next = (p.index + 1) % total
	}
	p.mu.Unlock()
	return p.PlayIndex(next)
}

// Prev goes to the previous track and plays it if it is available locally.
// The index always moves regardless of download status.
func (p *Player) Prev() error {
	p.mu.Lock()
	total := len(p.songs)
	if total == 0 {
		p.mu.Unlock()
		return nil
	}
	prev := (p.index - 1 + total) % total
	p.mu.Unlock()
	return p.PlayIndex(prev)
}

// SeekForward seeks forward by d seconds.
func (p *Player) SeekForward(d float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.stream == nil {
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
	if p.stream == nil {
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
	if p.stream != nil {
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
	if p.stream == nil {
		return 0, 0
	}
	return p.stream.Position(), p.stream.Duration()
}

// FFTData returns frequency magnitude data from the currently playing stream.
// Returns nil when no stream is active or the backend does not support FFT.
func (p *Player) FFTData() []float32 {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.stream == nil {
		return nil
	}
	return p.stream.FFTData()
}

// SetLoopMode changes the loop mode.
func (p *Player) SetLoopMode(m LoopMode) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.loopMode = m
}

// GetLoopMode returns the current loop mode.
func (p *Player) GetLoopMode() LoopMode {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.loopMode
}

// SetShuffle enables or disables shuffle mode.
// When enabled, Next() picks a random track instead of the sequential one.
func (p *Player) SetShuffle(on bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.shuffle = on
}

// IsShuffle reports whether shuffle mode is currently enabled.
func (p *Player) IsShuffle() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.shuffle
}

// WatchEnd polls for track end and auto-advances. Call in a goroutine.
func (p *Player) WatchEnd() {
	for {
		time.Sleep(500 * time.Millisecond)
		p.mu.Lock()
		if p.state == StatePlaying && p.stream != nil {
			if p.stream.IsActive() == audio.ActiveStopped {
				loop := p.loopMode
				idx := p.index
				total := len(p.songs)
				p.mu.Unlock()
				switch loop {
				case LoopOne:
					_ = p.PlayIndex(idx)
				case LoopOff:
					if idx >= total-1 {
						p.Stop()
					} else {
						_ = p.Next()
					}
				default: // LoopAll
					_ = p.Next()
				}
				continue
			}
		}
		p.mu.Unlock()
	}
}

package audio

import (
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/effects"
	"github.com/gopxl/beep/v2/flac"
	"github.com/gopxl/beep/v2/mp3"
	"github.com/gopxl/beep/v2/speaker"
	"github.com/gopxl/beep/v2/vorbis"
	"github.com/gopxl/beep/v2/wav"
)

const beepSampleRate = beep.SampleRate(44100)
const beepBufferSize = 4096

// ── BeepBackend ───────────────────────────────────────────────────────────────

// BeepBackend implements Backend using gopxl/beep — no external libraries required.
type BeepBackend struct{}

func NewBeepBackend() Backend { return &BeepBackend{} }

func (b *BeepBackend) Init() error {
	return speaker.Init(beepSampleRate, beepBufferSize)
}

func (b *BeepBackend) Free() {
	speaker.Close()
}

func (b *BeepBackend) OpenFile(path string) (Stream, error) {
	ext := strings.ToLower(path[strings.LastIndex(path, ".")+1:])

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	var streamer beep.StreamSeekCloser
	var format beep.Format

	switch ext {
	case "mp3":
		streamer, format, err = mp3.Decode(f)
	case "ogg":
		streamer, format, err = vorbis.Decode(f)
	case "wav":
		streamer, format, err = wav.Decode(f)
	case "flac":
		streamer, format, err = flac.Decode(f)
	default:
		f.Close()
		return nil, fmt.Errorf("%w: .%s", ErrUnsupportedFormat, ext)
	}
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("beep: decode %q: %w", path, err)
	}

	ctrl := &beep.Ctrl{Streamer: beep.Resample(4, format.SampleRate, beepSampleRate, streamer)}
	vol := &effects.Volume{Streamer: ctrl, Base: 2, Volume: 0, Silent: false}

	bs := &beepStream{
		raw:   streamer,
		ctrl:  ctrl,
		vol:   vol,
		sr:    format.SampleRate,
		state: int32(ActiveStopped),
	}
	return bs, nil
}

// ── beepStream ────────────────────────────────────────────────────────────────

type beepStream struct {
	raw   beep.StreamSeekCloser
	ctrl  *beep.Ctrl
	vol   *effects.Volume
	sr    beep.SampleRate
	state int32 // atomic: ActiveStopped / ActivePlaying / ActivePaused
}

func (s *beepStream) Play(restart bool) error {
	if restart {
		if err := s.raw.Seek(0); err != nil {
			return err
		}
	}
	atomic.StoreInt32(&s.state, int32(ActivePlaying))
	s.ctrl.Paused = false

	done := make(chan struct{})
	speaker.Play(beep.Seq(s.vol, beep.Callback(func() {
		// Only mark stopped if we weren't paused or already freed.
		if atomic.LoadInt32(&s.state) == int32(ActivePlaying) {
			atomic.StoreInt32(&s.state, int32(ActiveStopped))
		}
		close(done)
	})))
	_ = done // fire-and-forget; WatchEnd polls IsActive()
	return nil
}

func (s *beepStream) Pause() error {
	speaker.Lock()
	s.ctrl.Paused = true
	speaker.Unlock()
	atomic.StoreInt32(&s.state, int32(ActivePaused))
	return nil
}

func (s *beepStream) Stop() {
	atomic.StoreInt32(&s.state, int32(ActiveStopped))
	speaker.Lock()
	s.ctrl.Paused = true
	speaker.Unlock()
	// Give speaker a moment to drain the current buffer cleanly.
	time.Sleep(10 * time.Millisecond)
	speaker.Clear()
}

func (s *beepStream) Free() {
	s.raw.Close()
}

func (s *beepStream) Duration() float64 {
	return s.sr.D(s.raw.Len()).Seconds()
}

func (s *beepStream) Position() float64 {
	return s.sr.D(s.raw.Position()).Seconds()
}

func (s *beepStream) Seek(secs float64) {
	n := int(secs * float64(s.sr))
	speaker.Lock()
	_ = s.raw.Seek(n)
	speaker.Unlock()
}

// SetVolume accepts [0.0, 1.0] and maps to beep's logarithmic volume scale.
// beep Volume=0 is unity gain; we map 1.0→0, 0.5→-1, 0.0→silent.
func (s *beepStream) SetVolume(v float32) {
	speaker.Lock()
	if v <= 0 {
		s.vol.Silent = true
	} else {
		s.vol.Silent = false
		// Map [0,1] → [-4, 0] log scale (sounds natural).
		s.vol.Volume = float64(v-1) * 4
	}
	speaker.Unlock()
}

func (s *beepStream) Volume() float32 {
	if s.vol.Silent {
		return 0
	}
	// Inverse of SetVolume mapping.
	return float32(s.vol.Volume/4) + 1
}

func (s *beepStream) IsActive() int {
	return int(atomic.LoadInt32(&s.state))
}

func (s *beepStream) FFTData() []float32 { return nil }

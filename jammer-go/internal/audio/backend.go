package audio

import "errors"

// ErrUnsupportedFormat is returned by Backend.OpenFile when the file
// format is not supported by the active backend.
var ErrUnsupportedFormat = errors.New("audio: unsupported format")

// Stream is the interface for a single audio stream (one playing track).
type Stream interface {
	Play(restart bool) error
	Pause() error
	Stop()
	Free()
	Duration() float64
	Position() float64
	Seek(secs float64)
	SetVolume(v float32)
	Volume() float32
	// IsActive returns ActivePlaying, ActivePaused, or ActiveStopped.
	IsActive() int
	// FFTData returns frequency magnitude data from a 512-point FFT (256 bins, 0.0–1.0).
	// Returns nil when unavailable (stopped, unsupported backend).
	FFTData() []float32
}

// Backend is the interface for an audio backend.
type Backend interface {
	// Init initialises the audio device. Must be called once before OpenFile.
	Init() error
	// Free releases all resources held by the backend.
	Free()
	// OpenFile opens a file and returns a Stream ready to play.
	// Returns ErrUnsupportedFormat if the format is not supported.
	OpenFile(path string) (Stream, error)
}

const (
	ActiveStopped = 0
	ActivePlaying = 1
	ActivePaused  = 3
)

package audio

// ── NullBackend ───────────────────────────────────────────────────────────────

// NullBackend is a no-op Backend used in tests and headless environments.
type NullBackend struct{}

func NewNullBackend() Backend { return &NullBackend{} }

func (n *NullBackend) Init() error                       { return nil }
func (n *NullBackend) Free()                             {}
func (n *NullBackend) OpenFile(_ string) (Stream, error) { return &nullStream{}, nil }

// ── nullStream ────────────────────────────────────────────────────────────────

type nullStream struct{}

func (s *nullStream) Play(_ bool) error   { return nil }
func (s *nullStream) Pause() error        { return nil }
func (s *nullStream) Stop()               {}
func (s *nullStream) Free()               {}
func (s *nullStream) Duration() float64   { return 0 }
func (s *nullStream) Position() float64   { return 0 }
func (s *nullStream) Seek(_ float64)      {}
func (s *nullStream) SetVolume(_ float32) {}
func (s *nullStream) Volume() float32     { return 1 }
func (s *nullStream) IsActive() int       { return ActiveStopped }
func (s *nullStream) FFTData() []float32  { return nil }

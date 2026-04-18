//go:build !windows

package audio

/*
#cgo linux LDFLAGS: -ldl
#include <dlfcn.h>
#include <stdlib.h>
#include <stdint.h>

// BASS types
typedef uint32_t HSTREAM;
typedef uint32_t DWORD;
typedef float    FLOAT;
typedef int      BOOL;

// BASS flags
#define BASS_POS_BYTE       0
#define BASS_ATTRIB_VOL     2
#define BASS_ACTIVE_STOPPED 0
#define BASS_ACTIVE_PLAYING 1
#define BASS_ACTIVE_PAUSED  3
#define BASS_DATA_FFT512    0x80000001

// Function pointer types
typedef BOOL      (*pfn_BASS_Init)(int, DWORD, DWORD, void*, void*);
typedef BOOL      (*pfn_BASS_Free)(void);
typedef HSTREAM   (*pfn_BASS_StreamCreateFile)(BOOL, const char*, uint64_t, uint64_t, DWORD);
typedef BOOL      (*pfn_BASS_ChannelPlay)(HSTREAM, BOOL);
typedef BOOL      (*pfn_BASS_ChannelPause)(HSTREAM);
typedef BOOL      (*pfn_BASS_ChannelStop)(HSTREAM);
typedef BOOL      (*pfn_BASS_StreamFree)(HSTREAM);
typedef uint64_t  (*pfn_BASS_ChannelGetLength)(HSTREAM, DWORD);
typedef uint64_t  (*pfn_BASS_ChannelGetPosition)(HSTREAM, DWORD);
typedef BOOL      (*pfn_BASS_ChannelSetPosition)(HSTREAM, uint64_t, DWORD);
typedef double    (*pfn_BASS_ChannelBytes2Seconds)(HSTREAM, uint64_t);
typedef uint64_t  (*pfn_BASS_ChannelSeconds2Bytes)(HSTREAM, double);
typedef BOOL      (*pfn_BASS_ChannelSetAttribute)(HSTREAM, DWORD, FLOAT);
typedef BOOL      (*pfn_BASS_ChannelGetAttribute)(HSTREAM, DWORD, FLOAT*);
typedef DWORD     (*pfn_BASS_ChannelIsActive)(HSTREAM);
typedef DWORD     (*pfn_BASS_ErrorGetCode)(void);
typedef DWORD     (*pfn_BASS_ChannelGetData)(HSTREAM, void*, DWORD);

static void* bassLib = NULL;

// Resolved function pointers
static pfn_BASS_Init              fn_BASS_Init;
static pfn_BASS_Free              fn_BASS_Free;
static pfn_BASS_StreamCreateFile  fn_BASS_StreamCreateFile;
static pfn_BASS_ChannelPlay       fn_BASS_ChannelPlay;
static pfn_BASS_ChannelPause      fn_BASS_ChannelPause;
static pfn_BASS_ChannelStop       fn_BASS_ChannelStop;
static pfn_BASS_StreamFree        fn_BASS_StreamFree;
static pfn_BASS_ChannelGetLength  fn_BASS_ChannelGetLength;
static pfn_BASS_ChannelGetPosition fn_BASS_ChannelGetPosition;
static pfn_BASS_ChannelSetPosition fn_BASS_ChannelSetPosition;
static pfn_BASS_ChannelBytes2Seconds fn_BASS_ChannelBytes2Seconds;
static pfn_BASS_ChannelSeconds2Bytes fn_BASS_ChannelSeconds2Bytes;
static pfn_BASS_ChannelSetAttribute fn_BASS_ChannelSetAttribute;
static pfn_BASS_ChannelGetAttribute fn_BASS_ChannelGetAttribute;
static pfn_BASS_ChannelIsActive   fn_BASS_ChannelIsActive;
static pfn_BASS_ErrorGetCode      fn_BASS_ErrorGetCode;
static pfn_BASS_ChannelGetData    fn_BASS_ChannelGetData;

static int bass_load(const char* path) {
    bassLib = dlopen(path, RTLD_LAZY | RTLD_GLOBAL);
    if (!bassLib) return 0;
    fn_BASS_Init                = (pfn_BASS_Init)               dlsym(bassLib, "BASS_Init");
    fn_BASS_Free                = (pfn_BASS_Free)               dlsym(bassLib, "BASS_Free");
    fn_BASS_StreamCreateFile    = (pfn_BASS_StreamCreateFile)   dlsym(bassLib, "BASS_StreamCreateFile");
    fn_BASS_ChannelPlay         = (pfn_BASS_ChannelPlay)        dlsym(bassLib, "BASS_ChannelPlay");
    fn_BASS_ChannelPause        = (pfn_BASS_ChannelPause)       dlsym(bassLib, "BASS_ChannelPause");
    fn_BASS_ChannelStop         = (pfn_BASS_ChannelStop)        dlsym(bassLib, "BASS_ChannelStop");
    fn_BASS_StreamFree          = (pfn_BASS_StreamFree)         dlsym(bassLib, "BASS_StreamFree");
    fn_BASS_ChannelGetLength    = (pfn_BASS_ChannelGetLength)   dlsym(bassLib, "BASS_ChannelGetLength");
    fn_BASS_ChannelGetPosition  = (pfn_BASS_ChannelGetPosition) dlsym(bassLib, "BASS_ChannelGetPosition");
    fn_BASS_ChannelSetPosition  = (pfn_BASS_ChannelSetPosition) dlsym(bassLib, "BASS_ChannelSetPosition");
    fn_BASS_ChannelBytes2Seconds= (pfn_BASS_ChannelBytes2Seconds)dlsym(bassLib,"BASS_ChannelBytes2Seconds");
    fn_BASS_ChannelSeconds2Bytes= (pfn_BASS_ChannelSeconds2Bytes)dlsym(bassLib,"BASS_ChannelSeconds2Bytes");
    fn_BASS_ChannelSetAttribute = (pfn_BASS_ChannelSetAttribute)dlsym(bassLib, "BASS_ChannelSetAttribute");
    fn_BASS_ChannelGetAttribute = (pfn_BASS_ChannelGetAttribute)dlsym(bassLib, "BASS_ChannelGetAttribute");
    fn_BASS_ChannelIsActive     = (pfn_BASS_ChannelIsActive)    dlsym(bassLib, "BASS_ChannelIsActive");
    fn_BASS_ErrorGetCode        = (pfn_BASS_ErrorGetCode)       dlsym(bassLib, "BASS_ErrorGetCode");
    fn_BASS_ChannelGetData      = (pfn_BASS_ChannelGetData)     dlsym(bassLib, "BASS_ChannelGetData");
    return 1;
}

static int    bass_init(void)                         { return fn_BASS_Init ? fn_BASS_Init(-1, 44100, 0, NULL, NULL) : 0; }
static int    bass_free(void)                         { return fn_BASS_Free ? fn_BASS_Free() : 0; }
static HSTREAM bass_stream_create(const char* f)      { return fn_BASS_StreamCreateFile ? fn_BASS_StreamCreateFile(0, f, 0, 0, 0) : 0; }
static int    bass_play(HSTREAM h, int restart)       { return fn_BASS_ChannelPlay ? fn_BASS_ChannelPlay(h, restart) : 0; }
static int    bass_pause(HSTREAM h)                   { return fn_BASS_ChannelPause ? fn_BASS_ChannelPause(h) : 0; }
static int    bass_stop(HSTREAM h)                    { return fn_BASS_ChannelStop ? fn_BASS_ChannelStop(h) : 0; }
static int    bass_stream_free(HSTREAM h)             { return fn_BASS_StreamFree ? fn_BASS_StreamFree(h) : 0; }
static uint64_t bass_get_length(HSTREAM h)            { return fn_BASS_ChannelGetLength ? fn_BASS_ChannelGetLength(h, BASS_POS_BYTE) : 0; }
static uint64_t bass_get_position(HSTREAM h)          { return fn_BASS_ChannelGetPosition ? fn_BASS_ChannelGetPosition(h, BASS_POS_BYTE) : 0; }
static int    bass_set_position(HSTREAM h, uint64_t p){ return fn_BASS_ChannelSetPosition ? fn_BASS_ChannelSetPosition(h, p, BASS_POS_BYTE) : 0; }
static double bass_bytes2secs(HSTREAM h, uint64_t b)  { return fn_BASS_ChannelBytes2Seconds ? fn_BASS_ChannelBytes2Seconds(h, b) : 0.0; }
static uint64_t bass_secs2bytes(HSTREAM h, double s)  { return fn_BASS_ChannelSeconds2Bytes ? fn_BASS_ChannelSeconds2Bytes(h, s) : 0; }
static int    bass_set_volume(HSTREAM h, float v)     { return fn_BASS_ChannelSetAttribute ? fn_BASS_ChannelSetAttribute(h, BASS_ATTRIB_VOL, v) : 0; }
static float  bass_get_volume(HSTREAM h)              { float v=1.0f; if(fn_BASS_ChannelGetAttribute) fn_BASS_ChannelGetAttribute(h, BASS_ATTRIB_VOL, &v); return v; }
static DWORD  bass_is_active(HSTREAM h)               { return fn_BASS_ChannelIsActive ? fn_BASS_ChannelIsActive(h) : 0; }
static DWORD  bass_error(void)                        { return fn_BASS_ErrorGetCode ? fn_BASS_ErrorGetCode() : -1; }
static DWORD  bass_get_fft(HSTREAM h, float* buf) {
    if (!fn_BASS_ChannelGetData) return 0;
    return fn_BASS_ChannelGetData(h, buf, BASS_DATA_FFT512);
}
*/
import "C"
import (
	"fmt"
	"unsafe"
)

// ── BassBackend ───────────────────────────────────────────────────────────────

// BassBackend implements Backend using the BASS audio library loaded at runtime.
type BassBackend struct{}

// LoadBass dynamically loads libbass from the given path and returns a BassBackend.
// Call backend.Init() before use.
func LoadBass(libPath string) (Backend, error) {
	cpath := C.CString(libPath)
	defer C.free(unsafe.Pointer(cpath))
	if C.bass_load(cpath) == 0 {
		return nil, fmt.Errorf("failed to load BASS from %s: %s", libPath, C.GoString(C.dlerror()))
	}
	return &BassBackend{}, nil
}

func (b *BassBackend) Init() error {
	if C.bass_init() == 0 {
		return fmt.Errorf("BASS_Init failed (error %d)", C.bass_error())
	}
	return nil
}

func (b *BassBackend) Free() {
	C.bass_free()
}

func (b *BassBackend) OpenFile(path string) (Stream, error) {
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	h := C.bass_stream_create(cpath)
	if h == 0 {
		return nil, fmt.Errorf("BASS_StreamCreateFile failed (error %d) for: %s", C.bass_error(), path)
	}
	return &bassStream{h: h}, nil
}

// ── bassStream ────────────────────────────────────────────────────────────────

// bassStream implements Stream backed by a BASS channel handle.
type bassStream struct {
	h C.HSTREAM
}

func (s *bassStream) Play(restart bool) error {
	r := 0
	if restart {
		r = 1
	}
	if C.bass_play(s.h, C.int(r)) == 0 {
		return fmt.Errorf("BASS_ChannelPlay failed (error %d)", C.bass_error())
	}
	return nil
}

func (s *bassStream) Pause() error {
	if C.bass_pause(s.h) == 0 {
		return fmt.Errorf("BASS_ChannelPause failed (error %d)", C.bass_error())
	}
	return nil
}

func (s *bassStream) Stop() {
	C.bass_stop(s.h)
}

func (s *bassStream) Free() {
	C.bass_stream_free(s.h)
}

func (s *bassStream) Duration() float64 {
	bytes := C.bass_get_length(s.h)
	return float64(C.bass_bytes2secs(s.h, bytes))
}

func (s *bassStream) Position() float64 {
	bytes := C.bass_get_position(s.h)
	return float64(C.bass_bytes2secs(s.h, bytes))
}

func (s *bassStream) Seek(secs float64) {
	bytes := C.bass_secs2bytes(s.h, C.double(secs))
	C.bass_set_position(s.h, bytes)
}

func (s *bassStream) SetVolume(v float32) {
	C.bass_set_volume(s.h, C.float(v))
}

func (s *bassStream) Volume() float32 {
	return float32(C.bass_get_volume(s.h))
}

func (s *bassStream) IsActive() int {
	return int(C.bass_is_active(s.h))
}

// FFTData returns 256 frequency-magnitude values (0.0–1.0) from a 512-point FFT.
// Returns nil if BASS_ChannelGetData is unavailable or the channel is not playing.
func (s *bassStream) FFTData() []float32 {
	var buf [256]C.float
	n := C.bass_get_fft(s.h, &buf[0])
	if n == 0 {
		return nil
	}
	out := make([]float32, 256)
	for i := range out {
		out[i] = float32(buf[i])
	}
	return out
}

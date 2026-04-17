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
    return 1;
}

static int  bass_init(void)                        { return fn_BASS_Init ? fn_BASS_Init(-1, 44100, 0, NULL, NULL) : 0; }
static int  bass_free(void)                        { return fn_BASS_Free ? fn_BASS_Free() : 0; }
static HSTREAM bass_stream_create(const char* f)   { return fn_BASS_StreamCreateFile ? fn_BASS_StreamCreateFile(0, f, 0, 0, 0) : 0; }
static int  bass_play(HSTREAM h, int restart)      { return fn_BASS_ChannelPlay ? fn_BASS_ChannelPlay(h, restart) : 0; }
static int  bass_pause(HSTREAM h)                  { return fn_BASS_ChannelPause ? fn_BASS_ChannelPause(h) : 0; }
static int  bass_stop(HSTREAM h)                   { return fn_BASS_ChannelStop ? fn_BASS_ChannelStop(h) : 0; }
static int  bass_stream_free(HSTREAM h)            { return fn_BASS_StreamFree ? fn_BASS_StreamFree(h) : 0; }
static uint64_t bass_get_length(HSTREAM h)         { return fn_BASS_ChannelGetLength ? fn_BASS_ChannelGetLength(h, BASS_POS_BYTE) : 0; }
static uint64_t bass_get_position(HSTREAM h)       { return fn_BASS_ChannelGetPosition ? fn_BASS_ChannelGetPosition(h, BASS_POS_BYTE) : 0; }
static int  bass_set_position(HSTREAM h, uint64_t p){ return fn_BASS_ChannelSetPosition ? fn_BASS_ChannelSetPosition(h, p, BASS_POS_BYTE) : 0; }
static double bass_bytes2secs(HSTREAM h, uint64_t b){ return fn_BASS_ChannelBytes2Seconds ? fn_BASS_ChannelBytes2Seconds(h, b) : 0.0; }
static uint64_t bass_secs2bytes(HSTREAM h, double s){ return fn_BASS_ChannelSeconds2Bytes ? fn_BASS_ChannelSeconds2Bytes(h, s) : 0; }
static int  bass_set_volume(HSTREAM h, float v)    { return fn_BASS_ChannelSetAttribute ? fn_BASS_ChannelSetAttribute(h, BASS_ATTRIB_VOL, v) : 0; }
static float bass_get_volume(HSTREAM h)            { float v=1.0f; if(fn_BASS_ChannelGetAttribute) fn_BASS_ChannelGetAttribute(h, BASS_ATTRIB_VOL, &v); return v; }
static DWORD bass_is_active(HSTREAM h)             { return fn_BASS_ChannelIsActive ? fn_BASS_ChannelIsActive(h) : 0; }
static DWORD bass_error(void)                      { return fn_BASS_ErrorGetCode ? fn_BASS_ErrorGetCode() : -1; }
*/
import "C"
import (
	"fmt"
	"unsafe"
)

const (
	ActiveStopped = 0
	ActivePlaying = 1
	ActivePaused  = 3
)

// Load dynamically loads libbass.so from the given path.
func Load(libPath string) error {
	cpath := C.CString(libPath)
	defer C.free(unsafe.Pointer(cpath))
	if C.bass_load(cpath) == 0 {
		return fmt.Errorf("failed to load BASS from %s: %s", libPath, C.GoString(C.dlerror()))
	}
	return nil
}

// Init initialises the BASS audio device.
func Init() error {
	if C.bass_init() == 0 {
		return fmt.Errorf("BASS_Init failed (error %d)", C.bass_error())
	}
	return nil
}

// Free releases BASS resources.
func Free() {
	C.bass_free()
}

// Stream represents a BASS audio stream handle.
type Stream C.HSTREAM

// OpenFile creates a BASS stream from a file path.
func OpenFile(path string) (Stream, error) {
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	h := C.bass_stream_create(cpath)
	if h == 0 {
		return 0, fmt.Errorf("BASS_StreamCreateFile failed (error %d) for: %s", C.bass_error(), path)
	}
	return Stream(h), nil
}

func (s Stream) Play(restart bool) error {
	r := 0
	if restart {
		r = 1
	}
	if C.bass_play(C.HSTREAM(s), C.int(r)) == 0 {
		return fmt.Errorf("BASS_ChannelPlay failed (error %d)", C.bass_error())
	}
	return nil
}

func (s Stream) Pause() error {
	if C.bass_pause(C.HSTREAM(s)) == 0 {
		return fmt.Errorf("BASS_ChannelPause failed (error %d)", C.bass_error())
	}
	return nil
}

func (s Stream) Stop() {
	C.bass_stop(C.HSTREAM(s))
}

func (s Stream) Free() {
	C.bass_stream_free(C.HSTREAM(s))
}

// Duration returns total duration in seconds.
func (s Stream) Duration() float64 {
	bytes := C.bass_get_length(C.HSTREAM(s))
	return float64(C.bass_bytes2secs(C.HSTREAM(s), bytes))
}

// Position returns current position in seconds.
func (s Stream) Position() float64 {
	bytes := C.bass_get_position(C.HSTREAM(s))
	return float64(C.bass_bytes2secs(C.HSTREAM(s), bytes))
}

// Seek seeks to the given position in seconds.
func (s Stream) Seek(secs float64) {
	bytes := C.bass_secs2bytes(C.HSTREAM(s), C.double(secs))
	C.bass_set_position(C.HSTREAM(s), bytes)
}

// SetVolume sets volume in range [0.0, 1.0].
func (s Stream) SetVolume(v float32) {
	C.bass_set_volume(C.HSTREAM(s), C.float(v))
}

// Volume returns current volume.
func (s Stream) Volume() float32 {
	return float32(C.bass_get_volume(C.HSTREAM(s)))
}

// IsActive returns the channel state (ActivePlaying, ActivePaused, ActiveStopped).
func (s Stream) IsActive() int {
	return int(C.bass_is_active(C.HSTREAM(s)))
}

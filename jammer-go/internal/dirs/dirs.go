// Package dirs resolves jammer's data directories following the XDG Base
// Directory Specification, with a legacy fallback for existing users:
//
//   - If ~/jammer/ exists, every function returns that directory (legacy mode).
//   - Otherwise XDG dirs are used:
//     Data()   → $XDG_DATA_HOME/jammer   (default ~/.local/share/jammer)
//     Config() → $XDG_CONFIG_HOME/jammer (default ~/.config/jammer)
//     State()  → $XDG_STATE_HOME/jammer  (default ~/.local/state/jammer)
//     Cache()  → $XDG_CACHE_HOME/jammer  (default ~/.cache/jammer)
package dirs

import (
	"os"
	"path/filepath"
	"sync"
)

var (
	once       sync.Once
	legacyDir  string // set to ~/jammer if it exists, otherwise ""
	homeDir    string
)

func init() {
	once.Do(resolve)
}

func resolve() {
	h, err := os.UserHomeDir()
	if err != nil {
		return
	}
	homeDir = h
	candidate := filepath.Join(h, "jammer")
	if info, err := os.Stat(candidate); err == nil && info.IsDir() {
		legacyDir = candidate
	}
}

func xdgDir(envVar, defaultSub string) string {
	if legacyDir != "" {
		return legacyDir
	}
	if v := os.Getenv(envVar); v != "" {
		return filepath.Join(v, "jammer")
	}
	if homeDir == "" {
		return filepath.Join(".", "jammer")
	}
	return filepath.Join(homeDir, defaultSub, "jammer")
}

// Data returns the directory for persistent user data (songs, playlists).
func Data() string {
	return xdgDir("XDG_DATA_HOME", ".local/share")
}

// Config returns the directory for user configuration (settings.json, KeyData.ini).
func Config() string {
	return xdgDir("XDG_CONFIG_HOME", ".config")
}

// State returns the directory for non-essential runtime state (jammer.log).
func State() string {
	return xdgDir("XDG_STATE_HOME", ".local/state")
}

// Cache returns the directory for cached data (sc_client_id.json).
func Cache() string {
	return xdgDir("XDG_CACHE_HOME", ".cache")
}

// IsLegacy reports whether jammer is running in legacy mode (~/jammer/ exists).
func IsLegacy() bool {
	return legacyDir != ""
}

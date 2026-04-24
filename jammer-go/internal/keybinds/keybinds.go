package keybinds

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Keybinds holds all loaded keybindings as string-to-keystring mappings
type Keybinds struct {
	// bindings maps action name → key string (e.g. "Shuffle" → "s")
	bindings map[string]string
	// reverse maps normalized key strings back to action names (for help display)
	reverse map[string]string
}

// New loads keybindings from ~/jammer/KeyData.ini, falling back to defaults if not found
func New() *Keybinds {
	kb := &Keybinds{
		bindings: make(map[string]string),
		reverse:  make(map[string]string),
	}

	// First apply all defaults
	kb.applyDefaults()

	// Then try to load from file (overwrites defaults for any keys present in file)
	_ = kb.loadFromFile()

	// Build reverse map for help screen
	kb.buildReverse()

	return kb
}

// loadFromFile reads KeyData.ini and populates bindings
func (kb *Keybinds) loadFromFile() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	filePath := filepath.Join(homeDir, "jammer", "KeyData.ini")
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	inKeybinds := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, ";") {
			continue
		}

		// Check for [Keybinds] section
		if line == "[Keybinds]" {
			inKeybinds = true
			continue
		}

		// Check for other sections
		if strings.HasPrefix(line, "[") {
			inKeybinds = false
			continue
		}

		// Parse bindings within [Keybinds] section
		if inKeybinds {
			if parts := strings.SplitN(line, "=", 2); len(parts) == 2 {
				action := strings.TrimSpace(parts[0])
				keyStr := strings.TrimSpace(parts[1])
				kb.bindings[action] = normalizeKey(keyStr)
			}
		}
	}

	return scanner.Err()
}

// applyDefaults sets all original keybinding defaults
func (kb *Keybinds) applyDefaults() {
	defaults := map[string]string{
		"ToMainMenu":                "Escape",
		"PlayPause":                 "Spacebar",
		"Quit":                      "Q",
		"NextSong":                  "N",
		"PreviousSong":              "P",
		"PlaySong":                  "Shift + P",
		"Forward5s":                 "RightArrow",
		"Backwards5s":               "LeftArrow",
		"VolumeUp":                  "UpArrow",
		"VolumeDown":                "DownArrow",
		"VolumeUpByOne":             "Shift + UpArrow",
		"VolumeDownByOne":           "Shift + DownArrow",
		"Shuffle":                   "S",
		"SaveAsPlaylist":            "Shift + Alt + S",
		"SaveCurrentPlaylist":       "Shift + S",
		"ShufflePlaylist":           "Alt + S",
		"Loop":                      "L",
		"Mute":                      "M",
		"ShowHidePlaylist":          "F",
		"ListAllPlaylists":          "Shift + F",
		"Help":                      "H",
		"Settings":                  "C",
		"ToSongStart":               "0",
		"ToSongEnd":                 "9",
		"ToggleInfo":                "I",
		"SearchInPlaylist":          "F3",
		"SearchByAuthor":            "Shift + F3",
		"CurrentState":              "F12",
		"CommandHelpScreen":         "Tab",
		"DeleteCurrentSong":         "Delete",
		"HardDeleteCurrentSong":     "Shift + Delete",
		"AddSongToPlaylist":         "Shift + A",
		"AddCurrentSongToFavorites": "Ctrl + F",
		"ShowSongsInPlaylists":      "Shift + D",
		"PlayOtherPlaylist":         "Shift + O",
		"RedownloadCurrentSong":     "Shift + B",
		"EditKeybindings":           "Shift + E",
		"ChangeLanguage":            "Shift + L",
		"ChangeTheme":               "Shift + T",
		"PlayRandomSong":            "R",
		"ChangeSoundFont":           "Shift + G",
		"GroupMenu":                 "Ctrl + G",
		"AddToGroup":                "G",
		"PlaylistViewScrollup":      "PageUp",
		"PlaylistViewScrolldown":    "PageDown",
		"Choose":                    "Enter",
		"Search":                    "Ctrl + Y",
		"ShowLog":                   "Ctrl + L",
		"ExitRssFeed":               "E",
		"BackEndChange":             "B",
		"RenameSong":                "F2",
	}

	for action, keyStr := range defaults {
		kb.bindings[action] = normalizeKey(keyStr)
	}
}

// buildReverse builds a reverse map for help/display purposes
func (kb *Keybinds) buildReverse() {
	for action, keyStr := range kb.bindings {
		kb.reverse[keyStr] = action
	}
}

// Is checks if the key string matches the specified action
func (kb *Keybinds) Is(action string, keyStr string) bool {
	spec, exists := kb.bindings[action]
	if !exists {
		return false
	}
	return normalizeRuntimeKey(keyStr) == spec
}

// normalizeRuntimeKey normalizes a key string coming from a BubbleTea KeyPressMsg.
// Unlike normalizeKey (used for INI values), uppercase single letters are treated
// as Shift+letter because BubbleTea sends "A" when the user presses Shift+A.
func normalizeRuntimeKey(keyStr string) string {
	// Promote single uppercase letter → shift+<lower> before lowercasing.
	if len(keyStr) == 1 && keyStr[0] >= 'A' && keyStr[0] <= 'Z' {
		keyStr = "shift+" + strings.ToLower(keyStr)
	}
	return normalizeKey(keyStr)
}

// normalizeKey converts a key string from INI config format to canonical form.
// Single uppercase letters are treated as plain (unmodified) keys — the INI
// convention is to write "H" to mean "press H", not "press Shift+H".
func normalizeKey(keyStr string) string {
	// We store our ini keys normalized the same way
	keyStr = strings.ToLower(strings.TrimSpace(keyStr))

	// Convert some common substitutions from ini format to bubbletea format
	keyStr = strings.ReplaceAll(keyStr, "spacebar", "space")
	keyStr = strings.ReplaceAll(keyStr, "rightarrow", "right")
	keyStr = strings.ReplaceAll(keyStr, "leftarrow", "left")
	keyStr = strings.ReplaceAll(keyStr, "uparrow", "up")
	keyStr = strings.ReplaceAll(keyStr, "downarrow", "down")
	keyStr = strings.ReplaceAll(keyStr, "pageup", "pgup")
	keyStr = strings.ReplaceAll(keyStr, "pagedown", "pgdn")
	// BubbleTea sends "esc"; INI stores "Escape" → normalize both to "escape"
	if keyStr == "esc" {
		keyStr = "escape"
	}

	// Normalize modifier order to: ctrl+alt+shift
	if strings.Contains(keyStr, "+") {
		parts := strings.Split(keyStr, "+")
		var mods []string
		var key string

		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "ctrl" || part == "alt" || part == "shift" {
				mods = append(mods, part)
			} else {
				key = part
			}
		}

		// Sort modifiers in standard order
		modMap := make(map[string]bool)
		for _, m := range mods {
			modMap[m] = true
		}

		keyStr = ""
		if modMap["ctrl"] {
			keyStr += "ctrl+"
		}
		if modMap["alt"] {
			keyStr += "alt+"
		}
		if modMap["shift"] {
			keyStr += "shift+"
		}
		keyStr += key
	}

	return keyStr
}

// Get returns the key string for an action
func (kb *Keybinds) Get(action string) (string, bool) {
	spec, exists := kb.bindings[action]
	return spec, exists
}

// Set updates the binding for an action and rebuilds the reverse map.
func (kb *Keybinds) Set(action, keyStr string) {
	kb.bindings[action] = normalizeKey(keyStr)
	kb.buildReverse()
}

// GetAll returns all bindings (useful for help screen)
func (kb *Keybinds) GetAll() map[string]string {
	result := make(map[string]string)
	for k, v := range kb.bindings {
		result[k] = v
	}
	return result
}

// Save writes the current bindings back to ~/jammer/KeyData.ini.
func (kb *Keybinds) Save() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	jammerDir := filepath.Join(homeDir, "jammer")
	if err := os.MkdirAll(jammerDir, 0o755); err != nil {
		return err
	}
	filePath := filepath.Join(jammerDir, "KeyData.ini")
	f, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	_, _ = w.WriteString("[Keybinds]\n")
	for action, keyStr := range kb.bindings {
		display := GetDisplay(keyStr)
		_, _ = w.WriteString(fmt.Sprintf("%s = %s\n", action, display))
	}
	return w.Flush()
}

// GetDisplay returns a displayable key string for help screens
// E.g. "shift+s" → "Shift + S"
func GetDisplay(keyStr string) string {
	keyStr = strings.ToLower(keyStr)

	// Reverse the normalizations for display
	keyStr = strings.ReplaceAll(keyStr, "space", "Spacebar")
	keyStr = strings.ReplaceAll(keyStr, "right", "RightArrow")
	keyStr = strings.ReplaceAll(keyStr, "left", "LeftArrow")
	keyStr = strings.ReplaceAll(keyStr, "up", "UpArrow")
	keyStr = strings.ReplaceAll(keyStr, "down", "DownArrow")
	keyStr = strings.ReplaceAll(keyStr, "pgup", "PageUp")
	keyStr = strings.ReplaceAll(keyStr, "pgdn", "PageDown")

	// Capitalize and add spacing
	keyStr = strings.ReplaceAll(keyStr, "ctrl", "Ctrl")
	keyStr = strings.ReplaceAll(keyStr, "alt", "Alt")
	keyStr = strings.ReplaceAll(keyStr, "shift", "Shift")
	keyStr = strings.ReplaceAll(keyStr, "+", " + ")

	// Capitalize single letter keys
	parts := strings.Split(keyStr, " + ")
	for i, part := range parts {
		if len(part) == 1 && part >= "a" && part <= "z" {
			parts[i] = strings.ToUpper(part)
		} else if len(part) == 1 && part >= "0" && part <= "9" {
			parts[i] = part
		} else if !strings.Contains(part, "Ctrl") && !strings.Contains(part, "Alt") && !strings.Contains(part, "Shift") {
			// Capitalize first letter of function/named keys
			if len(part) > 0 {
				parts[i] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
			}
		}
	}
	return strings.Join(parts, " + ")
}

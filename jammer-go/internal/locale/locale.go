// Package locale provides a simple INI-based localisation system.
//
// Locale files live in the embedded "locales" directory (en.ini, fi.ini,
// pt-br.ini).  At runtime the caller may also supply a user-provided locale
// directory that shadows the built-ins.
//
// Usage:
//
//	locale.Load("fi")          // load Finnish; falls back to English on missing keys
//	locale.T("Player", "Volume") // returns the localised string
package locale

import (
	"bufio"
	"bytes"
	"embed"
	"strings"
	"sync"
)

//go:embed locales/*.ini
var builtinFS embed.FS

// table maps section → key → value.
type table map[string]map[string]string

var (
	mu      sync.RWMutex
	current table            // active (possibly merged) locale
	base    table            // English fallback
	lang    string = "en"    // active language code (matches filename stem)
	langs   []LangInfo       // discovered languages (built-in + user)
)

// LangInfo describes a discovered locale.
type LangInfo struct {
	Code     string // filename stem, e.g. "fi"
	Language string // [Country] Language value, e.g. "Finnish"
	Country  string // [Country] Country value
}

// Load loads the locale with the given code (e.g. "en", "fi", "pt-br").
// It always loads English first as the fallback, then overlays the requested
// language on top so missing keys gracefully degrade.
func Load(code string) error {
	mu.Lock()
	defer mu.Unlock()

	en, err := loadBuiltin("en")
	if err != nil {
		return err
	}
	base = en

	if code == "en" || code == "" {
		current = en
		lang = "en"
		return nil
	}

	overlay, err := loadBuiltin(code)
	if err != nil {
		// fallback: just use English, don't error
		current = en
		lang = "en"
		return nil
	}

	current = merge(en, overlay)
	lang = code
	return nil
}

// T returns the localised string for [section] key.
// If the key is absent in the active locale it falls back to English; if it
// is also absent in English it returns "section.key" as a sentinel.
func T(section, key string) string {
	mu.RLock()
	defer mu.RUnlock()
	if current != nil {
		if sec, ok := current[section]; ok {
			if v, ok := sec[key]; ok {
				return v
			}
		}
	}
	if base != nil {
		if sec, ok := base[section]; ok {
			if v, ok := sec[key]; ok {
				return v
			}
		}
	}
	return section + "." + key
}

// CurrentLang returns the active language code.
func CurrentLang() string {
	mu.RLock()
	defer mu.RUnlock()
	return lang
}

// AvailableLanguages returns the list of built-in locale codes.
func AvailableLanguages() []LangInfo {
	mu.Lock()
	defer mu.Unlock()
	if langs != nil {
		return langs
	}
	entries, _ := builtinFS.ReadDir("locales")
	var out []LangInfo
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".ini") {
			continue
		}
		code := strings.TrimSuffix(e.Name(), ".ini")
		t, err := loadBuiltin(code)
		if err != nil {
			continue
		}
		info := LangInfo{Code: code}
		if sec, ok := t["Country"]; ok {
			info.Language = sec["Language"]
			info.Country = sec["Country"]
		}
		if info.Language == "" {
			info.Language = code
		}
		out = append(out, info)
	}
	langs = out
	return langs
}

// ─── internal ────────────────────────────────────────────────────────────────

func loadBuiltin(code string) (table, error) {
	data, err := builtinFS.ReadFile("locales/" + code + ".ini")
	if err != nil {
		return nil, err
	}
	return parseINI(bytes.NewReader(data))
}

func merge(base, overlay table) table {
	out := make(table, len(base))
	for sec, kvs := range base {
		m := make(map[string]string, len(kvs))
		for k, v := range kvs {
			m[k] = v
		}
		out[sec] = m
	}
	for sec, kvs := range overlay {
		if _, ok := out[sec]; !ok {
			out[sec] = make(map[string]string)
		}
		for k, v := range kvs {
			out[sec][k] = v
		}
	}
	return out
}

func parseINI(r *bytes.Reader) (table, error) {
	t := make(table)
	section := ""
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		// strip BOM
		line = strings.TrimPrefix(line, "\uFEFF")
		line = strings.TrimSpace(line)
		if line == "" || line[0] == ';' || line[0] == '#' {
			continue
		}
		if line[0] == '[' {
			end := strings.Index(line, "]")
			if end > 1 {
				section = line[1:end]
			}
			continue
		}
		eqIdx := strings.IndexByte(line, '=')
		if eqIdx < 1 || section == "" {
			continue
		}
		key := strings.TrimSpace(line[:eqIdx])
		val := strings.TrimSpace(line[eqIdx+1:])
		if _, ok := t[section]; !ok {
			t[section] = make(map[string]string)
		}
		t[section][key] = val
	}
	return t, scanner.Err()
}

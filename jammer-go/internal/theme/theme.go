// Package theme defines the colour palette used by the TUI and ships a small
// set of built-in themes.  The active theme is stored as a string name in
// settings.json ("theme" key).
//
// Theme files follow the classic Jammer JSON schema so themes are compatible
// between the C# and Go versions.
package theme

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ── Classic Jammer JSON schema ────────────────────────────────────────────────

// ClassicTheme is the top-level structure of a classic Jammer .json theme file.
type ClassicTheme struct {
	Playlist        PlaylistSection        `json:"Playlist"`
	GeneralPlaylist GeneralPlaylistSection `json:"GeneralPlaylist"`
	WholePlaylist   WholePlaylistSection   `json:"WholePlaylist"`
	Time            TimeSection            `json:"Time"`
	GeneralHelp     GeneralHelpSection     `json:"GeneralHelp"`
	GeneralSettings GeneralSettingsSection `json:"GeneralSettings"`
	EditKeybinds    EditKeybindsSection    `json:"EditKeybinds"`
	LanguageChange  LanguageChangeSection  `json:"LanguageChange"`
	InputBox        InputBoxSection        `json:"InputBox"`
	Visualizer      VisualizerSection      `json:"Visualizer"`
	Rss             RssSection             `json:"Rss"`
}

// RGB is a JSON-deserializable [r, g, b] array.
type RGB [3]int

// Lipgloss converts the RGB triple to a lipgloss.Color hex string.
func (c RGB) Lipgloss() lipgloss.Color {
	return lipgloss.Color(fmt.Sprintf("#%02X%02X%02X", c[0], c[1], c[2]))
}

type PlaylistSection struct {
	BorderStyle           string `json:"BorderStyle"`
	BorderColor           RGB    `json:"BorderColor"`
	PathColor             string `json:"PathColor"`
	ErrorColor            string `json:"ErrorColor"`
	SuccessColor          string `json:"SuccessColor"`
	InfoColor             string `json:"InfoColor"`
	PlaylistNameColor     string `json:"PlaylistNameColor"`
	MiniHelpBorderStyle   string `json:"MiniHelpBorderStyle"`
	MiniHelpBorderColor   RGB    `json:"MiniHelpBorderColor"`
	HelpLetterColor       string `json:"HelpLetterColor"`
	ForHelpTextColor      string `json:"ForHelpTextColor"`
	SettingsLetterColor   string `json:"SettingsLetterColor"`
	ForSettingsTextColor  string `json:"ForSettingsTextColor"`
	PlaylistLetterColor   string `json:"PlaylistLetterColor"`
	ForPlaylistTextColor  string `json:"ForPlaylistTextColor"`
	ForSeperatorTextColor string `json:"ForSeperatorTextColor"`
	VisualizerColor       string `json:"VisualizerColor"`
	RandomTextColor       string `json:"RandomTextColor"`
}

type GeneralPlaylistSection struct {
	BorderColor       RGB    `json:"BorderColor"`
	BorderStyle       string `json:"BorderStyle"`
	CurrentSongColor  string `json:"CurrentSongColor"`
	PreviousSongColor string `json:"PreviousSongColor"`
	NextSongColor     string `json:"NextSongColor"`
	NoneSongColor     string `json:"NoneSongColor"`
}

type WholePlaylistSection struct {
	BorderColor      RGB    `json:"BorderColor"`
	BorderStyle      string `json:"BorderStyle"`
	ChoosingColor    string `json:"ChoosingColor"`
	NormalSongColor  string `json:"NormalSongColor"`
	CurrentSongColor string `json:"CurrentSongColor"`
}

type TimeSection struct {
	BorderColor              RGB    `json:"BorderColor"`
	BorderStyle              string `json:"BorderStyle"`
	PlayingLetterColor       string `json:"PlayingLetterColor"`
	PlayingLetterLetter      string `json:"PlayingLetterLetter"`
	PausedLetterColor        string `json:"PausedLetterColor"`
	PausedLetterLetter       string `json:"PausedLetterLetter"`
	StoppedLetterColor       string `json:"StoppedLetterColor"`
	StoppedLetterLetter      string `json:"StoppedLetterLetter"`
	NextLetterColor          string `json:"NextLetterColor"`
	NextLetterLetter         string `json:"NextLetterLetter"`
	PreviousLetterColor      string `json:"PreviousLetterColor"`
	PreviousLetterLetter     string `json:"PreviousLetterLetter"`
	ShuffleLetterOffColor    string `json:"ShuffleLetterOffColor"`
	ShuffleOffLetter         string `json:"ShuffleOffLetter"`
	ShuffleLetterOnColor     string `json:"ShuffleLetterOnColor"`
	ShuffleOnLetter          string `json:"ShuffleOnLetter"`
	LoopLetterOffColor       string `json:"LoopLetterOffColor"`
	LoopOffLetter            string `json:"LoopOffLetter"`
	LoopLetterOnColor        string `json:"LoopLetterOnColor"`
	LoopOnLetter             string `json:"LoopOnLetter"`
	LoopLetterOnceColor      string `json:"LoopLetterOnceColor"`
	LoopOnceLetter           string `json:"LoopOnceLetter"`
	TimeColor                string `json:"TimeColor"`
	VolumeColorNotMuted      string `json:"VolumeColorNotMuted"`
	VolumeColorMuted         string `json:"VolumeColorMuted"`
	TimebarColor             string `json:"TimebarColor"`
	TimebarLetter            string `json:"TimebarLetter"`
}

type GeneralHelpSection struct {
	BorderColor          RGB    `json:"BorderColor"`
	BorderStyle          string `json:"BorderStyle"`
	HeaderTextColor      string `json:"HeaderTextColor"`
	ControlTextColor     string `json:"ControlTextColor"`
	DescriptionTextColor string `json:"DescriptionTextColor"`
	ModifierTextColor_1  string `json:"ModifierTextColor_1"`
	ModifierTextColor_2  string `json:"ModifierTextColor_2"`
	ModifierTextColor_3  string `json:"ModifierTextColor_3"`
}

type GeneralSettingsSection struct {
	BorderColor               RGB    `json:"BorderColor"`
	BorderStyle               string `json:"BorderStyle"`
	HeaderTextColor           string `json:"HeaderTextColor"`
	SettingTextColor          string `json:"SettingTextColor"`
	SettingValueColor         string `json:"SettingValueColor"`
	SettingChangeValueColor   string `json:"SettingChangeValueColor"`
	SettingChangeValueValueColor string `json:"SettingChangeValueValueColor"`
}

type EditKeybindsSection struct {
	BorderColor         RGB    `json:"BorderColor"`
	BorderStyle         string `json:"BorderStyle"`
	HeaderTextColor     string `json:"HeaderTextColor"`
	DescriptionColor    string `json:"DescriptionColor"`
	CurrentControlColor string `json:"CurrentControlColor"`
	CurrentKeyColor     string `json:"CurrentKeyColor"`
	EnteredKeyColor     string `json:"EnteredKeyColor"`
}

type LanguageChangeSection struct {
	BorderColor          RGB    `json:"BorderColor"`
	BorderStyle          string `json:"BorderStyle"`
	TextColor            string `json:"TextColor"`
	CurrentLanguageColor string `json:"CurrentLanguageColor"`
}

type InputBoxSection struct {
	BorderColor                     RGB    `json:"BorderColor"`
	BorderStyle                     string `json:"BorderStyle"`
	InputTextColor                  string `json:"InputTextColor"`
	TitleColor                      string `json:"TitleColor"`
	InputBorderStyle                string `json:"InputBorderStyle"`
	InputBorderColor                RGB    `json:"InputBorderColor"`
	TitleColorIfError               string `json:"TitleColorIfError"`
	InputTextColorIfError           string `json:"InputTextColorIfError"`
	InputBorderStyleIfError         string `json:"InputBorderStyleIfError"`
	InputBorderColorIfError         RGB    `json:"InputBorderColorIfError"`
	MultiSelectMoreChoicesTextColor string `json:"MultiSelectMoreChoicesTextColor"`
}

type VisualizerSection struct {
	UnicodeMap          []string `json:"UnicodeMap"`
	PlayingColor        string   `json:"PlayingColor"`
	PausedColor         string   `json:"PausedColor"`
	// GradientColors defines 2+ color stops (low→high) for per-bar gradient
	// coloring while playing. Empty = flat PlayingColor (classic behavior).
	// Colors are Spectre.Console names or #RRGGBB hex strings.
	GradientColors      []string `json:"GradientColors"`
	// GradientPausedColors is the same but applied when paused.
	// Empty = falls back to flat PausedColor.
	GradientPausedColors []string `json:"GradientPausedColors"`
}

type RssSection struct {
	BorderColor      RGB    `json:"BorderColor"`
	BorderStyle      string `json:"BorderStyle"`
	TitleColor       string `json:"TitleColor"`
	AuthorColor      string `json:"AuthorColor"`
	DescriptionColor string `json:"DescriptionColor"`
	LinkColor        string `json:"LinkColor"`
	DetailBorderColor RGB   `json:"DetailBorderColor"`
}

// ── Palette (derived, ready-to-use by the UI renderer) ───────────────────────

// Palette holds all named colours and glyphs used by the TUI renderer.
// It is derived from a ClassicTheme (or built-in defaults) by ConvertClassic.
type Palette struct {
	// ── Playlist / General song list ─────────────────────────────────────
	PlaylistBorderColor   lipgloss.Color // outer box border
	PlaylistBorderStyle   string         // "Rounded", "Double", etc.
	PlaylistTitle         lipgloss.Color // playlist name / header
	PlaylistNormal        lipgloss.Color // regular row foreground
	PlaylistSelected      lipgloss.Color // highlighted row foreground
	PlaylistSelectedBg    lipgloss.Color // highlighted row background
	PlaylistPlaying       lipgloss.Color // currently-playing row
	PlaylistHelp          lipgloss.Color // dim help / status text
	PlaylistError         lipgloss.Color // error text
	PlaylistSuccess       lipgloss.Color // success text
	PlaylistInfo          lipgloss.Color // info text
	PlaylistNotDL         lipgloss.Color // not yet downloaded
	PlaylistDownloading   lipgloss.Color // downloading

	// ── GeneralPlaylist (prev/current/next snippet) ───────────────────────
	GeneralPlaylistBorderColor  lipgloss.Color
	GeneralPlaylistBorderStyle  string
	GeneralPlaylistCurrent      lipgloss.Color
	GeneralPlaylistPrevious     lipgloss.Color
	GeneralPlaylistNext         lipgloss.Color

	// ── WholePlaylist (full song list) ───────────────────────────────────
	WholePlaylistBorderColor  lipgloss.Color
	WholePlaylistBorderStyle  string
	WholePlaylistChoosing     lipgloss.Color
	WholePlaylistNormal       lipgloss.Color
	WholePlaylistCurrent      lipgloss.Color

	// ── Time / progress bar ───────────────────────────────────────────────
	TimeBorderColor       lipgloss.Color
	TimeBorderStyle       string
	TimeColor             lipgloss.Color
	TimebarColor          lipgloss.Color // progress bar unfilled
	TimebarFill           lipgloss.Color // progress bar filled
	TimebarLetter         string         // character used for fill
	VolumeColor           lipgloss.Color
	VolumeMutedColor      lipgloss.Color
	PlayingLetter         string // glyph shown when playing
	PausedLetter          string // glyph shown when paused
	StoppedLetter         string
	NextLetter            string
	PreviousLetter        string
	ShuffleOffColor       lipgloss.Color
	ShuffleOffLetter      string
	ShuffleOnColor        lipgloss.Color
	ShuffleOnLetter       string
	LoopOffColor          lipgloss.Color
	LoopOffLetter         string
	LoopOnColor           lipgloss.Color
	LoopOnLetter          string
	LoopOnceColor         lipgloss.Color
	LoopOnceLetter        string

	// ── Help screen ───────────────────────────────────────────────────────
	HelpBorderColor      lipgloss.Color
	HelpBorderStyle      string
	HelpHeader           lipgloss.Color
	HelpControl          lipgloss.Color
	HelpDescription      lipgloss.Color

	// ── Settings screen ───────────────────────────────────────────────────
	SettingsBorderColor    lipgloss.Color
	SettingsBorderStyle    string
	SettingsHeader         lipgloss.Color
	SettingsName           lipgloss.Color
	SettingsValue          lipgloss.Color
	SettingsChangeHint     lipgloss.Color

	// ── Edit Keybinds screen ──────────────────────────────────────────────
	KeybindsBorderColor   lipgloss.Color
	KeybindsBorderStyle   string
	KeybindsHeader        lipgloss.Color
	KeybindsDescription   lipgloss.Color
	KeybindsControl       lipgloss.Color
	KeybindsCurrentKey    lipgloss.Color
	KeybindsEnteredKey    lipgloss.Color

	// ── Input box / modals ────────────────────────────────────────────────
	InputBorderColor      lipgloss.Color
	InputBorderStyle      string
	InputTitle            lipgloss.Color
	InputText             lipgloss.Color
	InputTitleError       lipgloss.Color
	InputTextError        lipgloss.Color

	// ── Visualizer ────────────────────────────────────────────────────────
	VizUnicodeMap        []string       // bar characters (low → high)
	VizPlayingColor      lipgloss.Color // flat color when playing (used if no gradient)
	VizPausedColor       lipgloss.Color // flat color when paused (used if no gradient)
	VizGradient          []lipgloss.Color // per-bar gradient stops playing (low→high); nil = flat
	VizGradientPaused    []lipgloss.Color // per-bar gradient stops paused; nil = flat

	// ── RSS feed ──────────────────────────────────────────────────────────
	RssBorderColor   lipgloss.Color
	RssBorderStyle   string
	RssTitle         lipgloss.Color
	RssAuthor        lipgloss.Color
	RssDescription   lipgloss.Color

	// ── Tabs ──────────────────────────────────────────────────────────────
	TabActive   lipgloss.Color
	TabInactive lipgloss.Color
}

// ── Spectre → lipgloss color conversion ──────────────────────────────────────

// spectreToLipgloss converts a Spectre.Console color/style string
// (e.g. "cyan bold", "red", "grey strikethrough") to a lipgloss.Color.
// It strips style modifiers (bold, italic, etc.) and maps named colors to
// their nearest ANSI 256-color index or hex equivalent.
// Returns "" (terminal default) for empty or unrecognised inputs.
func spectreToLipgloss(s string) lipgloss.Color {
	if s == "" {
		return lipgloss.Color("")
	}
	// Strip modifiers: bold, italic, underline, strikethrough, dim, blink, invert
	parts := strings.Fields(strings.ToLower(s))
	modifiers := map[string]bool{
		"bold": true, "italic": true, "underline": true,
		"strikethrough": true, "dim": true, "blink": true,
		"invert": true, "reverse": true,
	}
	colorName := ""
	for _, p := range parts {
		if !modifiers[p] {
			colorName = p
			break
		}
	}
	if colorName == "" {
		return lipgloss.Color("")
	}

	// Mapping of Spectre.Console named colors → lipgloss hex / ANSI index
	named := map[string]lipgloss.Color{
		// Basic 16
		"black":   "0",
		"maroon":  "1",
		"green":   "2",
		"olive":   "3",
		"navy":    "4",
		"purple":  "5",
		"teal":    "6",
		"silver":  "7",
		"grey":    "8",
		"gray":    "8",
		"red":     "9",
		"lime":    "10",
		"yellow":  "11",
		"blue":    "12",
		"fuchsia": "13",
		"magenta": "13",
		"aqua":    "14",
		"cyan":    "14",
		"white":   "15",
		// Extended named colors (Spectre subset)
		"orange":     "214",
		"orangered":  "202",
		"gold":       "220",
		"khaki":      "185",
		"cornsilk":   "230",
		"pink":       "218",
		"hotpink":    "205",
		"deeppink":   "199",
		"brown":      "130",
		"chocolate":  "166",
		"salmon":     "209",
		"coral":      "203",
		"tomato":     "196",
		"violet":     "177",
		"orchid":     "170",
		"plum":       "183",
		"lavender":   "189",
		"indigo":     "54",
		"steelblue":  "67",
		"skyblue":    "117",
		"lightblue":  "153",
		"dodgerblue": "33",
		"royalblue":  "63",
		"blue1":      "21",
		"blue3":      "20",
		"navyblue":   "17",
		"darkblue":   "18",
		"chartreuse": "118",
		"springgreen": "48",
		"seagreen":   "29",
		"darkgreen":  "22",
		"forestgreen": "28",
		"limegreen":  "46",
		"greenyellow": "154",
		"turquoise":  "44",
		"cadetblue":  "72",
		"darkturquoise": "44",
		"lightyellow": "230",
		"lightsalmon": "216",
		"lightcoral": "210",
		"lightgreen": "120",
		"lightcyan":  "195",
		"lightpink":  "218",
		"darkred":    "88",
		"darkviolet": "92",
		"darkorange": "208",
		"darkcyan":   "36",
		"darkyellow": "142",
		"darkgrey":   "238",
		"darkgray":   "238",
		"lightgrey":  "252",
		"lightgray":  "252",
		"dimgrey":    "242",
		"dimgray":    "242",
		"slategrey":  "66",
		"slategray":  "66",
	}

	if c, ok := named[colorName]; ok {
		return c
	}
	// Return empty (terminal default) for anything we don't recognise
	return lipgloss.Color("")
}

// ── Built-in themes ───────────────────────────────────────────────────────────

// defaultClassicTheme returns the built-in default ClassicTheme, mirroring the
// defaults embedded in classic Jammer's Themes.cs.
func defaultClassicTheme() ClassicTheme {
	return ClassicTheme{
		Playlist: PlaylistSection{
			BorderStyle:           "Rounded",
			BorderColor:           RGB{255, 255, 255},
			PathColor:             "white",
			ErrorColor:            "red bold",
			SuccessColor:          "green bold",
			InfoColor:             "blue",
			PlaylistNameColor:     "white bold",
			MiniHelpBorderStyle:   "Rounded",
			MiniHelpBorderColor:   RGB{255, 255, 255},
			HelpLetterColor:       "yellow bold",
			ForHelpTextColor:      "white",
			SettingsLetterColor:   "yellow bold",
			ForSettingsTextColor:  "white",
			PlaylistLetterColor:   "yellow bold",
			ForPlaylistTextColor:  "white",
			ForSeperatorTextColor: "white",
			VisualizerColor:       "green",
			RandomTextColor:       "white",
		},
		GeneralPlaylist: GeneralPlaylistSection{
			BorderColor:       RGB{255, 255, 255},
			BorderStyle:       "Rounded",
			CurrentSongColor:  "green bold",
			PreviousSongColor: "grey",
			NextSongColor:     "grey",
			NoneSongColor:     "grey",
		},
		WholePlaylist: WholePlaylistSection{
			BorderColor:      RGB{255, 255, 255},
			BorderStyle:      "Rounded",
			ChoosingColor:    "yellow bold",
			NormalSongColor:  "white",
			CurrentSongColor: "green bold",
		},
		Time: TimeSection{
			BorderColor:           RGB{255, 255, 255},
			BorderStyle:           "Rounded",
			PlayingLetterColor:    "white",
			PlayingLetterLetter:   "❚❚",
			PausedLetterColor:     "white",
			PausedLetterLetter:    "▶ ",
			StoppedLetterColor:    "white",
			StoppedLetterLetter:   "■",
			NextLetterColor:       "white",
			NextLetterLetter:      "▶▶",
			PreviousLetterColor:   "white",
			PreviousLetterLetter:  "◀◀",
			ShuffleLetterOffColor: "red bold",
			ShuffleOffLetter:      "⇌ ",
			ShuffleLetterOnColor:  "green bold",
			ShuffleOnLetter:       "⇌ ",
			LoopLetterOffColor:    "white",
			LoopOffLetter:         " ↻  ",
			LoopLetterOnColor:     "green bold",
			LoopOnLetter:          " ⟳  ",
			LoopLetterOnceColor:   "yellow bold",
			LoopOnceLetter:        " 1  ",
			TimeColor:             "white",
			VolumeColorNotMuted:   "white",
			VolumeColorMuted:      "grey strikethrough bold",
			TimebarColor:          "white",
			TimebarLetter:         "█",
		},
		GeneralHelp: GeneralHelpSection{
			BorderColor:          RGB{255, 255, 255},
			BorderStyle:          "Rounded",
			HeaderTextColor:      "white bold",
			ControlTextColor:     "white",
			DescriptionTextColor: "white",
			ModifierTextColor_1:  "green bold",
			ModifierTextColor_2:  "yellow bold",
			ModifierTextColor_3:  "red bold",
		},
		GeneralSettings: GeneralSettingsSection{
			BorderColor:                  RGB{255, 255, 255},
			BorderStyle:                  "Rounded",
			HeaderTextColor:              "white bold",
			SettingTextColor:             "white",
			SettingValueColor:            "green bold",
			SettingChangeValueColor:      "white",
			SettingChangeValueValueColor: "green bold",
		},
		EditKeybinds: EditKeybindsSection{
			BorderColor:         RGB{255, 255, 255},
			BorderStyle:         "Rounded",
			HeaderTextColor:     "white bold",
			DescriptionColor:    "white",
			CurrentControlColor: "white",
			CurrentKeyColor:     "red bold",
			EnteredKeyColor:     "cyan bold",
		},
		LanguageChange: LanguageChangeSection{
			BorderColor:          RGB{255, 255, 255},
			BorderStyle:          "Rounded",
			TextColor:            "white",
			CurrentLanguageColor: "red bold",
		},
		InputBox: InputBoxSection{
			BorderColor:                     RGB{255, 255, 255},
			BorderStyle:                     "Rounded",
			InputTextColor:                  "white",
			TitleColor:                      "white bold",
			InputBorderStyle:                "Rounded",
			InputBorderColor:                RGB{255, 255, 255},
			TitleColorIfError:               "red bold",
			InputTextColorIfError:           "red",
			InputBorderStyleIfError:         "Rounded",
			InputBorderColorIfError:         RGB{255, 255, 255},
			MultiSelectMoreChoicesTextColor: "grey",
		},
		Visualizer: VisualizerSection{
			UnicodeMap:          []string{" ", "▁", "▂", "▃", "▄", "▅", "▆", "▇", "█"},
			PlayingColor:        "green",
			PausedColor:         "grey",
			GradientColors:      []string{"#00FF00", "#FFFF00", "#FF0000"},
			GradientPausedColors: []string{"#444444", "#888888"},
		},
		Rss: RssSection{
			BorderColor:       RGB{255, 255, 255},
			BorderStyle:       "Rounded",
			TitleColor:        "white bold",
			AuthorColor:       "grey",
			DescriptionColor:  "white",
			LinkColor:         "blue",
			DetailBorderColor: RGB{255, 255, 255},
		},
	}
}

// parseGradientColors converts a slice of Spectre.Console color name or
// "#RRGGBB" hex strings into lipgloss.Colors, skipping empty entries.
// Returns nil if the input is empty or all entries are blank.
func parseGradientColors(stops []string) []lipgloss.Color {
	var out []lipgloss.Color
	for _, s := range stops {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if strings.HasPrefix(s, "#") {
			out = append(out, lipgloss.Color(s))
		} else {
			out = append(out, spectreToLipgloss(s))
		}
	}
	if len(out) < 2 {
		return nil // need at least 2 stops to form a gradient
	}
	return out
}

// ConvertClassic converts a ClassicTheme into a Palette ready for the UI renderer.
func ConvertClassic(ct ClassicTheme) Palette {
	sp := spectreToLipgloss
	return Palette{
		// Playlist
		PlaylistBorderColor: ct.Playlist.BorderColor.Lipgloss(),
		PlaylistBorderStyle: ct.Playlist.BorderStyle,
		PlaylistTitle:       sp(ct.Playlist.PlaylistNameColor),
		PlaylistNormal:      sp(ct.Playlist.PathColor),
		PlaylistSelected:    sp(ct.WholePlaylist.ChoosingColor),
		PlaylistSelectedBg:  lipgloss.Color(""),
		PlaylistPlaying:     sp(ct.WholePlaylist.CurrentSongColor),
		PlaylistHelp:        sp(ct.Playlist.ForHelpTextColor),
		PlaylistError:       sp(ct.Playlist.ErrorColor),
		PlaylistSuccess:     sp(ct.Playlist.SuccessColor),
		PlaylistInfo:        sp(ct.Playlist.InfoColor),
		PlaylistNotDL:       sp("grey"),
		PlaylistDownloading: sp("yellow bold"),

		// GeneralPlaylist
		GeneralPlaylistBorderColor: ct.GeneralPlaylist.BorderColor.Lipgloss(),
		GeneralPlaylistBorderStyle: ct.GeneralPlaylist.BorderStyle,
		GeneralPlaylistCurrent:     sp(ct.GeneralPlaylist.CurrentSongColor),
		GeneralPlaylistPrevious:    sp(ct.GeneralPlaylist.PreviousSongColor),
		GeneralPlaylistNext:        sp(ct.GeneralPlaylist.NextSongColor),

		// WholePlaylist
		WholePlaylistBorderColor: ct.WholePlaylist.BorderColor.Lipgloss(),
		WholePlaylistBorderStyle: ct.WholePlaylist.BorderStyle,
		WholePlaylistChoosing:    sp(ct.WholePlaylist.ChoosingColor),
		WholePlaylistNormal:      sp(ct.WholePlaylist.NormalSongColor),
		WholePlaylistCurrent:     sp(ct.WholePlaylist.CurrentSongColor),

		// Time
		TimeBorderColor:  ct.Time.BorderColor.Lipgloss(),
		TimeBorderStyle:  ct.Time.BorderStyle,
		TimeColor:        sp(ct.Time.TimeColor),
		TimebarColor:     sp(ct.Time.TimebarColor),
		TimebarFill:      sp(ct.Time.TimebarColor),
		TimebarLetter:    ct.Time.TimebarLetter,
		VolumeColor:      sp(ct.Time.VolumeColorNotMuted),
		VolumeMutedColor: sp(ct.Time.VolumeColorMuted),
		PlayingLetter:    ct.Time.PlayingLetterLetter,
		PausedLetter:     ct.Time.PausedLetterLetter,
		StoppedLetter:    ct.Time.StoppedLetterLetter,
		NextLetter:       ct.Time.NextLetterLetter,
		PreviousLetter:   ct.Time.PreviousLetterLetter,
		ShuffleOffColor:  sp(ct.Time.ShuffleLetterOffColor),
		ShuffleOffLetter: ct.Time.ShuffleOffLetter,
		ShuffleOnColor:   sp(ct.Time.ShuffleLetterOnColor),
		ShuffleOnLetter:  ct.Time.ShuffleOnLetter,
		LoopOffColor:     sp(ct.Time.LoopLetterOffColor),
		LoopOffLetter:    ct.Time.LoopOffLetter,
		LoopOnColor:      sp(ct.Time.LoopLetterOnColor),
		LoopOnLetter:     ct.Time.LoopOnLetter,
		LoopOnceColor:    sp(ct.Time.LoopLetterOnceColor),
		LoopOnceLetter:   ct.Time.LoopOnceLetter,

		// Help
		HelpBorderColor: ct.GeneralHelp.BorderColor.Lipgloss(),
		HelpBorderStyle: ct.GeneralHelp.BorderStyle,
		HelpHeader:      sp(ct.GeneralHelp.HeaderTextColor),
		HelpControl:     sp(ct.GeneralHelp.ControlTextColor),
		HelpDescription: sp(ct.GeneralHelp.DescriptionTextColor),

		// Settings
		SettingsBorderColor: ct.GeneralSettings.BorderColor.Lipgloss(),
		SettingsBorderStyle: ct.GeneralSettings.BorderStyle,
		SettingsHeader:      sp(ct.GeneralSettings.HeaderTextColor),
		SettingsName:        sp(ct.GeneralSettings.SettingTextColor),
		SettingsValue:       sp(ct.GeneralSettings.SettingValueColor),
		SettingsChangeHint:  sp(ct.GeneralSettings.SettingChangeValueColor),

		// Keybinds editor
		KeybindsBorderColor: ct.EditKeybinds.BorderColor.Lipgloss(),
		KeybindsBorderStyle: ct.EditKeybinds.BorderStyle,
		KeybindsHeader:      sp(ct.EditKeybinds.HeaderTextColor),
		KeybindsDescription: sp(ct.EditKeybinds.DescriptionColor),
		KeybindsControl:     sp(ct.EditKeybinds.CurrentControlColor),
		KeybindsCurrentKey:  sp(ct.EditKeybinds.CurrentKeyColor),
		KeybindsEnteredKey:  sp(ct.EditKeybinds.EnteredKeyColor),

		// Input box
		InputBorderColor: ct.InputBox.BorderColor.Lipgloss(),
		InputBorderStyle: ct.InputBox.BorderStyle,
		InputTitle:       sp(ct.InputBox.TitleColor),
		InputText:        sp(ct.InputBox.InputTextColor),
		InputTitleError:  sp(ct.InputBox.TitleColorIfError),
		InputTextError:   sp(ct.InputBox.InputTextColorIfError),

		// Visualizer
		VizUnicodeMap:      ct.Visualizer.UnicodeMap,
		VizPlayingColor:    sp(ct.Visualizer.PlayingColor),
		VizPausedColor:     sp(ct.Visualizer.PausedColor),
		VizGradient:        parseGradientColors(ct.Visualizer.GradientColors),
		VizGradientPaused:  parseGradientColors(ct.Visualizer.GradientPausedColors),

		// RSS
		RssBorderColor: ct.Rss.BorderColor.Lipgloss(),
		RssBorderStyle: ct.Rss.BorderStyle,
		RssTitle:       sp(ct.Rss.TitleColor),
		RssAuthor:      sp(ct.Rss.AuthorColor),
		RssDescription: sp(ct.Rss.DescriptionColor),

		// Tabs (derived from playlist / settings accent colors)
		TabActive:   sp(ct.Playlist.PlaylistNameColor),
		TabInactive: sp(ct.Playlist.ForHelpTextColor),
	}
}

// ── Built-in theme registry ───────────────────────────────────────────────────

// all stores the built-in Palettes keyed by name.
var all map[string]Palette

func init() {
	def := ConvertClassic(defaultClassicTheme())

	dracula := def
	dracula.PlaylistBorderColor = "#6272a4"
	dracula.PlaylistTitle = "#ff79c6"
	dracula.PlaylistNormal = "#f8f8f2"
	dracula.PlaylistSelected = "#ff79c6"
	dracula.PlaylistSelectedBg = "#44475a"
	dracula.PlaylistPlaying = "#50fa7b"
	dracula.PlaylistHelp = "#6272a4"
	dracula.PlaylistError = "#ff5555"
	dracula.PlaylistNotDL = "#44475a"
	dracula.PlaylistDownloading = "#f1fa8c"
	dracula.GeneralPlaylistBorderColor = "#6272a4"
	dracula.GeneralPlaylistCurrent = "#50fa7b"
	dracula.GeneralPlaylistPrevious = "#6272a4"
	dracula.GeneralPlaylistNext = "#6272a4"
	dracula.WholePlaylistBorderColor = "#6272a4"
	dracula.WholePlaylistChoosing = "#ff79c6"
	dracula.WholePlaylistNormal = "#f8f8f2"
	dracula.WholePlaylistCurrent = "#50fa7b"
	dracula.TimeBorderColor = "#6272a4"
	dracula.TimeColor = "#f8f8f2"
	dracula.TimebarColor = "#6272a4"
	dracula.TimebarFill = "#ff79c6"
	dracula.VolumeColor = "#ffb86c"
	dracula.HelpBorderColor = "#6272a4"
	dracula.HelpHeader = "#ff79c6"
	dracula.HelpControl = "#f8f8f2"
	dracula.HelpDescription = "#f8f8f2"
	dracula.SettingsBorderColor = "#6272a4"
	dracula.SettingsHeader = "#ff79c6"
	dracula.SettingsName = "#f8f8f2"
	dracula.SettingsValue = "#50fa7b"
	dracula.KeybindsBorderColor = "#6272a4"
	dracula.KeybindsHeader = "#ff79c6"
	dracula.InputBorderColor = "#6272a4"
	dracula.InputTitle = "#ff79c6"
	dracula.VizPlayingColor = "#50fa7b"
	dracula.VizPausedColor = "#6272a4"
	dracula.VizGradient = []lipgloss.Color{"#50fa7b", "#f1fa8c", "#ff5555"}
	dracula.VizGradientPaused = []lipgloss.Color{"#44475a", "#6272a4"}
	dracula.RssBorderColor = "#6272a4"
	dracula.RssTitle = "#ff79c6"
	dracula.TabActive = "#ff79c6"
	dracula.TabInactive = "#6272a4"

	nord := def
	nord.PlaylistBorderColor = "#4c566a"
	nord.PlaylistTitle = "#88c0d0"
	nord.PlaylistNormal = "#d8dee9"
	nord.PlaylistSelected = "#88c0d0"
	nord.PlaylistSelectedBg = "#3b4252"
	nord.PlaylistPlaying = "#a3be8c"
	nord.PlaylistHelp = "#4c566a"
	nord.PlaylistError = "#bf616a"
	nord.PlaylistNotDL = "#3b4252"
	nord.PlaylistDownloading = "#ebcb8b"
	nord.GeneralPlaylistBorderColor = "#4c566a"
	nord.GeneralPlaylistCurrent = "#a3be8c"
	nord.GeneralPlaylistPrevious = "#4c566a"
	nord.GeneralPlaylistNext = "#4c566a"
	nord.WholePlaylistBorderColor = "#4c566a"
	nord.WholePlaylistChoosing = "#88c0d0"
	nord.WholePlaylistNormal = "#d8dee9"
	nord.WholePlaylistCurrent = "#a3be8c"
	nord.TimeBorderColor = "#4c566a"
	nord.TimeColor = "#d8dee9"
	nord.TimebarColor = "#4c566a"
	nord.TimebarFill = "#88c0d0"
	nord.VolumeColor = "#ebcb8b"
	nord.HelpBorderColor = "#4c566a"
	nord.HelpHeader = "#88c0d0"
	nord.HelpControl = "#d8dee9"
	nord.HelpDescription = "#d8dee9"
	nord.SettingsBorderColor = "#4c566a"
	nord.SettingsHeader = "#88c0d0"
	nord.SettingsName = "#d8dee9"
	nord.SettingsValue = "#a3be8c"
	nord.KeybindsBorderColor = "#4c566a"
	nord.KeybindsHeader = "#88c0d0"
	nord.InputBorderColor = "#4c566a"
	nord.InputTitle = "#88c0d0"
	nord.VizPlayingColor = "#a3be8c"
	nord.VizPausedColor = "#4c566a"
	nord.VizGradient = []lipgloss.Color{"#8fbcbb", "#a3be8c", "#ebcb8b", "#bf616a"}
	nord.VizGradientPaused = []lipgloss.Color{"#3b4252", "#4c566a"}
	nord.RssBorderColor = "#4c566a"
	nord.RssTitle = "#88c0d0"
	nord.TabActive = "#88c0d0"
	nord.TabInactive = "#4c566a"

	gruvbox := def
	gruvbox.PlaylistBorderColor = "#504945"
	gruvbox.PlaylistTitle = "#fabd2f"
	gruvbox.PlaylistNormal = "#ebdbb2"
	gruvbox.PlaylistSelected = "#fabd2f"
	gruvbox.PlaylistSelectedBg = "#3c3836"
	gruvbox.PlaylistPlaying = "#b8bb26"
	gruvbox.PlaylistHelp = "#665c54"
	gruvbox.PlaylistError = "#fb4934"
	gruvbox.PlaylistNotDL = "#3c3836"
	gruvbox.PlaylistDownloading = "#fabd2f"
	gruvbox.GeneralPlaylistBorderColor = "#504945"
	gruvbox.GeneralPlaylistCurrent = "#b8bb26"
	gruvbox.GeneralPlaylistPrevious = "#665c54"
	gruvbox.GeneralPlaylistNext = "#665c54"
	gruvbox.WholePlaylistBorderColor = "#504945"
	gruvbox.WholePlaylistChoosing = "#fabd2f"
	gruvbox.WholePlaylistNormal = "#ebdbb2"
	gruvbox.WholePlaylistCurrent = "#b8bb26"
	gruvbox.TimeBorderColor = "#504945"
	gruvbox.TimeColor = "#ebdbb2"
	gruvbox.TimebarColor = "#504945"
	gruvbox.TimebarFill = "#fabd2f"
	gruvbox.VolumeColor = "#fe8019"
	gruvbox.HelpBorderColor = "#504945"
	gruvbox.HelpHeader = "#fabd2f"
	gruvbox.HelpControl = "#ebdbb2"
	gruvbox.HelpDescription = "#ebdbb2"
	gruvbox.SettingsBorderColor = "#504945"
	gruvbox.SettingsHeader = "#fabd2f"
	gruvbox.SettingsName = "#ebdbb2"
	gruvbox.SettingsValue = "#b8bb26"
	gruvbox.KeybindsBorderColor = "#504945"
	gruvbox.KeybindsHeader = "#fabd2f"
	gruvbox.InputBorderColor = "#504945"
	gruvbox.InputTitle = "#fabd2f"
	gruvbox.VizPlayingColor = "#b8bb26"
	gruvbox.VizPausedColor = "#665c54"
	gruvbox.VizGradient = []lipgloss.Color{"#b8bb26", "#fabd2f", "#fe8019", "#fb4934"}
	gruvbox.VizGradientPaused = []lipgloss.Color{"#3c3836", "#665c54"}
	gruvbox.RssBorderColor = "#504945"
	gruvbox.RssTitle = "#fabd2f"
	gruvbox.TabActive = "#fabd2f"
	gruvbox.TabInactive = "#665c54"

	all = map[string]Palette{
		"default": def,
		"dracula": dracula,
		"nord":    nord,
		"gruvbox": gruvbox,
	}
}

// ── File loading ──────────────────────────────────────────────────────────────

// LoadFromFile loads a classic Jammer .json theme file and returns a Palette.
// Comments in the JSON (// and /* */) are stripped before parsing.
func LoadFromFile(path string) (Palette, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Palette{}, err
	}
	cleaned := stripJSONComments(data)
	var ct ClassicTheme
	if err := json.Unmarshal(cleaned, &ct); err != nil {
		return Palette{}, fmt.Errorf("parse %s: %w", path, err)
	}
	// Backfill missing fields from defaults.
	def := defaultClassicTheme()
	backfillClassicTheme(&ct, def)
	return ConvertClassic(ct), nil
}

// stripJSONComments removes // line comments and /* block comments */ from JSON.
func stripJSONComments(data []byte) []byte {
	var out []byte
	s := string(data)
	inLineComment := false
	inBlockComment := false
	inString := false

	for i := 0; i < len(s); i++ {
		if inString {
			out = append(out, s[i])
			if s[i] == '\\' && i+1 < len(s) {
				i++
				out = append(out, s[i])
			} else if s[i] == '"' {
				inString = false
			}
			continue
		}
		if inLineComment {
			if s[i] == '\n' {
				inLineComment = false
				out = append(out, s[i])
			}
			continue
		}
		if inBlockComment {
			if s[i] == '*' && i+1 < len(s) && s[i+1] == '/' {
				inBlockComment = false
				i++
			}
			continue
		}
		if s[i] == '"' {
			inString = true
			out = append(out, s[i])
			continue
		}
		if s[i] == '/' && i+1 < len(s) {
			if s[i+1] == '/' {
				inLineComment = true
				i++
				continue
			}
			if s[i+1] == '*' {
				inBlockComment = true
				i++
				continue
			}
		}
		out = append(out, s[i])
	}
	return out
}

// backfillClassicTheme fills zero/empty fields in dst from src (the defaults).
func backfillClassicTheme(dst *ClassicTheme, src ClassicTheme) {
	if dst.Playlist.BorderStyle == "" {
		dst.Playlist.BorderStyle = src.Playlist.BorderStyle
	}
	if dst.Playlist.PathColor == "" {
		dst.Playlist.PathColor = src.Playlist.PathColor
	}
	if dst.Playlist.ErrorColor == "" {
		dst.Playlist.ErrorColor = src.Playlist.ErrorColor
	}
	if dst.Playlist.SuccessColor == "" {
		dst.Playlist.SuccessColor = src.Playlist.SuccessColor
	}
	if dst.Playlist.PlaylistNameColor == "" {
		dst.Playlist.PlaylistNameColor = src.Playlist.PlaylistNameColor
	}
	if dst.Playlist.VisualizerColor == "" {
		dst.Playlist.VisualizerColor = src.Playlist.VisualizerColor
	}
	if dst.GeneralPlaylist.BorderStyle == "" {
		dst.GeneralPlaylist.BorderStyle = src.GeneralPlaylist.BorderStyle
	}
	if dst.GeneralPlaylist.CurrentSongColor == "" {
		dst.GeneralPlaylist.CurrentSongColor = src.GeneralPlaylist.CurrentSongColor
	}
	if dst.GeneralPlaylist.PreviousSongColor == "" {
		dst.GeneralPlaylist.PreviousSongColor = src.GeneralPlaylist.PreviousSongColor
	}
	if dst.GeneralPlaylist.NextSongColor == "" {
		dst.GeneralPlaylist.NextSongColor = src.GeneralPlaylist.NextSongColor
	}
	if dst.WholePlaylist.BorderStyle == "" {
		dst.WholePlaylist.BorderStyle = src.WholePlaylist.BorderStyle
	}
	if dst.WholePlaylist.ChoosingColor == "" {
		dst.WholePlaylist.ChoosingColor = src.WholePlaylist.ChoosingColor
	}
	if dst.WholePlaylist.NormalSongColor == "" {
		dst.WholePlaylist.NormalSongColor = src.WholePlaylist.NormalSongColor
	}
	if dst.WholePlaylist.CurrentSongColor == "" {
		dst.WholePlaylist.CurrentSongColor = src.WholePlaylist.CurrentSongColor
	}
	if dst.Time.BorderStyle == "" {
		dst.Time.BorderStyle = src.Time.BorderStyle
	}
	if dst.Time.PlayingLetterLetter == "" {
		dst.Time.PlayingLetterLetter = src.Time.PlayingLetterLetter
	}
	if dst.Time.PausedLetterLetter == "" {
		dst.Time.PausedLetterLetter = src.Time.PausedLetterLetter
	}
	if dst.Time.StoppedLetterLetter == "" {
		dst.Time.StoppedLetterLetter = src.Time.StoppedLetterLetter
	}
	if dst.Time.TimebarLetter == "" {
		dst.Time.TimebarLetter = src.Time.TimebarLetter
	}
	if dst.Time.ShuffleOffLetter == "" {
		dst.Time.ShuffleOffLetter = src.Time.ShuffleOffLetter
	}
	if dst.Time.ShuffleOnLetter == "" {
		dst.Time.ShuffleOnLetter = src.Time.ShuffleOnLetter
	}
	if dst.Time.LoopOffLetter == "" {
		dst.Time.LoopOffLetter = src.Time.LoopOffLetter
	}
	if dst.Time.LoopOnLetter == "" {
		dst.Time.LoopOnLetter = src.Time.LoopOnLetter
	}
	if dst.Time.LoopOnceLetter == "" {
		dst.Time.LoopOnceLetter = src.Time.LoopOnceLetter
	}
	if dst.Time.TimeColor == "" {
		dst.Time.TimeColor = src.Time.TimeColor
	}
	if dst.Time.VolumeColorNotMuted == "" {
		dst.Time.VolumeColorNotMuted = src.Time.VolumeColorNotMuted
	}
	if dst.Time.TimebarColor == "" {
		dst.Time.TimebarColor = src.Time.TimebarColor
	}
	if dst.GeneralHelp.BorderStyle == "" {
		dst.GeneralHelp.BorderStyle = src.GeneralHelp.BorderStyle
	}
	if dst.GeneralHelp.HeaderTextColor == "" {
		dst.GeneralHelp.HeaderTextColor = src.GeneralHelp.HeaderTextColor
	}
	if dst.GeneralHelp.ControlTextColor == "" {
		dst.GeneralHelp.ControlTextColor = src.GeneralHelp.ControlTextColor
	}
	if dst.GeneralHelp.DescriptionTextColor == "" {
		dst.GeneralHelp.DescriptionTextColor = src.GeneralHelp.DescriptionTextColor
	}
	if dst.GeneralSettings.BorderStyle == "" {
		dst.GeneralSettings.BorderStyle = src.GeneralSettings.BorderStyle
	}
	if dst.GeneralSettings.HeaderTextColor == "" {
		dst.GeneralSettings.HeaderTextColor = src.GeneralSettings.HeaderTextColor
	}
	if dst.GeneralSettings.SettingTextColor == "" {
		dst.GeneralSettings.SettingTextColor = src.GeneralSettings.SettingTextColor
	}
	if dst.GeneralSettings.SettingValueColor == "" {
		dst.GeneralSettings.SettingValueColor = src.GeneralSettings.SettingValueColor
	}
	if dst.EditKeybinds.BorderStyle == "" {
		dst.EditKeybinds.BorderStyle = src.EditKeybinds.BorderStyle
	}
	if dst.EditKeybinds.HeaderTextColor == "" {
		dst.EditKeybinds.HeaderTextColor = src.EditKeybinds.HeaderTextColor
	}
	if dst.EditKeybinds.CurrentKeyColor == "" {
		dst.EditKeybinds.CurrentKeyColor = src.EditKeybinds.CurrentKeyColor
	}
	if dst.InputBox.BorderStyle == "" {
		dst.InputBox.BorderStyle = src.InputBox.BorderStyle
	}
	if dst.InputBox.TitleColor == "" {
		dst.InputBox.TitleColor = src.InputBox.TitleColor
	}
	if dst.InputBox.InputTextColor == "" {
		dst.InputBox.InputTextColor = src.InputBox.InputTextColor
	}
	if dst.Visualizer.PlayingColor == "" {
		dst.Visualizer.PlayingColor = src.Visualizer.PlayingColor
	}
	if dst.Visualizer.PausedColor == "" {
		dst.Visualizer.PausedColor = src.Visualizer.PausedColor
	}
	if len(dst.Visualizer.UnicodeMap) == 0 {
		dst.Visualizer.UnicodeMap = src.Visualizer.UnicodeMap
	}
	// Gradient fields are intentionally NOT backfilled from default — an empty
	// GradientColors in a user theme means "use flat color", not "inherit default".
	if dst.Rss.BorderStyle == "" {
		dst.Rss.BorderStyle = src.Rss.BorderStyle
	}
	if dst.Rss.TitleColor == "" {
		dst.Rss.TitleColor = src.Rss.TitleColor
	}
	if dst.Rss.AuthorColor == "" {
		dst.Rss.AuthorColor = src.Rss.AuthorColor
	}
}

// ── Public API ────────────────────────────────────────────────────────────────

// themesDir is set by SetThemesDir before Names/Get are called from the UI.
var themesDir string

// SetThemesDir tells the theme package where to look for .json theme files.
// Call this once at startup with dirs.Data() + "/themes".
func SetThemesDir(dir string) {
	themesDir = dir
}

// Names returns all available theme names: built-ins first, then on-disk .json
// files (stem name only, e.g. "blue-neon").
func Names() []string {
	names := []string{"default", "dracula", "gruvbox", "nord"}
	if themesDir == "" {
		return names
	}
	entries, err := os.ReadDir(themesDir)
	if err != nil {
		return names
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		n := e.Name()
		if strings.ToLower(filepath.Ext(n)) == ".json" {
			stem := strings.TrimSuffix(n, filepath.Ext(n))
			// Don't add if it collides with a built-in name.
			collision := false
			for _, existing := range names {
				if strings.EqualFold(existing, stem) {
					collision = true
					break
				}
			}
			if !collision {
				names = append(names, stem)
			}
		}
	}
	return names
}

// Get returns the Palette for the given name.
// Built-in themes are returned directly; for on-disk theme names the
// corresponding .json file in themesDir is loaded.
// Falls back to "default" on any error.
func Get(name string) Palette {
	if p, ok := all[name]; ok {
		return p
	}
	// Try loading from disk.
	if themesDir != "" {
		path := filepath.Join(themesDir, name+".json")
		if p, err := LoadFromFile(path); err == nil {
			return p
		}
	}
	return all["default"]
}

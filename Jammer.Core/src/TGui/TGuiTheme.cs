using Terminal.Gui;

namespace Jammer.TGui
{
    /// <summary>
    /// Bridges Jammer's Spectre.Console-based theme system to Terminal.Gui v2 ColorSchemes.
    ///
    /// Call Apply() once after Application.Init() and after Themes.Init().
    /// Views read the public static color/scheme properties to color their labels.
    /// </summary>
    public static class TGuiTheme
    {
        // ── Per-element colors (populated by Apply) ────────────────────────
        public static Color CurrentSongColor    { get; private set; } = Color.BrightGreen;
        public static Color PreviousSongColor   { get; private set; } = Color.Gray;
        public static Color NextSongColor       { get; private set; } = Color.Gray;
        public static Color HelpLetterColor     { get; private set; } = Color.BrightRed;
        public static Color SettingsLetterColor { get; private set; } = Color.BrightYellow;
        public static Color PlaylistLetterColor { get; private set; } = Color.BrightGreen;

        // ── Shared schemes ────────────────────────────────────────────────
        public static ColorScheme Base { get; private set; } = BuildBase();
        public static ColorScheme Dim  { get; private set; } = BuildDim();

        // ── Init ──────────────────────────────────────────────────────────

        public static void Apply()
        {
            var t = Themes.CurrentTheme;

            CurrentSongColor    = SpectreToTGui(t?.GeneralPlaylist?.CurrentSongColor,  Color.BrightGreen);
            PreviousSongColor   = SpectreToTGui(t?.GeneralPlaylist?.PreviousSongColor, Color.Gray);
            NextSongColor       = SpectreToTGui(t?.GeneralPlaylist?.NextSongColor,      Color.Gray);
            HelpLetterColor     = SpectreToTGui(t?.Playlist?.HelpLetterColor,           Color.BrightRed);
            SettingsLetterColor = SpectreToTGui(t?.Playlist?.SettingsLetterColor,        Color.BrightYellow);
            PlaylistLetterColor = SpectreToTGui(t?.Playlist?.PlaylistLetterColor,        Color.BrightGreen);

            Base = BuildBase();
            Dim  = BuildDim();

            // Push our base scheme into the global color scheme table so all
            // views that don't override ColorScheme inherit it.
            Colors.ColorSchemes["Base"]   = Base;
            Colors.ColorSchemes["Dialog"] = MakeScheme(Color.White, Color.Black, CurrentSongColor);
            Colors.ColorSchemes["Menu"]   = MakeScheme(Color.White, Color.Black, CurrentSongColor);
            Colors.ColorSchemes["Error"]  = MakeScheme(Color.BrightRed, Color.Black, Color.White);
        }

        // ── Helpers for views ─────────────────────────────────────────────

        // v2 Color is RGB-based; use Black as the default background.
        private static readonly Color DefaultBg = Color.Black;

        /// <summary>Return a ColorScheme with the given foreground on the terminal default background.</summary>
        public static ColorScheme LabelScheme(Color fg) =>
            MakeScheme(fg, DefaultBg, fg);

        // ── Internal ──────────────────────────────────────────────────────

        private static ColorScheme BuildBase() =>
            MakeScheme(Color.White, DefaultBg, CurrentSongColor);

        private static ColorScheme BuildDim() =>
            MakeScheme(Color.Gray, DefaultBg, Color.Gray);

        private static ColorScheme MakeScheme(Color fg, Color bg, Color hotFg) =>
            new ColorScheme
            {
                Normal    = new Terminal.Gui.Attribute(fg, bg),
                Focus     = new Terminal.Gui.Attribute(Color.Black, fg),
                HotNormal = new Terminal.Gui.Attribute(hotFg, bg),
                HotFocus  = new Terminal.Gui.Attribute(Color.Black, hotFg),
                Disabled  = new Terminal.Gui.Attribute(Color.DarkGray, bg),
            };

        /// <summary>
        /// Map a Spectre.Console color name (e.g. "green", "red bold") to the
        /// nearest Terminal.Gui Color. Style modifiers (bold, italic, etc.) are
        /// ignored — Terminal.Gui v2 handles bold via Attribute flags.
        /// Returns <paramref name="fallback"/> for empty / unknown names.
        /// </summary>
        public static Color SpectreToTGui(string? name, Color fallback)
        {
            if (string.IsNullOrWhiteSpace(name))
                return fallback;

            // Strip modifiers like "bold", "italic" — use first token only.
            name = name.Split(' ')[0].ToLowerInvariant().Trim();

            return name switch
            {
                "black"                  => Color.Black,
                "white"                  => Color.White,
                "red"                    => Color.BrightRed,
                "green"                  => Color.BrightGreen,
                "blue"                   => Color.BrightBlue,
                "cyan"                   => Color.BrightCyan,
                "yellow"                 => Color.BrightYellow,
                "grey" or "gray"         => Color.Gray,
                "darkgrey" or "darkgray" => Color.DarkGray,
                "magenta"                => Color.BrightMagenta,
                "orange"                 => Color.Yellow,
                "purple"                 => Color.Magenta,
                _                        => fallback,
            };
        }
    }
}

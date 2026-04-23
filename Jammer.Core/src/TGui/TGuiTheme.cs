using Terminal.Gui;

namespace Jammer.TGui
{
    /// <summary>
    /// Bridges Jammer's Spectre.Console-based theme system to Terminal.Gui ColorSchemes.
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

        // The terminal's own default background color (captured once on Apply).
        private static Color _termBg = Color.Black;

        // ── Init ──────────────────────────────────────────────────────────

        public static void Apply()
        {
            // Do NOT read Colors.Base.Normal.Background here — after Application.Init()
            // Terminal.Gui has already overwritten it with its own default cyan scheme,
            // so reading it would capture cyan, not the terminal's real background.
            // _termBg stays as Color.Black (the field default), which produces a
            // neutral dark scheme that respects most terminal themes.

            var t = Themes.CurrentTheme;

            CurrentSongColor    = SpectreToTGui(t?.GeneralPlaylist?.CurrentSongColor,  Color.BrightGreen);
            PreviousSongColor   = SpectreToTGui(t?.GeneralPlaylist?.PreviousSongColor, Color.Gray);
            NextSongColor       = SpectreToTGui(t?.GeneralPlaylist?.NextSongColor,      Color.Gray);
            HelpLetterColor     = SpectreToTGui(t?.Playlist?.HelpLetterColor,           Color.BrightRed);
            SettingsLetterColor = SpectreToTGui(t?.Playlist?.SettingsLetterColor,        Color.BrightYellow);
            PlaylistLetterColor = SpectreToTGui(t?.Playlist?.PlaylistLetterColor,        Color.BrightGreen);

            Base = BuildBase();
            Dim  = BuildDim();

            // Apply scheme globally using the terminal's own background so the UI
            // stays transparent rather than painting everything black.
            Colors.Base    = Base;
            Colors.Dialog  = MakeScheme(Color.White, _termBg, CurrentSongColor);
            Colors.Menu    = MakeScheme(Color.White, _termBg, CurrentSongColor);
            Colors.Error   = MakeScheme(Color.BrightRed, _termBg, Color.White);
        }

        // ── Helpers for views ─────────────────────────────────────────────

        /// <summary>Return a ColorScheme with the given foreground on the terminal default background.</summary>
        public static ColorScheme LabelScheme(Color fg) =>
            MakeScheme(fg, _termBg, fg);

        // ── Internal ──────────────────────────────────────────────────────

        private static ColorScheme BuildBase() =>
            MakeScheme(Color.White, _termBg, CurrentSongColor);

        private static ColorScheme BuildDim() =>
            MakeScheme(Color.Gray, _termBg, Color.Gray);

        private static ColorScheme MakeScheme(Color fg, Color bg, Color hotFg) =>
            new ColorScheme
            {
                Normal    = Terminal.Gui.Attribute.Make(fg, bg),
                Focus     = Terminal.Gui.Attribute.Make(Color.Black, fg),
                HotNormal = Terminal.Gui.Attribute.Make(hotFg, bg),
                HotFocus  = Terminal.Gui.Attribute.Make(Color.Black, hotFg),
                Disabled  = Terminal.Gui.Attribute.Make(Color.DarkGray, bg),
            };

        /// <summary>
        /// Map a Spectre.Console color name (e.g. "green", "red bold") to the
        /// nearest Terminal.Gui Color. Style modifiers (bold, italic, etc.) are
        /// ignored — Terminal.Gui v1 does not support them.
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
                "orange"                 => Color.Brown,
                "purple"                 => Color.Magenta,
                _                        => fallback,
            };
        }
    }
}

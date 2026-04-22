using Terminal.Gui;

namespace Jammer.TGui.Views
{
    /// <summary>
    /// Phase 2.5: Default player view showing previous / current / next song labels.
    /// No list interaction — purely a display. Media keys are handled by JammerToplevel.
    /// </summary>
    public class PlayerWindow : FrameView
    {
        private readonly Label _prevLabel;
        private readonly Label _currentLabel;
        private readonly Label _nextLabel;

        // Hint bar: split into colored bracket labels + plain text labels
        private readonly Label _hBracket;
        private readonly Label _hText;
        private readonly Label _fBracket;
        private readonly Label _fText;
        private readonly Label _cBracket;
        private readonly Label _cText;
        private readonly Label _lBracket;
        private readonly Label _lText;

        public PlayerWindow()
        {
            Title = Locale.Player.Playlist;
            BorderStyle = LineStyle.Single;
            ColorScheme = TGuiTheme.Base;

            _prevLabel = new Label
            {
                X = 2,
                Y = Pos.Center() - 2,
                Width = Dim.Fill(2),
                Height = 1,
                ColorScheme = TGuiTheme.LabelScheme(TGuiTheme.PreviousSongColor),
            };

            _currentLabel = new Label
            {
                X = 2,
                Y = Pos.Center(),
                Width = Dim.Fill(2),
                Height = 1,
                ColorScheme = TGuiTheme.LabelScheme(TGuiTheme.CurrentSongColor),
            };

            _nextLabel = new Label
            {
                X = 2,
                Y = Pos.Center() + 2,
                Width = Dim.Fill(2),
                Height = 1,
                ColorScheme = TGuiTheme.LabelScheme(TGuiTheme.NextSongColor),
            };

            // ── Hint bar (colored key brackets + plain text) ──────────────
            _hBracket = new Label
            {
                X = 2,
                Y = Pos.AnchorEnd(2),
                Text = "[H]",
                ColorScheme = TGuiTheme.LabelScheme(TGuiTheme.HelpLetterColor),
            };
            _hText = new Label
            {
                X = Pos.Right(_hBracket),
                Y = Pos.AnchorEnd(2),
                Text = $" {Locale.Help.Description}  ",
            };
            _fBracket = new Label
            {
                X = Pos.Right(_hText),
                Y = Pos.AnchorEnd(2),
                Text = "[F]",
                ColorScheme = TGuiTheme.LabelScheme(TGuiTheme.PlaylistLetterColor),
            };
            _fText = new Label
            {
                X = Pos.Right(_fBracket),
                Y = Pos.AnchorEnd(2),
                Text = $" {Locale.Player.ForPlaylist}  ",
            };
            _cBracket = new Label
            {
                X = Pos.Right(_fText),
                Y = Pos.AnchorEnd(2),
                Text = "[C]",
                ColorScheme = TGuiTheme.LabelScheme(TGuiTheme.SettingsLetterColor),
            };
            _cText = new Label
            {
                X = Pos.Right(_cBracket),
                Y = Pos.AnchorEnd(2),
                Text = " Settings  ",
            };
            _lBracket = new Label
            {
                X = Pos.Right(_cText),
                Y = Pos.AnchorEnd(2),
                Text = "[Shift+L]",
                ColorScheme = TGuiTheme.LabelScheme(TGuiTheme.HelpLetterColor),
            };
            _lText = new Label
            {
                X = Pos.Right(_lBracket),
                Y = Pos.AnchorEnd(2),
                Text = $" {Locale.Help.ChangeLanguage}",
            };

            Add(_prevLabel, _currentLabel, _nextLabel,
                _hBracket, _hText, _fBracket, _fText,
                _cBracket, _cText, _lBracket, _lText);
            Refresh();
        }

        /// <summary>Update labels from current playback state. Call from UI thread.</summary>
        public void Refresh()
        {
            string prevTitle = PrevTitle();
            string curTitle  = CurTitle();
            string nextTitle = NextTitle();

            _prevLabel.Text    = $"  {Locale.Player.Previos}: {prevTitle}";
            _currentLabel.Text = $"▶ {Locale.Player.Current}: {curTitle}";
            _nextLabel.Text    = $"  {Locale.Player.Next}: {nextTitle}";

            SetNeedsDraw();
        }

        private static string PrevTitle()
        {
            int idx = Utils.CurrentSongIndex - 1;
            if (idx < 0) idx = Utils.Songs.Length - 1;
            return idx >= 0 && idx < Utils.Songs.Length
                ? SongExtensions.Title(Utils.Songs[idx])
                : "-";
        }

        private static string CurTitle()
        {
            return Utils.CurrentSongPath.Length > 0
                ? SongExtensions.Title(Utils.CurrentSongPath)
                : Locale.Player.NoSongsInPlaylist;
        }

        private static string NextTitle()
        {
            int idx = Utils.CurrentSongIndex + 1;
            if (idx >= Utils.Songs.Length) idx = 0;
            return idx >= 0 && idx < Utils.Songs.Length
                ? SongExtensions.Title(Utils.Songs[idx])
                : "-";
        }
    }
}

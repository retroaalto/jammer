using Terminal.Gui;
using ManagedBass;

namespace Jammer.TGui
{
    /// <summary>
    /// Single-row bar at the bottom of the screen showing playback state,
    /// progress, and volume. Does not handle key events — key binding is done
    /// in JammerToplevel.ProcessKey() so it is scoped to the whole application.
    /// </summary>
    public class PlayerStatusBar : View
    {
        private readonly Label _label;

        public PlayerStatusBar()
        {
            Height = 1;
            Width = Dim.Fill();
            X = 0;
            Y = Pos.AnchorEnd(1);
            CanFocus = false;
            ColorScheme = TGuiTheme.Base;

            _label = new Label
            {
                X = 0,
                Y = 0,
                Width = Dim.Fill(),
                Height = 1,
            };
            Add(_label);
        }

        /// <summary>
        /// Refresh the label text from current playback state.
        /// Must be called from the UI thread (inside Application.MainLoop.Invoke).
        /// </summary>
        public void UpdateProgress()
        {
            string playIcon = Bass.ChannelIsActive(Utils.CurrentMusic) == PlaybackState.Playing
                ? "▶" : "⏸";

            string loopIcon = Preferences.loopType switch
            {
                LoopType.Always => "↻",
                LoopType.Once   => "¹",
                _               => " "
            };

            string shuffleIcon = Preferences.isShuffle ? "⇀" : " ";

            double elapsed = Utils.TotalMusicDurationInSec;
            double total = Utils.SongDurationInSec;
            float pct = Utils.MusicTimePercentage;
            int vol = (int)(Preferences.volume * 100);

            string timeStr = $"{FormatTime(elapsed)} / {FormatTime(total)}";
            int barWidth = Math.Max(10, (Application.Driver?.Cols ?? 80) - 50);
            int filled = total > 0 ? (int)(pct / 100f * barWidth) : 0;
            filled = Math.Clamp(filled, 0, barWidth);
            string bar = new string('█', filled) + new string('░', barWidth - filled);

            string title = SongExtensions.Title(Utils.CurrentSongPath);
            if (title.Length > 30) title = title[..27] + "...";

            _label.Text = $" {playIcon} {loopIcon} {shuffleIcon}  {timeStr}  [{bar}]  {vol}%  {title}";
            SetNeedsDisplay();
        }

        private static string FormatTime(double seconds)
        {
            if (double.IsNaN(seconds) || double.IsInfinity(seconds) || seconds < 0) seconds = 0;
            var ts = TimeSpan.FromSeconds(seconds);
            return ts.TotalHours >= 1
                ? $"{(int)ts.TotalHours}:{ts.Minutes:D2}:{ts.Seconds:D2}"
                : $"{ts.Minutes}:{ts.Seconds:D2}";
        }
    }
}

using System.Collections.ObjectModel;
using Terminal.Gui;

namespace Jammer.TGui.Views
{
    /// <summary>
    /// Phase 2.8: RSS feed view.
    /// Shows feed title/author at the top, a scrollable episode list in the middle,
    /// and an exit hint at the bottom. Enter plays the selected episode.
    /// ExitRssFeed key restores the previous playlist and fires ExitRequested.
    /// </summary>
    public class RssFeedWindow : FrameView
    {
        private readonly ListView _list;

        /// <summary>Raised when the user exits the RSS feed (restores previous playlist).</summary>
        public event Action? ExitRequested;

        public RssFeedWindow()
        {
            Title = BuildTitle();
            ColorScheme = TGuiTheme.Base;

            _list = new ListView
            {
                X = 0,
                Y = 0,
                Width = Dim.Fill(),
                Height = Dim.Fill(1),
                CanFocus = true,
            };
            _list.OpenSelectedItem += (_, e) => OnEpisodeSelected(e);

            var hint = new Label
            {
                X = 1,
                Y = Pos.AnchorEnd(1),
                Width = Dim.Fill(1),
                Height = 1,
                Text = $"Enter: play  [{Keybindings.ExitRssFeed}]: exit feed  Esc: player",
            };

            Add(_list, hint);
            Populate();
        }

        private static string BuildTitle()
        {
            var feed = Utils.RssFeedSong;
            string title  = feed.Title  ?? "RSS Feed";
            string author = feed.Author ?? "";
            return string.IsNullOrEmpty(author) ? title : $"{title}  —  {author}";
        }

        private void Populate()
        {
            var items = Utils.Songs
                .Select(s =>
                {
                    var song = SongExtensions.ToSong(s);
                    if (song == null) return SongExtensions.Title(s);
                    string t = song.Title  ?? SongExtensions.Title(s);
                    string a = song.Author ?? "";
                    string d = song.PubDate != null ? $"  [{song.PubDate}]" : "";
                    return string.IsNullOrEmpty(a) ? t + d : $"{t}  —  {a}{d}";
                })
                .ToList();

            _list.SetSource<string>(new ObservableCollection<string>(items));

            int idx = Math.Clamp(Utils.CurrentSongIndex, 0, Math.Max(0, items.Count - 1));
            _list.SelectedItem = idx;
            _list.TopItem = Math.Max(0, idx - 5);
        }

        private void OnEpisodeSelected(ListViewItemEventArgs e)
        {
            Play.PlaySong(Utils.Songs, e.Item);
        }

        protected override bool OnKeyDown(Key key)
        {
            if (KeybindingParser.Matches(key, Keybindings.ExitRssFeed))
            {
                ExitFeed();
                return true;
            }
            return base.OnKeyDown(key);
        }

        private void ExitFeed()
        {
            if (!Funcs.IsInsideOfARssFeed()) return;

            if (Utils.BackUpSongs != null)
                Utils.Songs = Utils.BackUpSongs;
            if (Utils.BackUpPlaylistName != null)
                Utils.CurrentPlaylist = Utils.BackUpPlaylistName;
            Utils.CurrentSongIndex = Utils.lastPositionInPreviousPlaylist;
            Utils.CurrentPlaylistSongIndex = Utils.lastPositionInPreviousPlaylist;

            Funcs.ResetRssExitVariables();

            if (Utils.Songs.Length > 0)
                Play.PlaySong(Utils.Songs, Utils.CurrentSongIndex);

            ExitRequested?.Invoke();
        }
    }
}

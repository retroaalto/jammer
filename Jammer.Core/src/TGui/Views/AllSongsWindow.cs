using System.Collections.ObjectModel;
using Terminal.Gui;

namespace Jammer.TGui.Views
{
    /// <summary>
    /// Phase 2.6: Playlist view with current-song highlight and Delete-to-remove.
    /// Arrow keys scroll natively; Enter plays; Delete removes from playlist.
    /// </summary>
    public class AllSongsWindow : FrameView
    {
        private readonly JammerListView _list;
        private readonly Label _hint;

        /// <summary>Raised when the selected song is an RSS feed and the feed view should open.</summary>
        public event Action? RssFeedRequested;

        public AllSongsWindow()
        {
            Title = "Playlist";
            ColorScheme = TGuiTheme.Base;

            _list = new JammerListView
            {
                X = 0,
                Y = 0,
                Width = Dim.Fill(),
                Height = Dim.Fill(1),
            };

            _hint = new Label
            {
                X = 1,
                Y = Pos.AnchorEnd(1),
                Width = Dim.Fill(1),
                Height = 1,
                Text = "Enter: play  Del: remove  Esc: back",
            };

            _list.OpenSelectedItem += (_, e) => OnSongSelected(e);
            Add(_list, _hint);

            Refresh();
        }

        /// <summary>Re-populate the list and keep the current-song row visible.</summary>
        public void Refresh()
        {
            var items = Utils.Songs
                .Select(s =>
                {
                    string display = SongExtensions.Title(s);
                    bool isFav = SongExtensions.IsFavorite(s);
                    return (isFav ? "★ " : "  ") + display;
                })
                .ToList();

            _list.SetSource<string>(new ObservableCollection<string>(items));
            _list.CurrentSongIndex = Utils.CurrentSongIndex;

            int idx = Math.Clamp(Utils.CurrentSongIndex, 0, Math.Max(0, items.Count - 1));
            _list.SelectedItem = idx;
            _list.TopItem = Math.Max(0, idx - 5);
        }

        protected override bool OnKeyDown(Key key)
        {
            if (key == Key.Enter)
            {
                int idx = _list.SelectedItem;
                if (idx >= 0 && idx < Utils.Songs.Length)
                    OnSongSelected(new ListViewItemEventArgs(idx, Utils.Songs[idx]));
                return true;
            }
            if (KeybindingParser.Matches(key, Keybindings.DeleteCurrentSong))
            {
                RemoveSelected();
                return true;
            }
            return base.OnKeyDown(key);
        }

        private void OnSongSelected(ListViewItemEventArgs e)
        {
            // If the selected song is an RSS feed, enter the feed view
            if (e.Item >= 0 && e.Item < Utils.Songs.Length)
            {
                var song = SongExtensions.ToSong(Utils.Songs[e.Item]);
                if (song?.URI != null && URL.IsValidRssFeed(song.URI))
                {
                    Utils.CurrentPlaylistSongIndex = e.Item;
                    Task.Run(async () =>
                    {
                        await Funcs.ContinueToRss();
                        Application.Invoke(() => RssFeedRequested?.Invoke());
                    });
                    return;
                }
            }
            Play.PlaySong(Utils.Songs, e.Item);
        }

        private void RemoveSelected()
        {
            int idx = _list.SelectedItem;
            if (idx < 0 || idx >= Utils.Songs.Length) return;

            var newSongs = Utils.Songs
                .Where((_, i) => i != idx)
                .ToArray();
            Utils.Songs = newSongs;

            // If the removed song was playing, stop and move to next
            if (idx == Utils.CurrentSongIndex)
            {
                if (newSongs.Length > 0)
                    Play.PlaySong(newSongs, Math.Min(idx, newSongs.Length - 1));
                else
                    Play.StopSong();
            }
            else if (idx < Utils.CurrentSongIndex)
            {
                Utils.CurrentSongIndex--;
            }

            Refresh();
        }

        // ── Custom ListView that highlights the currently playing row ──────────

        private class JammerListView : ListView
        {
            public int CurrentSongIndex { get; set; } = -1;

            public JammerListView()
            {
                RowRender += OnRowRenderHandler;
            }

            private void OnRowRenderHandler(object? sender, ListViewRowEventArgs args)
            {
                if (args.Row == CurrentSongIndex)
                {
                    args.RowAttribute = new Terminal.Gui.Attribute(Color.Black, Color.Green);
                }
            }
        }
    }
}

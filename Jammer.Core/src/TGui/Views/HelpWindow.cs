using Terminal.Gui;

namespace Jammer.TGui.Views
{
    /// <summary>
    /// Phase 2.2: Help view showing all keybindings in a scrollable text view.
    /// Arrow keys scroll natively. Escape exits (handled by JammerToplevel).
    /// </summary>
    public class HelpWindow : FrameView
    {
        public event Action? ExitRequested;

        public HelpWindow()
        {
            Title = Locale.Help.Description;
            ColorScheme = TGuiTheme.Base;

            var tv = new TextView
            {
                X = 0,
                Y = 0,
                Width = Dim.Fill(),
                Height = Dim.Fill(),
                ReadOnly = true,
                CanFocus = true,
                Text = BuildText(),
            };

            Add(tv);
        }

        private static string BuildText()
        {
            var sb = new System.Text.StringBuilder();

            sb.AppendLine("Key            Description");
            sb.AppendLine(new string('-', 50));

            foreach (var (key, desc) in GetHelpPairs())
                sb.AppendLine($"  {key,-18} {desc}");

            sb.AppendLine();
            sb.AppendLine("Esc: back to player");

            return sb.ToString();
        }

        private static List<(string key, string desc)> GetHelpPairs()
        {
            return new List<(string, string)>
            {
                (Keybindings.PlayPause,       Locale.Help.PlayPause),
                (Keybindings.NextSong,        Locale.Help.NextSong),
                (Keybindings.PreviousSong,    Locale.Help.PreviousSong),
                (Keybindings.Quit,            Locale.Help.Quit),
                (Keybindings.VolumeUp,        Locale.Help.VolumeUp),
                (Keybindings.VolumeDown,      Locale.Help.VolumeDown),
                (Keybindings.VolumeUpByOne,   "Volume +1"),
                (Keybindings.VolumeDownByOne, "Volume -1"),
                (Keybindings.Forward5s,       $"{Locale.Help.Forward} 5 {Locale.Help.Seconds}"),
                (Keybindings.Backwards5s,     $"{Locale.Help.Rewind} 5 {Locale.Help.Seconds}"),
                (Keybindings.ToSongStart,     "Go to Start"),
                (Keybindings.ToSongEnd,       "Go to End"),
                (Keybindings.Mute,            Locale.Help.ToggleMute),
                (Keybindings.Loop,            Locale.Help.ToggleLooping),
                (Keybindings.Shuffle,         Locale.Help.ToggleShuffle),
                (Keybindings.ShufflePlaylist, Locale.Help.ShufflePlaylist),
                (Keybindings.ShowHidePlaylist,Locale.Help.ToShowPlaylist),
                (Keybindings.AddSongToPlaylist, Locale.Help.AddsongToPlaylist),
                (Keybindings.AddSongToQueue,  "Add to Queue"),
                (Keybindings.ShowSongsInPlaylists, Locale.Help.ListAllSongsInOtherPlaylist),
                (Keybindings.AddCurrentSongToFavorites, Locale.Help.AddCurrentSongToFavorites),
                (Keybindings.ListAllPlaylists, Locale.Help.ListAllPlaylists),
                (Keybindings.PlayOtherPlaylist, Locale.Help.PlayOtherPlaylist),
                (Keybindings.SaveCurrentPlaylist, Locale.Help.SavePlaylist),
                (Keybindings.SaveAsPlaylist,  Locale.Help.SaveAs),
                (Keybindings.PlaySong,        Locale.Help.PlaySongs),
                (Keybindings.PlayRandomSong,  Locale.Help.PlayRandomSong),
                (Keybindings.DeleteCurrentSong, Locale.Help.DeleteCurrentSongFromPlaylist),
                (Keybindings.HardDeleteCurrentSong, "Delete from PC"),
                (Keybindings.RenameSong,      "Rename"),
                (Keybindings.RedownloadCurrentSong, Locale.Help.RedownloadCurrentSong),
                (Keybindings.Search,          "Search"),
                (Keybindings.SearchInPlaylist,"Search Playlist"),
                (Keybindings.SearchByAuthor,  Locale.Help.SearchByAuthor),
                (Keybindings.Choose,          "Select Song"),
                (Keybindings.Settings,        "Settings"),
                (Keybindings.EditKeybindings, Locale.Help.EditKeybinds),
                (Keybindings.ChangeLanguage,  Locale.Help.ChangeLanguage),
                (Keybindings.ChangeTheme,     "Change Theme"),
                (Keybindings.Help,            "Show Help"),
                (Keybindings.ToMainMenu,      Locale.Help.ToMainMenu),
                (Keybindings.CommandHelpScreen, Locale.Help.ShowCmdHelp),
            };
        }
    }
}

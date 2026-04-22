using Terminal.Gui;
using Jammer;
using Jammer.TGui.Views;

namespace Jammer.TGui
{
    /// <summary>
    /// Root Terminal.Gui Toplevel for Jammer.
    ///
    /// Layout (bottom-anchored, fills on resize automatically):
    ///   ┌─ active view (Dim.Fill minus 1 for PlayerStatusBar) ───────────────┐
    ///   └────────────────────────────────────────────────────────────────────┘
    ///   ┌─ PlayerStatusBar (1 row, anchored to bottom) ──────────────────────┐
    ///   └────────────────────────────────────────────────────────────────────┘
    ///
    /// All dimensions use Dim.Fill() / Pos.AnchorEnd() — no magic numbers.
    /// Media keys are handled here so they are available globally but do not
    /// fire when a focused child (e.g. ListView) has consumed the key first.
    /// </summary>
    public class JammerToplevel : Toplevel
    {
        private readonly PlayerStatusBar _statusBar;
        private readonly VisualizerBar _visualizerBar;
        private View? _currentView;
        private System.Threading.Timer? _uiTimer;

        private JammerToplevel()
        {
            _statusBar = new PlayerStatusBar();
            _visualizerBar = new VisualizerBar();
            Add(_visualizerBar);
            Add(_statusBar);
        }

        public static JammerToplevel Build()
        {
            var top = new JammerToplevel();
            top.ShowPlayer();          // default view
            top.StartUiTimer();

            // Re-layout when terminal is resized
            top.Resized += size =>
            {
                top.Width = size.Width;
                top.Height = size.Height;
                top.LayoutSubviews();
                top.SetNeedsDisplay();
                Application.Refresh();
            };

            Application.Resized = args =>
            {
                top.Width = args.Cols;
                top.Height = args.Rows;
                top.LayoutSubviews();
                top.SetNeedsDisplay();
                Application.Refresh();
            };

            return top;
        }

        // ── View switching ──────────────────────────────────────────────────

        public void ShowPlayer()
        {
            var w = new PlayerWindow();
            SetContentView(w);
        }

        public void ShowAllSongs()
        {
            var w = new AllSongsWindow();
            w.RssFeedRequested += ShowRssFeed;
            SetContentView(w);
        }

        public void ShowHelp()
        {
            var w = new HelpWindow();
            w.ExitRequested += ShowPlayer;
            SetContentView(w);
        }

        public void ShowSettings()
        {
            var w = new SettingsWindow();
            w.ExitRequested += ShowPlayer;
            SetContentView(w);
        }

        public void ShowChangeLanguage()
        {
            var w = new ChangeLanguageWindow();
            w.ExitRequested += ShowPlayer;
            SetContentView(w);
        }

        public void ShowEditKeybindings()
        {
            var w = new EditKeybindingsWindow();
            w.ExitRequested += ShowPlayer;
            SetContentView(w);
        }

        public void ShowRssFeed()
        {
            var w = new RssFeedWindow();
            w.ExitRequested += ShowAllSongs;   // return to playlist after exiting feed
            SetContentView(w);
        }

        private void SetContentView(View view)
        {
            if (_currentView != null)
                Remove(_currentView);

            _currentView = view;
            // Fill everything except the bottom status bar row
            _currentView.X = 0;
            _currentView.Y = 0;
            _currentView.Width = Dim.Fill();
            // Fill everything except the bottom two rows (visualizer + status bar)
            _currentView.Height = Dim.Fill(2);

            Add(_currentView);
            SetNeedsDisplay();
            _currentView.SetNeedsDisplay();
            Application.Refresh();
            // SetFocus on the container, then drill into the first focusable child
            // (e.g. the ListView inside SettingsWindow) so key events route correctly.
            _currentView.SetFocus();
            _currentView.FocusFirst();
        }

        // ── Global key handling ─────────────────────────────────────────────
        //
        // ProcessHotKey fires BEFORE the focused child — all global keys live here
        // so that ListView quick-jump or other child handlers cannot swallow them.
        //
        // ProcessColdKey handles volume/seek so a focused child gets arrows first
        // (e.g. ListView scrolling), and only falls through to volume if unhandled.

        public override bool ProcessHotKey(KeyEvent keyEvent)
        {
            bool M(string kb) => KeybindingParser.Matches(keyEvent, kb);

            // ── View switching ─────────────────────────────────────────────
            if (M(Keybindings.Help))             { ShowHelp();                return true; }
            if (M(Keybindings.Settings))         { ShowSettings();            return true; }
            if (M(Keybindings.ShowHidePlaylist)) { ShowAllSongs();            return true; }
            if (M(Keybindings.ChangeLanguage))   { ShowChangeLanguage();      return true; }
            if (M(Keybindings.EditKeybindings))  { ShowEditKeybindings();     return true; }
            if (M(Keybindings.Quit))             { Preferences.SaveSettings(); Application.RequestStop(); return true; }

            // ── Playback ───────────────────────────────────────────────────
            if (M(Keybindings.PlayPause))        { Play.TogglePause();        return true; }
            if (M(Keybindings.NextSong))         { Play.NextSong();           return true; }
            if (M(Keybindings.PreviousSong))     { Play.PrevSong();           return true; }
            if (M(Keybindings.PlayRandomSong))   { Play.RandomSong();         return true; }
            if (M(Keybindings.ToSongStart))      { Play.SeekSong(0, false);   return true; }
            if (M(Keybindings.ToSongEnd))
            {
                Play.SeekSong((float)Utils.SongDurationInSec, false);
                return true;
            }

            // ── Toggles ────────────────────────────────────────────────────
            if (M(Keybindings.Loop))
            {
                Preferences.loopType = Preferences.loopType switch
                {
                    LoopType.None   => LoopType.Always,
                    LoopType.Always => LoopType.Once,
                    _               => LoopType.None
                };
                return true;
            }
            if (M(Keybindings.Mute))
            {
                Play.ToggleMute();
                return true;
            }
            if (M(Keybindings.Shuffle))
            {
                Preferences.isShuffle = !Preferences.isShuffle;
                return true;
            }
            if (M(Keybindings.ShufflePlaylist))
            {
                Funcs.ShufflePlaylist();
                return true;
            }

            // ── Volume (Shift+arrow — always work, even in list views) ─────
            if (M(Keybindings.VolumeUpByOne))
            {
                AdjustVolume(0.01f);
                return true;
            }
            if (M(Keybindings.VolumeDownByOne))
            {
                AdjustVolume(-0.01f);
                return true;
            }

            // ── Volume + seek (plain arrows — only when no list view focused) ──
            // In AllSongsWindow / Settings / etc. Up/Down scroll the list,
            // so we skip volume handling there.
            bool listViewFocused = _currentView is AllSongsWindow
                                                or SettingsWindow
                                                or EditKeybindingsWindow
                                                or ChangeLanguageWindow
                                                or RssFeedWindow;
            if (!listViewFocused)
            {
                if (M(Keybindings.VolumeUp))    { AdjustVolume(Preferences.changeVolumeBy);   return true; }
                if (M(Keybindings.VolumeDown))  { AdjustVolume(-Preferences.changeVolumeBy);  return true; }
                if (M(Keybindings.Forward5s))   { Play.SeekSong(Preferences.forwardSeconds, true);  return true; }
                if (M(Keybindings.Backwards5s)) { Play.SeekSong(-Preferences.rewindSeconds, true);  return true; }
            }

            // ── Playlist management ────────────────────────────────────────
            if (M(Keybindings.SaveCurrentPlaylist))
            {
                if (!string.IsNullOrEmpty(Utils.CurrentPlaylist))
                    Playlists.Save(Utils.CurrentPlaylist, true);
                return true;
            }
            if (M(Keybindings.SaveAsPlaylist))
            {
                Application.MainLoop?.AddIdle(() =>
                {
                    string? name = TGuiDialogs.Input(
                        Locale.Player.SaveAsPlaylistMessage1,
                        Locale.Player.SaveAsPlaylistMessage2);
                    if (!string.IsNullOrWhiteSpace(name))
                        Playlists.Save(name);
                    return false;
                });
                return true;
            }
            if (M(Keybindings.AddSongToPlaylist))
            {
                Application.MainLoop?.AddIdle(() =>
                {
                    string? song = TGuiDialogs.Input(
                        Locale.Player.AddSongToPlaylistMessage1,
                        Locale.Player.AddSongToPlaylistMessage2);
                    if (string.IsNullOrWhiteSpace(song)) return false;
                    if (!Funcs.IsValidSong(song))
                    {
                        TGuiDialogs.Data(
                            Locale.Player.AddSongToPlaylistError3 + " " + song,
                            Locale.Player.AddSongToPlaylistError4,
                            true);
                        return false;
                    }
                    Play.AddSong(song);
                    return false;
                });
                return true;
            }

            return base.ProcessHotKey(keyEvent);
        }

        public override bool ProcessKey(KeyEvent keyEvent)
        {
            // Escape → back to Player view (ToMainMenu keybinding)
            if (KeybindingParser.Matches(keyEvent, Keybindings.ToMainMenu))
            {
                ShowPlayer();
                return true;
            }

            return base.ProcessKey(keyEvent);
        }

        // ProcessColdKey is intentionally left minimal.  All global keys now
        // live in ProcessHotKey (which fires before any focused child) so they
        // can never be swallowed by ListView or FrameView internals.
        public override bool ProcessColdKey(KeyEvent keyEvent) => base.ProcessColdKey(keyEvent);

        // ── Helpers ─────────────────────────────────────────────────────────

        private static void AdjustVolume(float delta)
        {
            Preferences.volume = Math.Clamp(Preferences.volume + delta, 0f, 1f);
            ManagedBass.Bass.GlobalStreamVolume = (int)(Preferences.volume * 10000);
        }

        // ── Periodic UI update timer ────────────────────────────────────────

        private void StartUiTimer()
        {
            _uiTimer = new System.Threading.Timer(_ =>
            {
                Application.MainLoop?.Invoke(() =>
                {
                    _statusBar.UpdateProgress();
                    _visualizerBar.SetNeedsDisplay();
                    // Refresh PlayerWindow if it is the active view
                    if (_currentView is PlayerWindow pw)
                        pw.Refresh();
                    Application.Refresh();
                });
            }, null, 0, Visual.refreshTime);
        }

        protected override void Dispose(bool disposing)
        {
            if (disposing)
            {
                _uiTimer?.Dispose();
            }
            base.Dispose(disposing);
        }
    }
}

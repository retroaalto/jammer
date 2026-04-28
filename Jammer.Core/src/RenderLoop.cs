using ManagedBass;
using Spectre.Console;

namespace Jammer
{
    /// <summary>
    /// Single-threaded render loop driven by a PeriodicTimer at 30 Hz (33ms).
    ///
    /// Replaces the two previous threads:
    ///   - Start.Loop()          (was Thread.Sleep(1)  = ~1000 Hz)
    ///   - Start.EqualizerLoop() (was while(true) with no exit, ~100 Hz)
    ///
    /// Tick schedule:
    ///   Every tick  (30 Hz)  : visualizer redraw if enabled + in player view
    ///   Every 3rd   (10 Hz)  : BASS position query, playback state machine, time bar
    ///   Every 30th  (1 Hz)   : console resize check
    ///   On dirty flag        : full DrawPlayer() rebuild
    /// </summary>
    public static class RenderLoop
    {
        private static CancellationTokenSource _cts = new();
        private static Task _task = Task.CompletedTask;

        // Tick counters
        private const int TickIntervalMs = 33;   // ~30 Hz
        private const int PlaybackTickInterval = 3;   // every 3rd tick  = ~10 Hz
        private const int ResizeTickInterval = 30;  // every 30th tick = ~1 Hz

        public static void Start()
        {
            _cts = new CancellationTokenSource();
            _task = Task.Run(() => RunAsync(_cts.Token));
        }

        public static void Stop()
        {
            _cts.Cancel();
            // Wait briefly for the loop to exit cleanly
            try { _task.Wait(500); } catch { }
        }

        private static async Task RunAsync(CancellationToken ct)
        {
            // One-time initialisation (mirrors what Loop() did on first entry)
            Jammer.Start.lastSeconds = -1;
            Jammer.Start.treshhold = 1;
            Utils.IsInitialized = true;

            AnsiConsole.Clear();
            TUI.RefreshCurrentView();
            AnsiConsole.Cursor.Hide();

            var timer = new PeriodicTimer(TimeSpan.FromMilliseconds(TickIntervalMs));
            int tick = 0;

            while (!ct.IsCancellationRequested)
            {
                try
                {
                    await timer.WaitForNextTickAsync(ct);
                }
                catch (OperationCanceledException)
                {
                    break;
                }

                tick++;
                AnsiConsole.Cursor.Hide();

                // ── 1 Hz: console resize ──────────────────────────────────────
                if (tick % ResizeTickInterval == 0)
                {
                    int w = Console.WindowWidth;
                    int h = Console.WindowHeight;
                    if (w != Jammer.Start.consoleWidth || h != Jammer.Start.consoleHeight)
                    {
                        Jammer.Start.consoleWidth = w;
                        Jammer.Start.consoleHeight = h;
                        AnsiConsole.Clear();
                        RenderState.NeedsFullRedraw = true;
                    }
                }

                // ── 10 Hz: playback state machine + BASS queries ──────────────
                if (tick % PlaybackTickInterval == 0)
                {
                    TickPlayback();
                }

                // ── 30 Hz: visualizer ─────────────────────────────────────────
                string view = Jammer.Start.playerView;
                bool inPlayerView = view == "default" || view == "all" || view == "rss";

                if (inPlayerView && Preferences.isVisualizer)
                {
                    var s = Jammer.Start.state;
                    if (s == MainStates.playing || s == MainStates.pause ||
                        s == MainStates.stop    || s == MainStates.idle)
                    {
                        TUI.DrawVisualizer();
                    }
                }

                // ── Dirty flags: time bar and full redraw ─────────────────────
                if (RenderState.NeedsTimeRedraw)
                {
                    RenderState.NeedsTimeRedraw = false;
                    TUI.DrawTime();
                }

                if (RenderState.NeedsFullRedraw || Jammer.Start.previousView != view)
                {
                    RenderState.NeedsFullRedraw = false;
                    Jammer.Start.previousView = view;
                    TUI.RefreshCurrentView();
                }

                // ── Keyboard (non-blocking) ───────────────────────────────────
                // Only check when loop is in a state that expects input
                if (Jammer.Start.state == MainStates.idle ||
                    Jammer.Start.state == MainStates.playing)
                {
                    await Jammer.Start.CheckKeyboardAsync();
                }
            }
        }

        /// <summary>
        /// Playback state machine + BASS position update.
        /// Runs at 10 Hz. Mirrors the logic from Start.Loop().
        /// </summary>
        private static void TickPlayback()
        {
            // Handle pending song queue entry (first slot empty = more songs queued)
            if (Utils.Songs.Length != 0)
            {
                if (Utils.Songs[0] == "" && Utils.Songs.Length > 1)
                {
                    Jammer.Start.state = MainStates.play;
                    Play.DeleteSong(0, false);
                    Play.PlaySong();
                }
            }

            switch (Jammer.Start.state)
            {
                case MainStates.play:
                    if (Utils.Songs.Length > 0)
                    {
                        Play.PlaySong();
                        TUI.ClearScreen();
                        TUI.DrawPlayer();
                        Utils.TotalMusicDurationInSec = 0;
                        Jammer.Start.state = MainStates.playing;
                    }
                    else
                    {
                        RenderState.NeedsFullRedraw = true;
                        Jammer.Start.state = MainStates.idle;
                    }
                    break;

                case MainStates.playing:
                    // Update time bar once per second of playback
                    if (Utils.TotalMusicDurationInSec - Jammer.Start.prevMusicTimePlayed >= 1)
                    {
                        RenderState.NeedsTimeRedraw = true;
                        Jammer.Start.prevMusicTimePlayed = Utils.TotalMusicDurationInSec;
                    }

                    // Song finished
                    if (Bass.ChannelIsActive(Utils.CurrentMusic) == PlaybackState.Stopped
                        && Utils.TotalMusicDurationInSec > 0)
                    {
                        Jammer.Start.prevMusicTimePlayed = 0;
                        RenderState.NeedsTimeRedraw = true;
                    }

                    // RSS auto-skip
                    if (Play.ShouldSkipRss())
                    {
                        Play.MaybeNextSong(forceNoLoop: true);
                    }
                    break;

                case MainStates.pause:
                    Play.PauseSong();
                    Jammer.Start.state = MainStates.idle;
                    break;

                case MainStates.stop:
                    Play.StopSong();
                    Jammer.Start.state = MainStates.idle;
                    break;

                case MainStates.next:
                    Play.NextSong();
                    break;

                case MainStates.previous:
                    if (Utils.TotalMusicDurationInSec > 3)
                    {
                        Play.SeekSong(0, false);
                        Jammer.Start.state = MainStates.playing;
                    }
                    else
                    {
                        Play.PrevSong();
                    }
                    break;
            }

            // Update BASS position values (was done every tick in the old loop)
            Utils.PreciseTime = Bass.ChannelBytes2Seconds(
                Utils.CurrentMusic, Bass.ChannelGetPosition(Utils.CurrentMusic));
            Utils.TotalMusicDurationInSec = Utils.PreciseTime;
            Utils.SongDurationInSec = Bass.ChannelBytes2Seconds(
                Utils.CurrentMusic, Bass.ChannelGetLength(Utils.CurrentMusic));
            Utils.MusicTimePercentage = (float)(
                Utils.TotalMusicDurationInSec / Utils.SongDurationInSec * 100);

            if (Utils.Songs.Length == 0)
            {
                Utils.CurrentSongPath = "";
            }
        }
    }
}

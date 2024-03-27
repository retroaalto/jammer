using ManagedBass;
using Spectre.Console;
using System;
using System.Runtime.InteropServices;

namespace jammer
{
    //NOTES(ra) A way to fix the drawonce - prevState

    // idle - the program wait for user input. Song is not played
    // play - Start playing - Play.PlaySong
    // playing - The music is playing. Update screen once a second or if a
    // button is pressed
    // pause - Pause song, returns to idle state

    public enum MainStates
    {
        idle,
        play,
        playing,
        pause,
        stop,
        next,
        previous
    }

    public partial class Start
    {
        //NOTE(ra) Starting state to playing.
        // public static MainStates state = MainStates.idle;
        public static MainStates state = MainStates.playing;
        public static bool drawOnce = false;
        private static Thread loopThread = new Thread(() => { });
        public static int consoleWidth = Console.WindowWidth;
        public static int consoleHeight = Console.WindowHeight;
        public static double lastSeconds = -1;
        public static double lastPlaybackTime = -1;
        public static double treshhold = 1;
        private static bool initWMP = false;
        public static double prevMusicTimePlayed = 0;

        //
        // Run
        //
        public static void Run(string[] args)
        {
            Debug.dprint("Run");

            for (int i = 0; i < args.Length; i++)
            {
                if (args[i] == "-d")
                {
                    Utils.isDebug = true;
                    Debug.dprint("\n--- Debug Started ---\n");
                    List<string> argumentsList = new List<string>(args);
                    argumentsList.RemoveAt(0);
                    args = argumentsList.ToArray();
                    break;
                }
                if (args[i] == "-help" || args[i] == "-h" || args[i] == "--help" || args[i] == "--h" || args[i] == "-?" || args[i] == "?" || args[i] == "help")
                {
                    TUI.ClearScreen();
                    TUI.Help();
                    return;
                }
                if (args[i] == "-v" || args[i] == "--version" || args[i] == "version")
                {
                    TUI.ClearScreen();
                    TUI.Version();
                    return;
                }
            }

            Utils.songs = args;
            if (Utils.songs.Length != 0)
            {
                if (Utils.songs[0] == "playlist" || Utils.songs[0] == "pl")
                {
                    // TUI.ClearScreen();
                    TUI.PlaylistCli(Utils.songs);
                }
                if (Utils.songs[0] == "start")
                {
                    // open explorer in jammer folder
                    AnsiConsole.MarkupLine("[green]Opening Jammer folder...[/]");
                    // if windows
                    if (RuntimeInformation.IsOSPlatform(OSPlatform.Windows))
                    {
                        System.Diagnostics.Process.Start("explorer.exe", Utils.jammerPath);
                    }
                    // if linux
                    else if (RuntimeInformation.IsOSPlatform(OSPlatform.Linux))
                    {
                        System.Diagnostics.Process.Start("xdg-open", Utils.jammerPath);
                    }
                    return;
                }
                if (Utils.songs[0] == "update")
                {
                    if (RuntimeInformation.IsOSPlatform(OSPlatform.Linux))
                    {
                        AnsiConsole.MarkupLine("[red]Run the update command[/]");
                        return;
                    }
                    AnsiConsole.MarkupLine("[green]Checking for updates...[/]");

                    string latestVersion = Update.CheckForUpdate(Utils.version);
                    if (latestVersion != "")
                    {
                        AnsiConsole.MarkupLine("[green]Update found![/]" + "\n" + "Version: [green]" + latestVersion + "[/]");
                        AnsiConsole.MarkupLine("[green]Downloading...[/]");
                        string downloadPath = Update.UpdateJammer(latestVersion);

                        AnsiConsole.MarkupLine("[green]Downloaded to: " + downloadPath + "[/]");
                        AnsiConsole.MarkupLine("[cyan]Installing...[/]");
                        // Run run_command.bat with argument as the path to the downloaded file
                        System.Diagnostics.Process.Start("run_command.bat", downloadPath);
                    }
                    else
                    {
                        AnsiConsole.MarkupLine("[green]Jammer is up to date![/]");
                    }
                    Environment.Exit(0);
                }
            }

            Preferences.CheckJammerFolderExists();

            StartUp();
            
        }

        public static void StartUp() {

            if (!Bass.Init())
            {
                Message.Data("Can't initialize device", "Error", true);
                return;
            }

            state = MainStates.idle; // Start in idle state if no songs are given
            if (Utils.songs.Length != 0)
            {
                Utils.songs = Absolute.Correctify(Utils.songs);
                Utils.currentSong = Utils.songs[0];
                Utils.currentSongIndex = 0;
                state = MainStates.playing; // Start in play state if songs are given
                Play.PlaySong(Utils.songs, Utils.currentSongIndex);
            }

            Debug.dprint("Start Loop");
            loopThread = new Thread(Loop);
            loopThread.Start();
        }

        //
        // Main loop
        //
        public static void Loop()
        {

            if (initWMP == false)
            {
                initWMP = true;
            }

            lastSeconds = -1;
            treshhold = 1;
            // if (Utils.audioStream == null || Utils.currentMusic == null) {
            //     Debug.dprint("Audiostream");
            //     return;
            // }

            TUI.ClearScreen();
            drawOnce = true;
            TUI.RehreshCurrentView();
            while (true)
            {
                if (Utils.songs.Length != 0)
                {
                    // if the first song is "" then there are more songs
                    if (Utils.songs[0] == "" && Utils.songs.Length > 1)
                    {
                        state = MainStates.play;
                        Play.DeleteSong(0);
                        Play.PlaySong();
                    }
                }

                if (consoleWidth != Console.WindowWidth || consoleHeight != Console.WindowHeight)
                {
                    consoleHeight = Console.WindowHeight;
                    consoleWidth = Console.WindowWidth;
                    TUI.RehreshCurrentView();
                }

                switch (state)
                {
                    case MainStates.idle:
                        TUI.ClearScreen();
                        CheckKeyboard();
                        //FIXME(ra) This is a workaround for screen to update once when entering the state.
                        if (drawOnce)
                        {
                            TUI.DrawPlayer();
                            drawOnce = false;
                        }
                        break;

                    case MainStates.play:
                        Debug.dprint("Play");
                        if (Utils.songs.Length > 0)
                        {
                            Debug.dprint("Play - len");
                            Play.PlaySong();
                            TUI.ClearScreen();
                            TUI.DrawPlayer();
                            drawOnce = true;
                            Utils.MusicTimePlayed = 0;
                            state = MainStates.playing;
                        }
                        break;

                    case MainStates.playing:
                        // get current time

                        Utils.preciseTime = Bass.ChannelBytes2Seconds(Utils.currentMusic, Bass.ChannelGetPosition(Utils.currentMusic));
                        // get current time in seconds
                        Utils.MusicTimePlayed = Bass.ChannelBytes2Seconds(Utils.currentMusic, Bass.ChannelGetPosition(Utils.currentMusic));
                        // get whole song length in seconds
                        //Utils.currentMusicLength = Utils.audioStream.Length / Utils.audioStream.WaveFormat.AverageBytesPerSecond;
                        Utils.currentMusicLength = Bass.ChannelBytes2Seconds(Utils.currentMusic, Bass.ChannelGetLength(Utils.currentMusic));


                        //FIXME(ra) This is a workaround for screen to update once when entering the state.
                        if (drawOnce)
                        {
                            TUI.DrawPlayer();
                            drawOnce = false;
                        }

                        // every second, update screen, use MusicTimePlayed, and prevMusicTimePlayed
                        if (Utils.MusicTimePlayed - prevMusicTimePlayed >= 1)
                        {
                            TUI.RehreshCurrentView();
                            prevMusicTimePlayed = Utils.MusicTimePlayed;
                        }

                        // If the song is finished, play next song
                        if (Bass.ChannelIsActive(Utils.currentMusic) == PlaybackState.Stopped && Utils.MusicTimePlayed > 0)
                        {
                            Play.MaybeNextSong();
                            prevMusicTimePlayed = 0;
                            TUI.RehreshCurrentView();
                        }

                        CheckKeyboard();
                        break;

                    case MainStates.pause:
                        Play.PauseSong();
                        state = MainStates.idle;
                        break;

                    case MainStates.stop:
                        Play.StopSong();
                        state = MainStates.idle;
                        break;

                    case MainStates.next:
                        Debug.dprint("next");
                        Play.NextSong();
                        TUI.ClearScreen();
                        break;

                    case MainStates.previous:
                        Play.PrevSong();
                        TUI.ClearScreen();
                        break;
                }
            }
        }

        public static void SetLastseconds(float s)
        {
            lastSeconds = s;
        }
    }
}

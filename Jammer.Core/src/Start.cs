using ManagedBass;
using Jammer;
using System.Runtime.InteropServices;
using Spectre.Console;
using System.Diagnostics;
using System.Text.RegularExpressions;


namespace Jammer
{
    //NOTES(ra) A way to fix the drawonce - prevState

    // idle - the program wait for user input. Song is not played
    // play - Start playing - Play.PlaySong
    // playing - The music is playing. Update screen once a second or if a -
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

    public static partial class Start
    {
        //NOTE(ra) Starting state to playing.
        // public static MainStates state = MainStates.idle;
        // ! Translations needed to locales
        public static MainStates state = MainStates.playing;
        public static int consoleWidth = Console.WindowWidth;
        public static int consoleHeight = Console.WindowHeight;
        public static bool CLI = false;
        public static double lastSeconds = -1;
        public static double lastPlaybackTime = -1;
        public static double treshhold = 1;
        public static double prevMusicTimePlayed = 0;

        //
        // Run
        //

        public static void Run(string[] args)
        {
            Log.Info("Starting Jammer...");
            try
            {
                Console.OutputEncoding = System.Text.Encoding.UTF8;
                Log.Info("Output encoding set to UTF8");
            }
            catch (Exception e)
            {
                Console.WriteLine(e.Message);
                Log.Error("Error setting output encoding to UTF8");
            }

            Utils.Songs = args;
            // Theme init
            Themes.Init();
            Log.Info("Themes initialized");
            Debug.dprint("Run");
            if (args.Length > 0)
            {
                CheckArgs(args);
            }

            Preferences.CheckJammerFolderExists();
            IniFileHandling.Create_KeyDataIni(0);
            IniFileHandling.Create_KeyDataIni(2);
            StartUp();
        }

        public static void StartUp()
        {
            try
            {
                if (!Bass.Init())
                {
                    /* Message.Data(Locale.OutsideItems.InitializeError, Locale.OutsideItems.Error, true); */
                    Log.Error("BASS initialization failed");
                    return;
                }
                // Additional code if initialization is successful
            }
            catch (Exception ex)
            {
                // Log the exception message and stack trace
                Console.WriteLine($"Exception during BASS initialization: {ex.Message}");
                Console.WriteLine(ex.StackTrace);
            }
            Log.Info("BASS initialized");

            // Initialize the keyboard hook
            Log.Info("Initializing keyboard hook");
            InitializeSharpHook();
            // Or specify a specific name in the current dir
            state = MainStates.idle; // Start in idle state if no songs are given
            if (Utils.Songs.Length != 0)
            {
                Utils.Songs = Absolute.Correctify(Utils.Songs);
                //NOTE(ra) Correctify removes filenames from Utils.Songs. 
                //If there is one file that doesn't exist this is a fix
                if (Utils.Songs.Length == 0)
                {
                    Debug.dprint("No songs found");
                    AnsiConsole.WriteLine("No songs found. Exiting...");
                    Environment.Exit(1);
                }
                Utils.CurrentSongPath = Utils.Songs[0];
                Utils.CurrentSongIndex = 0;
                state = MainStates.playing; // Start in play state if songs are given
                Play.PlaySong(Utils.Songs, Utils.CurrentSongIndex);
            }

            Console.CancelKeyPress += new ConsoleCancelEventHandler(Exit.OnExit);
            AppDomain.CurrentDomain.ProcessExit += new EventHandler(Exit.OnProcessExit);

            Debug.dprint("Start RenderLoop");
            RenderLoop.Start();
        }

        //
        // Render dirty flags — set by keyboard/state handlers, consumed by RenderLoop.
        // Keep these as pass-through properties pointing to RenderState for
        // backwards compatibility with any callers not yet migrated.
        //
        public static bool drawTime
        {
            get => RenderState.NeedsTimeRedraw;
            set => RenderState.NeedsTimeRedraw = value;
        }
        public static bool drawWhole
        {
            get => RenderState.NeedsFullRedraw;
            set => RenderState.NeedsFullRedraw = value;
        }
        // drawVisualizer is no longer needed — visualizer runs every tick in RenderLoop.
        public static bool drawVisualizer = false; // kept for source compatibility only

        public static string previousView = "default";
        public static bool debug = false;
        /// <summary>
        /// Removes "[" and "]" from a string to prevent Spectre.Console from blowing up.
        /// </summary>
        /// <param name="input">The string to sanitize</param>
        /// <returns>The sanitized string</returns>
        /// <remarks>
        /// This is a workaround for a bug in Spectre.Console that causes it to crash when it encounters "[" or "]" in a string.
        /// </remarks>
        /// <example>
        /// <code>
        /// string sanitized = Sanitize("Hello world [lol]");
        /// Output: "Hello world lol"
        /// </code>        
        public static string? Sanitize(string? input, bool removeBrakets = false)
        {
            if (string.IsNullOrEmpty(input))
            {
                return input;
            }

            if (removeBrakets)
            {
                input = input.Replace("[", "");
                input = input.Replace("]", "");
            }
            else
            {
                input = input.Replace("[", "[[");
                input = input.Replace("]", "]]");
            }
            input = input.Replace("\"", "\'");
            return input;
        }

        // replace inputSaying every character inside of [] @"\[.*?\]
        /// <summary>
        /// Removes all occurrences of text enclosed in square brackets from the input string.
        /// </summary>
        /// <param name="input">The input string to be purged.</param>
        /// <returns>The input string with all occurrences of text enclosed in square brackets removed.</returns>
        public static string Purge(string input)
        {
            string pattern = @"\[.*?\]";
            string replacement = "";
            Regex rgx = new(pattern);
            string text = rgx.Replace(input, replacement);
            return text;
        }

        /// <summary>
        /// Sanitizes an array of strings by calling the Sanitize method on each element.
        /// </summary>
        /// <param name="input">The array of strings to be sanitized.</param>
        /// <returns>The sanitized array of strings.</returns>
        public static string[] Sanitize(string[] input)
        {
            for (int i = 0; i < input.Length; i++)
            {
                input[i] = Sanitize(input[i]);
            }
            return input;
        }
    }
}


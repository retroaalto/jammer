using Terminal.Gui;

namespace Jammer.TGui
{
    /// <summary>
    /// Thin wrapper around Terminal.Gui v2 Application lifetime.
    /// All new-UI work goes through this class; old TUI.cs is untouched.
    /// </summary>
    public static class TGuiApp
    {
        public static bool Enabled { get; private set; }

        public static void Init()
        {
            Application.Init();

            // v2: Color.Default maps to the terminal's own background automatically.
            // No need to call UseDefaultColors() — the driver handles it internally.

            TGuiTheme.Apply();
            Enabled = true;
        }

        public static void Shutdown()
        {
            Application.Shutdown();
            Enabled = false;
        }

        public static void Run(Toplevel top) => Application.Run(top, (ex) =>
        {
            // Log but don't crash — non-fatal render errors are survived.
            Log.Error($"TGui render error (non-fatal): {ex.GetType().Name}: {ex.Message}");
            return true; // true = handled, continue running
        });

        /// <summary>
        /// Schedule an action to run on the UI thread. Safe to call from any thread.
        /// </summary>
        public static void Invoke(Action action)
        {
            Application.Invoke(action);
        }
    }
}

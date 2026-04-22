using Terminal.Gui;

namespace Jammer.TGui
{
    /// <summary>
    /// Thin wrapper around Terminal.Gui Application lifetime.
    /// All new-UI work goes through this class; old TUI.cs is untouched.
    /// </summary>
    public static class TGuiApp
    {
        public static bool Enabled { get; private set; }

        public static void Init()
        {
            Application.Init();
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
            // Log but don't crash — Terminal.Gui v1 NStack rendering bugs with
            // certain string widths can throw from Redraw; we survive them.
            Log.Error($"TGui render error (non-fatal): {ex.GetType().Name}: {ex.Message}");
            return true; // true = handled, continue running
        });

        /// <summary>
        /// Schedule an action to run on the UI thread. Safe to call from any thread.
        /// </summary>
        public static void Invoke(Action action)
        {
            Application.MainLoop.Invoke(action);
        }
    }
}

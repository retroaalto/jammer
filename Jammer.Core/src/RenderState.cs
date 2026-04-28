namespace Jammer
{
    /// <summary>
    /// Thread-safe render dirty flags. Set by keyboard/state handlers,
    /// consumed and cleared by RenderLoop on each tick.
    /// </summary>
    public static class RenderState
    {
        /// <summary>Full player view rebuild needed (view switch, song change, resize, etc.)</summary>
        public static volatile bool NeedsFullRedraw = false;

        /// <summary>Time bar / progress bar redraw needed (playback position changed).</summary>
        public static volatile bool NeedsTimeRedraw = false;
    }
}

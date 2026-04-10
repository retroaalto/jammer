using Spectre.Console;

namespace Jammer.Components
{
    /// <summary>
    /// Component responsible for rendering the audio visualizer
    /// Extracted from TUI.DrawVisualizer method
    /// </summary>
    public class VisualizerComponent : IDirectRenderer, IStatefulComponent
    {
        private bool _isPlaying;

        // Cache the last rendered line so we can skip the AnsiConsole write
        // (and the cursor move) when the visual output has not changed.
        // The cache is shared across instances because the visualizer is a single
        // stateless line and all instances render the same content.
        private static string? _lastRenderedLine = null;
        private static int    _lastRenderedWidth  = -1;

        // Running total of cache-hit skips since process start (monotonically increasing).
        // Callers may snapshot this to detect whether a given RenderDirect call was a cache hit.
        public static int SkipCount { get; private set; } = 0;

        public VisualizerComponent()
        {
            UpdateState();
        }

        public void UpdateState()
        {
            _isPlaying = Start.state == MainStates.playing || Start.state == MainStates.play;
        }

        public (int X, int Y) CalculatePosition(LayoutConfig layout)
        {
            return layout.GetVisualizerPosition();
        }

        public void RenderDirect(LayoutConfig layout)
        {
            int visualWidth = layout.CalculateVisualWidth();
            string line = Visual.GetSongVisual(visualWidth, _isPlaying);

            // Skip the cursor-move + terminal write if nothing has changed.
            // This is the primary guard against redundant AnsiConsole work at 30 Hz.
            if (line == _lastRenderedLine && visualWidth == _lastRenderedWidth)
            {
                SkipCount++;
                return;
            }

            _lastRenderedLine  = line;
            _lastRenderedWidth = visualWidth;

            var position = CalculatePosition(layout);
            AnsiConsole.Cursor.SetPosition(position.X, position.Y);
            AnsiConsole.MarkupLine(line);
        }

        /// <summary>
        /// Renders the visualizer directly to console at the calculated position
        /// </summary>
        /// <param name="layout">Layout configuration for positioning</param>
        public static void DrawVisualizerToConsole(LayoutConfig layout)
        {
            var component = new VisualizerComponent();
            component.RenderDirect(layout);
        }

        /// <summary>
        /// Invalidates the render cache so the next frame is always written.
        /// Call this after a full-screen redraw (resize, view change, etc.) to
        /// ensure the visualizer line is repainted even if the FFT data is the same.
        /// </summary>
        public static void InvalidateCache()
        {
            _lastRenderedLine  = null;
            _lastRenderedWidth = -1;
        }

        /// <summary>
        /// Checks if visualizer should be rendered based on preferences
        /// </summary>
        /// <returns>True if visualizer should be shown</returns>
        public static bool ShouldShowVisualizer()
        {
            return Preferences.isVisualizer;
        }
    }
}
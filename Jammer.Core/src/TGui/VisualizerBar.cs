using System.Drawing;
using Terminal.Gui;
using ManagedBass;

namespace Jammer.TGui
{
    /// <summary>
    /// Single-row Unicode FFT visualizer bar with temporal smoothing.
    ///
    /// Uses fast-attack / slow-decay per-column smoothing so bars snap up
    /// instantly when a frequency spikes but decay gradually, giving a fluid
    /// VU-meter feel instead of raw per-frame jumps.
    /// </summary>
    public class VisualizerBar : View
    {
        // Per-column smoothed height values (same scale as Visual.GetSongVisualRaw output).
        private float[] _smoothed = Array.Empty<float>();

        // Decay multiplier applied each frame when the new value is below the smoothed one.
        // 0.75 = drops ~75% of the way to silence in ~4 frames (~140 ms at 30 fps).
        private const float Decay = 0.75f;

        public VisualizerBar()
        {
            Height = 1;
            Width = Dim.Fill();
            X = 0;
            Y = Pos.AnchorEnd(2);
            CanFocus = false;
            ColorScheme = TGuiTheme.LabelScheme(TGuiTheme.CurrentSongColor);

            DrawingContent += OnDrawingContent;
        }

        private void OnDrawingContent(object? sender, DrawEventArgs e)
        {
            bool isPlaying = Bass.ChannelIsActive(Utils.CurrentMusic) == PlaybackState.Playing;
            int width = Viewport.Width;
            if (width <= 0)
                return;

            float[] raw = Visual.GetSongVisualRaw(width, isPlaying);
            string[] map = Visual.GetUnicodeMap();

            // Resize smoothing buffer when the terminal width changes.
            if (_smoothed.Length != raw.Length)
                _smoothed = new float[raw.Length];

            // Fast attack, slow decay.
            for (int i = 0; i < raw.Length; i++)
            {
                if (raw[i] >= _smoothed[i])
                    _smoothed[i] = raw[i];          // instant attack
                else
                    _smoothed[i] *= Decay;          // gradual decay
            }

            // Map smoothed values to Unicode block characters.
            var sb = new System.Text.StringBuilder(raw.Length);
            for (int i = 0; i < _smoothed.Length; i++)
            {
                int idx = (int)(_smoothed[i] * (map.Length - 1));
                idx = Math.Clamp(idx, 0, map.Length - 1);
                sb.Append(map[idx]);
            }

            string line = sb.ToString();

            // Pad or truncate to exactly `width` columns.
            if (line.Length < width)
                line = line.PadRight(width);
            else if (line.Length > width)
                line = line[..width];

            Driver?.SetAttribute(ColorScheme.Normal);
            Move(0, 0);
            Driver?.AddStr(line);
        }
    }
}

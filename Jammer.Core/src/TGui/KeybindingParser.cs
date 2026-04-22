using Terminal.Gui;

namespace Jammer.TGui
{
    /// <summary>
    /// Converts a Keybindings string (as produced by <see cref="Keybindings.CheckValue"/>)
    /// into a Terminal.Gui v2 <see cref="Key"/> value for use in key-event comparisons.
    /// </summary>
    public static class KeybindingParser
    {
        /// <summary>
        /// Parse a normalised keybinding string into a Terminal.Gui v2 Key.
        /// Returns <see cref="Key.Empty"/> if the string cannot be parsed.
        /// </summary>
        public static Key Parse(string keybinding)
        {
            if (string.IsNullOrWhiteSpace(keybinding))
                return Key.Empty;

            var parts = keybinding
                .Split('+')
                .Select(p => p.Trim())
                .Where(p => p.Length > 0)
                .ToList();

            if (parts.Count == 0)
                return Key.Empty;

            // Collect modifier flags
            bool ctrl = false, alt = false, shift = false;
            var remaining = new List<string>();
            foreach (var part in parts)
            {
                switch (part.ToLowerInvariant())
                {
                    case "ctrl":  ctrl  = true; break;
                    case "alt":   alt   = true; break;
                    case "shift": shift = true; break;
                    default:      remaining.Add(part); break;
                }
            }

            if (remaining.Count != 1)
                return Key.Empty;

            string token = remaining[0];

            Key baseKey;

            // Single character
            if (token.Length == 1)
            {
                char ch = token[0];
                // Use the Key implicit char operator which correctly maps chars to KeyCode values.
                // (Key)'h' = KeyCode.H (no shift), (Key)'H' = KeyCode.H | ShiftMask.
                // Do NOT normalize case here — the character's case determines whether Shift is implied.
                baseKey = (Key)ch;
            }
            else
            {
                // Named special keys
                Key? named = token.ToLowerInvariant() switch
                {
                    "spacebar" or "space"     => Key.Space,
                    "escape" or "esc"         => Key.Esc,
                    "enter" or "return"       => Key.Enter,
                    "tab"                     => Key.Tab,
                    "backspace" or "bspace"   => Key.Backspace,
                    "delete" or "del"         => Key.Delete,
                    "insert" or "ins"         => Key.InsertChar,
                    "home"                    => Key.Home,
                    "end"                     => Key.End,
                    "pageup"                  => Key.PageUp,
                    "pagedown"                => Key.PageDown,
                    "uparrow"   or "up"       => Key.CursorUp,
                    "downarrow" or "down"     => Key.CursorDown,
                    "leftarrow" or "left"     => Key.CursorLeft,
                    "rightarrow" or "right"   => Key.CursorRight,
                    "f1"  => Key.F1,  "f2"  => Key.F2,  "f3"  => Key.F3,  "f4"  => Key.F4,
                    "f5"  => Key.F5,  "f6"  => Key.F6,  "f7"  => Key.F7,  "f8"  => Key.F8,
                    "f9"  => Key.F9,  "f10" => Key.F10, "f11" => Key.F11, "f12" => Key.F12,
                    _ => null,
                };

                if (named == null)
                    return Key.Empty;

                baseKey = named;
            }

            if (ctrl)  baseKey = baseKey.WithCtrl;
            if (alt)   baseKey = baseKey.WithAlt;
            if (shift) baseKey = baseKey.WithShift;

            return baseKey;
        }

        /// <summary>
        /// Returns true if <paramref name="key"/> matches the given keybinding string.
        /// </summary>
        public static bool Matches(Key key, string keybinding)
        {
            Key parsed = Parse(keybinding);
            if (parsed == Key.Empty) return false;
            return key == parsed;
        }
    }
}

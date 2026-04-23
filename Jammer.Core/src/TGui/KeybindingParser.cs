using Terminal.Gui;

namespace Jammer.TGui
{
    /// <summary>
    /// Converts a Keybindings string (as produced by <see cref="Keybindings.CheckValue"/>)
    /// into a Terminal.Gui <see cref="Key"/> value for use in key-event comparisons.
    ///
    /// CheckValue normalises modifier syntax so the strings arriving here look like:
    ///   "q"          single lowercase letter (no modifier)
    ///   "Q"          single uppercase letter (Shift was present but consumed)
    ///   "Ctrl + q"   ctrl modifier + letter
    ///   "Alt + q"    alt modifier + letter
    ///   "Spacebar"   named special key
    ///   "RightArrow" named special key
    ///   etc.
    /// </summary>
    public static class KeybindingParser
    {
        /// <summary>
        /// Parse a normalised keybinding string into a Terminal.Gui Key.
        /// Returns <see cref="Key.Unknown"/> if the string cannot be parsed.
        /// </summary>
        public static Key Parse(string keybinding)
        {
            if (string.IsNullOrWhiteSpace(keybinding))
                return Key.Unknown;

            var parts = keybinding
                .Split('+')
                .Select(p => p.Trim())
                .Where(p => p.Length > 0)
                .ToList();

            if (parts.Count == 0)
                return Key.Unknown;

            // Collect modifier masks
            Key mask = 0;
            var remaining = new List<string>();
            foreach (var part in parts)
            {
                switch (part.ToLowerInvariant())
                {
                    case "ctrl":  mask |= Key.CtrlMask;  break;
                    case "alt":   mask |= Key.AltMask;   break;
                    case "shift": mask |= Key.ShiftMask; break;
                    default:      remaining.Add(part);   break;
                }
            }

            if (remaining.Count != 1)
                return Key.Unknown;

            string token = remaining[0];

            // Single character
            if (token.Length == 1)
            {
                char ch = token[0];
                Key baseKey = (Key)ch;
                return baseKey | mask;
            }

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

            if (named.HasValue)
                return named.Value | mask;

            return Key.Unknown;
        }

        /// <summary>
        /// Returns true if <paramref name="keyEvent"/> matches the given keybinding string.
        /// </summary>
        public static bool Matches(KeyEvent keyEvent, string keybinding)
        {
            Key parsed = Parse(keybinding);
            if (parsed == Key.Unknown) return false;
            return keyEvent.Key == parsed;
        }
    }
}

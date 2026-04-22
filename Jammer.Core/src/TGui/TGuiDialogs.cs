using Terminal.Gui;

namespace Jammer.TGui
{
    /// <summary>
    /// Phase 2.7: Terminal.Gui equivalents of Message.Input, Message.Data and
    /// Message.CustomMenuSelect. All methods block on the UI thread via
    /// Application.Run(dialog) and return when the user dismisses the dialog.
    /// </summary>
    public static class TGuiDialogs
    {
        // ── Text input ──────────────────────────────────────────────────────

        /// <summary>
        /// Prompt the user for a single line of text.
        /// Returns the entered string, or null if the user cancelled.
        /// </summary>
        public static string? Input(string prompt, string title, string prefill = "")
        {
            string? result = null;

            var dialog = new Dialog(title, 64, 9);

            var label = new Label
            {
                X = 1, Y = 0,
                Width = Dim.Fill(1),
                Text = prompt,
            };

            var field = new TextField(prefill)
            {
                X = 1, Y = 2,
                Width = Dim.Fill(2),
            };

            var ok     = new Button("OK", is_default: true);
            var cancel = new Button("Cancel");

            ok.Clicked     += () => { result = field.Text?.ToString(); dialog.Running = false; };
            cancel.Clicked += () => { result = null;                   dialog.Running = false; };

            dialog.AddButton(ok);
            dialog.AddButton(cancel);
            dialog.Add(label, field);

            Application.Run(dialog);
            return result;
        }

        // ── Informational / error message ───────────────────────────────────

        /// <summary>
        /// Show an informational or error message with an OK button.
        /// </summary>
        public static void Data(string message, string title, bool isError = false)
        {
            MessageBox.Query(
                Math.Min(Math.Max(message.Length + 6, 40), Application.Driver?.Cols ?? 80),
                7,
                title,
                message,
                "OK");
        }

        // ── Custom menu / list selection ────────────────────────────────────

        /// <summary>
        /// Show a scrollable list dialog and return the DataURI of the selected item,
        /// or null / "__CANCELLED__" if the user pressed Escape.
        /// </summary>
        public static string? CustomMenuSelect(
            CustomSelectInput[] options,
            string title,
            CustomSelectInputSettings? settings = null)
        {
            if (options == null || options.Length == 0)
                return "__CANCELLED__";

            settings ??= new CustomSelectInputSettings();

            string? result = "__CANCELLED__";

            var dialog = new Dialog(title, 70, 20);

            // Build display strings: title  [author]
            var items = options
                .Select(o =>
                {
                    string left  = o.Title ?? "";
                    string right = string.IsNullOrEmpty(o.Author) ? "" : $"  [{o.Author}]";
                    return left + right;
                })
                .ToList();

            var list = new ListView
            {
                X = 1, Y = 0,
                Width = Dim.Fill(1),
                Height = Dim.Fill(3),
                CanFocus = true,
            };
            list.SetSource(items);
            list.SelectedItem = Math.Clamp(settings.StartIndex, 0, Math.Max(0, options.Length - 1));

            var hint = new Label
            {
                X = 1,
                Y = Pos.AnchorEnd(2),
                Width = Dim.Fill(1),
                Height = 1,
                Text = "Enter: select  Esc: cancel",
            };

            var ok     = new Button("OK", is_default: true);
            var cancel = new Button("Cancel");

            ok.Clicked += () =>
            {
                int idx = list.SelectedItem;
                result = (idx >= 0 && idx < options.Length)
                    ? options[idx].DataURI
                    : "__CANCELLED__";
                dialog.Running = false;
            };
            cancel.Clicked += () => { result = "__CANCELLED__"; dialog.Running = false; };

            // Enter on list row = same as OK
            list.OpenSelectedItem += _ => ok.OnClicked();

            dialog.AddButton(ok);
            dialog.AddButton(cancel);
            dialog.Add(list, hint);

            Application.Run(dialog);
            return result;
        }
    }
}

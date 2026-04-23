using Terminal.Gui;

namespace Jammer.TGui.Views
{
    /// <summary>
    /// Phase 2.4: Edit keybindings view.
    /// Shows all keybindings in a two-column list. Enter opens a TextField dialog
    /// where the user types the new binding (e.g. "Shift + E"). Escape exits.
    /// </summary>
    public class EditKeybindingsWindow : FrameView
    {
        private readonly ListView _list;
        private readonly Label _hint;
        private List<(string KeyName, string Value, string Description)> _binds = new();

        public event Action? ExitRequested;

        public EditKeybindingsWindow()
        {
            Title = Locale.Help.EditKeybinds;
            Border.BorderStyle = BorderStyle.Single;

            _list = new ListView
            {
                X = 0,
                Y = 0,
                Width = Dim.Fill(),
                Height = Dim.Fill(1),
                CanFocus = true,
            };
            _list.OpenSelectedItem += OnActivate;

            _hint = new Label
            {
                X = 1,
                Y = Pos.AnchorEnd(1),
                Width = Dim.Fill(1),
                Height = 1,
                Text = "Enter: edit  Del+Shift+Alt: reset all  Esc: back"
            };

            Add(_list, _hint);
            Reload();
        }

        public override bool ProcessKey(KeyEvent keyEvent)
        {
            if (keyEvent.Key == Key.Enter)
            {
                int idx = _list.SelectedItem;
                if (idx >= 0 && idx < _binds.Count)
                {
                    var bind = _binds[idx];
                    // Defer so Application.Run(dialog) is not called from within ProcessKey.
                    Application.MainLoop?.AddIdle(() =>
                    {
                        OnActivate(new ListViewItemEventArgs(idx, bind));
                        return false;
                    });
                }
                return true;
            }
            if (keyEvent.Key == Key.Esc)
            {
                ExitRequested?.Invoke();
                return true;
            }
            return base.ProcessKey(keyEvent);
        }

        private void Reload()
        {
            _binds = IniFileHandling.GetAllKeybinds();
            var rows = _binds
                .Select(b => $"{b.Description,-35}  {b.Value,-25}  [{b.KeyName}]")
                .ToList();
            int sel = _list.SelectedItem;
            _list.SetSource(rows);
            _list.SelectedItem = Math.Clamp(sel, 0, Math.Max(0, rows.Count - 1));
        }

        private void OnActivate(ListViewItemEventArgs e)
        {
            if (e.Item < 0 || e.Item >= _binds.Count) return;
            var bind = _binds[e.Item];

            string? newValue = PromptKeybind(bind.Description, bind.Value);
            if (newValue == null) return;

            IniFileHandling.WriteKeybind(bind.KeyName, newValue);
            Reload();
        }

        private static string? PromptKeybind(string actionName, string current)
        {
            string? result = null;
            var dialog = new Dialog($"Edit: {actionName}", 60, 9);

            var hint = new Label
            {
                X = 1, Y = 0,
                Text = "Type binding (e.g. Shift + E, Ctrl + Alt + Delete)"
            };
            var currentLabel = new Label
            {
                X = 1, Y = 1,
                Text = $"Current: {current}"
            };
            var field = new TextField(current)
            {
                X = 1, Y = 3, Width = Dim.Fill(2)
            };
            var ok = new Button("OK", is_default: true);
            var cancel = new Button("Cancel");

            ok.Clicked += () => { result = field.Text?.ToString()?.Trim(); dialog.Running = false; };
            cancel.Clicked += () => { result = null; dialog.Running = false; };

            dialog.AddButton(ok);
            dialog.AddButton(cancel);
            dialog.Add(hint, currentLabel, field);
            Application.Run(dialog);
            return string.IsNullOrWhiteSpace(result) ? null : result;
        }
    }
}

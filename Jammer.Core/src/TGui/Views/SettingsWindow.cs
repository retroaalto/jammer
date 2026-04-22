using System.Collections.ObjectModel;
using Terminal.Gui;

namespace Jammer.TGui.Views
{
    /// <summary>
    /// Phase 2.3: Settings view.
    /// Arrow keys navigate rows, Enter activates the setting action, Escape exits.
    /// </summary>
    public class SettingsWindow : FrameView
    {
        private readonly ListView _list;
        private readonly Label _hint;
        private List<SettingRow> _rows = new();

        public event Action? ExitRequested;

        private record SettingRow(string Name, Func<string> ValueGetter, Action Activate);

        public SettingsWindow()
        {
            Title = "Settings";
            BorderStyle = LineStyle.Single;

            _list = new ListView
            {
                X = 0,
                Y = 0,
                Width = Dim.Fill(),
                Height = Dim.Fill(1),
                CanFocus = true,
            };
            _list.OpenSelectedItem += (_, e) => OnActivate(e);

            _hint = new Label
            {
                X = 1,
                Y = Pos.AnchorEnd(1),
                Width = Dim.Fill(1),
                Height = 1,
                Text = "Enter: change  Esc: back"
            };

            Add(_list, _hint);
            BuildRows();
            RenderList();
        }

        protected override bool OnKeyDown(Key key)
        {
            if (key == Key.Enter)
            {
                int idx = _list.SelectedItem;
                if (idx >= 0 && idx < _rows.Count)
                {
                    var row = _rows[idx];
                    // Defer so Application.Run(dialog) is not called from within ProcessKey.
                    Application.AddIdle(() =>
                    {
                        row.Activate();
                        RenderList();
                        return false;
                    });
                }
                return true;
            }
            if (key == Key.Esc)
            {
                ExitRequested?.Invoke();
                return true;
            }
            return base.OnKeyDown(key);
        }

        private void OnActivate(ListViewItemEventArgs e)
        {
            if (e.Item < 0 || e.Item >= _rows.Count) return;
            _rows[e.Item].Activate();
            RenderList();
        }

        private void RenderList()
        {
            var display = _rows
                .Select(r => $"{r.Name,-35}  {r.ValueGetter()}")
                .ToList();
            int sel = _list.SelectedItem;
            _list.SetSource<string>(new ObservableCollection<string>(display));
            _list.SelectedItem = Math.Clamp(sel, 0, Math.Max(0, display.Count - 1));
        }

        // ── Setting input helpers ────────────────────────────────────────────

        private static string? PromptText(string title, string current)
        {
            string? result = null;
            var dialog = new Dialog { Title = title, Width = 60, Height = 8 };

            var label = new Label { X = 1, Y = 0, Text = $"Current: {current}" };
            var field = new TextField { X = 1, Y = 2, Width = Dim.Fill(2), Text = current };
            var ok = new Button { Title = "OK", IsDefault = true };
            var cancel = new Button { Title = "Cancel" };

            ok.Accepting += (_, _) => { result = field.Text?.ToString(); Application.RequestStop(); };
            cancel.Accepting += (_, _) => { result = null; Application.RequestStop(); };

            dialog.AddButton(ok);
            dialog.AddButton(cancel);
            dialog.Add(label, field);
            Application.Run(dialog);
            dialog.Dispose();
            return result;
        }

        // ── Row definitions ──────────────────────────────────────────────────

        private void BuildRows()
        {
            _rows = new List<SettingRow>
            {
                new(Locale.Settings.Forwardseconds,
                    () => $"{Preferences.forwardSeconds} sec",
                    () =>
                    {
                        var v = PromptText(Locale.Settings.Forwardseconds, Preferences.forwardSeconds.ToString());
                        if (int.TryParse(v, out int n)) { Preferences.forwardSeconds = n; Preferences.SaveSettings(); }
                    }),

                new(Locale.Settings.Rewindseconds,
                    () => $"{Preferences.rewindSeconds} sec",
                    () =>
                    {
                        var v = PromptText(Locale.Settings.Rewindseconds, Preferences.rewindSeconds.ToString());
                        if (int.TryParse(v, out int n)) { Preferences.rewindSeconds = n; Preferences.SaveSettings(); }
                    }),

                new(Locale.Settings.ChangeVolumeBy,
                    () => $"{(int)(Preferences.changeVolumeBy * 100)} %",
                    () =>
                    {
                        var v = PromptText(Locale.Settings.ChangeVolumeBy, ((int)(Preferences.changeVolumeBy * 100)).ToString());
                        if (int.TryParse(v, out int n)) { Preferences.changeVolumeBy = n / 100f; Preferences.SaveSettings(); }
                    }),

                new(Locale.Settings.AutoSave,
                    () => Preferences.isAutoSave ? Locale.Miscellaneous.True : Locale.Miscellaneous.False,
                    () => { Preferences.isAutoSave = !Preferences.isAutoSave; Preferences.SaveSettings(); }),

                new("Toggle Media Buttons",
                    () => Preferences.isMediaButtons ? Locale.Miscellaneous.True : Locale.Miscellaneous.False,
                    () => { Preferences.isMediaButtons = !Preferences.isMediaButtons; Preferences.SaveSettings(); }),

                new("Toggle Visualizer",
                    () => Preferences.isVisualizer ? Locale.Miscellaneous.True : Locale.Miscellaneous.False,
                    () => { Preferences.isVisualizer = !Preferences.isVisualizer; Preferences.SaveSettings(); }),

                new("Toggle Key Modifier Helpers",
                    () => Preferences.isModifierKeyHelper ? Locale.Miscellaneous.True : Locale.Miscellaneous.False,
                    () => { Preferences.isModifierKeyHelper = !Preferences.isModifierKeyHelper; Preferences.SaveSettings(); }),

                new("Toggle Skip Errors",
                    () => Preferences.isSkipErrors ? Locale.Miscellaneous.True : Locale.Miscellaneous.False,
                    () => { Preferences.isSkipErrors = !Preferences.isSkipErrors; Preferences.SaveSettings(); }),

                new("Toggle Playlist Position",
                    () => Preferences.showPlaylistPosition ? Locale.Miscellaneous.True : Locale.Miscellaneous.False,
                    () => { Preferences.showPlaylistPosition = !Preferences.showPlaylistPosition; Preferences.SaveSettings(); }),

                new("Skip RSS after time",
                    () => Preferences.rssSkipAfterTime.ToString(),
                    () => { Preferences.rssSkipAfterTime = !Preferences.rssSkipAfterTime; Preferences.SaveSettings(); }),

                new("RSS skip time value (sec)",
                    () => Preferences.rssSkipAfterTimeValue.ToString(),
                    () =>
                    {
                        var v = PromptText("RSS skip time value", Preferences.rssSkipAfterTimeValue.ToString());
                        if (int.TryParse(v, out int n)) { Preferences.rssSkipAfterTimeValue = n; Preferences.SaveSettings(); }
                    }),

                new("Toggle Quick Search",
                    () => Preferences.isQuickSearch ? Locale.Miscellaneous.True : Locale.Miscellaneous.False,
                    () => { Preferences.isQuickSearch = !Preferences.isQuickSearch; Preferences.SaveSettings(); }),

                new("Toggle Quick Play From Search",
                    () => Preferences.isQuickPlayFromSearch ? Locale.Miscellaneous.True : Locale.Miscellaneous.False,
                    () => { Preferences.isQuickPlayFromSearch = !Preferences.isQuickPlayFromSearch; Preferences.SaveSettings(); }),

                new("Favorite Explainer",
                    () => Preferences.favoriteExplainer ? Locale.Miscellaneous.True : Locale.Miscellaneous.False,
                    () => { Preferences.favoriteExplainer = !Preferences.favoriteExplainer; Preferences.SaveSettings(); }),

                new("Load Effects",
                    () => "",
                    () => { Effects.ReadEffects(); if (Utils.Songs.Length > 0) Play.SetEffectsToChannel(); }),

                new("Load Visualizer Settings",
                    () => "",
                    () => Visual.Read()),

                new("Set Soundcloud Client ID",
                    () => string.IsNullOrEmpty(Preferences.clientID) ? "(not set)" : "***",
                    () =>
                    {
                        var v = PromptText("Soundcloud Client ID", Preferences.clientID ?? "");
                        if (v != null && v != "cancel")
                        {
                            Preferences.clientID = v == "reset" ? "" : v;
                            Preferences.SaveSettings();
                        }
                    }),
            };
        }
    }
}

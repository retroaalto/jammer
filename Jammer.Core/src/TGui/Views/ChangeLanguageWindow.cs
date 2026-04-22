using System.Collections.ObjectModel;
using Terminal.Gui;

namespace Jammer.TGui.Views
{
    /// <summary>
    /// Phase 2.1: Language selection window.
    /// Lists all .ini files in the locales directory.
    /// Arrow keys navigate, Enter selects and loads the language, Escape exits.
    /// </summary>
    public class ChangeLanguageWindow : FrameView
    {
        private readonly ListView _list;
        private readonly Label _hint;
        private string[] _filePaths = Array.Empty<string>();

        /// <summary>Raised when the user presses Escape or after a successful language change.</summary>
        public event Action? ExitRequested;

        public ChangeLanguageWindow()
        {
            Title = Locale.Help.ChangeLanguage;
            BorderStyle = LineStyle.Single;

            _list = new ListView
            {
                X = 0,
                Y = 0,
                Width = Dim.Fill(),
                Height = Dim.Fill(1),
            };
            _list.OpenSelectedItem += (_, e) => OnItemSelected(e);

            _hint = new Label
            {
                X = 1,
                Y = Pos.AnchorEnd(1),
                Width = Dim.Fill(1),
                Height = 1,
                Text = "Enter: select  Esc: back"
            };

            Add(_list, _hint);
            LoadLocales();
        }

        protected override bool OnKeyDown(Key key)
        {
            if (key == Key.Enter)
            {
                int idx = _list.SelectedItem;
                if (idx >= 0 && idx < _filePaths.Length)
                {
                    var path = _filePaths[idx];
                    // Defer so MessageBox.Query is not called from within ProcessKey.
                    Application.AddIdle(() =>
                    {
                        OnItemSelected(new ListViewItemEventArgs(idx, path));
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

        private void LoadLocales()
        {
            string path = Path.Combine(Utils.JammerPath, "locales");
            if (!Directory.Exists(path))
            {
                _list.SetSource<string>(new ObservableCollection<string>(new[] { $"(locales directory not found: {path})" }));
                return;
            }

            _filePaths = Directory.GetFiles(path, "*.ini");
            var displayNames = _filePaths
                .Select(f => Path.GetFileNameWithoutExtension(f))
                .ToList();

            _list.SetSource<string>(new ObservableCollection<string>(displayNames));

            // Pre-select the currently active locale
            int current = Array.FindIndex(_filePaths,
                f => string.Equals(
                    Path.GetFileNameWithoutExtension(f),
                    Preferences.localeLanguage,
                    StringComparison.OrdinalIgnoreCase));
            if (current >= 0)
                _list.SelectedItem = current;
        }

        private void OnItemSelected(ListViewItemEventArgs e)
        {
            if (e.Item < 0 || e.Item >= _filePaths.Length)
                return;

            string countryCode = Path.GetFileNameWithoutExtension(_filePaths[e.Item]);

            try
            {
                Preferences.localeLanguage = countryCode;
                IniFileHandling.SetLocaleData();
                Preferences.SaveSettings();

                MessageBox.Query(
                    50, 5,
                    "Language",
                    Locale.LocaleKeybind.Ini_LoadNewLocaleMessage1,
                    "OK");
            }
            catch (Exception ex)
            {
                Log.Error($"ChangeLanguageWindow: {ex.Message}");
                MessageBox.Query(
                    50, 5,
                    "Error",
                    Locale.LocaleKeybind.Ini_LoadNewLocaleError1,
                    "OK");
            }

            ExitRequested?.Invoke();
        }
    }
}

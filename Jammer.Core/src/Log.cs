namespace Jammer
{
    public static class Log
    {
        private const int MaxLogEntries = 1000;
        private static List<string> _log = new();
        public static string[] log => _log.ToArray();
        private static void New(string txt, bool isErr = false)
        {
            var time = DateTime.Now.ToString("HH:mm:ss"); // case sensitive

            var curPlaylist = Playlists.GetJammerPlaylistVisualPath(Utils.CurrentPlaylist);
            if (curPlaylist == "")
            {
                curPlaylist = "No playlist";
            }

            string entry;
            if (isErr)
            {
                entry = "[red]" + time + "[/]" + ";ERROR;[cyan]" + Start.Sanitize(curPlaylist) + "[/]: " + Start.Sanitize(txt);
            }
            else
            {
                entry = "[green3_1]" + time + "[/]" + ";INFO;[cyan]" + Start.Sanitize(curPlaylist) + "[/]: " + Start.Sanitize(txt);
            }
            _log.Add(entry);
            if (_log.Count > MaxLogEntries)
                _log.RemoveAt(0);
        }

        public static void Info(string txt)
        {
            New(txt);
        }

        public static void Error(string txt)
        {
            New(txt, true);
        }

        public static string GetLog()
        {
            return string.Join("\n", _log);
        }
    }
}
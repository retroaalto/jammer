namespace Jammer
{
    public static class Log
    {
        private const int MaxEntries = 500;
        private static readonly Queue<string> _log = new Queue<string>(MaxEntries + 1);

        // Read-only view for external consumers that previously accessed log[]
        public static string[] log => _log.ToArray();

        private static void New(string txt, bool isErr = false)
        {
            var time = DateTime.Now.ToString("HH:mm:ss"); // case sensitive

            var curPlaylist = Playlists.GetJammerPlaylistVisualPath(Utils.CurrentPlaylist);
            if (curPlaylist == "")
            {
                curPlaylist = "No playlist";
            }

            string entry = isErr
                ? "[red]" + time + "[/]" + ";ERROR;[cyan]" + Start.Sanitize(curPlaylist) + "[/]: " + Start.Sanitize(txt)
                : "[green3_1]" + time + "[/]" + ";INFO;[cyan]" + Start.Sanitize(curPlaylist) + "[/]: " + Start.Sanitize(txt);

            _log.Enqueue(entry);
            if (_log.Count > MaxEntries)
            {
                _log.Dequeue(); // drop the oldest entry to keep memory bounded
            }
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
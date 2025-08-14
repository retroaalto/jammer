namespace Jammer
{
    public static class Log
    {
        private static readonly List<string> log = new List<string>();
        private static readonly int maxLogEntries = 1000; // Prevent unbounded growth
        private static readonly object logLock = new object(); // Thread safety
        
        private static void New(string txt, bool isErr = false)
        {
            var time = DateTime.Now.ToString("HH:mm:ss"); // case sensitive

            var curPlaylist = Playlists.GetJammerPlaylistVisualPath(Utils.CurrentPlaylist);
            if (curPlaylist == "")
            {
                curPlaylist = "No playlist";
            }

            string logEntry;
            if (isErr)
            {
                logEntry = "[red]" + time + "[/]" + ";ERROR;[cyan]" + Start.Sanitize(curPlaylist) + "[/]: " + Start.Sanitize(txt);
            }
            else
            {
                logEntry = "[green3_1]" + time + "[/]" + ";INFO;[cyan]" + Start.Sanitize(curPlaylist) + "[/]: " + Start.Sanitize(txt);
            }
            
            lock (logLock)
            {
                log.Add(logEntry);
                
                // Implement log rotation: remove oldest entries when exceeding limit
                if (log.Count > maxLogEntries)
                {
                    log.RemoveAt(0); // Remove oldest entry
                }
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
            lock (logLock)
            {
                return string.Join("\n", log);
            }
        }
    }
}
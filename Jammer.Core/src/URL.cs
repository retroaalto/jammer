using System.Text.RegularExpressions;

namespace Jammer
{
    public class URL
    {
        // Pre-compiled static regexes — avoids recompiling on every call.
        private static readonly Regex _scSongRegex      = new Regex(Utils.SCSongPattern,      RegexOptions.IgnoreCase | RegexOptions.Compiled);
        private static readonly Regex _scPlaylistRegex  = new Regex(Utils.SCPlaylistPattern,  RegexOptions.IgnoreCase | RegexOptions.Compiled);
        private static readonly Regex _ytPlaylistRegex  = new Regex(Utils.YTPlaylistPattern,  RegexOptions.IgnoreCase | RegexOptions.Compiled);
        private static readonly Regex _ytSongRegex      = new Regex(Utils.YTSongPattern,      RegexOptions.IgnoreCase | RegexOptions.Compiled);
        private static readonly Regex _urlHttpsRegex    = new Regex(Utils.UrlPatternHTTPS,    RegexOptions.IgnoreCase | RegexOptions.Compiled);
        private static readonly Regex _urlHttpRegex     = new Regex(Utils.UrlPatternHTTP,     RegexOptions.IgnoreCase | RegexOptions.Compiled);

        public static bool IsValidSoundcloudSong(string uri)
        {
            return _scSongRegex.IsMatch(uri);
        }

        public static bool isValidSoundCloudPlaylist(string uri)
        {
            return _scPlaylistRegex.IsMatch(uri);
        }

        public static bool IsValidYoutubePlaylist(string uri)
        {
            return _ytPlaylistRegex.IsMatch(uri);
        }
        public static bool IsValidYoutubeSong(string uri)
        {
            return _ytSongRegex.IsMatch(uri);
        }

        public static bool IsUrl(string uri)
        {
            return IsUrlHTTPS(uri) || IsUrlHTTP(uri);
        }


        public static bool IsUrlHTTPS(string uri)
        {
            return _urlHttpsRegex.IsMatch(uri);
        }

        public static bool IsUrlHTTP(string uri)
        {
            return _urlHttpRegex.IsMatch(uri);
        }

        /// <summary>
        /// Checks if the given URI is a valid RSS feed URL.
        /// </summary>
        public static bool IsValidRssFeed(string uri)
        {
            return IsUrl(uri) && uri.EndsWith(".rss", StringComparison.OrdinalIgnoreCase) || IsUrl(uri) && uri.Contains("rss", StringComparison.OrdinalIgnoreCase);
        }
    }
}

using System.Text.RegularExpressions;

namespace Jammer
{
    public class URL
    {
        public static bool IsValidSoundcloudSong(string uri)
        {
            Regex regex = new Regex(Utils.SCSongPattern, RegexOptions.IgnoreCase);
            return regex.IsMatch(uri);
        }

        public static bool isValidSoundCloudPlaylist(string uri)
        {
            Regex regex = new Regex(Utils.SCPlaylistPattern, RegexOptions.IgnoreCase);
            return regex.IsMatch(uri);
        }

        public static bool IsValidYoutubePlaylist(string uri)
        {
            Regex regex = new Regex(Utils.YTPlaylistPattern, RegexOptions.IgnoreCase);
            return regex.IsMatch(uri);
        }
        public static bool IsValidYoutubeSong(string uri)
        {
            Regex regex = new Regex(Utils.YTSongPattern, RegexOptions.IgnoreCase);
            return regex.IsMatch(uri);
        }

        public static bool IsUrl(string uri)
        {
            return IsUrlHTTPS(uri) || IsUrlHTTP(uri);
        }


        public static bool IsUrlHTTPS(string uri)
        {
            Regex regex = new Regex(Utils.UrlPatternHTTPS, RegexOptions.IgnoreCase);
            return regex.IsMatch(uri);
        }

        public static bool IsUrlHTTP(string uri)
        {
            Regex regex = new Regex(Utils.UrlPatternHTTP, RegexOptions.IgnoreCase);
            return regex.IsMatch(uri);
        }
    }
}

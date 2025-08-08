namespace Jammer.Tests.TestUtilities.Fixtures;

public static class TestPlaylistData
{
    public static class ValidPlaylists
    {
        public const string PlaylistName = "Test Playlist";
        public const string PlaylistExtension = ".jammer";
        
        public static readonly Dictionary<string, object> SamplePlaylistData = new()
        {
            ["name"] = "My Test Playlist",
            ["songs"] = new List<string>
            {
                "/path/to/song1.mp3",
                "/path/to/song2.flac",
                "/path/to/song3.wav"
            },
            ["created"] = DateTime.Now.ToString("yyyy-MM-dd HH:mm:ss"),
            ["description"] = "A test playlist for unit testing"
        };
        
        public static string SamplePlaylistJson => 
            """
            {
              "name": "My Test Playlist",
              "songs": [
                "/path/to/song1.mp3",
                "/path/to/song2.flac", 
                "/path/to/song3.wav"
              ],
              "created": "2024-01-01 12:00:00",
              "description": "A test playlist for unit testing"
            }
            """;
    }
    
    public static class InvalidPlaylists
    {
        public static string InvalidJsonPlaylist => 
            """
            {
              "name": "Broken Playlist",
              "songs": [
                "/path/to/song1.mp3"
              "missing_comma": true
            }
            """;
            
        public static string EmptyPlaylist => "{}";
    }
}
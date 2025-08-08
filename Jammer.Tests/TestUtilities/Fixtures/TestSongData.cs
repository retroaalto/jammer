namespace Jammer.Tests.TestUtilities.Fixtures;

public static class TestSongData
{
    public static class ValidSongs
    {
        public const string Mp3File = "test_song.mp3";
        public const string FlacFile = "test_song.flac";
        public const string WavFile = "test_song.wav";
        
        public static readonly string[] SupportedExtensions = 
        [
            ".mp3", ".flac", ".wav", ".ogg", ".aac", ".m4a"
        ];
    }
    
    public static class InvalidSongs
    {
        public const string UnsupportedFile = "test.txt";
        public const string NonExistentFile = "nonexistent.mp3";
        
        public static readonly string[] UnsupportedExtensions = 
        [
            ".txt", ".doc", ".pdf", ".exe"
        ];
    }
    
    public static class TestMetadata
    {
        public const string Title = "Test Song";
        public const string Artist = "Test Artist";
        public const string Album = "Test Album";
        public const int Duration = 180; // 3 minutes
        public const string Genre = "Test Genre";
    }
}
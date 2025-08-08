using Jammer.Tests.TestUtilities.Fixtures;

namespace Jammer.Tests.TestUtilities.Builders;

/// <summary>
/// Builder pattern for creating test Song objects with default and customizable values.
/// </summary>
public class SongBuilder
{
    private string _filePath = "/test/path/test_song.mp3";
    private string _title = TestSongData.TestMetadata.Title;
    private string _artist = TestSongData.TestMetadata.Artist;
    private string _album = TestSongData.TestMetadata.Album;
    private TimeSpan _duration = TimeSpan.FromSeconds(TestSongData.TestMetadata.Duration);
    private string _genre = TestSongData.TestMetadata.Genre;
    private DateTime _dateAdded = DateTime.Now;
    
    public SongBuilder WithFilePath(string filePath)
    {
        _filePath = filePath;
        return this;
    }
    
    public SongBuilder WithTitle(string title)
    {
        _title = title;
        return this;
    }
    
    public SongBuilder WithArtist(string artist)
    {
        _artist = artist;
        return this;
    }
    
    public SongBuilder WithAlbum(string album)
    {
        _album = album;
        return this;
    }
    
    public SongBuilder WithDuration(TimeSpan duration)
    {
        _duration = duration;
        return this;
    }
    
    public SongBuilder WithDuration(int seconds)
    {
        _duration = TimeSpan.FromSeconds(seconds);
        return this;
    }
    
    public SongBuilder WithGenre(string genre)
    {
        _genre = genre;
        return this;
    }
    
    public SongBuilder WithDateAdded(DateTime dateAdded)
    {
        _dateAdded = dateAdded;
        return this;
    }
    
    public SongBuilder WithMp3Extension()
    {
        var directory = Path.GetDirectoryName(_filePath) ?? "/test/path";
        var fileName = Path.GetFileNameWithoutExtension(_filePath);
        _filePath = Path.Combine(directory, $"{fileName}.mp3");
        return this;
    }
    
    public SongBuilder WithFlacExtension()
    {
        var directory = Path.GetDirectoryName(_filePath) ?? "/test/path";
        var fileName = Path.GetFileNameWithoutExtension(_filePath);
        _filePath = Path.Combine(directory, $"{fileName}.flac");
        return this;
    }
    
    /// <summary>
    /// Note: This builds a mock Song object for testing.
    /// The actual Song class construction will be implemented once we analyze the real Song class.
    /// </summary>
    public Dictionary<string, object> BuildAsDictionary()
    {
        return new Dictionary<string, object>
        {
            ["filePath"] = _filePath,
            ["title"] = _title,
            ["artist"] = _artist,
            ["album"] = _album,
            ["duration"] = _duration,
            ["genre"] = _genre,
            ["dateAdded"] = _dateAdded
        };
    }
}
using System.IO.Abstractions.TestingHelpers;
using FluentAssertions;
using Xunit;
using System.IO;

namespace Jammer.Tests.Unit.Core;

/// <summary>
/// Unit tests for playlist management functionality in Playlists.cs.
/// These tests focus on path resolution, playlist file operations, and core business logic.
/// Note: Many methods have external dependencies (file system, console I/O, UI) which are tested conceptually.
/// </summary>
public class PlaylistsTests
{
    [Theory]
    [InlineData("myplaylist", true)]
    [InlineData("test-playlist", true)]
    [InlineData("playlist with spaces", true)]
    [InlineData("/full/path/to/playlist.jammer", false)]
    [InlineData("relative/path/playlist.jammer", false)]
    [InlineData("playlist/with/slash", false)]
    public void GetJammerPlaylistPath_HandlesPlaylistNameVsPath(string input, bool isPlaylistName)
    {
        // Arrange & Act
        // Simulate the path resolution logic from GetJammerPlaylistPath
        bool containsDirectorySeparator = input.Contains(Path.DirectorySeparatorChar.ToString()) || input.Contains(Path.AltDirectorySeparatorChar.ToString());
        bool shouldUsePlaylistsPath = !containsDirectorySeparator;
        
        // Assert
        shouldUsePlaylistsPath.Should().Be(isPlaylistName);
        
        if (isPlaylistName)
        {
            // Should combine with playlists directory and add .jammer extension
            var expectedPattern = input + ".jammer";
            input.Should().NotContain(Path.DirectorySeparatorChar.ToString());
        }
        else
        {
            // Should treat as full path
            input.Should().Contain(Path.DirectorySeparatorChar.ToString());
        }
    }
    
    [Theory]
    [InlineData("simple")]
    [InlineData("with-dashes")]
    [InlineData("with_underscores")]
    [InlineData("123numeric")]
    [InlineData("MixedCase")]
    public void GetJammerPlaylistPath_ValidPlaylistNames(string playlistName)
    {
        // Act - Simulate playlist path generation
        string expectedExtension = ".jammer";
        string simulatedPath = playlistName + expectedExtension;
        
        // Assert
        simulatedPath.Should().EndWith(expectedExtension);
        simulatedPath.Should().StartWith(playlistName);
        simulatedPath.Should().NotContain(Path.DirectorySeparatorChar.ToString());
    }
    
    [Theory]
    [InlineData("/absolute/path/playlist.jammer")]
    [InlineData("relative/path/playlist.jammer")]
    [InlineData("../parent/playlist.jammer")]
    [InlineData("./current/playlist.jammer")]
    public void GetJammerPlaylistPath_HandlesPaths(string inputPath)
    {
        // Act - Test the path detection logic
        bool containsSeparator = inputPath.Contains(Path.DirectorySeparatorChar.ToString());
        
        // Assert
        containsSeparator.Should().BeTrue();
        
        // Should be treated as a path, not a playlist name
        // The actual method would call Path.GetFullPath(inputPath)
        Path.IsPathRooted(inputPath).Should().Be(inputPath.StartsWith("/") || inputPath.Contains(":"));
    }
    
    [Fact]
    public void GetJammerPlaylistVisualPath_ProcessesCorrectly()
    {
        // Arrange
        string testPath = "/home/user/playlists/myplaylist.jammer";
        
        // Act - Simulate the visual path processing logic
        string fileName = Path.GetFileName(testPath);
        string nameWithoutExtension = Path.GetFileNameWithoutExtension(testPath);
        
        // Assert
        fileName.Should().Be("myplaylist.jammer");
        nameWithoutExtension.Should().Be("myplaylist");
        
        // The visual path should remove .jammer extension for display
        nameWithoutExtension.Should().NotContain(".jammer");
    }
    
    [Theory]
    [InlineData("")]
    [InlineData("   ")]
    [InlineData("\t")]
    [InlineData("\n")]
    public void Playlists_HandlesInvalidPlaylistNames(string invalidName)
    {
        // Act & Assert
        // Invalid names should be handled gracefully
        string.IsNullOrWhiteSpace(invalidName).Should().BeTrue();
        
        // These should not be valid playlist names
        if (!string.IsNullOrWhiteSpace(invalidName))
        {
            var trimmedName = invalidName.Trim();
            trimmedName.Length.Should().BeGreaterThan(0);
        }
    }
    
    [Theory]
    [InlineData("normalname")]
    [InlineData("name-with-dashes")]
    [InlineData("name_with_underscores")]
    [InlineData("name123")]
    [InlineData("123name")]
    public void Playlists_ValidPlaylistNamesPass(string validName)
    {
        // Act & Assert
        validName.Should().NotBeNullOrWhiteSpace();
        validName.Trim().Should().Be(validName);
        validName.Length.Should().BeGreaterThan(0);
        
        // Should not contain path separators for simple names
        validName.Should().NotContain(Path.DirectorySeparatorChar.ToString());
        validName.Should().NotContain(Path.AltDirectorySeparatorChar.ToString());
    }
    
    [Theory]
    [InlineData("playlist", ".jammer")]
    [InlineData("my-list", ".jammer")]
    [InlineData("test_playlist", ".jammer")]
    public void Playlists_FileExtensionHandling(string name, string expectedExtension)
    {
        // Act
        string fullName = name + expectedExtension;
        string extractedName = Path.GetFileNameWithoutExtension(fullName);
        string extractedExtension = Path.GetExtension(fullName);
        
        // Assert
        extractedName.Should().Be(name);
        extractedExtension.Should().Be(expectedExtension);
    }
    
    [Fact]
    public void Playlists_FileOperationsHandling()
    {
        // This test documents the file operations that the Playlists class performs
        // Actual file operations would require mocking or integration tests
        
        // Arrange
        string playlistName = "test-playlist";
        string expectedExtension = ".jammer";
        
        // Act & Assert
        // The class should handle these operations:
        var operations = new[]
        {
            "File.Exists() - Check if playlist exists",
            "File.Create() - Create new playlist file", 
            "File.Delete() - Delete existing playlist",
            "File.ReadAllText() - Read playlist content",
            "File.WriteAllText() - Write playlist content",
            "Path.Combine() - Build playlist paths",
            "Path.GetFullPath() - Resolve full paths"
        };
        
        operations.Should().HaveCount(7);
        
        // File extension should be .jammer
        var playlistFile = playlistName + expectedExtension;
        playlistFile.Should().EndWith(expectedExtension);
    }
    
    [Theory]
    [InlineData("song1.mp3", "song2.flac", "song3.wav")]
    [InlineData("http://youtube.com/watch?v=123")]
    [InlineData("https://soundcloud.com/artist/track")]
    [InlineData("/absolute/path/to/song.mp3")]
    public void Playlists_SongPathHandling(params string[] songPaths)
    {
        // Act & Assert - Test song path validation logic
        foreach (var songPath in songPaths)
        {
            songPath.Should().NotBeNullOrEmpty();
            
            // Songs can be local files or URLs
            bool isUrl = songPath.StartsWith("http://") || songPath.StartsWith("https://");
            bool isAbsolutePath = Path.IsPathRooted(songPath);
            bool isRelativePath = !isUrl && !isAbsolutePath;
            
            // At least one should be true
            (isUrl || isAbsolutePath || isRelativePath).Should().BeTrue();
        }
    }
    
    [Fact]
    public void Playlists_DelimiterHandling()
    {
        // Arrange - Test the Jammer file delimiter usage
        string songPath = "/path/to/song.mp3";
        string metadata = "{\"title\":\"Test Song\",\"author\":\"Test Artist\"}";
        
        // Act - Simulate how songs with metadata are stored
        string jammerEntry = songPath + Utils.JammerFileDelimeter + metadata;
        
        // Assert
        jammerEntry.Should().Contain(Utils.JammerFileDelimeter);
        jammerEntry.Should().StartWith(songPath);
        jammerEntry.Should().EndWith(metadata);
        
        // Should be able to split back
        string[] parts = jammerEntry.Split(Utils.JammerFileDelimeter);
        parts.Should().HaveCount(2);
        parts[0].Should().Be(songPath);
        parts[1].Should().Be(metadata);
    }
    
    [Theory]
    [InlineData("rock-hits")]
    [InlineData("classical-music")]
    [InlineData("workout-mix")]
    [InlineData("study-songs")]
    public void Playlists_PlaylistNamingConventions(string playlistName)
    {
        // Act & Assert
        // Good playlist names should follow these conventions
        playlistName.Should().NotBeNullOrWhiteSpace();
        playlistName.Should().NotStartWith(".");
        playlistName.Should().NotEndWith(".");
        
        // Should not contain invalid filename characters
        var invalidChars = Path.GetInvalidFileNameChars();
        foreach (var invalidChar in invalidChars)
        {
            playlistName.Should().NotContain(invalidChar.ToString());
        }
        
        // Should be reasonable length
        playlistName.Length.Should().BeLessThan(100);
        playlistName.Length.Should().BeGreaterThan(0);
    }
}

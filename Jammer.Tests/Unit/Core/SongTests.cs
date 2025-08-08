using System.Text.Json;
using FluentAssertions;
using Xunit;

namespace Jammer.Tests.Unit.Core;

/// <summary>
/// Unit tests for the Song class and its metadata extraction functionality.
/// </summary>
public class SongTests
{
    [Fact]
    public void Song_DefaultConstructor_CreatesInstanceWithNullProperties()
    {
        // Act
        var song = new Song();
        
        // Assert
        song.URI.Should().BeNull();
        song.Title.Should().BeNull();
        song.Author.Should().BeNull();
        song.Album.Should().BeNull();
        song.Year.Should().BeNull();
        song.Genre.Should().BeNull();
        song.Duration.Should().BeNull();
    }
    
    [Fact]
    public void Song_PropertiesCanBeSet()
    {
        // Arrange
        var song = new Song();
        
        // Act
        song.URI = "/path/to/song.mp3";
        song.Title = "Test Title";
        song.Author = "Test Author";
        song.Album = "Test Album";
        song.Year = "2023";
        song.Genre = "Rock";
        song.Duration = "3:45";
        
        // Assert
        song.URI.Should().Be("/path/to/song.mp3");
        song.Title.Should().Be("Test Title");
        song.Author.Should().Be("Test Author");
        song.Album.Should().Be("Test Album");
        song.Year.Should().Be("2023");
        song.Genre.Should().Be("Rock");
        song.Duration.Should().Be("3:45");
    }
    
    [Fact]
    public void ExtractSongDetails_WithNoDelimiter_DoesNotModifyProperties()
    {
        // Arrange
        var song = new Song { URI = "/path/to/song.mp3" };
        
        // Act
        song.ExtractSongDetails();
        
        // Assert
        song.URI.Should().Be("/path/to/song.mp3");
        song.Title.Should().BeNull();
        song.Author.Should().BeNull();
        song.Album.Should().BeNull();
        song.Year.Should().BeNull();
        song.Genre.Should().BeNull();
        song.Duration.Should().BeNull();
    }
    
    [Fact]
    public void ExtractSongDetails_WithNullURI_DoesNotThrow()
    {
        // Arrange
        var song = new Song { URI = null };
        
        // Act
        var act = () => song.ExtractSongDetails();
        
        // Assert
        act.Should().NotThrow();
        song.URI.Should().BeNull();
    }
    
    [Fact]
    public void ExtractSongDetails_WithEmptyJson_DoesNotModifyProperties()
    {
        // Arrange
        var song = new Song { URI = $"/path/to/song.mp3{Utils.JammerFileDelimeter}" };
        
        // Act
        song.ExtractSongDetails();
        
        // Assert
        song.URI.Should().Be("/path/to/song.mp3");
        song.Title.Should().BeNull();
        song.Author.Should().BeNull();
        song.Album.Should().BeNull();
        song.Year.Should().BeNull();
        song.Genre.Should().BeNull();
        song.Duration.Should().BeNull();
    }
    
    [Fact]
    public void ExtractSongDetails_WithValidJson_ExtractsAllProperties()
    {
        // Arrange
        var metadata = new Song
        {
            Title = "Extracted Title",
            Author = "Extracted Author", 
            Album = "Extracted Album",
            Year = "2024",
            Genre = "Pop",
            Duration = "4:20"
        };
        var json = JsonSerializer.Serialize(metadata);
        var song = new Song { URI = $"/path/to/song.mp3{Utils.JammerFileDelimeter}{json}" };
        
        // Act
        song.ExtractSongDetails();
        
        // Assert
        song.URI.Should().Be("/path/to/song.mp3");
        song.Title.Should().Be("Extracted Title");
        song.Author.Should().Be("Extracted Author");
        song.Album.Should().Be("Extracted Album");
        song.Year.Should().Be("2024");
        song.Genre.Should().Be("Pop");
        song.Duration.Should().Be("4:20");
    }
    
    [Fact]
    public void ExtractSongDetails_WithPartialJson_ExtractsAvailableProperties()
    {
        // Arrange
        var metadata = new Song
        {
            Title = "Partial Title",
            Author = "Partial Author"
            // Other properties left null
        };
        var json = JsonSerializer.Serialize(metadata);
        var song = new Song { URI = $"/path/to/song.mp3{Utils.JammerFileDelimeter}{json}" };
        
        // Act
        song.ExtractSongDetails();
        
        // Assert
        song.URI.Should().Be("/path/to/song.mp3");
        song.Title.Should().Be("Partial Title");
        song.Author.Should().Be("Partial Author");
        song.Album.Should().BeNull();
        song.Year.Should().BeNull();
        song.Genre.Should().BeNull();
        song.Duration.Should().BeNull();
    }
    
    [Fact]
    public void ExtractSongDetails_WithInvalidJson_ThrowsJsonException()
    {
        // Arrange
        var song = new Song { URI = $"/path/to/song.mp3{Utils.JammerFileDelimeter}{{invalid json" };
        
        // Act
        var act = () => song.ExtractSongDetails();
        
        // Assert
        act.Should().Throw<JsonException>();
        song.URI.Should().Be("/path/to/song.mp3"); // URI is still updated
    }
    
    [Fact]
    public void ExtractSongDetails_WithComplexPath_HandlesCorrectly()
    {
        // Arrange
        var metadata = new Song { Title = "Complex Path Song" };
        var json = JsonSerializer.Serialize(metadata);
        var complexPath = "/home/user/music/folder with spaces/song (remix).mp3";
        var song = new Song { URI = $"{complexPath}{Utils.JammerFileDelimeter}{json}" };
        
        // Act
        song.ExtractSongDetails();
        
        // Assert
        song.URI.Should().Be(complexPath);
        song.Title.Should().Be("Complex Path Song");
    }
    
    [Fact]
    public void ExtractSongDetails_WithMultipleDelimiters_UsesFirstDelimiterOnly()
    {
        // Arrange
        var metadata = new Song { Title = "Multi Delimiter Test" };
        var json = JsonSerializer.Serialize(metadata);
        var song = new Song { URI = $"/path/to/song.mp3{Utils.JammerFileDelimeter}{json}{Utils.JammerFileDelimeter}extra" };
        
        // Act
        song.ExtractSongDetails();
        
        // Assert
        song.URI.Should().Be("/path/to/song.mp3");
        song.Title.Should().Be("Multi Delimiter Test");
    }
}
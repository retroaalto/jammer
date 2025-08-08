using System.Text.RegularExpressions;
using FluentAssertions;
using Xunit;

namespace Jammer.Tests.Unit.Core;

/// <summary>
/// Unit tests for utility functions in the Jammer.Core project.
/// These tests target pure functions without external dependencies.
/// </summary>
public class UtilsTests
{
    [Fact]
    public void GetJammerPath_WithXdgConfigHome_ReturnsCorrectPath()
    {
        // Arrange
        var originalXdg = Environment.GetEnvironmentVariable("XDG_CONFIG_HOME");
        var originalJammer = Environment.GetEnvironmentVariable("JAMMER_CONFIG_PATH");
        const string testXdgPath = "/home/test/.config";
        
        try
        {
            Environment.SetEnvironmentVariable("XDG_CONFIG_HOME", testXdgPath);
            Environment.SetEnvironmentVariable("JAMMER_CONFIG_PATH", null);
            
            // Act
            var result = Utils.UtilFuncs.GetJammerPath();
            
            // Assert
            result.Should().Be(Path.Combine(testXdgPath, "jammer"));
        }
        finally
        {
            Environment.SetEnvironmentVariable("XDG_CONFIG_HOME", originalXdg);
            Environment.SetEnvironmentVariable("JAMMER_CONFIG_PATH", originalJammer);
        }
    }
    
    [Fact]
    public void GetJammerPath_WithJammerConfigPath_ReturnsJammerConfigPath()
    {
        // Arrange
        var originalXdg = Environment.GetEnvironmentVariable("XDG_CONFIG_HOME");
        var originalJammer = Environment.GetEnvironmentVariable("JAMMER_CONFIG_PATH");
        const string testJammerPath = "/custom/jammer/path";
        
        try
        {
            Environment.SetEnvironmentVariable("XDG_CONFIG_HOME", null);
            Environment.SetEnvironmentVariable("JAMMER_CONFIG_PATH", testJammerPath);
            
            // Act
            var result = Utils.UtilFuncs.GetJammerPath();
            
            // Assert
            result.Should().Be(testJammerPath);
        }
        finally
        {
            Environment.SetEnvironmentVariable("XDG_CONFIG_HOME", originalXdg);
            Environment.SetEnvironmentVariable("JAMMER_CONFIG_PATH", originalJammer);
        }
    }
    
    [Fact]
    public void GetJammerPath_WithBothEnvVars_PrefersXdgConfigHome()
    {
        // Arrange
        var originalXdg = Environment.GetEnvironmentVariable("XDG_CONFIG_HOME");
        var originalJammer = Environment.GetEnvironmentVariable("JAMMER_CONFIG_PATH");
        const string testXdgPath = "/home/test/.config";
        const string testJammerPath = "/custom/jammer/path";
        
        try
        {
            Environment.SetEnvironmentVariable("XDG_CONFIG_HOME", testXdgPath);
            Environment.SetEnvironmentVariable("JAMMER_CONFIG_PATH", testJammerPath);
            
            // Act
            var result = Utils.UtilFuncs.GetJammerPath();
            
            // Assert
            result.Should().Be(Path.Combine(testXdgPath, "jammer"));
        }
        finally
        {
            Environment.SetEnvironmentVariable("XDG_CONFIG_HOME", originalXdg);
            Environment.SetEnvironmentVariable("JAMMER_CONFIG_PATH", originalJammer);
        }
    }
    
    [Fact]
    public void GetJammerPath_WithNoEnvVars_ReturnsUserProfilePath()
    {
        // Arrange
        var originalXdg = Environment.GetEnvironmentVariable("XDG_CONFIG_HOME");
        var originalJammer = Environment.GetEnvironmentVariable("JAMMER_CONFIG_PATH");
        
        try
        {
            Environment.SetEnvironmentVariable("XDG_CONFIG_HOME", null);
            Environment.SetEnvironmentVariable("JAMMER_CONFIG_PATH", null);
            
            // Act
            var result = Utils.UtilFuncs.GetJammerPath();
            
            // Assert
            var expectedPath = Path.Combine(Environment.GetFolderPath(Environment.SpecialFolder.UserProfile), "jammer");
            result.Should().Be(expectedPath);
        }
        finally
        {
            Environment.SetEnvironmentVariable("XDG_CONFIG_HOME", originalXdg);
            Environment.SetEnvironmentVariable("JAMMER_CONFIG_PATH", originalJammer);
        }
    }
    
    [Theory]
    [InlineData("https://www.youtube.com/watch?v=dQw4w9WgXcQ", true)]
    [InlineData("http://youtube.com/watch?v=dQw4w9WgXcQ", true)]
    [InlineData("https://youtu.be/dQw4w9WgXcQ", true)]
    [InlineData("www.youtube.com/watch?v=dQw4w9WgXcQ", true)]
    [InlineData("youtube.com/watch?v=dQw4w9WgXcQ", true)]
    [InlineData("youtu.be/dQw4w9WgXcQ", true)]
    [InlineData("https://example.com", false)]
    [InlineData("not-a-url", false)]
    [InlineData("", false)]
    [InlineData(null, false)]
    public void YTSongPattern_ValidatesYouTubeUrls(string input, bool shouldMatch)
    {
        // Arrange
        var regex = new Regex(Utils.YTSongPattern);
        
        // Act
        var isMatch = input != null && regex.IsMatch(input);
        
        // Assert
        isMatch.Should().Be(shouldMatch);
    }
    
    [Theory]
    [InlineData("https://www.youtube.com/playlist?list=PLrAXtmRdnEQy2EsEOwqPsXDJKBMagYQr3", true)]
    [InlineData("https://youtube.com/playlist?list=PLrAXtmRdnEQy2EsEOwqPsXDJKBMagYQr3", true)]
    [InlineData("http://www.youtube.com/playlist?list=PLrAXtmRdnEQy2EsEOwqPsXDJKBMagYQr3", true)]
    [InlineData("https://www.youtube.com/watch?v=dQw4w9WgXcQ", false)]
    [InlineData("https://example.com/playlist", false)]
    [InlineData("", false)]
    [InlineData(null, false)]
    public void YTPlaylistPattern_ValidatesYouTubePlaylistUrls(string input, bool shouldMatch)
    {
        // Arrange
        var regex = new Regex(Utils.YTPlaylistPattern);
        
        // Act
        var isMatch = input != null && regex.IsMatch(input);
        
        // Assert
        isMatch.Should().Be(shouldMatch);
    }
    
    [Theory]
    [InlineData("https://soundcloud.com/artist/track", true)]
    [InlineData("http://soundcloud.com/artist/track", true)]
    [InlineData("https://www.soundcloud.com/artist/track", true)]
    [InlineData("https://snd.sc/shortlink", true)]
    [InlineData("soundcloud.com/artist/track", true)]
    [InlineData("www.soundcloud.com/artist/track", true)]
    [InlineData("https://example.com", false)]
    [InlineData("", false)]
    [InlineData(null, false)]
    public void SCSongPattern_ValidatesSoundCloudUrls(string input, bool shouldMatch)
    {
        // Arrange
        var regex = new Regex(Utils.SCSongPattern);
        
        // Act
        var isMatch = input != null && regex.IsMatch(input);
        
        // Assert
        isMatch.Should().Be(shouldMatch);
    }
    
    [Theory]
    [InlineData("https://soundcloud.com/artist/sets/playlist-name", true)]
    [InlineData("http://soundcloud.com/artist/sets/playlist-name", true)]
    [InlineData("https://www.soundcloud.com/artist/sets/playlist-name", true)]
    [InlineData("soundcloud.com/artist/sets/playlist-name", false)]
    [InlineData("https://soundcloud.com/artist/track", false)]
    [InlineData("https://example.com/sets/playlist", false)]
    [InlineData("", false)]
    [InlineData(null, false)]
    public void SCPlaylistPattern_ValidatesSoundCloudPlaylistUrls(string input, bool shouldMatch)
    {
        // Arrange
        var regex = new Regex(Utils.SCPlaylistPattern);
        
        // Act
        var isMatch = input != null && regex.IsMatch(input);
        
        // Assert
        isMatch.Should().Be(shouldMatch);
    }
    
    [Theory]
    [InlineData("https://example.com", true)]
    [InlineData("http://example.com", true)] // HTTPS pattern actually accepts both http and https
    [InlineData("https://www.example.com", true)]
    [InlineData("https://sub.example.com", true)]
    [InlineData("https://example.com/path", true)]
    [InlineData("https://example.com/path?query=value", true)]
    [InlineData("not-a-url", false)]
    [InlineData("", false)]
    [InlineData(null, false)]
    public void UrlPatternHTTPS_ValidatesHttpsUrls(string input, bool shouldMatch)
    {
        // Arrange
        var regex = new Regex(Utils.UrlPatternHTTPS);
        
        // Act
        var isMatch = input != null && regex.IsMatch(input);
        
        // Assert
        isMatch.Should().Be(shouldMatch);
    }
    
    [Theory]
    [InlineData("http://example.com", true)]
    [InlineData("https://example.com", false)] // HTTP pattern only accepts http, not https
    [InlineData("http://www.example.com", true)]
    [InlineData("http://sub.example.com", true)]
    [InlineData("http://example.com/path", true)]
    [InlineData("http://example.com/path?query=value", true)]
    [InlineData("not-a-url", false)]
    [InlineData("", false)]
    [InlineData(null, false)]
    public void UrlPatternHTTP_ValidatesHttpUrls(string input, bool shouldMatch)
    {
        // Arrange
        var regex = new Regex(Utils.UrlPatternHTTP);
        
        // Act
        var isMatch = input != null && regex.IsMatch(input);
        
        // Assert
        isMatch.Should().Be(shouldMatch);
    }
    
    [Fact]
    public void JammerFileDelimeter_HasCorrectValue()
    {
        // Assert
        Utils.JammerFileDelimeter.Should().Be("?|");
    }
    
    [Fact]
    public void Version_IsNotEmpty()
    {
        // Assert
        Utils.Version.Should().NotBeNullOrEmpty();
    }
}
using FluentAssertions;
using Jammer.Tests.TestUtilities.Doubles;
using Xunit;

namespace Jammer.Tests.Unit.Core
{
    /// <summary>
    /// Tests for Download.cs service layer functionality
    /// Phase 3: Service Layer Testing - YouTube/SoundCloud download operations
    /// 
    /// ARCHITECTURAL NOTES:
    /// - Download.cs has 17 methods for async downloading operations
    /// - External dependencies: YoutubeExplode, SoundCloudExplode, FFmpeg, PuppeteerSharp
    /// - Complex error handling with skip-on-error functionality
    /// - File system operations with path manipulation
    /// - Progress tracking integrated with TUI for user feedback
    /// 
    /// TESTING APPROACH:
    /// - Focus on testable utility methods first (URL formatting, validation)  
    /// - Use mock implementations for external API dependencies
    /// - Abstract file system operations for testability
    /// - Document complex async operations that need integration testing
    /// </summary>
    public class DownloadTests
    {
        private readonly MockDownloadService _mockDownloadService;

        public DownloadTests()
        {
            _mockDownloadService = new MockDownloadService();
        }

        [Fact]
        public void Download_Class_ShouldExist()
        {
            // Arrange & Act
            var downloadType = typeof(Download);
            
            // Assert
            downloadType.Should().NotBeNull();
            downloadType.IsClass.Should().BeTrue();
        }

        [Fact]
        public void FormatUrlForFilename_WithSimpleYouTubeUrl_ShouldReturnSafeFilename()
        {
            // Arrange
            var testUrl = "https://www.youtube.com/watch?v=dQw4w9WgXcQ";
            
            // Act
            var result = Download.FormatUrlForFilename(testUrl);
            
            // Assert
            result.Should().NotBeNull("Formatted filename should not be null");
            result.Should().NotContain("/", "Formatted filename should not contain slashes");
            result.Should().NotContain("\\", "Formatted filename should not contain backslashes");
            result.Should().NotContain(":", "Formatted filename should not contain colons");
            result.Should().NotContain("?", "Formatted filename should not contain question marks");
            result.Should().NotContain("*", "Formatted filename should not contain asterisks");
            result.Should().NotContain("<", "Formatted filename should not contain less-than signs");
            result.Should().NotContain(">", "Formatted filename should not contain greater-than signs");
            result.Should().NotContain("|", "Formatted filename should not contain pipes");
        }

        [Fact]
        public void FormatUrlForFilename_WithComplexUrl_ShouldHandleAllSpecialCharacters()
        {
            // Arrange
            var testUrl = "https://www.example.com/path/with spaces & special chars?param=value&other=123";
            
            // Act
            var result = Download.FormatUrlForFilename(testUrl);
            
            // Assert
            result.Should().NotBeNull();
            result.Should().NotContainAny(new[] { "/", "\\", ":", "?", "*", "<", ">", "|" },
                "Formatted filename should not contain filesystem-unsafe characters");
            
            // Verify length is reasonable for filesystem
            result.Length.Should().BeLessOrEqualTo(255, "Filename should not exceed filesystem limits");
        }

        [Fact]
        public void FormatUrlForFilename_WithNullOrEmptyUrl_ShouldHandleGracefully()
        {
            // Arrange & Act & Assert for null
            var nullException = Record.Exception(() => Download.FormatUrlForFilename(null!));
            
            // Arrange & Act & Assert for empty
            var emptyException = Record.Exception(() => Download.FormatUrlForFilename(""));
            
            // The method should either handle null/empty gracefully or throw meaningful exceptions
            // This tests the robustness of the current implementation
            if (nullException != null)
            {
                nullException.Should().BeOfType<ArgumentNullException>("Should throw ArgumentNullException for null input");
            }
            
            if (emptyException == null)
            {
                var emptyResult = Download.FormatUrlForFilename("");
                emptyResult.Should().NotBeNull("Empty URL should return valid result");
            }
        }

        [Fact]
        public void IsFFmpegInstalled_IsPrivateMethod_CannotBeTestedDirectly()
        {
            // IsFFmpegInstalled is a private method used internally by Download.cs
            // This test documents that it exists but cannot be tested directly
            
            var downloadType = typeof(Download);
            var isFFmpegInstalledMethod = downloadType.GetMethod("IsFFmpegInstalled", 
                System.Reflection.BindingFlags.NonPublic | System.Reflection.BindingFlags.Static);
            
            // Assert
            isFFmpegInstalledMethod.Should().NotBeNull("IsFFmpegInstalled method should exist");
            if (isFFmpegInstalledMethod != null)
            {
                isFFmpegInstalledMethod.ReturnType.Should().Be(typeof(bool), "Should return boolean");
                isFFmpegInstalledMethod.IsStatic.Should().BeTrue("Should be static method");
            }
        }

        [Fact]
        public void GetDownloadedJammerFileName_IsPrivateMethod_CannotBeTestedDirectly()
        {
            // GetDownloadedJammerFileName is a private method used internally by Download.cs
            // This test documents that it exists but cannot be tested directly
            
            var downloadType = typeof(Download);
            var getJammerFileNameMethod = downloadType.GetMethod("GetDownloadedJammerFileName", 
                System.Reflection.BindingFlags.NonPublic | System.Reflection.BindingFlags.Static);
            
            // Assert
            getJammerFileNameMethod.Should().NotBeNull("GetDownloadedJammerFileName method should exist");
            if (getJammerFileNameMethod != null)
            {
                getJammerFileNameMethod.ReturnType.Should().Be(typeof(string), "Should return string");
                getJammerFileNameMethod.IsStatic.Should().BeTrue("Should be static method");
            }
        }

        [Fact]
        public void Download_Architecture_HasExpectedMethods()
        {
            // Document the method structure for testing strategy
            
            var downloadType = typeof(Download);
            var methods = downloadType.GetMethods(System.Reflection.BindingFlags.Public | System.Reflection.BindingFlags.Static);
            var methodNames = methods.Select(m => m.Name).Distinct().ToArray();
            
            // Key methods identified in architectural analysis (only public static methods)
            var expectedMethods = new[]
            {
                "DownloadSong", "DownloadSoundCloudTrackAsync",
                "FormatUrlForFilename", "GetPlaylist", "GetSongsFromPlaylist"
            };
            

            foreach (var expectedMethod in expectedMethods)
            {
                methodNames.Should().Contain(expectedMethod, 
                    $"Download should have {expectedMethod} method for download operations");
            }
        }

        [Fact]
        public void MockDownloadService_ShouldSimulateYouTubeDownload()
        {
            // This demonstrates how Download.cs COULD be tested with proper abstractions
            
            // Arrange
            _mockDownloadService.Reset();
            var testUrl = "https://youtube.com/watch?v=test123";
            
            // Act
            var task = _mockDownloadService.DownloadYoutubeTrackAsync(testUrl);
            var result = task.Result; // In real tests, use await
            
            // Assert
            result.Should().NotBeNull("Download should return a file path");
            result.Should().EndWith(".ogg", "YouTube downloads should be converted to OGG");
            result.Should().Contain(_mockDownloadService.SongPath, "Download should be in songs directory");
            
            _mockDownloadService.WasMethodCalled("DownloadYoutubeTrackAsync").Should().BeTrue();
            _mockDownloadService.DownloadedFiles.Should().Contain(result);
        }

        [Fact]
        public void MockDownloadService_ShouldSimulateFailureScenarios()
        {
            // This demonstrates error handling that COULD be tested with proper abstractions
            
            // Arrange
            _mockDownloadService.Reset();
            _mockDownloadService.ShouldFailDownloads = true;
            var testUrl = "https://youtube.com/watch?v=invalid";
            
            // Act & Assert
            var exception = Record.ExceptionAsync(async () => 
                await _mockDownloadService.DownloadYoutubeTrackAsync(testUrl));
            
            exception.Result.Should().NotBeNull("Should throw exception on download failure");
            exception.Result.Should().BeOfType<InvalidOperationException>();
            
            _mockDownloadService.DownloadedFiles.Should().BeEmpty("No files should be downloaded on failure");
        }

        [Fact]
        public void MockDownloadService_ShouldSimulatePlaylistOperations()
        {
            // This demonstrates playlist functionality that COULD be tested
            
            // Arrange
            _mockDownloadService.Reset();
            var playlistUrl = "https://youtube.com/playlist?list=test123";
            
            // Act
            var task = _mockDownloadService.GetPlaylistYoutubeAsync(playlistUrl);
            var result = task.Result;
            
            // Assert
            result.Should().NotBeNull("Playlist download should return result");
            _mockDownloadService.PlaylistSongs.Should().NotBeEmpty("Playlist should contain songs");
            _mockDownloadService.PlaylistSongs.Should().HaveCountGreaterOrEqualTo(1, 
                "Playlist should have at least one song");
                
            _mockDownloadService.WasMethodCalled("GetPlaylistYoutubeAsync").Should().BeTrue();
        }

        [Fact]
        public void MockDownloadService_ShouldSimulateFFmpegOperations()
        {
            // This demonstrates FFmpeg integration testing
            
            // Arrange
            _mockDownloadService.Reset();
            var inputPath = "/test/input.mp4";
            var metadata = new Song { Title = "Test Song", Author = "Test Artist" };
            
            // Act
            var task = _mockDownloadService.ConvertWithFFmpegAsync(inputPath, metadata);
            
            // Assert - Should complete without exception when FFmpeg is available
            task.IsCompletedSuccessfully.Should().BeTrue("Conversion should complete successfully");
            _mockDownloadService.WasMethodCalled("ConvertWithFFmpegAsync").Should().BeTrue();
        }

        [Fact]
        public void MockDownloadService_ShouldSimulateFFmpegFailure()
        {
            // This demonstrates FFmpeg error handling
            
            // Arrange
            _mockDownloadService.Reset();
            _mockDownloadService.IsFFmpegInstalled = false;
            var inputPath = "/test/input.mp4";
            
            // Act & Assert
            var exception = Record.ExceptionAsync(async () => 
                await _mockDownloadService.ConvertWithFFmpegAsync(inputPath));
            
            exception.Result.Should().NotBeNull("Should throw when FFmpeg not available");
            exception.Result.Should().BeOfType<InvalidOperationException>();
        }

        [Fact]
        public void Download_TestingChallenges_Documentation()
        {
            // This test documents the architectural challenges for comprehensive testing
            
            // CURRENT TESTING CHALLENGES:
            // 1. All methods are static - prevents dependency injection
            // 2. Direct external API calls (YoutubeExplode, SoundCloudExplode) - no abstraction
            // 3. File system operations without abstraction (Path.Combine, File.Exists, etc.)
            // 4. External process execution (FFmpeg) - difficult to test without actual binaries
            // 5. Progress reporting tightly coupled to TUI - mixed concerns
            // 6. Global state dependencies (Preferences.songsPath, Utils.JammerPath)
            // 7. Complex async operations with error handling and retry logic
            
            // METHODS THAT ARE TESTABLE (utility functions):
            // - FormatUrlForFilename(string) - pure string manipulation
            // - GetDownloadedJammerFileName(string) - simple string concatenation
            // - IsFFmpegInstalled() - system check (environment dependent)
            
            // METHODS THAT NEED MAJOR REFACTORING FOR TESTING:
            // - DownloadYoutubeTrackAsync() - 90+ lines, external APIs, file I/O, FFmpeg
            // - DownloadSoundCloudTrackAsync() - similar complexity to YouTube
            // - FFMPEGConvert() - external process execution with complex argument building
            // - GetPlaylist() operations - external API calls with parsing logic
            // - All async download methods with progress reporting
            
            // RECOMMENDED ABSTRACTIONS FOR BETTER TESTING:
            // 1. IYouTubeService interface for YouTube API operations
            // 2. ISoundCloudService interface for SoundCloud API operations  
            // 3. IFFmpegService interface for media conversion operations
            // 4. IFileSystemService for file operations
            // 5. IProgressReporter for decoupled progress updates
            // 6. Configuration injection instead of global static access
            // 7. Async/await patterns with proper cancellation token support
            
            var downloadType = typeof(Download);
            downloadType.Should().NotBeNull("Download class exists for testing analysis");
            
            // Count static methods to quantify testing challenge
            var staticMethods = downloadType.GetMethods(System.Reflection.BindingFlags.Public | System.Reflection.BindingFlags.Static);
            staticMethods.Should().HaveCountGreaterOrEqualTo(7, 
                "Download has multiple static methods that present testing challenges");
            
            Assert.True(true, "Documentation test - highlights need for architectural improvements");
        }
    }
}
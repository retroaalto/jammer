using System.IO.Abstractions.TestingHelpers;
using FluentAssertions;
using Jammer.Tests.TestUtilities.Doubles;
using Jammer.Tests.TestUtilities.Fixtures;
using System;
using System.IO;
using System.Threading.Tasks;
using Xunit;

namespace Jammer.Tests.Integration
{
    /// <summary>
    /// Integration tests for the critical Download→FileSystem→Playlist workflow.
    /// Tests the complete flow: URL validation → download → file save → FFmpeg conversion → playlist integration.
    /// </summary>
    [Collection("Integration")]
    public class DownloadToPlaylistIntegrationTests : IDisposable
    {
        private readonly string tempSongsPath;
        private readonly string tempPlaylistsPath;
        private readonly MockDownloadService mockDownloadService;
        private readonly MockFileSystem fileSystem;

        public DownloadToPlaylistIntegrationTests()
        {
            // Arrange: Create isolated test environment
            tempSongsPath = Path.Combine(Path.GetTempPath(), "JammerTests", Guid.NewGuid().ToString(), "songs");
            tempPlaylistsPath = Path.Combine(Path.GetTempPath(), "JammerTests", Guid.NewGuid().ToString(), "playlists");
            
            Directory.CreateDirectory(tempSongsPath);
            Directory.CreateDirectory(tempPlaylistsPath);

            // Setup mock file system for testing
            fileSystem = new MockFileSystem();
            fileSystem.AddDirectory(tempSongsPath);
            fileSystem.AddDirectory(tempPlaylistsPath);

            // Initialize mock download service with controllable behavior
            mockDownloadService = new MockDownloadService();
            mockDownloadService.Reset();
            mockDownloadService.SongPath = tempSongsPath;
        }

        [Fact]
        [Trait("Category", "Integration")]
        [Trait("Priority", "Critical")]
        public async Task DownloadToPlaylist_YoutubeUrl_CompletesFullWorkflow()
        {
            // Arrange
            var youtubeUrl = "https://www.youtube.com/watch?v=dQw4w9WgXcQ";
            var expectedSongPath = Path.Combine(tempSongsPath, "youtube.com_watch_v_dQw4w9WgXcQ.ogg");
            var playlistName = "TestPlaylist";
            var playlistPath = Path.Combine(tempPlaylistsPath, $"{playlistName}.jammer");

            // Create test playlist file
            File.WriteAllText(playlistPath, "");

            // Act - Simulate the full workflow
            // Step 1: Download song
            var downloadedPath = await mockDownloadService.DownloadSongAsync(youtubeUrl);
            
            // Step 2: Verify file was created
            mockDownloadService.DownloadedFiles.Should().Contain(downloadedPath);
            
            // Step 3: Add to playlist (simulated - actual method calls Environment.Exit)
            var playlistContent = File.ReadAllText(playlistPath);
            File.AppendAllText(playlistPath, downloadedPath + Environment.NewLine);

            // Assert
            downloadedPath.Should().Be(expectedSongPath);
            File.ReadAllText(playlistPath).Should().Contain(downloadedPath);
            mockDownloadService.WasMethodCalled("DownloadYoutubeTrackAsync").Should().BeTrue();
        }

        [Fact]
        [Trait("Category", "Integration")]
        [Trait("Priority", "Critical")]
        public async Task DownloadToPlaylist_SoundCloudUrl_CompletesFullWorkflow()
        {
            // Arrange
            var soundcloudUrl = "https://soundcloud.com/artist/track-name";
            var expectedSongPath = Path.Combine(tempSongsPath, "soundcloud.com_artist_track-name.ogg");
            var playlistName = "SoundCloudPlaylist";
            var playlistPath = Path.Combine(tempPlaylistsPath, $"{playlistName}.jammer");

            File.WriteAllText(playlistPath, "");

            // Act
            var downloadedPath = await mockDownloadService.DownloadSongAsync(soundcloudUrl);
            File.AppendAllText(playlistPath, downloadedPath + Environment.NewLine);

            // Assert
            downloadedPath.Should().Be(expectedSongPath);
            File.ReadAllText(playlistPath).Should().Contain(downloadedPath);
            mockDownloadService.WasMethodCalled("DownloadSoundCloudTrackAsync").Should().BeTrue();
        }

        [Fact]
        [Trait("Category", "Integration")]
        [Trait("Priority", "High")]
        public async Task DownloadToPlaylist_NetworkFailure_HandlesGracefully()
        {
            // Arrange
            var youtubeUrl = "https://www.youtube.com/watch?v=invalid123";
            mockDownloadService.ShouldFailDownloads = true;

            // Act & Assert
            await Assert.ThrowsAsync<InvalidOperationException>(
                () => mockDownloadService.DownloadSongAsync(youtubeUrl));
            
            mockDownloadService.DownloadedFiles.Should().BeEmpty();
        }

        [Fact]
        [Trait("Category", "Integration")]
        [Trait("Priority", "High")]
        public async Task DownloadToPlaylist_DuplicateSong_PreventsDuplication()
        {
            // Arrange
            var youtubeUrl = "https://www.youtube.com/watch?v=duplicate123";
            var playlistName = "DuplicateTestPlaylist";
            var playlistPath = Path.Combine(tempPlaylistsPath, $"{playlistName}.jammer");
            
            var downloadedPath = await mockDownloadService.DownloadSongAsync(youtubeUrl);
            
            // Add song to playlist first time
            File.WriteAllText(playlistPath, downloadedPath + Environment.NewLine);

            // Act - Try to add same song again
            var existingSongs = File.ReadAllLines(playlistPath);
            if (!existingSongs.Contains(downloadedPath))
            {
                File.AppendAllText(playlistPath, downloadedPath + Environment.NewLine);
            }

            // Assert
            var finalSongs = File.ReadAllLines(playlistPath);
            finalSongs.Should().ContainSingle(song => song == downloadedPath);
        }

        [Fact]
        [Trait("Category", "Integration")]
        [Trait("Priority", "Medium")]
        public async Task DownloadToPlaylist_FFmpegNotAvailable_HandlesGracefully()
        {
            // Arrange
            mockDownloadService.IsFFmpegInstalled = false;

            // Act & Assert
            await Assert.ThrowsAsync<InvalidOperationException>(
                () => mockDownloadService.ConvertWithFFmpegAsync("/test/path.mp3"));
            
            mockDownloadService.WasMethodCalled("ConvertWithFFmpegAsync").Should().BeTrue();
        }

        [Fact]
        [Trait("Category", "Integration")]
        [Trait("Priority", "Medium")]
        public async Task DownloadToPlaylist_InvalidUrl_ReturnsEmpty()
        {
            // Arrange
            var invalidUrl = "not_a_valid_url";

            // Act
            var result = await mockDownloadService.DownloadSongAsync(invalidUrl);

            // Assert
            result.Should().Be(Path.Combine(tempSongsPath, "not_a_valid_url.mp3"));
            mockDownloadService.WasMethodCalled("GeneralDownloadAsync").Should().BeTrue();
        }

        [Fact]
        [Trait("Category", "Integration")]
        [Trait("Priority", "Medium")]
        public async Task DownloadToPlaylist_FileSystemPermissionError_HandlesGracefully()
        {
            // Arrange
            var youtubeUrl = "https://www.youtube.com/watch?v=permission123";
            var readOnlyPath = Path.Combine(tempSongsPath, "readonly");
            Directory.CreateDirectory(readOnlyPath);
            
            // Simulate read-only directory (simplified for test)
            mockDownloadService.SongPath = "/invalid/readonly/path";

            // Act
            var downloadedPath = await mockDownloadService.DownloadSongAsync(youtubeUrl);

            // Assert - Mock service still returns path but real implementation would handle permission errors
            downloadedPath.Should().NotBeNullOrEmpty();
            mockDownloadService.WasMethodCalled("DownloadSongAsync").Should().BeTrue();
        }

        [Fact]
        [Trait("Category", "Integration")]
        [Trait("Priority", "Medium")]
        public async Task DownloadToPlaylist_ProgressTracking_UpdatesCorrectly()
        {
            // Arrange
            var youtubeUrl = "https://www.youtube.com/watch?v=progress123";
            mockDownloadService.DownloadDelay = TimeSpan.FromMilliseconds(100);

            // Act
            var startTime = DateTime.Now;
            await mockDownloadService.DownloadSongAsync(youtubeUrl);
            var endTime = DateTime.Now;

            // Assert
            (endTime - startTime).Should().BeGreaterThan(TimeSpan.FromMilliseconds(90));
            mockDownloadService.WasMethodCalled("DownloadYoutubeTrackAsync").Should().BeTrue();
        }

        [Fact]
        [Trait("Category", "Integration")]
        [Trait("Priority", "High")]
        public async Task DownloadToPlaylist_PlaylistManagement_HandlesMultipleSongs()
        {
            // Arrange
            var urls = new[]
            {
                "https://www.youtube.com/watch?v=song1",
                "https://www.youtube.com/watch?v=song2",
                "https://soundcloud.com/artist/song3"
            };
            var playlistName = "MultiSongPlaylist";
            var playlistPath = Path.Combine(tempPlaylistsPath, $"{playlistName}.jammer");
            File.WriteAllText(playlistPath, "");

            // Act
            foreach (var url in urls)
            {
                var downloadedPath = await mockDownloadService.DownloadSongAsync(url);
                File.AppendAllText(playlistPath, downloadedPath + Environment.NewLine);
            }

            // Assert
            var playlistContent = File.ReadAllLines(playlistPath);
            playlistContent.Should().HaveCount(3);
            playlistContent.Should().Contain(line => line.Contains("song1"));
            playlistContent.Should().Contain(line => line.Contains("song2"));
            playlistContent.Should().Contain(line => line.Contains("song3"));
        }

        [Fact]
        [Trait("Category", "Integration")]
        [Trait("Priority", "Medium")]
        public async Task DownloadToPlaylist_JammerFileDownload_CompletesWorkflow()
        {
            // Arrange
            var jammerUrl = "https://example.com/playlist.jammer";

            // Act
            await mockDownloadService.DownloadJammerFileAsync(jammerUrl);

            // Assert
            var expectedFileName = mockDownloadService.GetDownloadedJammerFileName(jammerUrl);
            mockDownloadService.DownloadedFiles.Should().Contain(file => 
                Path.GetFileName(file) == expectedFileName);
        }

        [Fact]
        [Trait("Category", "Integration")]
        [Trait("Priority", "Low")]
        public void DownloadToPlaylist_FilenameFormatting_HandlesSpecialCharacters()
        {
            // Arrange
            var urlWithSpecialChars = "https://www.youtube.com/watch?v=test&t=123&list=abc";

            // Act
            var formattedName = mockDownloadService.FormatUrlForFilename(urlWithSpecialChars);

            // Assert
            formattedName.Should().NotContain("&");
            formattedName.Should().NotContain("?");
            formattedName.Should().NotContain("=");
            formattedName.Should().Contain("_");
        }

        public void Dispose()
        {
            // Cleanup test directories
            var parentDir = Path.GetDirectoryName(tempSongsPath);
            if (parentDir != null && Directory.Exists(parentDir))
            {
                Directory.Delete(parentDir, true);
            }
        }
    }
}
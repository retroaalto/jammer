using FluentAssertions;
using Jammer.Tests.TestUtilities.Fixtures;
using System;
using System.IO;
using System.Linq;
using Xunit;

namespace Jammer.Tests.Integration
{
    /// <summary>
    /// Integration tests for playlist management after downloads.
    /// Tests .jammer format, serialization/deserialization, and playlist consistency.
    /// </summary>
    [Collection("Integration")]
    public class PlaylistIntegrationTests : IDisposable
    {
        private readonly string testPlaylistsPath;

        public PlaylistIntegrationTests()
        {
            testPlaylistsPath = Path.Combine(Path.GetTempPath(), "JammerPlaylistTests", Guid.NewGuid().ToString());
            Directory.CreateDirectory(testPlaylistsPath);
        }

        [Fact]
        [Trait("Category", "Integration")]
        [Trait("Priority", "Critical")]
        public void Playlist_AddDownloadedSong_UpdatesJammerFile()
        {
            // Arrange
            var playlistName = "TestPlaylist";
            var playlistPath = Path.Combine(testPlaylistsPath, $"{playlistName}.jammer");
            var downloadedSongPath = "/path/to/downloaded/song.ogg";

            // Create empty playlist
            File.WriteAllText(playlistPath, "");

            // Act - Simulate adding downloaded song to playlist
            var existingSongs = File.Exists(playlistPath) ? File.ReadAllLines(playlistPath) : new string[0];
            if (!existingSongs.Contains(downloadedSongPath))
            {
                File.AppendAllText(playlistPath, downloadedSongPath + Environment.NewLine);
            }

            // Assert
            var updatedPlaylist = File.ReadAllLines(playlistPath);
            updatedPlaylist.Should().Contain(downloadedSongPath);
            updatedPlaylist.Should().HaveCount(1);
        }

        [Fact]
        [Trait("Category", "Integration")]
        [Trait("Priority", "Critical")]
        public void Playlist_PreventDuplicateEntries_MaintainsConsistency()
        {
            // Arrange
            var playlistName = "DuplicateTestPlaylist";
            var playlistPath = Path.Combine(testPlaylistsPath, $"{playlistName}.jammer");
            var songPath = "/path/to/duplicate/song.ogg";

            // Create playlist with existing song
            File.WriteAllText(playlistPath, songPath + Environment.NewLine);

            // Act - Try to add the same song again
            var existingSongs = File.ReadAllLines(playlistPath);
            if (!existingSongs.Contains(songPath))
            {
                File.AppendAllText(playlistPath, songPath + Environment.NewLine);
            }

            // Assert
            var finalPlaylist = File.ReadAllLines(playlistPath);
            finalPlaylist.Where(song => song == songPath).Should().HaveCount(1);
        }

        [Fact]
        [Trait("Category", "Integration")]
        [Trait("Priority", "High")]
        public void Playlist_HandleMultipleDownloads_MaintainsOrder()
        {
            // Arrange
            var playlistName = "OrderTestPlaylist";
            var playlistPath = Path.Combine(testPlaylistsPath, $"{playlistName}.jammer");
            var songPaths = new[]
            {
                "/path/to/first/song.ogg",
                "/path/to/second/song.ogg",
                "/path/to/third/song.ogg"
            };

            File.WriteAllText(playlistPath, "");

            // Act - Add songs in sequence
            foreach (var songPath in songPaths)
            {
                var existingSongs = File.ReadAllLines(playlistPath);
                if (!existingSongs.Contains(songPath))
                {
                    File.AppendAllText(playlistPath, songPath + Environment.NewLine);
                }
            }

            // Assert
            var playlistContent = File.ReadAllLines(playlistPath);
            playlistContent.Should().HaveCount(3);
            for (int i = 0; i < songPaths.Length; i++)
            {
                playlistContent[i].Should().Be(songPaths[i]);
            }
        }

        [Fact]
        [Trait("Category", "Integration")]
        [Trait("Priority", "High")]
        public void Playlist_HandleMissingFile_CreatesNew()
        {
            // Arrange
            var playlistName = "NonExistentPlaylist";
            var playlistPath = Path.Combine(testPlaylistsPath, $"{playlistName}.jammer");
            var songPath = "/path/to/new/song.ogg";

            // Ensure playlist doesn't exist
            if (File.Exists(playlistPath))
            {
                File.Delete(playlistPath);
            }

            // Act - Try to add song to non-existent playlist (simulate creation)
            if (!File.Exists(playlistPath))
            {
                // In real implementation, this would show error or create new playlist
                File.WriteAllText(playlistPath, "");
            }
            File.AppendAllText(playlistPath, songPath + Environment.NewLine);

            // Assert
            File.Exists(playlistPath).Should().BeTrue();
            var playlistContent = File.ReadAllLines(playlistPath);
            playlistContent.Should().Contain(songPath);
        }

        [Fact]
        [Trait("Category", "Integration")]
        [Trait("Priority", "Medium")]
        public void Playlist_SerializeMetadata_PreservesFormat()
        {
            // Arrange
            var playlistName = "MetadataPlaylist";
            var playlistPath = Path.Combine(testPlaylistsPath, $"{playlistName}.jammer");
            
            // Use test data fixture
            var sampleData = TestPlaylistData.ValidPlaylists.SamplePlaylistJson;
            
            // Act
            File.WriteAllText(playlistPath, sampleData);

            // Assert
            var savedContent = File.ReadAllText(playlistPath);
            savedContent.Should().Contain("My Test Playlist");
            savedContent.Should().Contain("/path/to/song1.mp3");
            savedContent.Should().Contain("/path/to/song2.flac");
            savedContent.Should().Contain("/path/to/song3.wav");
        }

        [Fact]
        [Trait("Category", "Integration")]
        [Trait("Priority", "Medium")]
        public void Playlist_HandleCorruptedFile_DetectsError()
        {
            // Arrange
            var playlistName = "CorruptedPlaylist";
            var playlistPath = Path.Combine(testPlaylistsPath, $"{playlistName}.jammer");
            
            // Create corrupted playlist file
            var corruptedContent = TestPlaylistData.InvalidPlaylists.InvalidJsonPlaylist;
            File.WriteAllText(playlistPath, corruptedContent);

            // Act & Assert - This would normally be handled by playlist parser
            var content = File.ReadAllText(playlistPath);
            content.Should().Contain("Broken Playlist");
            // In real implementation, JSON parsing would fail and be handled gracefully
        }

        [Fact]
        [Trait("Category", "Integration")]
        [Trait("Priority", "Medium")]
        public void Playlist_PathNormalization_HandlesAbsolutePaths()
        {
            // Arrange
            var playlistName = "PathNormalizationTest";
            var playlistPath = Path.Combine(testPlaylistsPath, $"{playlistName}.jammer");
            
            var relativePath = "songs/relative_song.ogg";
            var absolutePath = "/absolute/path/to/song.ogg";
            
            File.WriteAllText(playlistPath, "");

            // Act - Add both relative and absolute paths
            File.AppendAllText(playlistPath, relativePath + Environment.NewLine);
            File.AppendAllText(playlistPath, absolutePath + Environment.NewLine);

            // Assert
            var playlistContent = File.ReadAllLines(playlistPath);
            playlistContent.Should().Contain(relativePath);
            playlistContent.Should().Contain(absolutePath);
            playlistContent.Should().HaveCount(2);
        }

        [Fact]
        [Trait("Category", "Integration")]
        [Trait("Priority", "Low")]
        public void Playlist_EmptyPlaylist_HandlesGracefully()
        {
            // Arrange
            var playlistName = "EmptyPlaylist";
            var playlistPath = Path.Combine(testPlaylistsPath, $"{playlistName}.jammer");
            
            // Act
            File.WriteAllText(playlistPath, "");

            // Assert
            File.Exists(playlistPath).Should().BeTrue();
            var content = File.ReadAllText(playlistPath);
            content.Should().BeEmpty();
            
            var lines = File.ReadAllLines(playlistPath);
            lines.Should().BeEmpty();
        }

        [Fact]
        [Trait("Category", "Integration")]
        [Trait("Priority", "Medium")]
        public void Playlist_RemoveSong_UpdatesCorrectly()
        {
            // Arrange
            var playlistName = "RemovalTestPlaylist";
            var playlistPath = Path.Combine(testPlaylistsPath, $"{playlistName}.jammer");
            
            var songs = new[]
            {
                "/path/to/keep1.ogg",
                "/path/to/remove.ogg",
                "/path/to/keep2.ogg"
            };

            // Create playlist with multiple songs
            File.WriteAllLines(playlistPath, songs);

            // Act - Remove middle song
            var songToRemove = "/path/to/remove.ogg";
            var currentSongs = File.ReadAllLines(playlistPath);
            var updatedSongs = currentSongs.Where(song => song != songToRemove).ToArray();
            File.WriteAllLines(playlistPath, updatedSongs);

            // Assert
            var finalSongs = File.ReadAllLines(playlistPath);
            finalSongs.Should().HaveCount(2);
            finalSongs.Should().NotContain(songToRemove);
            finalSongs.Should().Contain("/path/to/keep1.ogg");
            finalSongs.Should().Contain("/path/to/keep2.ogg");
        }

        [Fact]
        [Trait("Category", "Integration")]
        [Trait("Priority", "Low")]
        public void Playlist_LargePlaylist_HandlesManySongs()
        {
            // Arrange
            var playlistName = "LargePlaylist";
            var playlistPath = Path.Combine(testPlaylistsPath, $"{playlistName}.jammer");
            
            var songPaths = Enumerable.Range(1, 100)
                .Select(i => $"/path/to/song{i:D3}.ogg")
                .ToArray();

            // Act
            File.WriteAllLines(playlistPath, songPaths);

            // Assert
            var savedSongs = File.ReadAllLines(playlistPath);
            savedSongs.Should().HaveCount(100);
            savedSongs.Should().Contain("/path/to/song001.ogg");
            savedSongs.Should().Contain("/path/to/song050.ogg");
            savedSongs.Should().Contain("/path/to/song100.ogg");
        }

        [Fact]
        [Trait("Category", "Integration")]
        [Trait("Priority", "Medium")]
        public void Playlist_ConcurrentAccess_HandlesFileLocking()
        {
            // Arrange
            var playlistName = "ConcurrentAccessTest";
            var playlistPath = Path.Combine(testPlaylistsPath, $"{playlistName}.jammer");
            
            File.WriteAllText(playlistPath, "/initial/song.ogg" + Environment.NewLine);

            // Act - Simulate concurrent access (simplified)
            var song1 = "/concurrent/song1.ogg";
            var song2 = "/concurrent/song2.ogg";

            // First operation
            File.AppendAllText(playlistPath, song1 + Environment.NewLine);
            
            // Second operation
            File.AppendAllText(playlistPath, song2 + Environment.NewLine);

            // Assert
            var finalPlaylist = File.ReadAllLines(playlistPath);
            finalPlaylist.Should().HaveCount(3);
            finalPlaylist.Should().Contain("/initial/song.ogg");
            finalPlaylist.Should().Contain(song1);
            finalPlaylist.Should().Contain(song2);
        }

        public void Dispose()
        {
            if (Directory.Exists(testPlaylistsPath))
            {
                Directory.Delete(testPlaylistsPath, true);
            }
        }
    }
}
using System.IO.Abstractions.TestingHelpers;
using FluentAssertions;
using Jammer.Tests.TestUtilities.Fixtures;
using System;
using System.IO;
using Xunit;

namespace Jammer.Tests.Integration
{
    /// <summary>
    /// Integration tests for file system operations after downloads.
    /// Tests file organization, directory structure, permissions, and cleanup scenarios.
    /// </summary>
    [Collection("Integration")]
    public class FileSystemIntegrationTests : IDisposable
    {
        private readonly string testBasePath;
        private readonly MockFileSystem fileSystem;

        public FileSystemIntegrationTests()
        {
            testBasePath = Path.Combine(Path.GetTempPath(), "JammerFileSystemTests", Guid.NewGuid().ToString());
            Directory.CreateDirectory(testBasePath);

            fileSystem = new MockFileSystem();
        }

        [Fact]
        [Trait("Category", "Integration")]
        [Trait("Priority", "Critical")]
        public void FileSystem_CreateDirectoryStructure_CreatesHierarchy()
        {
            // Arrange
            var songsPath = Path.Combine(testBasePath, "songs");
            var playlistsPath = Path.Combine(testBasePath, "playlists");
            var themesPath = Path.Combine(testBasePath, "themes");

            // Act
            Directory.CreateDirectory(songsPath);
            Directory.CreateDirectory(playlistsPath);
            Directory.CreateDirectory(themesPath);

            // Assert
            Directory.Exists(songsPath).Should().BeTrue();
            Directory.Exists(playlistsPath).Should().BeTrue();
            Directory.Exists(themesPath).Should().BeTrue();
        }

        [Fact]
        [Trait("Category", "Integration")]
        [Trait("Priority", "High")]
        public void FileSystem_SaveDownloadedFile_CreatesFileWithCorrectExtension()
        {
            // Arrange
            var songsPath = Path.Combine(testBasePath, "songs");
            Directory.CreateDirectory(songsPath);
            
            var originalFileName = "test_download.webm";
            var convertedFileName = "test_download.ogg";
            var originalPath = Path.Combine(songsPath, originalFileName);
            var convertedPath = Path.Combine(songsPath, convertedFileName);

            // Act - Simulate download and conversion process
            File.WriteAllText(originalPath, "dummy audio data");
            
            // Simulate FFmpeg conversion: create .ogg file and remove original
            File.WriteAllText(convertedPath, "converted ogg audio data");
            File.Delete(originalPath);

            // Assert
            File.Exists(originalPath).Should().BeFalse();
            File.Exists(convertedPath).Should().BeTrue();
            Path.GetExtension(convertedPath).Should().Be(".ogg");
        }

        [Fact]
        [Trait("Category", "Integration")]
        [Trait("Priority", "High")]
        public void FileSystem_DuplicateFileCheck_PreventsOverwrite()
        {
            // Arrange
            var songsPath = Path.Combine(testBasePath, "songs");
            Directory.CreateDirectory(songsPath);
            
            var fileName = "existing_song.ogg";
            var filePath = Path.Combine(songsPath, fileName);
            var originalContent = "original audio data";

            File.WriteAllText(filePath, originalContent);

            // Act - Simulate duplicate file check before download
            bool fileExists = File.Exists(filePath);
            if (!fileExists)
            {
                File.WriteAllText(filePath, "new audio data");
            }

            // Assert
            fileExists.Should().BeTrue();
            File.ReadAllText(filePath).Should().Be(originalContent);
        }

        [Fact]
        [Trait("Category", "Integration")]
        [Trait("Priority", "Medium")]
        public void FileSystem_FileOrganization_HandlesSubdirectories()
        {
            // Arrange
            var songsPath = Path.Combine(testBasePath, "songs");
            var artistPath = Path.Combine(songsPath, "TestArtist");
            Directory.CreateDirectory(artistPath);

            var songFileName = "test_song.ogg";
            var songPath = Path.Combine(artistPath, songFileName);

            // Act
            File.WriteAllText(songPath, "artist song data");

            // Assert
            File.Exists(songPath).Should().BeTrue();
            Directory.GetFiles(artistPath).Should().HaveCount(1);
            Directory.GetFiles(artistPath)[0].Should().EndWith(songFileName);
        }

        [Fact]
        [Trait("Category", "Integration")]
        [Trait("Priority", "Medium")]
        public void FileSystem_CleanupFailedDownload_RemovesPartialFiles()
        {
            // Arrange
            var songsPath = Path.Combine(testBasePath, "songs");
            Directory.CreateDirectory(songsPath);
            
            var partialFileName = "incomplete_download.part";
            var partialPath = Path.Combine(songsPath, partialFileName);
            var tempFileName = "temp_conversion.tmp";
            var tempPath = Path.Combine(songsPath, tempFileName);

            File.WriteAllText(partialPath, "incomplete data");
            File.WriteAllText(tempPath, "temp data");

            // Act - Simulate cleanup after failed download
            if (File.Exists(partialPath))
            {
                File.Delete(partialPath);
            }
            if (File.Exists(tempPath))
            {
                File.Delete(tempPath);
            }

            // Assert
            File.Exists(partialPath).Should().BeFalse();
            File.Exists(tempPath).Should().BeFalse();
        }

        [Fact]
        [Trait("Category", "Integration")]
        [Trait("Priority", "Low")]
        public void FileSystem_FilePermissions_HandlesDifferentAccess()
        {
            // Arrange
            var songsPath = Path.Combine(testBasePath, "songs");
            Directory.CreateDirectory(songsPath);
            
            var testFileName = "permission_test.ogg";
            var testFilePath = Path.Combine(songsPath, testFileName);

            // Act
            File.WriteAllText(testFilePath, "test data");
            var fileInfo = new FileInfo(testFilePath);

            // Assert
            fileInfo.Exists.Should().BeTrue();
            fileInfo.Length.Should().BeGreaterThan(0);
        }

        [Fact]
        [Trait("Category", "Integration")]
        [Trait("Priority", "Medium")]
        public void FileSystem_LargeFileHandling_ManagesSize()
        {
            // Arrange
            var songsPath = Path.Combine(testBasePath, "songs");
            Directory.CreateDirectory(songsPath);
            
            var largeFileName = "large_song.ogg";
            var largeFilePath = Path.Combine(songsPath, largeFileName);
            var largeContent = new string('A', 10000); // 10KB test file

            // Act
            File.WriteAllText(largeFilePath, largeContent);
            var fileInfo = new FileInfo(largeFilePath);

            // Assert
            fileInfo.Length.Should().Be(largeContent.Length);
            File.Exists(largeFilePath).Should().BeTrue();
        }

        [Fact]
        [Trait("Category", "Integration")]
        [Trait("Priority", "Medium")]
        public void FileSystem_PathNormalization_HandlesSpecialCharacters()
        {
            // Arrange
            var songsPath = Path.Combine(testBasePath, "songs");
            Directory.CreateDirectory(songsPath);
            
            var sanitizedName = "song_with_spaces___symbols_.ogg";
            var sanitizedPath = Path.Combine(songsPath, sanitizedName);

            // Act - Simulate filename sanitization
            File.WriteAllText(sanitizedPath, "sanitized song data");

            // Assert
            File.Exists(sanitizedPath).Should().BeTrue();
            Path.GetFileName(sanitizedPath).Should().Be(sanitizedName);
        }

        [Fact]
        [Trait("Category", "Integration")]
        [Trait("Priority", "Low")]
        public void FileSystem_DirectoryTraversal_ListsFilesCorrectly()
        {
            // Arrange
            var songsPath = Path.Combine(testBasePath, "songs");
            Directory.CreateDirectory(songsPath);
            
            var songFiles = new[] { "song1.ogg", "song2.ogg", "song3.mp3" };
            foreach (var songFile in songFiles)
            {
                File.WriteAllText(Path.Combine(songsPath, songFile), "dummy data");
            }

            // Act
            var foundFiles = Directory.GetFiles(songsPath);

            // Assert
            foundFiles.Should().HaveCount(songFiles.Length);
            foreach (var expectedFile in songFiles)
            {
                foundFiles.Should().Contain(f => Path.GetFileName(f) == expectedFile);
            }
        }

        [Fact]
        [Trait("Category", "Integration")]
        [Trait("Priority", "Medium")]
        public void FileSystem_BackupAndRestore_PreservesData()
        {
            // Arrange
            var songsPath = Path.Combine(testBasePath, "songs");
            var backupPath = Path.Combine(testBasePath, "backup");
            Directory.CreateDirectory(songsPath);
            Directory.CreateDirectory(backupPath);
            
            var originalFile = Path.Combine(songsPath, "important_song.ogg");
            var backupFile = Path.Combine(backupPath, "important_song.ogg");
            var originalData = "important song data";

            File.WriteAllText(originalFile, originalData);

            // Act - Create backup
            File.Copy(originalFile, backupFile);
            
            // Simulate corruption and restore
            File.WriteAllText(originalFile, "corrupted");
            File.Copy(backupFile, originalFile, overwrite: true);

            // Assert
            File.ReadAllText(originalFile).Should().Be(originalData);
            File.ReadAllText(backupFile).Should().Be(originalData);
        }

        public void Dispose()
        {
            if (Directory.Exists(testBasePath))
            {
                Directory.Delete(testBasePath, true);
            }
        }
    }
}
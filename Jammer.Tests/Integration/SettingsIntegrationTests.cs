using FluentAssertions;
using Jammer.Tests.TestUtilities.Doubles;
using System;
using System.IO;
using System.Text.Json;
using Xunit;

namespace Jammer.Tests.Integration
{
    /// <summary>
    /// Integration tests for Settings→Runtime Behavior pipeline.
    /// Tests complete integration between settings persistence, audio effects, UI themes, and path configuration.
    /// Verifies settings changes trigger immediate runtime effects without restart requirement.
    /// </summary>
    [Collection("Integration")]
    public class SettingsIntegrationTests : IDisposable
    {
        private readonly string tempJammerPath;
        private readonly string tempSettingsPath;
        private readonly string originalJammerPath;
        private readonly MockAudioPlayer mockAudioPlayer;

        public SettingsIntegrationTests()
        {
            // Arrange: Create isolated test environment
            originalJammerPath = Utils.JammerPath;
            tempJammerPath = Path.Combine(Path.GetTempPath(), "JammerSettingsTests", Guid.NewGuid().ToString());
            tempSettingsPath = Path.Combine(tempJammerPath, "settings.json");
            
            Directory.CreateDirectory(tempJammerPath);
            Utils.JammerPath = tempJammerPath;

            mockAudioPlayer = new MockAudioPlayer();
            mockAudioPlayer.Reset();
        }

        [Fact]
        [Trait("Category", "Integration")]
        [Trait("Priority", "Critical")]
        public void Settings_Persistence_SaveAndLoad_WorksCorrectly()
        {
            // Arrange
            Preferences.volume = 0.8f;
            Preferences.isLoop = true;
            Preferences.isMuted = false;
            Preferences.theme = "TestTheme";

            // Act - Save settings
            Preferences.SaveSettings();

            // Reset static values to test loading
            Preferences.volume = 0f;
            Preferences.isLoop = false;
            Preferences.isMuted = true;
            Preferences.theme = "";

            // Reload from saved file
            var loadedVolume = Preferences.GetVolume();
            var loadedIsLoop = Preferences.GetIsLoop();
            var loadedIsMuted = Preferences.GetIsMuted();
            var loadedTheme = Preferences.GetTheme();

            // Assert
            File.Exists(tempSettingsPath).Should().BeTrue();
            loadedVolume.Should().Be(0.8f);
            loadedIsLoop.Should().BeTrue();
            loadedIsMuted.Should().BeFalse();
            loadedTheme.Should().Be("TestTheme");
        }

        [Fact]
        [Trait("Category", "Integration")]
        [Trait("Priority", "Critical")]
        public void Settings_JSONCorruption_FallsBackToDefaults()
        {
            // Arrange - Create corrupted JSON file
            File.WriteAllText(tempSettingsPath, "Not valid JSON at all!");

            // Act - Try to load settings
            var volume = Preferences.GetVolume();
            var isLoop = Preferences.GetIsLoop();
            var isMuted = Preferences.GetIsMuted();
            var theme = Preferences.GetTheme();

            // Assert - Should return defaults
            volume.Should().Be(0.5f);
            isLoop.Should().BeFalse();
            isMuted.Should().BeFalse();
            theme.Should().Be("Jammer Default");
        }

        [Fact]
        [Trait("Category", "Integration")]
        [Trait("Priority", "Critical")]
        public void Settings_VolumeChange_ImmediatelyAffectsAudio()
        {
            // Arrange
            var mockHandle = mockAudioPlayer.CreateStream("test.mp3", 0, 100000, ManagedBass.BassFlags.Default);
            Utils.CurrentMusic = mockHandle;
            Preferences.volume = 0.5f;

            // Act - Modify volume through Play.ModifyVolume
            Play.ModifyVolume(0.3f);

            // Assert - Volume should be updated immediately
            Preferences.volume.Should().Be(0.8f);
            
            // Verify BASS operation was called
            mockAudioPlayer.WasMethodCalled("ChannelSetAttribute").Should().BeTrue();
        }

        [Fact]
        [Trait("Category", "Integration")]
        [Trait("Priority", "Critical")]
        public void Settings_VolumeBoundaryClamp_EnforcesLimits()
        {
            // Arrange
            var mockHandle = mockAudioPlayer.CreateStream("test.mp3", 0, 100000, ManagedBass.BassFlags.Default);
            Utils.CurrentMusic = mockHandle;
            Preferences.volume = 1.0f;

            // Act - Try to exceed maximum volume (1.5f)
            Play.ModifyVolume(1.0f);

            // Assert - Should be clamped to 1.5f
            Preferences.volume.Should().Be(1.5f);

            // Act - Try to go below minimum (0f)
            Play.ModifyVolume(-2.0f);

            // Assert - Should be clamped to 0f
            Preferences.volume.Should().Be(0f);
        }

        [Fact]
        [Trait("Category", "Integration")]
        [Trait("Priority", "High")]
        public void Settings_MuteToggle_PreservesVolumeState()
        {
            // Arrange
            var mockHandle = mockAudioPlayer.CreateStream("test.mp3", 0, 100000, ManagedBass.BassFlags.Default);
            Utils.CurrentMusic = mockHandle;
            var originalVolume = 0.7f;
            Preferences.volume = originalVolume;
            Preferences.isMuted = false;

            // Act - Toggle mute on
            Play.ToggleMute();

            // Assert - Should preserve old volume and set current to 0
            Preferences.isMuted.Should().BeTrue();
            Preferences.oldVolume.Should().Be(originalVolume);
            Preferences.volume.Should().Be(0f);

            // Act - Toggle mute off
            Play.ToggleMute();

            // Assert - Should restore original volume
            Preferences.isMuted.Should().BeFalse();
            Preferences.volume.Should().Be(originalVolume);
        }

        [Fact]
        [Trait("Category", "Integration")]
        [Trait("Priority", "High")]
        public void Settings_GetChangeVolumeBy_HandlesJSONError()
        {
            // Arrange - Create JSON with malformed changeVolumeBy
            var malformedJson = @"{""Volume"": 0.5, ""changeVolumeBy"": ""not_a_number""}";
            File.WriteAllText(tempSettingsPath, malformedJson);

            // Act - Should catch exception and return default
            var changeBy = Preferences.GetChangeVolumeBy();

            // Assert
            changeBy.Should().Be(0.05f);
        }

        [Fact]
        [Trait("Category", "Integration")]
        [Trait("Priority", "High")]
        public void Settings_ThemeSwitch_UpdatesImmediately()
        {
            // Arrange - Create mock theme file
            var themesPath = Path.Combine(tempJammerPath, "themes");
            Directory.CreateDirectory(themesPath);
            var themeFilePath = Path.Combine(themesPath, "TestTheme.json");
            
            var mockTheme = @"{
                ""Playlist"": {
                    ""BorderStyle"": ""Rounded"",
                    ""PathColor"": ""red""
                }
            }";
            File.WriteAllText(themeFilePath, mockTheme);

            // Set theme path for Themes class
            typeof(Themes).GetField("themePath", System.Reflection.BindingFlags.NonPublic | System.Reflection.BindingFlags.Static)
                ?.SetValue(null, themesPath);

            // Act - Set theme
            var result = Themes.SetTheme("TestTheme");

            // Assert
            result.Should().BeTrue();
            Preferences.theme.Should().Be("TestTheme");
            Themes.CurrentTheme.Should().NotBeNull();
        }

        [Fact]
        [Trait("Category", "Integration")]
        [Trait("Priority", "High")]
        public void Settings_InvalidTheme_HandlesGracefully()
        {
            // Arrange - Set theme path but don't create theme file
            var themesPath = Path.Combine(tempJammerPath, "themes");
            Directory.CreateDirectory(themesPath);
            typeof(Themes).GetField("themePath", System.Reflection.BindingFlags.NonPublic | System.Reflection.BindingFlags.Static)
                ?.SetValue(null, themesPath);

            // Act - Try to set non-existent theme
            var result = Themes.SetTheme("NonExistentTheme");

            // Assert
            result.Should().BeFalse();
        }

        [Fact]
        [Trait("Category", "Integration")]
        [Trait("Priority", "High")]
        public void Settings_FolderCreation_CreatesRequiredDirectories()
        {
            // Arrange - Delete test directory to simulate fresh install
            if (Directory.Exists(tempJammerPath))
            {
                Directory.Delete(tempJammerPath, true);
            }
            Directory.CreateDirectory(tempJammerPath);

            // Act - Call CheckJammerFolderExists
            Preferences.CheckJammerFolderExists();

            // Assert - All required directories should be created
            Directory.Exists(tempJammerPath).Should().BeTrue();
            Directory.Exists(Preferences.GetPlaylistsPath()).Should().BeTrue();
            Directory.Exists(Path.Combine(tempJammerPath, "soundfonts")).Should().BeTrue();
            Directory.Exists(Path.Combine(tempJammerPath, "locales")).Should().BeTrue();
            Directory.Exists(Preferences.songsPath).Should().BeTrue();
        }

        [Fact]
        [Trait("Category", "Integration")]
        [Trait("Priority", "Medium")]
        public void Settings_EnvironmentVariablePrecedence_OverridesConfig()
        {
            // Arrange - Set environment variable for songs path
            var customSongsPath = Path.Combine(tempJammerPath, "custom_songs");
            Environment.SetEnvironmentVariable("JAMMER_SONGS_PATH", customSongsPath);

            try
            {
                // Act - Get songs path (should use environment variable)
                var songsPath = Preferences.GetSongsPath();

                // Assert
                songsPath.Should().Be(customSongsPath);
            }
            finally
            {
                // Cleanup
                Environment.SetEnvironmentVariable("JAMMER_SONGS_PATH", null);
            }
        }

        [Fact]
        [Trait("Category", "Integration")]
        [Trait("Priority", "Medium")]
        public void Settings_PlaylistsPath_RespectsEnvironmentVariable()
        {
            // Arrange
            var customPlaylistsPath = Path.Combine(tempJammerPath, "custom_playlists");
            Environment.SetEnvironmentVariable("JAMMER_PLAYLISTS_PATH", customPlaylistsPath);

            try
            {
                // Act
                var playlistsPath = Preferences.GetPlaylistsPath();

                // Assert
                playlistsPath.Should().Be(customPlaylistsPath);
            }
            finally
            {
                // Cleanup
                Environment.SetEnvironmentVariable("JAMMER_PLAYLISTS_PATH", null);
            }
        }

        [Fact]
        [Trait("Category", "Integration")]
        [Trait("Priority", "Medium")]
        public void Settings_CrossComponentIntegration_VolumeAffectsUI()
        {
            // Arrange
            var mockHandle = mockAudioPlayer.CreateStream("test.mp3", 0, 100000, ManagedBass.BassFlags.Default);
            Utils.CurrentMusic = mockHandle;
            Utils.TotalMusicDurationInSec = 50.0;
            Utils.SongDurationInSec = 200.0;
            Preferences.volume = 0.6f;
            Preferences.isMuted = false;

            // Act - Generate progress bar (which includes volume display)
            var progressBar = TUI.ProgressBar(25.0, 200.0);

            // Assert - Progress bar should contain volume information
            progressBar.Should().NotBeNullOrEmpty();
            progressBar.Should().Contain("60%"); // Volume percentage display
        }

        [Fact]
        [Trait("Category", "Integration")]
        [Trait("Priority", "Medium")]
        public void Settings_Persistence_PartialJSON_GracefulFallback()
        {
            // Arrange - Create partial settings file missing some properties
            var partialJson = @"{""Volume"": 0.9, ""IsLoop"": true}";
            File.WriteAllText(tempSettingsPath, partialJson);

            // Act - Load settings
            var volume = Preferences.GetVolume();
            var isLoop = Preferences.GetIsLoop();
            var isMuted = Preferences.GetIsMuted(); // Missing from JSON
            var changeVolumeBy = Preferences.GetChangeVolumeBy(); // Missing from JSON

            // Assert - Should use saved values where available, defaults otherwise
            volume.Should().Be(0.9f);
            isLoop.Should().BeTrue();
            isMuted.Should().BeFalse(); // Default
            changeVolumeBy.Should().Be(0.05f); // Default
        }

        [Fact]
        [Trait("Category", "Integration")]
        [Trait("Priority", "Low")]
        public void Settings_LocaleLanguage_DefaultHandling()
        {
            // Arrange - No settings file exists
            if (File.Exists(tempSettingsPath))
            {
                File.Delete(tempSettingsPath);
            }

            // Act
            var locale = Preferences.GetLocaleLanguage();

            // Assert - Should return default locale
            locale.Should().Be("en");
        }

        [Fact]
        [Trait("Category", "Integration")]
        [Trait("Priority", "Low")]
        public void Settings_BooleanDefaults_HandleMissingValues()
        {
            // Arrange - Empty settings file
            File.WriteAllText(tempSettingsPath, "{}");

            // Act
            var isMediaButtons = Preferences.GetIsMediaButtons();
            var isVisualizer = Preferences.GetIsVisualizer();
            var isShuffle = Preferences.GetIsShuffle();
            var isAutoSave = Preferences.GetIsAutoSave();

            // Assert - Should use appropriate defaults
            isMediaButtons.Should().BeTrue();
            isVisualizer.Should().BeTrue();
            isShuffle.Should().BeFalse();
            isAutoSave.Should().BeFalse();
        }

        public void Dispose()
        {
            // Cleanup: Restore original state and remove temporary directory
            Utils.JammerPath = originalJammerPath;
            
            if (Directory.Exists(tempJammerPath))
            {
                try
                {
                    Directory.Delete(tempJammerPath, true);
                }
                catch
                {
                    // Ignore cleanup errors in tests
                }
            }

            mockAudioPlayer?.Reset();
        }
    }
}
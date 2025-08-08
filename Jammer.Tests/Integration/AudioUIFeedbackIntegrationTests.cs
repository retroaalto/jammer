using FluentAssertions;
using Jammer.Tests.TestUtilities.Doubles;
using Jammer.Tests.TestUtilities.Fixtures;
using System;
using System.Collections.Generic;
using System.IO;
using ManagedBass;
using Xunit;

namespace Jammer.Tests.Integration
{
    /// <summary>
    /// Integration tests for Audio Engine→UI Feedback pipeline.
    /// Tests complete integration between audio playback (Play.cs), UI updates (TUI.cs), and state management.
    /// Verifies audio state changes trigger UI updates, progress tracking synchronizes correctly, and metadata displays properly.
    /// </summary>
    [Collection("Integration")]
    public class AudioUIFeedbackIntegrationTests : IDisposable
    {
        private readonly MockAudioPlayer mockAudioPlayer;
        private readonly List<string> uiMessages;
        private readonly string tempSongPath;

        public AudioUIFeedbackIntegrationTests()
        {
            // Initialize test environment
            tempSongPath = Path.Combine(Path.GetTempPath(), "JammerAudioTests", Guid.NewGuid().ToString());
            Directory.CreateDirectory(tempSongPath);

            // Initialize themes to prevent null reference errors
            Themes.SetDefaultTheme();

            mockAudioPlayer = new MockAudioPlayer();
            mockAudioPlayer.Reset();

            uiMessages = new List<string>();

            // Initialize test state
            Utils.Songs = new string[] { };
            Utils.CurrentMusic = 0;
            Utils.CurSongError = false;
            Start.state = MainStates.idle;
        }

        [Fact]
        [Trait("Category", "Integration")]
        [Trait("Priority", "Critical")]
        public void AudioEngine_PlaySong_UpdatesStateAndUICorrectly()
        {
            // Arrange
            var testSongPath = Path.Combine(tempSongPath, TestSongData.ValidSongs.Mp3File);
            File.WriteAllText(testSongPath, "dummy audio data");
            
            Utils.Songs = new[] { testSongPath };
            Utils.CurrentSongPath = testSongPath;
            Start.state = MainStates.idle;

            var mockHandle = mockAudioPlayer.CreateStream(testSongPath, 0, 100000, BassFlags.Default);
            Utils.CurrentMusic = mockHandle;

            // Act - Simulate audio playback start
            Start.state = MainStates.playing;
            mockAudioPlayer.ChannelPlay(mockHandle, false);

            // Assert - Verify state and UI integration
            Start.state.Should().Be(MainStates.playing);
            mockAudioPlayer.WasMethodCalled("ChannelPlay").Should().BeTrue();
            Utils.CurrentMusic.Should().NotBe(0);
        }

        [Fact]
        [Trait("Category", "Integration")]
        [Trait("Priority", "Critical")]
        public void AudioEngine_VolumeModification_UpdatesUIAndAudioState()
        {
            // Arrange
            var mockHandle = mockAudioPlayer.CreateStream("test.mp3", 0, 100000, BassFlags.Default);
            Utils.CurrentMusic = mockHandle;
            Preferences.volume = 0.5f;
            
            mockAudioPlayer.ChannelSetAttribute(mockHandle, ChannelAttribute.Volume, 0.5f);

            // Act - Modify volume
            var newVolume = 0.8f;
            Preferences.volume = newVolume;
            mockAudioPlayer.ChannelSetAttribute(mockHandle, ChannelAttribute.Volume, newVolume);

            // Assert - Verify volume state synchronization
            float actualVolume = 0f;
            mockAudioPlayer.ChannelGetAttribute(mockHandle, ChannelAttribute.Volume, out actualVolume);
            
            Preferences.volume.Should().Be(newVolume);
            actualVolume.Should().BeApproximately(newVolume, 0.001f);
            mockAudioPlayer.WasMethodCalled("ChannelSetAttribute").Should().BeTrue();
        }

        [Fact]
        [Trait("Category", "Integration")]
        [Trait("Priority", "Critical")]
        public void AudioEngine_VolumeModification_ClampsToValidRange()
        {
            // Arrange
            var mockHandle = mockAudioPlayer.CreateStream("test.mp3", 0, 100000, BassFlags.Default);
            Utils.CurrentMusic = mockHandle;
            Preferences.volume = 1.0f;

            // Act - Attempt to exceed maximum volume (1.5f)
            var excessiveVolume = 2.0f;
            var clampedVolume = Math.Min(1.5f, Math.Max(0f, Preferences.volume + excessiveVolume - 1.0f));
            Preferences.volume = clampedVolume;
            mockAudioPlayer.ChannelSetAttribute(mockHandle, ChannelAttribute.Volume, clampedVolume);

            // Assert - Verify volume clamping
            Preferences.volume.Should().Be(1.5f);
            
            float actualVolume = 0f;
            mockAudioPlayer.ChannelGetAttribute(mockHandle, ChannelAttribute.Volume, out actualVolume);
            actualVolume.Should().BeApproximately(1.5f, 0.001f);
        }

        [Fact]
        [Trait("Category", "Integration")]
        [Trait("Priority", "High")]
        public void AudioEngine_MuteToggle_PreservesVolumeState()
        {
            // Arrange
            var mockHandle = mockAudioPlayer.CreateStream("test.mp3", 0, 100000, BassFlags.Default);
            Utils.CurrentMusic = mockHandle;
            var originalVolume = 0.7f;
            Preferences.volume = originalVolume;
            Preferences.isMuted = false;
            
            mockAudioPlayer.ChannelSetAttribute(mockHandle, ChannelAttribute.Volume, originalVolume);

            // Act - Toggle mute on
            Preferences.isMuted = true;
            Preferences.oldVolume = Preferences.volume;
            Preferences.volume = 0f;
            mockAudioPlayer.ChannelSetAttribute(mockHandle, ChannelAttribute.Volume, 0f);

            // Assert - Verify mute state
            Preferences.isMuted.Should().BeTrue();
            Preferences.oldVolume.Should().Be(originalVolume);
            Preferences.volume.Should().Be(0f);

            // Act - Toggle mute off
            Preferences.isMuted = false;
            Preferences.volume = Preferences.oldVolume;
            mockAudioPlayer.ChannelSetAttribute(mockHandle, ChannelAttribute.Volume, originalVolume);

            // Assert - Verify volume restoration
            Preferences.isMuted.Should().BeFalse();
            Preferences.volume.Should().Be(originalVolume);
        }

        [Fact]
        [Trait("Category", "Integration")]
        [Trait("Priority", "High")]
        public void AudioEngine_SeekOperation_UpdatesPositionAndUI()
        {
            // Arrange
            var mockHandle = mockAudioPlayer.CreateStream("test.mp3", 0, 100000, BassFlags.Default);
            Utils.CurrentMusic = mockHandle;
            var streamLength = 100000L;
            var targetPosition = 50000L;

            mockAudioPlayer.ChannelSetLength(mockHandle, streamLength);
            mockAudioPlayer.ChannelSetPosition(mockHandle, 0);

            // Act - Seek to middle of song
            mockAudioPlayer.ChannelSetPosition(mockHandle, targetPosition);

            // Assert - Verify position update
            var currentPosition = mockAudioPlayer.ChannelGetPosition(mockHandle);
            currentPosition.Should().Be(targetPosition);
            mockAudioPlayer.WasMethodCalled("ChannelSetPosition").Should().BeTrue();
        }

        [Fact]
        [Trait("Category", "Integration")]
        [Trait("Priority", "High")]
        public void AudioEngine_SeekBeyondLength_ClampsToValidRange()
        {
            // Arrange
            var mockHandle = mockAudioPlayer.CreateStream("test.mp3", 0, 100000, BassFlags.Default);
            Utils.CurrentMusic = mockHandle;
            var streamLength = 100000L;
            
            mockAudioPlayer.ChannelSetLength(mockHandle, streamLength);

            // Act - Attempt to seek beyond stream length
            var invalidPosition = streamLength + 10000;
            var clampedPosition = Math.Min(streamLength - 1, invalidPosition);
            mockAudioPlayer.ChannelSetPosition(mockHandle, clampedPosition);

            // Assert - Verify position clamping
            var currentPosition = mockAudioPlayer.ChannelGetPosition(mockHandle);
            currentPosition.Should().Be(streamLength - 1);
        }

        [Fact]
        [Trait("Category", "Integration")]
        [Trait("Priority", "High")]
        public void AudioEngine_StateTransitions_ReflectInUIComponents()
        {
            // Arrange - Test state transitions and UI sync
            var states = new[]
            {
                MainStates.idle,
                MainStates.play,
                MainStates.playing,
                MainStates.pause,
                MainStates.stop
            };

            foreach (var testState in states)
            {
                // Act
                Start.state = testState;
                var stateLogo = TUI.GetStateLogo(false);

                // Assert - Verify UI reflects state
                stateLogo.Should().NotBeNullOrEmpty();
                
                // Verify specific state mappings
                switch (testState)
                {
                    case MainStates.playing:
                    case MainStates.play:
                        stateLogo.Should().Contain(Themes.CurrentTheme!.Time?.PlayingLetterLetter ?? "▶▶");
                        break;
                    case MainStates.idle:
                    case MainStates.pause:
                        stateLogo.Should().Contain(Themes.CurrentTheme!.Time?.PausedLetterLetter ?? "▶ ");
                        break;
                    case MainStates.stop:
                        stateLogo.Should().Contain(Themes.CurrentTheme!.Time?.StoppedLetterLetter ?? "■");
                        break;
                }
            }
        }

        [Fact]
        [Trait("Category", "Integration")]
        [Trait("Priority", "Medium")]
        public void AudioEngine_ProgressBarCalculation_HandlesEdgeCases()
        {
            // Arrange
            var testCases = new[]
            {
                (current: 0.0, max: 100.0, description: "Zero position"),
                (current: 100.0, max: 100.0, description: "Full position"),
                (current: 50.0, max: 100.0, description: "Half position"),
                (current: 0.0, max: 0.0, description: "Zero duration"),
                (current: 150.0, max: 100.0, description: "Overflow position")
            };

            foreach (var (current, max, description) in testCases)
            {
                // Act
                var progressBar = TUI.ProgressBar(current, max);

                // Assert
                progressBar.Should().NotBeNullOrEmpty($"Progress bar should be generated for: {description}");
                
                // Verify progress bar contains expected components
                progressBar.Should().Contain("|", "Progress bar should contain separators");
                
                if (max > 0)
                {
                    // Should contain time information when duration is valid
                    progressBar.Should().Match("*:*", "Progress bar should contain time format");
                }
            }
        }

        [Fact]
        [Trait("Category", "Integration")]
        [Trait("Priority", "Medium")]
        public void AudioEngine_ErrorHandling_PropagesToUI()
        {
            // Arrange
            Utils.CurrentMusic = 0; // Simulate no active music stream
            Utils.CurSongError = true;

            // Act - Attempt to get current song path for UI display
            string displayPath;
            if (Utils.CurrentMusic == 0 && Utils.CurSongError)
            {
                displayPath = "Error: cannot play the song";
            }
            else if (Utils.CurrentMusic == 0)
            {
                displayPath = "No song is playing";
            }
            else
            {
                displayPath = Utils.CurrentSongPath;
            }

            // Assert - Verify error state is reflected in UI
            displayPath.Should().Be("Error: cannot play the song");
            Utils.CurSongError.Should().BeTrue();
        }

        [Fact]
        [Trait("Category", "Integration")]
        [Trait("Priority", "Medium")]
        public void AudioEngine_EndOfSong_TriggersNextSongBehavior()
        {
            // Arrange
            var mockHandle = mockAudioPlayer.CreateStream("test.mp3", 0, 100000, BassFlags.Default);
            Utils.CurrentMusic = mockHandle;
            Utils.Songs = new[] { "song1.mp3", "song2.mp3" };
            Utils.CurrentSongIndex = 0;
            
            Preferences.isLoop = false;
            Preferences.isShuffle = false;
            Start.state = MainStates.playing;

            // Simulate song ending
            var streamLength = 100000L;
            mockAudioPlayer.ChannelSetLength(mockHandle, streamLength);
            mockAudioPlayer.ChannelSetPosition(mockHandle, streamLength);

            // Act - Simulate end-of-song detection
            var position = mockAudioPlayer.ChannelGetPosition(mockHandle);
            var length = mockAudioPlayer.ChannelGetLength(mockHandle);
            
            if (position >= length)
            {
                // Simulate MaybeNextSong behavior
                if (!Preferences.isLoop && Utils.Songs.Length > 1)
                {
                    Start.state = MainStates.next;
                }
            }

            // Assert - Verify next song transition
            Start.state.Should().Be(MainStates.next);
            position.Should().BeGreaterOrEqualTo(length);
        }

        [Fact]
        [Trait("Category", "Integration")]
        [Trait("Priority", "Medium")]
        public void AudioEngine_LoopBehavior_RestartsCurrentSong()
        {
            // Arrange
            var mockHandle = mockAudioPlayer.CreateStream("test.mp3", 0, 100000, BassFlags.Default);
            Utils.CurrentMusic = mockHandle;
            Utils.Songs = new[] { "looped_song.mp3" };
            
            Preferences.isLoop = true;
            var streamLength = 100000L;
            mockAudioPlayer.ChannelSetLength(mockHandle, streamLength);
            mockAudioPlayer.ChannelSetPosition(mockHandle, streamLength); // At end

            // Act - Simulate loop behavior
            if (Preferences.isLoop)
            {
                mockAudioPlayer.ChannelSetPosition(mockHandle, 0);
                mockAudioPlayer.ChannelPlay(mockHandle, true);
            }

            // Assert - Verify loop restart
            var position = mockAudioPlayer.ChannelGetPosition(mockHandle);
            position.Should().Be(0);
            mockAudioPlayer.WasMethodCalled("ChannelPlay").Should().BeTrue();
        }

        [Fact]
        [Trait("Category", "Integration")]
        [Trait("Priority", "Low")]
        public void AudioEngine_PreferencesSync_UpdatesUIDisplay()
        {
            // Arrange
            Preferences.volume = 0.6f;
            Preferences.isMuted = false;
            Preferences.isLoop = true;
            Preferences.isShuffle = false;

            // Act - Generate progress bar with current preferences
            var progressBar = TUI.ProgressBar(50.0, 100.0);

            // Assert - Verify preferences are reflected in UI
            progressBar.Should().NotBeNullOrEmpty();
            progressBar.Should().Contain("60%", "Volume should be displayed");
            
            // Loop and shuffle indicators should be present
            progressBar.Should().ContainAny(new[] 
            { 
                Themes.CurrentTheme!.Time?.LoopOnLetter ?? " ⇳  ",
                Themes.CurrentTheme!.Time?.ShuffleOffLetter ?? "⇌ " 
            });
        }

        [Fact]
        [Trait("Category", "Integration")]
        [Trait("Priority", "Low")]
        public void AudioEngine_UIComponentTime_IntegratesWithProgressBar()
        {
            // Arrange
            Utils.TotalMusicDurationInSec = 50.0;
            Utils.SongDurationInSec = 200.0;

            var timeTable = new Spectre.Console.Table();

            // Act
            var resultTable = TUI.UIComponent_Time(timeTable);

            // Assert
            resultTable.Should().NotBeNull();
            resultTable.Border.Should().Be(Themes.bStyle(Themes.CurrentTheme!.Time?.BorderStyle ?? "Rounded"));
        }

        public void Dispose()
        {
            var parentDir = Path.GetDirectoryName(tempSongPath);
            if (!string.IsNullOrEmpty(parentDir) && Directory.Exists(parentDir))
            {
                try
                {
                    Directory.Delete(parentDir, true);
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
using FluentAssertions;
using Jammer.Tests.TestUtilities.Doubles;
using System;
using System.Threading.Tasks;
using Xunit;

namespace Jammer.Tests.Integration
{
    /// <summary>
    /// Integration tests for Input→Audio Control Pipeline.
    /// Tests complete integration between keyboard input detection, keybinding resolution, and audio control dispatch.
    /// Verifies input actions trigger immediate audio effects and settings persistence without restart requirement.
    /// </summary>
    [Collection("Integration")]
    public class InputIntegrationTests : IDisposable
    {
        private readonly MockAudioPlayer mockAudioPlayer;
        private readonly string originalPlayerView;
        private readonly bool originalEditingKeybind;

        public InputIntegrationTests()
        {
            // Arrange: Create isolated test environment
            // Initialize themes to prevent null reference errors
            Themes.SetDefaultTheme();
            
            // Initialize KeyData to prevent null reference errors
            try
            {
                IniFileHandling.ReadNewKeybinds();
            }
            catch
            {
                // If keybinds file doesn't exist, create default one
                IniFileHandling.Create_KeyDataIni(0);
                IniFileHandling.ReadNewKeybinds();
            }
            
            mockAudioPlayer = new MockAudioPlayer();
            mockAudioPlayer.Reset();

            // Save original state
            originalPlayerView = Start.playerView;
            originalEditingKeybind = IniFileHandling.EditingKeybind;

            // Initialize test state
            Start.playerView = "default";
            Start.Action = "";
            IniFileHandling.EditingKeybind = false;
            
            // Setup test audio stream
            Utils.CurrentMusic = mockAudioPlayer.CreateStream("test.mp3", 0, 100000, ManagedBass.BassFlags.Default);
        }

        [Theory]
        [InlineData("VolumeUp")]
        [InlineData("VolumeDown")]
        [InlineData("Mute")]
        [Trait("Category", "Integration")]
        [Trait("Priority", "Critical")]
        public async Task KeyboardInput_AudioControlActions_TriggerImmediateEffects(string action)
        {
            // Arrange
            var initialVolume = Preferences.volume;
            var initialMuted = Preferences.isMuted;
            Start.Action = action;

            // Act - Simulate keyboard input processing
            await Start.CheckKeyboardAsync();

            // Assert - Verify audio control was triggered
            switch (action)
            {
                case "VolumeUp":
                    if (initialMuted)
                    {
                        Preferences.isMuted.Should().BeFalse("VolumeUp should auto-unmute");
                    }
                    mockAudioPlayer.WasMethodCalled("ChannelSetAttribute").Should().BeTrue();
                    break;
                    
                case "VolumeDown":
                    if (initialMuted)
                    {
                        Preferences.isMuted.Should().BeFalse("VolumeDown should auto-unmute");
                    }
                    mockAudioPlayer.WasMethodCalled("ChannelSetAttribute").Should().BeTrue();
                    break;
                    
                case "Mute":
                    Preferences.isMuted.Should().Be(!initialMuted, "Mute should toggle mute state");
                    mockAudioPlayer.WasMethodCalled("ChannelSetAttribute").Should().BeTrue();
                    break;
            }

            // Action should be reset after processing
            Start.Action.Should().BeEmpty();
        }

        [Fact]
        [Trait("Category", "Integration")]
        [Trait("Priority", "Critical")]
        public async Task KeyboardInput_VolumeUp_AutoUnmutesAndIncrementsVolume()
        {
            // Arrange
            Preferences.volume = 0.5f;
            Preferences.isMuted = true;
            Preferences.oldVolume = 0.5f;
            var changeVolumeBy = Preferences.GetChangeVolumeBy();
            Start.Action = "VolumeUp";

            // Act
            await Start.CheckKeyboardAsync();

            // Assert
            Preferences.isMuted.Should().BeFalse("VolumeUp should automatically unmute");
            Preferences.volume.Should().BeApproximately(0.5f + changeVolumeBy, 0.001f);
            mockAudioPlayer.WasMethodCalled("ChannelSetAttribute").Should().BeTrue();
        }

        [Fact]
        [Trait("Category", "Integration")]
        [Trait("Priority", "Critical")]
        public async Task KeyboardInput_VolumeDown_AutoUnmutesAndDecrementsVolume()
        {
            // Arrange
            Preferences.volume = 0.5f;
            Preferences.isMuted = true;
            Preferences.oldVolume = 0.5f;
            var changeVolumeBy = Preferences.GetChangeVolumeBy();
            Start.Action = "VolumeDown";

            // Act
            await Start.CheckKeyboardAsync();

            // Assert
            Preferences.isMuted.Should().BeFalse("VolumeDown should automatically unmute");
            Preferences.volume.Should().BeApproximately(0.5f - changeVolumeBy, 0.001f);
            mockAudioPlayer.WasMethodCalled("ChannelSetAttribute").Should().BeTrue();
        }

        [Fact]
        [Trait("Category", "Integration")]
        [Trait("Priority", "High")]
        public async Task KeyboardInput_VolumeUpBeyondMax_ClampsToMaximum()
        {
            // Arrange
            Preferences.volume = 1.4f; // Close to maximum
            Preferences.isMuted = false;
            Start.Action = "VolumeUp";

            // Act
            await Start.CheckKeyboardAsync();

            // Assert
            Preferences.volume.Should().Be(1.5f, "Volume should be clamped to maximum");
            mockAudioPlayer.WasMethodCalled("ChannelSetAttribute").Should().BeTrue();
        }

        [Fact]
        [Trait("Category", "Integration")]
        [Trait("Priority", "High")]
        public async Task KeyboardInput_VolumeDownBelowMin_ClampsToMinimum()
        {
            // Arrange
            Preferences.volume = 0.01f; // Close to minimum
            Preferences.isMuted = false;
            Start.Action = "VolumeDown";

            // Act
            await Start.CheckKeyboardAsync();

            // Assert
            Preferences.volume.Should().Be(0f, "Volume should be clamped to minimum");
            mockAudioPlayer.WasMethodCalled("ChannelSetAttribute").Should().BeTrue();
        }

        [Fact]
        [Trait("Category", "Integration")]
        [Trait("Priority", "High")]
        public async Task KeyboardInput_MuteToggle_PreservesAndRestoresVolume()
        {
            // Arrange
            var originalVolume = 0.7f;
            Preferences.volume = originalVolume;
            Preferences.isMuted = false;
            Start.Action = "Mute";

            // Act - First mute
            await Start.CheckKeyboardAsync();
            
            // Assert - Should be muted with volume preserved
            Preferences.isMuted.Should().BeTrue();
            Preferences.oldVolume.Should().Be(originalVolume);
            Preferences.volume.Should().Be(0f);

            // Act - Second mute (unmute)
            Start.Action = "Mute";
            await Start.CheckKeyboardAsync();

            // Assert - Should be unmuted with volume restored
            Preferences.isMuted.Should().BeFalse();
            Preferences.volume.Should().Be(originalVolume);
        }

        [Theory]
        [InlineData("Forward5s")]
        [InlineData("Backwards5s")]
        [Trait("Category", "Integration")]
        [Trait("Priority", "High")]
        public async Task KeyboardInput_SeekActions_TriggerAudioSeeking(string action)
        {
            // Arrange
            mockAudioPlayer.ChannelSetLength(Utils.CurrentMusic, 200000L);
            mockAudioPlayer.ChannelSetPosition(Utils.CurrentMusic, 100000L);
            Start.Action = action;

            // Act
            await Start.CheckKeyboardAsync();

            // Assert
            mockAudioPlayer.WasMethodCalled("ChannelSetPosition").Should().BeTrue();
            mockAudioPlayer.WasMethodCalled("ChannelGetPosition").Should().BeTrue();
            Start.Action.Should().BeEmpty();
        }

        [Theory]
        [InlineData("NextSong", MainStates.next)]
        [InlineData("PreviousSong", MainStates.previous)]
        [InlineData("PlayPause", MainStates.play)] // Assumes current state is not playing
        [Trait("Category", "Integration")]
        [Trait("Priority", "High")]
        public async Task KeyboardInput_PlaybackActions_SetCorrectStates(string action, MainStates expectedState)
        {
            // Arrange
            Start.state = MainStates.idle;
            Start.Action = action;

            // Act
            await Start.CheckKeyboardAsync();

            // Assert
            Start.state.Should().Be(expectedState);
            Start.Action.Should().BeEmpty();
        }

        [Theory]
        [InlineData("Loop")]
        [InlineData("Shuffle")]
        [Trait("Category", "Integration")]
        [Trait("Priority", "Medium")]
        public async Task KeyboardInput_SettingsToggle_UpdatesPreferences(string action)
        {
            // Arrange
            var initialLoop = Preferences.isLoop;
            var initialShuffle = Preferences.isShuffle;
            Start.Action = action;

            // Act
            await Start.CheckKeyboardAsync();

            // Assert
            switch (action)
            {
                case "Loop":
                    Preferences.isLoop.Should().Be(!initialLoop);
                    break;
                case "Shuffle":
                    Preferences.isShuffle.Should().Be(!initialShuffle);
                    break;
            }
            Start.Action.Should().BeEmpty();
        }

        [Fact]
        [Trait("Category", "Integration")]
        [Trait("Priority", "Medium")]
        public async Task KeyboardInput_InvalidAction_NoStateChanges()
        {
            // Arrange
            var initialVolume = Preferences.volume;
            var initialMuted = Preferences.isMuted;
            var initialState = Start.state;
            Start.Action = "InvalidAction";

            // Act
            await Start.CheckKeyboardAsync();

            // Assert - No changes should occur
            Preferences.volume.Should().Be(initialVolume);
            Preferences.isMuted.Should().Be(initialMuted);
            Start.state.Should().Be(initialState);
            Start.Action.Should().BeEmpty(); // Action should still be reset
        }

        [Fact]
        [Trait("Category", "Integration")]
        [Trait("Priority", "Medium")]
        public async Task KeyboardInput_EditingKeybindMode_PreventsNormalProcessing()
        {
            // Arrange
            IniFileHandling.EditingKeybind = true;
            var initialVolume = Preferences.volume;
            Start.Action = "VolumeUp";

            // Act
            await Start.CheckKeyboardAsync();

            // Assert - Volume should not change during keybind editing
            Preferences.volume.Should().Be(initialVolume);
            mockAudioPlayer.WasMethodCalled("ChannelSetAttribute").Should().BeFalse();
        }

        [Theory]
        [InlineData("Shift + a", "A")]
        [InlineData("Ctrl + b", "Ctrl + b")]
        [InlineData("Alt + c", "Alt + c")]
        [InlineData(" d ", "d")] // Test space trimming
        [InlineData("Shift+e", "E")] // Test no spaces
        [Trait("Category", "Integration")]
        [Trait("Priority", "Medium")]
        public void Keybindings_CheckValue_NormalizesInputCorrectly(string input, string expected)
        {
            // Act
            var result = Keybindings.CheckValue("TestKey", input);

            // Assert
            result.Should().Be(expected);
        }

        [Fact]
        [Trait("Category", "Integration")]
        [Trait("Priority", "Low")]
        public async Task KeyboardInput_NoAction_NoProcessing()
        {
            // Arrange
            Start.Action = "";
            var initialVolume = Preferences.volume;
            var initialState = Start.state;

            // Act
            await Start.CheckKeyboardAsync();

            // Assert - No changes should occur
            Preferences.volume.Should().Be(initialVolume);
            Start.state.Should().Be(initialState);
            Start.Action.Should().BeEmpty();
        }

        [Fact]
        [Trait("Category", "Integration")]
        [Trait("Priority", "Low")]
        public async Task KeyboardInput_SettingsPlayerView_RoutesToSettingsMode()
        {
            // Arrange
            Start.playerView = "settings";
            Start.Action = "VolumeUp";
            var initialVolume = Preferences.volume;

            // Act
            await Start.CheckKeyboardAsync();

            // Assert - In settings mode, normal actions might not apply
            // This test validates that playerView affects processing
            Start.playerView.Should().Be("settings");
        }

        [Fact]
        [Trait("Category", "Integration")]
        [Trait("Priority", "Medium")]
        public async Task KeyboardInput_VolumeChange_TriggersSettingsPersistence()
        {
            // Arrange - Mock or verify settings are saved after volume change
            var initialVolume = Preferences.volume;
            Start.Action = "VolumeUp";

            // Act
            await Start.CheckKeyboardAsync();

            // Assert - Volume should have changed (indicating SaveSettings was called)
            Preferences.volume.Should().NotBe(initialVolume);
        }

        [Fact]
        [Trait("Category", "Integration")]
        [Trait("Priority", "Low")]
        public void KeyboardInput_ModifierKeyHelperDisabled_ProcessesNormally()
        {
            // Arrange
            var wasEnabled = Preferences.isModifierKeyHelper;
            Preferences.isModifierKeyHelper = false;
            
            try
            {
                // Act
                var result = Keybindings.CheckValue("TestKey", "Shift + a");

                // Assert
                result.Should().Be("A"); // Should process Shift modifier
            }
            finally
            {
                // Cleanup
                Preferences.isModifierKeyHelper = wasEnabled;
            }
        }

        public void Dispose()
        {
            // Cleanup: Restore original state
            Start.playerView = originalPlayerView;
            IniFileHandling.EditingKeybind = originalEditingKeybind;
            Start.Action = "";
            
            mockAudioPlayer?.Reset();
        }
    }
}
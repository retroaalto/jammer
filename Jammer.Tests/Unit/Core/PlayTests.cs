using FluentAssertions;
using Jammer.Tests.TestUtilities.Doubles;
using Xunit;

namespace Jammer.Tests.Unit.Core
{
    /// <summary>
    /// Tests for Play.cs service layer functionality  
    /// Phase 3: Service Layer Testing - Audio playback and BASS library integration
    /// 
    /// ARCHITECTURAL NOTES:
    /// - Play.cs is a 942-line monolithic class with 29 methods
    /// - Heavy static method usage prevents proper dependency injection
    /// - Direct BASS library calls throughout (needs abstraction)
    /// - Complex audio format detection and channel management
    /// - Mixed concerns: playbook control, effects, volume, seeking, playlist navigation
    /// 
    /// TESTING APPROACH:
    /// - Focus on testable pure functions first  
    /// - Document architectural challenges for future refactoring
    /// - Create abstractions for BASS library operations
    /// - Use mocks to simulate audio operations without real hardware
    /// </summary>
    public class PlayTests
    {
        private readonly MockAudioPlayer _mockAudioPlayer;

        public PlayTests()
        {
            _mockAudioPlayer = new MockAudioPlayer();
        }

        [Fact]
        public void Play_Class_ShouldExist()
        {
            // Arrange & Act
            var playType = typeof(Play);
            
            // Assert
            playType.Should().NotBeNull();
            playType.IsClass.Should().BeTrue();
        }

        [Fact]
        public void isValidExtension_WithValidMp3Extension_ShouldReturnTrue()
        {
            // Arrange
            var testExtension = ".mp3";
            var songExtensions = new[] { ".mp3", ".ogg", ".wav", ".flac" };
            
            // Act
            var result = Play.isValidExtension(testExtension, songExtensions);
            
            // Assert
            result.Should().BeTrue("MP3 should be a valid audio extension");
        }

        [Fact]
        public void isValidExtension_WithInvalidExtension_ShouldReturnFalse()
        {
            // Arrange
            var testExtension = ".txt";
            var songExtensions = new[] { ".mp3", ".ogg", ".wav", ".flac" };
            
            // Act
            var result = Play.isValidExtension(testExtension, songExtensions);
            
            // Assert
            result.Should().BeFalse("TXT should not be a valid audio extension");
        }

        [Fact]
        public void isValidExtension_WithEmptyExtension_ShouldReturnFalse()
        {
            // Arrange
            var testExtension = "";
            var songExtensions = new[] { ".mp3", ".ogg", ".wav", ".flac" };
            
            // Act
            var result = Play.isValidExtension(testExtension, songExtensions);
            
            // Assert
            result.Should().BeFalse("Empty extension should not be valid");
        }

        [Fact]
        public void isValidExtension_WithNullExtension_ShouldHandleGracefully()
        {
            // Arrange
            string? testExtension = null;
            var songExtensions = new[] { ".mp3", ".ogg", ".wav", ".flac" };
            
            // Act & Assert
            var exception = Record.Exception(() => Play.isValidExtension(testExtension!, songExtensions));
            
            // The method should either handle null gracefully or throw a meaningful exception
            if (exception != null)
            {
                exception.Should().BeOfType<ArgumentNullException>("Should throw ArgumentNullException for null input");
            }
        }

        [Fact]
        public void isValidExtension_WithCaseVariations_ShouldHandleConsistently()
        {
            // Arrange
            var songExtensions = new[] { ".mp3", ".ogg", ".wav", ".flac" };
            var testCases = new[]
            {
                (".MP3", "Upper case extension"),
                (".Mp3", "Mixed case extension"),
                (".mp3", "Lower case extension")
            };
            
            // Act & Assert
            foreach (var (extension, description) in testCases)
            {
                var result = Play.isValidExtension(extension, songExtensions);
                
                // Document current behavior - method appears to be case-sensitive
                // Only .mp3 (lowercase) matches the array, uppercase variations don't match
                if (extension == ".mp3")
                {
                    result.Should().BeTrue($"{description} should be handled properly");
                }
                else
                {
                    // Upper/mixed case don't match - this is the current behavior
                    result.Should().BeFalse($"{description} - method is case-sensitive");
                }
            }
        }

        [Fact]
        public void EmptySpaces_WithStringContainingOnlySpaces_ShouldReturnTrue()
        {
            // Arrange
            var testString = "   ";
            
            // Act
            var result = Play.EmptySpaces(testString);
            
            // Assert
            result.Should().BeTrue("String with only spaces should return true");
        }

        [Fact]
        public void EmptySpaces_WithStringContainingNonSpaceCharacters_ShouldReturnFalse()
        {
            // Arrange
            var testString = " a ";
            
            // Act
            var result = Play.EmptySpaces(testString);
            
            // Assert
            result.Should().BeFalse("String with non-space characters should return false");
        }

        [Fact]
        public void EmptySpaces_WithEmptyString_ShouldReturnTrue()
        {
            // Arrange
            var testString = "";
            
            // Act
            var result = Play.EmptySpaces(testString);
            
            // Assert
            result.Should().BeTrue("Empty string should return true");
        }

        [Fact]
        public void Play_AudioExtensionArrays_ShouldBeDefinedCorrectly()
        {
            // Test that the static extension arrays are properly defined
            
            // Act & Assert
            Play.songExtensions.Should().NotBeNull("songExtensions should be defined");
            Play.songExtensions.Should().NotBeEmpty("songExtensions should not be empty");
            Play.songExtensions.Should().Contain(".mp3", "songExtensions should include MP3");
            Play.songExtensions.Should().Contain(".ogg", "songExtensions should include OGG");
            
            Play.aacExtensions.Should().NotBeNull("aacExtensions should be defined");
            Play.aacExtensions.Should().Contain(".aac", "aacExtensions should include AAC");
            Play.aacExtensions.Should().Contain(".m4a", "aacExtensions should include M4A");
            
            Play.mp4Extensions.Should().NotBeNull("mp4Extensions should be defined");
            Play.mp4Extensions.Should().Contain(".mp4", "mp4Extensions should include MP4");
            
            Play.midiExtensions.Should().NotBeNull("midiExtensions should be defined");
            Play.midiExtensions.Should().Contain(".mid", "midiExtensions should include MIDI");
            Play.midiExtensions.Should().Contain(".midi", "midiExtensions should include MIDI variant");
        }

        [Fact]
        public void Play_Architecture_HasExpectedMethods()
        {
            // Document the method structure for testing strategy
            
            var playType = typeof(Play);
            var methods = playType.GetMethods(System.Reflection.BindingFlags.Public | System.Reflection.BindingFlags.Static);
            var methodNames = methods.Select(m => m.Name).Distinct().ToArray();
            
            // Key methods identified in architectural analysis
            var expectedMethods = new[]
            {
                "PlaySong", "StartPlaying", "PauseSong", "ResumeSong", "StopSong",
                "NextSong", "PrevSong", "RandomSong", "SeekSong", "SetVolume", "ModifyVolume",
                "ToggleMute", "SetFXs", "isValidExtension", "EmptySpaces", "AddSong", "DeleteSong"
            };
            
            foreach (var expectedMethod in expectedMethods)
            {
                methodNames.Should().Contain(expectedMethod, 
                    $"Play should have {expectedMethod} method for audio control");
            }
        }

        [Fact]
        public void Play_TestingChallenges_Documentation()
        {
            // This test documents the architectural challenges for comprehensive testing
            
            // CURRENT TESTING CHALLENGES:
            // 1. All methods are static - prevents dependency injection
            // 2. Direct BASS library calls throughout - no abstraction layer
            // 3. Global state dependencies (Utils.CurrentMusic, Utils.Songs, etc.)
            // 4. UI dependencies (TUI calls) mixed with business logic
            // 5. File system operations without abstraction
            // 6. External service calls (Download.* methods) directly invoked
            // 7. Complex state machine interactions (Start.state, MainStates)
            
            // METHODS THAT ARE TESTABLE (pure functions):
            // - isValidExtension(string, string[]) - pure string comparison
            // - EmptySpaces(string) - pure string validation
            // - Extension array definitions - static data
            
            // METHODS THAT NEED MAJOR REFACTORING FOR TESTING:
            // - StartPlaying() - 100+ lines, multiple BASS library calls, error handling
            // - PlaySong(string[], int) - 200+ lines, file I/O, downloads, format detection
            // - SetFXs() - 120+ lines, complex audio effects configuration  
            // - All audio control methods (pause, resume, seek, volume)
            // - Playlist management methods (add, delete, shuffle)
            
            // RECOMMENDED ABSTRACTIONS FOR BETTER TESTING:
            // 1. IAudioPlayer interface for BASS library operations
            // 2. IPlaylistManager for song array operations
            // 3. IAudioEffects for effects processing  
            // 4. IFileSystemService for file operations
            // 5. Separate pure business logic from external dependencies
            // 6. State management through dependency injection
            // 7. Event-driven architecture for UI updates
            
            var playType = typeof(Play);
            playType.Should().NotBeNull("Play class exists for testing analysis");
            
            // Count static methods to quantify testing challenge
            var staticMethods = playType.GetMethods(System.Reflection.BindingFlags.Public | System.Reflection.BindingFlags.Static);
            staticMethods.Should().HaveCountGreaterThan(20, 
                "Play has many static methods that present testing challenges");
            
            Assert.True(true, "Documentation test - highlights need for architectural improvements");
        }

        [Fact]
        public void MockAudioPlayer_Integration_ShouldWorkWithPlayAbstractions()
        {
            // This test demonstrates how Play.cs COULD be tested with proper abstractions
            
            // Arrange  
            _mockAudioPlayer.Reset();
            var testFilePath = "/test/audio.mp3";
            
            // Act - Simulate what StartPlaying() would do with abstraction
            var handle = _mockAudioPlayer.CreateStream(testFilePath, 0, 0, ManagedBass.BassFlags.Default);
            var playSuccess = _mockAudioPlayer.ChannelPlay(handle);
            _mockAudioPlayer.ChannelSetAttribute(handle, ManagedBass.ChannelAttribute.Volume, 0.8f);
            
            // Assert
            handle.Should().BeGreaterThan(0, "Stream should be created successfully");
            playSuccess.Should().BeTrue("Stream should start playing");
            
            var stream = _mockAudioPlayer.GetStream(handle);
            stream.Should().NotBeNull("Stream should exist in mock");
            stream!.IsPlaying.Should().BeTrue("Stream should be in playing state");
            stream.Volume.Should().Be(0.8f, "Volume should be set correctly");
            
            _mockAudioPlayer.MethodCallLog.Should().Contain(call => call.Contains("CreateStream"));
            _mockAudioPlayer.MethodCallLog.Should().Contain(call => call.Contains("ChannelPlay"));
            _mockAudioPlayer.MethodCallLog.Should().Contain(call => call.Contains("ChannelSetAttribute"));
        }

        [Fact]
        public void MockAudioPlayer_ErrorHandling_ShouldSimulateFailures()
        {
            // This test shows how error conditions could be tested with proper abstractions
            
            // Arrange
            _mockAudioPlayer.Reset();
            _mockAudioPlayer.ShouldFailStreamCreation = true;
            _mockAudioPlayer.LastErrorValue = ManagedBass.Errors.FileOpen;
            
            // Act
            var handle = _mockAudioPlayer.CreateStream("/nonexistent.mp3", 0, 0, ManagedBass.BassFlags.Default);
            
            // Assert
            handle.Should().Be(0, "Stream creation should fail for invalid file");
            _mockAudioPlayer.LastError.Should().Be(ManagedBass.Errors.FileOpen, "Error should be set appropriately");
            
            // This demonstrates how Play.StartPlaying() COULD handle errors with proper abstraction
        }
    }
}
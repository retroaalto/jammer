# MockAudioPlayer Interface Contract

This document defines the complete interface contract for MockAudioPlayer, documenting all required methods, signatures, expected behavior patterns, and usage examples.

## Overview

The `MockAudioPlayer` class provides a complete mock implementation of the `IAudioPlayer` interface, enabling comprehensive testing of audio functionality without requiring actual BASS library dependencies.

**Location**: `Jammer.Tests/TestUtilities/Doubles/MockAudioPlayer.cs`  
**Interface**: `IAudioPlayer` (defined in `Jammer.Core/src/Interfaces/IAudioPlayer.cs`)

## Interface Contract

### Core Interface Methods

All methods from `IAudioPlayer` interface must be implemented:

#### Stream Creation and Management

```csharp
int CreateStream(string file, long offset, long length, BassFlags flags)
int CreateAacStream(string file, long offset, long length, BassFlags flags)
int CreateMp4Stream(string file, long offset, long length, BassFlags flags)
int CreateOpusStream(string file, long offset, long length, BassFlags flags)  
int CreateMidiStream(string file, long offset, long length, BassFlags flags)
```

**Behavior Contract:**
- Returns unique handle (>0) on success, 0 on failure
- Respects `ShouldFailStreamCreation` test configuration
- Logs method calls with parameters
- Creates `MockAudioStream` with default properties

**Default Stream Properties:**
- Length: 300,000 bytes (5 minutes)
- Position: 0
- Volume: 1.0f
- IsPlaying: false

#### Playback Control

```csharp
bool ChannelPlay(int handle)
bool ChannelPlay(int handle, bool restart)  // Extended overload
bool ChannelPause(int handle)
bool ChannelStop(int handle)
bool StreamFree(int handle)
```

**Behavior Contract:**
- `ChannelPlay(int)`: Starts playback, sets IsPlaying = true
- `ChannelPlay(int, bool)`: If restart=true, resets Position to 0
- `ChannelPause(int)`: Sets IsPlaying = false, preserves position  
- `ChannelStop(int)`: Sets IsPlaying = false, resets Position to 0
- `StreamFree(int)`: Removes stream from internal collection
- All return false for invalid handles or when `ShouldFailPlayback = true`

#### Channel Properties

```csharp
bool ChannelSetAttribute(int handle, ChannelAttribute attribute, float value)
bool ChannelGetAttribute(int handle, ChannelAttribute attribute, out float value)
long ChannelGetPosition(int handle)
bool ChannelSetPosition(int handle, long position)  
long ChannelGetLength(int handle)
void ChannelSetLength(int handle, long length)  // Extended method
```

**Behavior Contract:**
- Attribute operations primarily handle `ChannelAttribute.Volume`
- Position operations clamp to valid range [0, Length]
- Invalid handles return false/0 as appropriate
- `ChannelSetLength` allows test control over stream length

#### Effects and Processing

```csharp
int ChannelSetFX(int handle, EffectType type, int priority)
bool FXSetParameters(int fxHandle, object parameters)
```

**Behavior Contract:**
- `ChannelSetFX`: Returns unique effect handle, creates `MockAudioEffect`
- `FXSetParameters`: Stores parameters in effect object

#### Format Conversion

```csharp
long ChannelSeconds2Bytes(int handle, double seconds)
double ChannelBytes2Seconds(int handle, long bytes)
```

**Behavior Contract:**
- Mock conversion assumes: 44.1kHz * 2 channels * 2 bytes = 176,400 bytes/second
- `Seconds2Bytes`: `(long)(seconds * 176400)`
- `Bytes2Seconds`: `bytes / 176400.0`

#### Sync and Callbacks

```csharp
int ChannelSetSync(int handle, SyncFlags type, long param, Action<int, int, int, IntPtr> callback, IntPtr user)
```

**Behavior Contract:**
- Returns unique sync handle
- Stores callback for potential test triggering (advanced testing)

#### Font Management (MIDI)

```csharp
int FontInit(string file, FontInitFlags flags)
bool FontFree(int handle)  
bool StreamSetFonts(int handle, MidiFont[] fonts, int count)
```

**Behavior Contract:**
- `FontInit`: Returns handle or 0 if `ShouldFailStreamCreation = true`
- Font operations succeed for valid inputs

#### Error Handling

```csharp
Errors LastError { get; }
```

**Behavior Contract:**
- Returns `LastErrorValue` property 
- Configurable via test setup

### Extended Test Methods

Methods beyond `IAudioPlayer` interface for test support:

#### Method Call Verification

```csharp
bool WasMethodCalled(string methodName)
List<string> MethodCallLog { get; }
```

**Usage:**
```csharp
mockPlayer.ChannelPlay(handle);
Assert.True(mockPlayer.WasMethodCalled("ChannelPlay"));

// Check specific parameters
var playCall = mockPlayer.MethodCallLog.First(c => c.Contains("ChannelPlay"));
Assert.Contains("1000", playCall); // Verify handle
```

#### Test Configuration

```csharp
bool ShouldFailStreamCreation { get; set; }
bool ShouldFailPlayback { get; set; }
Errors LastErrorValue { get; set; }
```

**Usage:**
```csharp
mockPlayer.ShouldFailStreamCreation = true;
var handle = mockPlayer.CreateStream("test.mp3", 0, 0, BassFlags.Default);
Assert.Equal(0, handle); // Should fail
```

#### State Management

```csharp
void Reset()
MockAudioStream? GetStream(int handle)
MockAudioEffect? GetEffect(int handle)
```

**Usage:**
```csharp
mockPlayer.Reset(); // Clear all state
var stream = mockPlayer.GetStream(handle); // Access internal state
Assert.NotNull(stream);
Assert.Equal("test.mp3", stream.FilePath);
```

## Data Structures

### MockAudioStream

```csharp
public class MockAudioStream
{
    public string FilePath { get; set; } = "";
    public long Position { get; set; }      // Current playback position
    public long Length { get; set; }        // Total stream length  
    public float Volume { get; set; }       // Volume level [0.0-1.0]
    public bool IsPlaying { get; set; }     // Playback state
}
```

### MockAudioEffect

```csharp
public class MockAudioEffect  
{
    public int ChannelHandle { get; set; }    // Associated channel
    public EffectType EffectType { get; set; } // Effect type
    public int Priority { get; set; }         // Effect priority
    public object? Parameters { get; set; }   // Effect parameters
}
```

## Expected Behavior Patterns

### Stream Lifecycle

```csharp
// 1. Create stream
var handle = mockPlayer.CreateStream("song.mp3", 0, 0, BassFlags.Default);
Assert.True(handle > 0);

// 2. Play stream  
Assert.True(mockPlayer.ChannelPlay(handle));
var stream = mockPlayer.GetStream(handle);
Assert.True(stream!.IsPlaying);

// 3. Control playback
mockPlayer.ChannelPause(handle);
Assert.False(stream.IsPlaying);
Assert.True(stream.Position >= 0); // Position preserved

// 4. Clean up
mockPlayer.StreamFree(handle);
Assert.Null(mockPlayer.GetStream(handle));
```

### Volume Control

```csharp
var handle = mockPlayer.CreateStream("song.mp3", 0, 0, BassFlags.Default);

// Set volume
mockPlayer.ChannelSetAttribute(handle, ChannelAttribute.Volume, 0.8f);

// Verify volume
mockPlayer.ChannelGetAttribute(handle, ChannelAttribute.Volume, out float volume);
Assert.Equal(0.8f, volume);

// Check internal state
var stream = mockPlayer.GetStream(handle);
Assert.Equal(0.8f, stream!.Volume);
```

### Position Management

```csharp
var handle = mockPlayer.CreateStream("song.mp3", 0, 0, BassFlags.Default);

// Seek to position
mockPlayer.ChannelSetPosition(handle, 150000);
Assert.Equal(150000, mockPlayer.ChannelGetPosition(handle));

// Position clamping
mockPlayer.ChannelSetLength(handle, 200000);
mockPlayer.ChannelSetPosition(handle, 300000); // Beyond length
Assert.Equal(200000, mockPlayer.ChannelGetPosition(handle)); // Clamped
```

### Effect Management

```csharp
var handle = mockPlayer.CreateStream("song.mp3", 0, 0, BassFlags.Default);

// Add effect
var fxHandle = mockPlayer.ChannelSetFX(handle, EffectType.Reverb, 0);
Assert.True(fxHandle > 0);

// Configure effect
var reverbParams = new { RoomSize = 0.5f };
Assert.True(mockPlayer.FXSetParameters(fxHandle, reverbParams));

// Verify effect state
var effect = mockPlayer.GetEffect(fxHandle);
Assert.NotNull(effect);
Assert.Equal(handle, effect.ChannelHandle);
Assert.Equal(EffectType.Reverb, effect.EffectType);
```

## Usage Examples

### Basic Test Setup

```csharp
[Fact]
public void AudioEngine_PlaySong_UpdatesState()
{
    // Arrange
    var mockPlayer = new MockAudioPlayer();
    var audioEngine = new AudioEngine(mockPlayer);
    
    // Act
    audioEngine.PlaySong("test.mp3");
    
    // Assert
    Assert.True(mockPlayer.WasMethodCalled("CreateStream"));
    Assert.True(mockPlayer.WasMethodCalled("ChannelPlay"));
}
```

### Failure Simulation

```csharp
[Fact] 
public void AudioEngine_StreamCreationFails_HandlesGracefully()
{
    // Arrange
    var mockPlayer = new MockAudioPlayer();
    mockPlayer.ShouldFailStreamCreation = true;
    var audioEngine = new AudioEngine(mockPlayer);
    
    // Act & Assert
    Assert.False(audioEngine.PlaySong("test.mp3"));
    Assert.True(mockPlayer.WasMethodCalled("CreateStream"));
    Assert.False(mockPlayer.WasMethodCalled("ChannelPlay"));
}
```

### State Verification

```csharp
[Fact]
public void AudioEngine_VolumeChange_UpdatesMockState()
{
    // Arrange  
    var mockPlayer = new MockAudioPlayer();
    var audioEngine = new AudioEngine(mockPlayer);
    var handle = audioEngine.LoadSong("test.mp3");
    
    // Act
    audioEngine.SetVolume(0.7f);
    
    // Assert - via mock
    Assert.True(mockPlayer.WasMethodCalled("ChannelSetAttribute"));
    
    // Assert - via internal state
    var stream = mockPlayer.GetStream(handle);
    Assert.Equal(0.7f, stream!.Volume);
}
```

### Call Parameter Verification

```csharp
[Fact]
public void AudioEngine_SeekToPosition_PassesCorrectParameters()
{
    // Arrange
    var mockPlayer = new MockAudioPlayer();  
    var audioEngine = new AudioEngine(mockPlayer);
    var handle = audioEngine.LoadSong("test.mp3");
    
    // Act
    audioEngine.SeekToPosition(TimeSpan.FromMinutes(2));
    
    // Assert  
    var positionCalls = mockPlayer.MethodCallLog
        .Where(call => call.Contains("ChannelSetPosition"))
        .ToList();
    Assert.Single(positionCalls);
    
    // Verify position calculation (2 minutes = 120 seconds)
    var expectedBytes = mockPlayer.ChannelSeconds2Bytes(handle, 120.0);
    Assert.Contains(expectedBytes.ToString(), positionCalls[0]);
}
```

## Quality Assurance Checklist

When implementing or maintaining MockAudioPlayer:

### Interface Compliance
- [ ] All `IAudioPlayer` methods implemented
- [ ] Method signatures match exactly (parameters, return types, nullability)
- [ ] All method overloads provided (especially `ChannelPlay`)
- [ ] Property getters/setters implemented correctly

### Test Support Methods  
- [ ] `WasMethodCalled(string)` method available
- [ ] `ChannelSetLength(int, long)` method available  
- [ ] `MethodCallLog` property accessible
- [ ] Test configuration properties available

### Behavior Consistency
- [ ] Method calls logged with parameters
- [ ] Failure modes respect configuration flags
- [ ] Internal state updates correctly
- [ ] Handle generation is unique and consistent
- [ ] Position/length clamping enforced

### State Management
- [ ] `Reset()` method clears all state
- [ ] `GetStream(int)` provides state access
- [ ] `GetEffect(int)` provides effect access
- [ ] Collections properly maintained

### Error Handling
- [ ] Invalid handles handled gracefully
- [ ] Configured failures properly simulated
- [ ] LastError property functional

## Maintenance Procedures

### When Updating IAudioPlayer Interface

1. **Add methods to interface**
2. **Update MockAudioPlayer implementation**:
   ```csharp
   public ReturnType NewMethod(Parameters params)
   {
       MethodCallLog.Add($"NewMethod({params})");
       // Implement mock behavior
       return appropriateValue;
   }
   ```
3. **Add corresponding tests**
4. **Update this documentation**
5. **Verify all existing tests still pass**

### When Adding Test Helper Methods

1. **Add method to MockAudioPlayer**:
   ```csharp
   public bool WasMethodCalledWithParams(string methodName, params object[] expectedParams)
   {
       return MethodCallLog.Any(call => call.Contains(methodName) && 
           expectedParams.All(param => call.Contains(param.ToString())));
   }
   ```
2. **Document method in this contract**
3. **Add usage examples**

### When Debugging Mock Issues

1. **Check method call log**:
   ```csharp
   foreach(var call in mockPlayer.MethodCallLog)
       Console.WriteLine(call);
   ```
2. **Verify internal state**:
   ```csharp
   var stream = mockPlayer.GetStream(handle);
   Console.WriteLine($"Position: {stream?.Position}, Playing: {stream?.IsPlaying}");
   ```
3. **Confirm configuration**:
   ```csharp
   Console.WriteLine($"Fail Creation: {mockPlayer.ShouldFailStreamCreation}");
   ```

---

*This interface contract ensures MockAudioPlayer provides comprehensive, consistent mock behavior for all Jammer audio testing scenarios while maintaining compatibility with the production IAudioPlayer interface.*
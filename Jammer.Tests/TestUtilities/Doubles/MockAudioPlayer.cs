using Jammer.Core.Interfaces;
using ManagedBass;
using ManagedBass.Midi;

namespace Jammer.Tests.TestUtilities.Doubles
{
    /// <summary>
    /// Mock implementation of IAudioPlayer for testing
    /// Provides controllable behavior for all BASS audio library operations
    /// </summary>
    public class MockAudioPlayer : IAudioPlayer
    {
        private int _nextHandle = 1000;
        private readonly Dictionary<int, MockAudioStream> _streams = new();
        private readonly Dictionary<int, MockAudioEffect> _effects = new();
        
        // Properties for test configuration
        public bool ShouldFailStreamCreation { get; set; } = false;
        public bool ShouldFailPlayback { get; set; } = false;
        public Errors LastErrorValue { get; set; } = Errors.OK;
        public List<string> MethodCallLog { get; } = new();
        
        public Errors LastError => LastErrorValue;

        // Stream creation methods
        public int CreateStream(string file, long offset, long length, BassFlags flags)
        {
            MethodCallLog.Add($"CreateStream({file}, {offset}, {length}, {flags})");
            
            if (ShouldFailStreamCreation)
                return 0;
                
            var handle = _nextHandle++;
            _streams[handle] = new MockAudioStream
            {
                FilePath = file,
                Position = 0,
                Length = 300000, // 5 minutes default
                Volume = 1.0f,
                IsPlaying = false
            };
            return handle;
        }

        public int CreateAacStream(string file, long offset, long length, BassFlags flags)
        {
            MethodCallLog.Add($"CreateAacStream({file}, {offset}, {length}, {flags})");
            return ShouldFailStreamCreation ? 0 : CreateStream(file, offset, length, flags);
        }

        public int CreateMp4Stream(string file, long offset, long length, BassFlags flags)
        {
            MethodCallLog.Add($"CreateMp4Stream({file}, {offset}, {length}, {flags})");
            return ShouldFailStreamCreation ? 0 : CreateStream(file, offset, length, flags);
        }

        public int CreateOpusStream(string file, long offset, long length, BassFlags flags)
        {
            MethodCallLog.Add($"CreateOpusStream({file}, {offset}, {length}, {flags})");
            return ShouldFailStreamCreation ? 0 : CreateStream(file, offset, length, flags);
        }

        public int CreateMidiStream(string file, long offset, long length, BassFlags flags)
        {
            MethodCallLog.Add($"CreateMidiStream({file}, {offset}, {length}, {flags})");
            return ShouldFailStreamCreation ? 0 : CreateStream(file, offset, length, flags);
        }

        // Playback control
        public bool ChannelPlay(int handle)
        {
            MethodCallLog.Add($"ChannelPlay({handle})");
            
            if (ShouldFailPlayback || !_streams.ContainsKey(handle))
                return false;
                
            _streams[handle].IsPlaying = true;
            return true;
        }

        public bool ChannelPause(int handle)
        {
            MethodCallLog.Add($"ChannelPause({handle})");
            
            if (!_streams.ContainsKey(handle))
                return false;
                
            _streams[handle].IsPlaying = false;
            return true;
        }

        public bool ChannelStop(int handle)
        {
            MethodCallLog.Add($"ChannelStop({handle})");
            
            if (!_streams.ContainsKey(handle))
                return false;
                
            _streams[handle].IsPlaying = false;
            _streams[handle].Position = 0;
            return true;
        }

        public bool StreamFree(int handle)
        {
            MethodCallLog.Add($"StreamFree({handle})");
            return _streams.Remove(handle);
        }

        // Channel properties
        public bool ChannelSetAttribute(int handle, ChannelAttribute attribute, float value)
        {
            MethodCallLog.Add($"ChannelSetAttribute({handle}, {attribute}, {value})");
            
            if (!_streams.ContainsKey(handle))
                return false;
                
            if (attribute == ChannelAttribute.Volume)
                _streams[handle].Volume = value;
                
            return true;
        }

        public bool ChannelGetAttribute(int handle, ChannelAttribute attribute, out float value)
        {
            MethodCallLog.Add($"ChannelGetAttribute({handle}, {attribute})");
            value = 0f;
            
            if (!_streams.ContainsKey(handle))
                return false;
                
            if (attribute == ChannelAttribute.Volume)
                value = _streams[handle].Volume;
                
            return true;
        }

        public long ChannelGetPosition(int handle)
        {
            MethodCallLog.Add($"ChannelGetPosition({handle})");
            return _streams.ContainsKey(handle) ? _streams[handle].Position : 0;
        }

        public bool ChannelSetPosition(int handle, long position)
        {
            MethodCallLog.Add($"ChannelSetPosition({handle}, {position})");
            
            if (!_streams.ContainsKey(handle))
                return false;
                
            _streams[handle].Position = Math.Max(0, Math.Min(position, _streams[handle].Length));
            return true;
        }

        public long ChannelGetLength(int handle)
        {
            MethodCallLog.Add($"ChannelGetLength({handle})");
            return _streams.ContainsKey(handle) ? _streams[handle].Length : 0;
        }

        // Effects and processing
        public int ChannelSetFX(int handle, EffectType type, int priority)
        {
            MethodCallLog.Add($"ChannelSetFX({handle}, {type}, {priority})");
            
            var effectHandle = _nextHandle++;
            _effects[effectHandle] = new MockAudioEffect
            {
                ChannelHandle = handle,
                EffectType = type,
                Priority = priority
            };
            return effectHandle;
        }

        public bool FXSetParameters(int fxHandle, object parameters)
        {
            MethodCallLog.Add($"FXSetParameters({fxHandle}, {parameters?.GetType().Name})");
            
            if (!_effects.ContainsKey(fxHandle))
                return false;
                
            _effects[fxHandle].Parameters = parameters;
            return true;
        }

        // Format conversion
        public long ChannelSeconds2Bytes(int handle, double seconds)
        {
            MethodCallLog.Add($"ChannelSeconds2Bytes({handle}, {seconds})");
            // Simple mock conversion: assume 44100 Hz * 2 channels * 2 bytes = 176400 bytes/second
            return (long)(seconds * 176400);
        }

        public double ChannelBytes2Seconds(int handle, long bytes)
        {
            MethodCallLog.Add($"ChannelBytes2Seconds({handle}, {bytes})");
            // Simple mock conversion
            return bytes / 176400.0;
        }

        // Sync and callbacks  
        public int ChannelSetSync(int handle, SyncFlags type, long param, Action<int, int, int, IntPtr> callback, IntPtr user)
        {
            MethodCallLog.Add($"ChannelSetSync({handle}, {type}, {param})");
            // Store callback for potential test triggering
            var syncHandle = _nextHandle++;
            return syncHandle;
        }

        // Font management for MIDI
        public int FontInit(string file, FontInitFlags flags)
        {
            MethodCallLog.Add($"FontInit({file}, {flags})");
            return ShouldFailStreamCreation ? 0 : _nextHandle++;
        }

        public bool FontFree(int handle)
        {
            MethodCallLog.Add($"FontFree({handle})");
            return true;
        }

        public bool StreamSetFonts(int handle, MidiFont[] fonts, int count)
        {
            MethodCallLog.Add($"StreamSetFonts({handle}, fonts[{fonts.Length}], {count})");
            return true;
        }

        // Missing methods required by tests
        public bool WasMethodCalled(string methodName)
        {
            return MethodCallLog.Any(call => call.Contains(methodName));
        }

        public void ChannelSetLength(int handle, long length)
        {
            MethodCallLog.Add($"ChannelSetLength({handle}, {length})");
            
            if (_streams.ContainsKey(handle))
            {
                _streams[handle].Length = length;
            }
        }

        public bool ChannelPlay(int handle, bool restart)
        {
            MethodCallLog.Add($"ChannelPlay({handle}, {restart})");
            
            if (ShouldFailPlayback || !_streams.ContainsKey(handle))
                return false;
                
            if (restart)
            {
                _streams[handle].Position = 0;
            }
            
            _streams[handle].IsPlaying = true;
            return true;
        }

        // Test helper methods
        public void Reset()
        {
            _streams.Clear();
            _effects.Clear();
            MethodCallLog.Clear();
            ShouldFailStreamCreation = false;
            ShouldFailPlayback = false;
            LastErrorValue = Errors.OK;
            _nextHandle = 1000;
        }
        
        public MockAudioStream? GetStream(int handle)
        {
            return _streams.TryGetValue(handle, out var stream) ? stream : null;
        }
        
        public MockAudioEffect? GetEffect(int handle)
        {
            return _effects.TryGetValue(handle, out var effect) ? effect : null;
        }
    }
    
    public class MockAudioStream
    {
        public string FilePath { get; set; } = "";
        public long Position { get; set; }
        public long Length { get; set; }
        public float Volume { get; set; }
        public bool IsPlaying { get; set; }
    }
    
    public class MockAudioEffect
    {
        public int ChannelHandle { get; set; }
        public EffectType EffectType { get; set; }
        public int Priority { get; set; }
        public object? Parameters { get; set; }
    }
}
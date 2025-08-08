using ManagedBass;
using ManagedBass.Midi;

namespace Jammer.Core.Interfaces
{
    /// <summary>
    /// Interface abstraction for BASS audio library operations
    /// Enables testing and dependency injection for audio playback functionality
    /// 
    /// This interface abstracts the core BASS library calls used throughout Play.cs
    /// allowing for mock implementations during testing and better separation of concerns
    /// </summary>
    public interface IAudioPlayer
    {
        // Stream creation and management
        int CreateStream(string file, long offset, long length, BassFlags flags);
        int CreateAacStream(string file, long offset, long length, BassFlags flags);
        int CreateMp4Stream(string file, long offset, long length, BassFlags flags);  
        int CreateOpusStream(string file, long offset, long length, BassFlags flags);
        int CreateMidiStream(string file, long offset, long length, BassFlags flags);
        
        // Playback control
        bool ChannelPlay(int handle);
        bool ChannelPause(int handle);
        bool ChannelStop(int handle);
        bool StreamFree(int handle);
        
        // Channel properties
        bool ChannelSetAttribute(int handle, ChannelAttribute attribute, float value);
        bool ChannelGetAttribute(int handle, ChannelAttribute attribute, out float value);
        long ChannelGetPosition(int handle);
        bool ChannelSetPosition(int handle, long position);
        long ChannelGetLength(int handle);
        
        // Effects and processing
        int ChannelSetFX(int handle, EffectType type, int priority);
        bool FXSetParameters(int fxHandle, object parameters);
        
        // Format conversion
        long ChannelSeconds2Bytes(int handle, double seconds);
        double ChannelBytes2Seconds(int handle, long bytes);
        
        // Sync and callbacks
        int ChannelSetSync(int handle, SyncFlags type, long param, Action<int, int, int, IntPtr> callback, IntPtr user);
        
        // Font management for MIDI
        int FontInit(string file, FontInitFlags flags);
        bool FontFree(int handle);
        bool StreamSetFonts(int handle, MidiFont[] fonts, int count);
        
        // Error handling
        Errors LastError { get; }
    }
}
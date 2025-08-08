namespace Jammer.Tests.TestUtilities.Doubles;

/// <summary>
/// A null implementation of audio player functionality for testing purposes.
/// This implementation does nothing and avoids dependencies on native BASS libraries.
/// </summary>
public class NullAudioPlayer
{
    public bool IsPlaying { get; private set; }
    public bool IsPaused { get; private set; }
    public double Volume { get; set; } = 1.0;
    public TimeSpan Position { get; set; } = TimeSpan.Zero;
    public TimeSpan Duration { get; set; } = TimeSpan.Zero;
    
    public List<string> PlayedTracks { get; } = new();
    public List<string> StoppedTracks { get; } = new();
    
    public void Play(string trackPath)
    {
        PlayedTracks.Add(trackPath);
        IsPlaying = true;
        IsPaused = false;
    }
    
    public void Pause()
    {
        IsPaused = true;
        IsPlaying = false;
    }
    
    public void Resume()
    {
        if (IsPaused)
        {
            IsPlaying = true;
            IsPaused = false;
        }
    }
    
    public void Stop()
    {
        if (IsPlaying || IsPaused)
        {
            StoppedTracks.Add(PlayedTracks.LastOrDefault() ?? "");
        }
        IsPlaying = false;
        IsPaused = false;
        Position = TimeSpan.Zero;
    }
    
    public void Seek(TimeSpan position)
    {
        Position = position;
    }
    
    public void SetVolume(double volume)
    {
        Volume = Math.Clamp(volume, 0.0, 1.0);
    }
    
    public void Reset()
    {
        Stop();
        PlayedTracks.Clear();
        StoppedTracks.Clear();
        Volume = 1.0;
        Duration = TimeSpan.Zero;
    }
}
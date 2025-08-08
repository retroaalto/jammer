namespace Jammer.Core.Interfaces
{
    /// <summary>
    /// Interface abstraction for download operations
    /// Enables testing and dependency injection for YouTube/SoundCloud download functionality
    /// 
    /// This interface abstracts the external API calls and file operations used in Download.cs
    /// allowing for mock implementations during testing and better separation of concerns
    /// </summary>
    public interface IDownloadService
    {
        // Main download operations
        Task<string> DownloadSongAsync(string url);
        Task<string> DownloadYoutubeTrackAsync(string url);
        Task<string> DownloadSoundCloudTrackAsync(string url);
        Task DownloadJammerFileAsync(string url);
        
        // Playlist operations
        Task<string> GetPlaylistAsync(string url);
        Task<string> GetPlaylistYoutubeAsync(string url);
        Task<string> GetSongsFromPlaylistAsync(string url, string platform);
        
        // Utility operations
        Task<bool> IsFFmpegInstalledAsync();
        Task ConvertWithFFmpegAsync(string inputPath, Song? metadata = null);
        Task DownloadThumbnailAsync(string url, string outputPath);
        string FormatUrlForFilename(string url);
        string GetDownloadedJammerFileName(string url);
        
        // Configuration
        string SongPath { get; set; }
        IList<string> PlaylistSongs { get; }
    }
}
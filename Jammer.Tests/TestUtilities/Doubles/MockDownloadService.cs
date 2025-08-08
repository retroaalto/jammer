using Jammer.Core.Interfaces;

namespace Jammer.Tests.TestUtilities.Doubles
{
    /// <summary>
    /// Mock implementation of IDownloadService for testing
    /// Provides controllable behavior for all download operations
    /// </summary>
    public class MockDownloadService : IDownloadService
    {
        private readonly Dictionary<string, string> _downloadResults = new();
        private readonly List<string> _downloadedFiles = new();
        private readonly List<string> _methodCalls = new();
        
        // Configuration properties for testing
        public bool ShouldFailDownloads { get; set; } = false;
        public bool IsFFmpegInstalled { get; set; } = true;
        public TimeSpan DownloadDelay { get; set; } = TimeSpan.Zero;
        public List<string> MethodCalls => _methodCalls;
        public List<string> DownloadedFiles => _downloadedFiles;
        
        public string SongPath { get; set; } = "/test/songs";
        public IList<string> PlaylistSongs { get; } = new List<string>();

        // Main download operations
        public async Task<string> DownloadSongAsync(string url)
        {
            _methodCalls.Add($"DownloadSongAsync({url})");
            
            if (string.IsNullOrEmpty(url))
                throw new ArgumentException("URL cannot be null or empty");
            
            if (DownloadDelay > TimeSpan.Zero)
                await Task.Delay(DownloadDelay);
            
            if (ShouldFailDownloads)
                throw new InvalidOperationException($"Mock download failed for {url}");
            
            // Determine download method based on URL
            if (url.Contains("youtube.com") || url.Contains("youtu.be"))
                return await DownloadYoutubeTrackAsync(url);
            else if (url.Contains("soundcloud.com"))
                return await DownloadSoundCloudTrackAsync(url);
            else
                return await GeneralDownloadAsync(url);
        }

        public async Task<string> DownloadYoutubeTrackAsync(string url)
        {
            _methodCalls.Add($"DownloadYoutubeTrackAsync({url})");
            
            if (DownloadDelay > TimeSpan.Zero)
                await Task.Delay(DownloadDelay);
            
            if (!IsFFmpegInstalled)
                throw new InvalidOperationException($"FFmpeg is required for YouTube downloads but is not installed");
            
            if (ShouldFailDownloads)
                throw new InvalidOperationException($"Mock YouTube download failed for {url}");
            
            var fileName = FormatUrlForFilename(url) + ".ogg";
            var filePath = Path.Combine(SongPath, fileName);
            _downloadedFiles.Add(filePath);
            
            return filePath;
        }

        public async Task<string> DownloadSoundCloudTrackAsync(string url)
        {
            _methodCalls.Add($"DownloadSoundCloudTrackAsync({url})");
            
            if (DownloadDelay > TimeSpan.Zero)
                await Task.Delay(DownloadDelay);
            
            if (ShouldFailDownloads)
                throw new InvalidOperationException($"Mock SoundCloud download failed for {url}");
            
            var fileName = FormatUrlForFilename(url) + ".ogg";
            var filePath = Path.Combine(SongPath, fileName);
            _downloadedFiles.Add(filePath);
            
            return filePath;
        }

        public async Task DownloadJammerFileAsync(string url)
        {
            _methodCalls.Add($"DownloadJammerFileAsync({url})");
            
            if (DownloadDelay > TimeSpan.Zero)
                await Task.Delay(DownloadDelay);
            
            if (ShouldFailDownloads)
                throw new InvalidOperationException($"Mock jammer file download failed for {url}");
            
            var fileName = GetDownloadedJammerFileName(url);
            var filePath = Path.Combine(SongPath, fileName);
            _downloadedFiles.Add(filePath);
        }

        // Playlist operations
        public async Task<string> GetPlaylistAsync(string url)
        {
            _methodCalls.Add($"GetPlaylistAsync({url})");
            
            if (DownloadDelay > TimeSpan.Zero)
                await Task.Delay(DownloadDelay);
            
            if (ShouldFailDownloads)
                throw new InvalidOperationException($"Mock playlist download failed for {url}");
            
            // Mock behavior - determine platform and delegate
            if (url.Contains("youtube.com") || url.Contains("youtu.be"))
                return await GetPlaylistYoutubeAsync(url);
            else
                return await GetSongsFromPlaylistAsync(url, "soundcloud");
        }

        public async Task<string> GetPlaylistYoutubeAsync(string url)
        {
            _methodCalls.Add($"GetPlaylistYoutubeAsync({url})");
            
            if (DownloadDelay > TimeSpan.Zero)
                await Task.Delay(DownloadDelay);
            
            if (ShouldFailDownloads)
                throw new InvalidOperationException($"Mock YouTube playlist download failed for {url}");
            
            // Mock: Add some test songs to playlist
            PlaylistSongs.Add("https://youtube.com/watch?v=test1");
            PlaylistSongs.Add("https://youtube.com/watch?v=test2");
            PlaylistSongs.Add("https://youtube.com/watch?v=test3");
            
            return $"Downloaded YouTube playlist with {PlaylistSongs.Count} songs";
        }

        public async Task<string> GetSongsFromPlaylistAsync(string url, string platform)
        {
            _methodCalls.Add($"GetSongsFromPlaylistAsync({url}, {platform})");
            
            if (DownloadDelay > TimeSpan.Zero)
                await Task.Delay(DownloadDelay);
            
            if (ShouldFailDownloads)
                throw new InvalidOperationException($"Mock playlist songs download failed for {url}");
            
            // Mock: Add platform-specific test songs
            if (platform.ToLower() == "youtube")
            {
                PlaylistSongs.Add("https://youtube.com/watch?v=playlist1");
                PlaylistSongs.Add("https://youtube.com/watch?v=playlist2");
            }
            else if (platform.ToLower() == "soundcloud")
            {
                PlaylistSongs.Add("https://soundcloud.com/artist/track1");
                PlaylistSongs.Add("https://soundcloud.com/artist/track2");
            }
            
            return $"Downloaded {platform} playlist with {PlaylistSongs.Count} songs";
        }

        // Utility operations
        public async Task<bool> IsFFmpegInstalledAsync()
        {
            _methodCalls.Add("IsFFmpegInstalledAsync()");
            
            if (DownloadDelay > TimeSpan.Zero)
                await Task.Delay(DownloadDelay);
            
            return IsFFmpegInstalled;
        }

        public async Task ConvertWithFFmpegAsync(string inputPath, Song? metadata = null)
        {
            _methodCalls.Add($"ConvertWithFFmpegAsync({inputPath}, {metadata?.Title})");
            
            if (DownloadDelay > TimeSpan.Zero)
                await Task.Delay(DownloadDelay);
            
            if (ShouldFailDownloads || !IsFFmpegInstalled)
                throw new InvalidOperationException($"Mock FFmpeg conversion failed for {inputPath}");
            
            // Mock: Simulate conversion by changing extension
            var outputPath = Path.ChangeExtension(inputPath, ".ogg");
            if (!_downloadedFiles.Contains(outputPath))
                _downloadedFiles.Add(outputPath);
        }

        public async Task DownloadThumbnailAsync(string url, string outputPath)
        {
            _methodCalls.Add($"DownloadThumbnailAsync({url}, {outputPath})");
            
            if (DownloadDelay > TimeSpan.Zero)
                await Task.Delay(DownloadDelay);
            
            if (ShouldFailDownloads)
                throw new InvalidOperationException($"Mock thumbnail download failed for {url}");
            
            _downloadedFiles.Add(outputPath);
        }

        public string FormatUrlForFilename(string url)
        {
            _methodCalls.Add($"FormatUrlForFilename({url})");
            
            // Mock implementation: simple URL cleaning
            var formatted = url
                .Replace("https://", "")
                .Replace("http://", "")
                .Replace("www.", "")
                .Replace("/", "_")
                .Replace("?", "_")
                .Replace("&", "_")
                .Replace("=", "_");
            
            // Truncate if too long
            if (formatted.Length > 50)
                formatted = formatted.Substring(0, 50);
            
            return formatted;
        }

        public string GetDownloadedJammerFileName(string url)
        {
            _methodCalls.Add($"GetDownloadedJammerFileName({url})");
            return FormatUrlForFilename(url) + ".jammer";
        }

        // Test helper methods
        private async Task<string> GeneralDownloadAsync(string url)
        {
            _methodCalls.Add($"GeneralDownloadAsync({url})");
            
            if (DownloadDelay > TimeSpan.Zero)
                await Task.Delay(DownloadDelay);
            
            var fileName = FormatUrlForFilename(url) + ".mp3";
            var filePath = Path.Combine(SongPath, fileName);
            _downloadedFiles.Add(filePath);
            
            return filePath;
        }

        public void Reset()
        {
            _downloadResults.Clear();
            _downloadedFiles.Clear();
            _methodCalls.Clear();
            PlaylistSongs.Clear();
            ShouldFailDownloads = false;
            IsFFmpegInstalled = true;
            DownloadDelay = TimeSpan.Zero;
            SongPath = "/test/songs";
        }
        
        public void SetDownloadResult(string url, string result)
        {
            _downloadResults[url] = result;
        }
        
        public bool WasMethodCalled(string methodName)
        {
            return _methodCalls.Any(call => call.Contains(methodName));
        }
        
        public int GetMethodCallCount(string methodName)
        {
            return _methodCalls.Count(call => call.Contains(methodName));
        }
    }
}
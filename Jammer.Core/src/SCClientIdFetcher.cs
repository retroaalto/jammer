using PuppeteerSharp;
using System.Runtime.InteropServices;
using System.Text;
using System.Text.RegularExpressions;

namespace Jammer
{
    public static class SCClientIdFetcher
    {
        private const string DefaultSoundCloudUrl = "https://soundcloud.com/";
        private const string SmokeTestUrl = "https://example.com/";
        private const int DefaultTimeoutMs = 30000;

        public static async Task<string> MonitorNetwork(string url, bool useUiMessages = true, int timeoutMs = DefaultTimeoutMs)
        {
            ReportStatus("Starting Puppeteer...", "Please wait.", useUiMessages);

            var fetcher = new BrowserFetcher();
            var installedBrowser = await fetcher.DownloadAsync();
            string executablePath = installedBrowser.GetExecutablePath();

            ReportStatus("Launching browser...", "Please wait..", useUiMessages);

            IBrowser browser;
            string launchMode;
            (browser, launchMode) = await LaunchBrowserWithFallbackAsync(executablePath);

            await using (browser.ConfigureAwait(false))
            {
                Log.Info($"Puppeteer launched using mode: {launchMode}");
                ReportStatus("Opening page...", "Please wait...", useUiMessages);

                await using var page = await browser.NewPageAsync();
                var clientIdTask = new TaskCompletionSource<string>(TaskCreationOptions.RunContinuationsAsynchronously);

                page.Request += (_, e) =>
                {
                    string requestUrl = e.Request.Url;

                    if (requestUrl.Contains("client_id"))
                    {
                        var clientIdMatch = Regex.Match(requestUrl, @"client_id=([^&]+)");
                        if (clientIdMatch.Success)
                        {
                            clientIdTask.TrySetResult(clientIdMatch.Groups[1].Value);
                        }
                    }
                };

                await page.GoToAsync(url, new NavigationOptions
                {
                    Timeout = timeoutMs,
                    WaitUntil = new[] { WaitUntilNavigation.DOMContentLoaded }
                });

                var completedTask = await Task.WhenAny(clientIdTask.Task, Task.Delay(timeoutMs));
                if (completedTask != clientIdTask.Task)
                {
                    throw new TimeoutException("Timed out waiting for a SoundCloud request containing client_id.");
                }

                return await clientIdTask.Task;
            }
        }

        public static async Task<string> GetClientId()
        {
            return await MonitorNetwork(DefaultSoundCloudUrl);
        }

        public static async Task<string> RunSelfTestAsync(int timeoutMs = DefaultTimeoutMs)
        {
            var report = new StringBuilder();
            var fetcher = new BrowserFetcher();
            Action<string> progress = CreateProgressReporter();

            report.AppendLine("Puppeteer self-test starting");
            progress("Puppeteer self-test starting");
            report.AppendLine($"Cache directory: {fetcher.CacheDir}");
            progress($"Using cache directory: {fetcher.CacheDir}");

            progress("Downloading or locating Chromium...");
            var installedBrowser = await fetcher.DownloadAsync();
            string executablePath = installedBrowser.GetExecutablePath();

            report.AppendLine($"Chromium executable: {executablePath}");
            progress($"Chromium ready: {executablePath}");

            progress("Launching browser...");
            IBrowser browser;
            string launchMode;
            (browser, launchMode) = await LaunchBrowserWithFallbackAsync(executablePath);

            await using (browser.ConfigureAwait(false))
            {
                report.AppendLine($"Launch mode: {launchMode}");
                progress($"Browser launched using mode: {launchMode}");
                report.AppendLine($"Browser version: {await browser.GetVersionAsync()}");
                progress("Browser launch succeeded");

                progress("Running smoke test against example.com...");
                await using var smokePage = await browser.NewPageAsync();
                await smokePage.GoToAsync(SmokeTestUrl, new NavigationOptions
                {
                    Timeout = timeoutMs,
                    WaitUntil = new[] { WaitUntilNavigation.Networkidle0 }
                });

                report.AppendLine($"Smoke test title: {await smokePage.GetTitleAsync()}");
                progress("Smoke test completed successfully");

                progress("Opening SoundCloud and watching network requests...");
                await using var soundCloudPage = await browser.NewPageAsync();
                var clientIdTask = new TaskCompletionSource<string>(TaskCreationOptions.RunContinuationsAsynchronously);
                int requestsSeen = 0;

                soundCloudPage.Request += (_, e) =>
                {
                    requestsSeen++;
                    string requestUrl = e.Request.Url;

                    if (requestUrl.Contains("client_id"))
                    {
                        var clientIdMatch = Regex.Match(requestUrl, @"client_id=([^&]+)");
                        if (clientIdMatch.Success)
                        {
                            clientIdTask.TrySetResult(clientIdMatch.Groups[1].Value);
                        }
                    }
                };

                try
                {
                    await soundCloudPage.GoToAsync(DefaultSoundCloudUrl, new NavigationOptions
                    {
                        Timeout = timeoutMs,
                        WaitUntil = new[] { WaitUntilNavigation.DOMContentLoaded }
                    });

                    var completedTask = await Task.WhenAny(clientIdTask.Task, Task.Delay(timeoutMs));
                    report.AppendLine($"SoundCloud requests observed: {requestsSeen}");

                    if (completedTask == clientIdTask.Task)
                    {
                        string clientId = await clientIdTask.Task;
                        report.AppendLine($"SoundCloud client_id detected: {clientId}");
                        progress("SoundCloud client_id detected successfully");
                    }
                    else
                    {
                        report.AppendLine("SoundCloud client_id not detected within timeout");
                        progress("SoundCloud loaded, but client_id was not detected before timeout");
                    }
                }
                catch (Exception ex)
                {
                    report.AppendLine($"SoundCloud check failed: {ex.Message}");
                    progress("SoundCloud check failed: " + ex.Message);
                }
            }

            return report.ToString().TrimEnd();
        }

        private static Action<string> CreateProgressReporter()
        {
            return message =>
            {
                Console.WriteLine("[puppeteer] " + message);
                Console.Out.Flush();
            };
        }

        private static async Task<(IBrowser Browser, string LaunchMode)> LaunchBrowserWithFallbackAsync(string executablePath)
        {
            try
            {
                return (await Puppeteer.LaunchAsync(CreateLaunchOptions(executablePath, disableSandbox: false)), "default");
            }
            catch (Exception ex) when (RuntimeInformation.IsOSPlatform(OSPlatform.Linux))
            {
                Log.Error("Default Puppeteer launch failed: " + ex.Message);
                return (await Puppeteer.LaunchAsync(CreateLaunchOptions(executablePath, disableSandbox: true)), "linux-no-sandbox-fallback");
            }
        }

        private static LaunchOptions CreateLaunchOptions(string executablePath, bool disableSandbox)
        {
            var options = new LaunchOptions
            {
                Headless = true,
                ExecutablePath = executablePath
            };

            if (RuntimeInformation.IsOSPlatform(OSPlatform.Linux))
            {
                options.Args = disableSandbox
                    ? new[] { "--no-sandbox", "--disable-setuid-sandbox", "--disable-dev-shm-usage", "--disable-gpu" }
                    : new[] { "--disable-dev-shm-usage", "--disable-gpu" };
            }

            return options;
        }

        private static void ReportStatus(string title, string message, bool useUiMessages)
        {
            Log.Info(title + " " + message);

            if (useUiMessages)
            {
                Message.Data(title, message, false, false);
            }
        }
    }
}

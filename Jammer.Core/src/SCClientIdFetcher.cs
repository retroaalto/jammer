using System.Net.Http.Headers;
using System.Text.RegularExpressions;

namespace Jammer
{
    public static class SCClientIdFetcher
    {
        private const string SoundCloudUrl = "https://soundcloud.com/";

        // Matches JS bundle URLs embedded in the SoundCloud homepage
        private static readonly Regex ScriptUrlRe = new Regex(
            @"https://a-v2\.sndcdn\.com/assets/[^""']+\.js",
            RegexOptions.Compiled);

        // Matches client_id in JS bundle content: client_id:"VALUE", client_id="VALUE", client_id\x3a"VALUE"
        private static readonly Regex ClientIdRe = new Regex(
            @"client_id[=:\x3a][""']?([a-zA-Z0-9_\-]{20,})",
            RegexOptions.Compiled);

        private static readonly HttpClient Http = new HttpClient(new HttpClientHandler
        {
            AllowAutoRedirect = true
        })
        {
            Timeout = TimeSpan.FromSeconds(30),
            DefaultRequestHeaders =
            {
                UserAgent = { ProductInfoHeaderValue.Parse("Mozilla/5.0") }
            }
        };

        /// <summary>
        /// Returns a valid SoundCloud client ID, using the cached value if still within TTL,
        /// otherwise scraping a fresh one from the SoundCloud homepage JS bundles.
        /// </summary>
        public static async Task<string> GetClientId()
        {
            // Use cached value if present and within TTL
            if (!string.IsNullOrEmpty(Preferences.clientID) && Preferences.clientIDFetchedAt.HasValue)
            {
                int ttlDays = Preferences.scClientIdTTLDays > 0 ? Preferences.scClientIdTTLDays : 7;
                if ((DateTime.UtcNow - Preferences.clientIDFetchedAt.Value).TotalDays < ttlDays)
                {
                    return Preferences.clientID;
                }
                Log.Info("SoundCloud client ID TTL expired, fetching fresh one.");
            }

            return await FetchAndSave();
        }

        /// <summary>
        /// Forces a fresh scrape of the client ID, saves it, and returns it.
        /// </summary>
        public static async Task<string> FetchAndSave()
        {
            Log.Info("Fetching SoundCloud client ID from JS bundles...");
            Message.Data("SoundCloud", "Fetching client ID...", false, false);

            string id = await Fetch();

            if (!string.IsNullOrEmpty(id))
            {
                Preferences.clientID = id;
                Preferences.clientIDFetchedAt = DateTime.UtcNow;
                Preferences.SaveSettings();
                Log.Info("SoundCloud client ID fetched and cached: " + id);
            }
            else
            {
                Log.Error("Failed to fetch SoundCloud client ID from JS bundles.");
            }

            return id;
        }

        /// <summary>
        /// Scrapes the SoundCloud homepage, finds JS bundle URLs, and extracts the client ID.
        /// </summary>
        private static async Task<string> Fetch()
        {
            string html;
            try
            {
                html = await Http.GetStringAsync(SoundCloudUrl);
            }
            catch (Exception ex)
            {
                Log.Error("Failed to fetch SoundCloud homepage: " + ex.Message);
                return "";
            }

            // Find all JS bundle URLs in the page
            var scriptUrls = ScriptUrlRe.Matches(html)
                .Select(m => m.Value)
                .Distinct()
                .ToList();

            // Try the last 5 bundles (most likely to contain client_id)
            foreach (var jsUrl in scriptUrls.TakeLast(5).Reverse())
            {
                string id = await TryExtractFromUrl(jsUrl);
                if (!string.IsNullOrEmpty(id))
                    return id;
            }

            // Fallback: check inline in the homepage HTML itself
            string inlineId = ExtractClientId(html);
            if (!string.IsNullOrEmpty(inlineId))
                return inlineId;

            return "";
        }

        private static async Task<string> TryExtractFromUrl(string url)
        {
            try
            {
                string js = await Http.GetStringAsync(url);
                return ExtractClientId(js);
            }
            catch (Exception ex)
            {
                Log.Error($"Failed to fetch JS bundle {url}: {ex.Message}");
                return "";
            }
        }

        private static string ExtractClientId(string content)
        {
            // Normalize escaped colon used in some minified bundles
            string normalized = content.Replace(@"\x3a", ":");
            var match = ClientIdRe.Match(normalized);
            return match.Success ? match.Groups[1].Value : "";
        }

        /// <summary>
        /// Simple self-test: fetches a fresh client ID and reports the result.
        /// </summary>
        public static async Task<string> RunSelfTestAsync()
        {
            var lines = new System.Text.StringBuilder();
            lines.AppendLine("SoundCloud client ID self-test starting");
            lines.AppendLine($"Fetching {SoundCloudUrl}...");

            try
            {
                string id = await Fetch();
                if (!string.IsNullOrEmpty(id))
                {
                    lines.AppendLine($"client_id detected: {id}");
                }
                else
                {
                    lines.AppendLine("client_id not found in any JS bundle or inline HTML.");
                }
            }
            catch (Exception ex)
            {
                lines.AppendLine("Self-test failed: " + ex.Message);
            }

            return lines.ToString().TrimEnd();
        }
    }
}

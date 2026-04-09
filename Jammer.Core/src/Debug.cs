using System.IO;

namespace Jammer
{
    public class Debug
    {
        // Resolved once so every write goes to the same file even if CWD changes.
        private static readonly string LogPath =
            Path.Combine(Utils.JammerPath, "debug.log");

        // State for CPU-usage delta calculation inside dperf().
        private static TimeSpan _lastCpuTime   = TimeSpan.Zero;
        private static DateTime _lastCpuSample = DateTime.MinValue;

        /// <summary>
        /// Appends a line to ~/.jammer/debug.log.
        /// Format: HH:mm:ss.fff;ClassName;MethodName: message
        /// Only writes when the app was started with -D.
        /// </summary>
        public static void dprint(string txt)
        {
            if (!Utils.IsDebug) return;

            try
            {
                // Capture call-site info before any allocation that could shift frames.
                var frame  = new System.Diagnostics.StackTrace(1, false).GetFrame(0);
                var method = frame?.GetMethod()?.Name ?? "?";
                var cls    = frame?.GetMethod()?.DeclaringType?.Name ?? "?";
                var time   = DateTime.Now.ToString("HH:mm:ss.fff");

                Directory.CreateDirectory(Path.GetDirectoryName(LogPath)!);
                using var writer = new StreamWriter(LogPath, append: true);
                writer.WriteLine($"{time};{cls};{method}: {txt}");
            }
            catch { /* never crash the app due to debug I/O */ }
        }

        /// <summary>
        /// Appends a performance snapshot line to debug.log.
        /// Captures: managed heap bytes, GC collection counts per generation,
        /// live thread count, process CPU usage % since last call, and an
        /// optional caller-supplied label.
        /// Deliberately avoids StackTrace capture to keep overhead low.
        /// Only writes when the app was started with -D.
        /// </summary>
        public static void dperf(string label = "")
        {
            if (!Utils.IsDebug) return;

            try
            {
                var now        = DateTime.Now;
                var proc       = System.Diagnostics.Process.GetCurrentProcess();

                long heapBytes = GC.GetTotalMemory(forceFullCollection: false);
                int gen0       = GC.CollectionCount(0);
                int gen1       = GC.CollectionCount(1);
                int gen2       = GC.CollectionCount(2);
                int threads    = proc.Threads.Count;

                // CPU % = (delta CPU time) / (delta wall time * logical CPUs) * 100
                string cpuStr = "?";
                var currentCpuTime = proc.TotalProcessorTime;
                if (_lastCpuSample != DateTime.MinValue)
                {
                    double wallMs = (now - _lastCpuSample).TotalMilliseconds;
                    if (wallMs > 0)
                    {
                        double cpuMs      = (currentCpuTime - _lastCpuTime).TotalMilliseconds;
                        int    cpuCount   = Environment.ProcessorCount;
                        double cpuPercent = cpuMs / (wallMs * cpuCount) * 100.0;
                        cpuStr = $"{cpuPercent:F1}%";
                    }
                }
                _lastCpuTime   = currentCpuTime;
                _lastCpuSample = now;

                Directory.CreateDirectory(Path.GetDirectoryName(LogPath)!);
                using var writer = new StreamWriter(LogPath, append: true);
                writer.WriteLine(
                    $"{now:HH:mm:ss.fff};PERF;{label}: " +
                    $"cpu={cpuStr} " +
                    $"heap={heapBytes / 1024}KB " +
                    $"gc0={gen0} gc1={gen1} gc2={gen2} " +
                    $"threads={threads}");
            }
            catch { }
        }

        /// <summary>
        /// Writes the entire in-memory Log queue to debug.log as a labelled block.
        /// Useful for capturing app-level events (playlist loads, errors, etc.)
        /// at the moment you notice a problem.
        /// Only writes when the app was started with -D.
        /// </summary>
        public static void FlushAppLog()
        {
            if (!Utils.IsDebug) return;

            try
            {
                var time = DateTime.Now.ToString("HH:mm:ss.fff");
                Directory.CreateDirectory(Path.GetDirectoryName(LogPath)!);
                using var writer = new StreamWriter(LogPath, append: true);
                writer.WriteLine($"--- App log flush at {time} ---");
                // Strip Spectre markup before writing to a plain-text file.
                foreach (var entry in Log.log)
                {
                    writer.WriteLine(Start.Purge(entry));
                }
                writer.WriteLine($"--- end flush ---");
            }
            catch { }
        }
    }
}

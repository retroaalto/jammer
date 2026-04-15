using System.IO;
using System.Collections.Generic;

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

        // State for /proc/<pid>/stat jiffies delta (mirrors what top reports).
        private static long     _lastJiffies    = -1;
        private static DateTime _lastJiffySample = DateTime.MinValue;
        private static readonly long TicksPerSecond = GetTicksPerSecond();

        // Per-thread jiffies from previous dperf() call: tid -> jiffies.
        private static Dictionary<int, long> _lastThreadJiffies = new();
        private static DateTime _lastThreadSample = DateTime.MinValue;

        private static long GetTicksPerSecond()
        {
            try
            {
                // sysconf(_SC_CLK_TCK) via getconf — almost always 100 on Linux.
                var psi = new System.Diagnostics.ProcessStartInfo("getconf", "CLK_TCK")
                {
                    RedirectStandardOutput = true,
                    UseShellExecute        = false
                };
                using var p = System.Diagnostics.Process.Start(psi);
                if (p != null)
                {
                    string s = p.StandardOutput.ReadToEnd().Trim();
                    p.WaitForExit();
                    if (long.TryParse(s, out long hz)) return hz;
                }
            }
            catch { }
            return 100; // safe fallback
        }

        /// <summary>
        /// Reads utime+stime from /proc/self/stat and returns CPU% since last
        /// call, matching what top displays (single-core %, not divided by ncpu).
        /// Returns "?" on first call or on any read failure.
        /// </summary>
        private static string ReadProcStatCpu()
        {
            try
            {
                var now    = DateTime.UtcNow;
                // Fields in /proc/self/stat are space-separated; field 14 = utime, 15 = stime (0-indexed).
                var fields = File.ReadAllText("/proc/self/stat").Split(' ');
                if (fields.Length < 15) return "?";
                long utime   = long.Parse(fields[13]);
                long stime   = long.Parse(fields[14]);
                long jiffies = utime + stime;

                string result = "?";
                if (_lastJiffies >= 0)
                {
                    double wallSec   = (now - _lastJiffySample).TotalSeconds;
                    if (wallSec > 0)
                    {
                        double cpuPercent = (jiffies - _lastJiffies) / (wallSec * TicksPerSecond) * 100.0;
                        result = $"{cpuPercent:F1}%";
                    }
                }
                _lastJiffies     = jiffies;
                _lastJiffySample = now;
                return result;
            }
            catch { return "?"; }
        }

        /// <summary>
        /// Reads per-thread CPU% from /proc/self/task/*/stat, returns a compact
        /// string listing threads that consumed >= 1% since the last call.
        /// Format: "tid=NAME:CPU% tid=NAME:CPU% ..."  (unnamed threads show tid only)
        /// Only available on Linux; returns "" on other platforms or on error.
        /// </summary>
        private static string ReadPerThreadCpu(DateTime now)
        {
            try
            {
                double wallSec = _lastThreadSample == DateTime.MinValue
                    ? 0
                    : (now - _lastThreadSample).TotalSeconds;

                var taskDir = $"/proc/{System.Diagnostics.Process.GetCurrentProcess().Id}/task";
                if (!Directory.Exists(taskDir)) return "";

                var newJiffies = new Dictionary<int, long>();
                var results    = new List<string>();

                foreach (var tidDir in Directory.GetDirectories(taskDir))
                {
                    var statPath = Path.Combine(tidDir, "stat");
                    var commPath = Path.Combine(tidDir, "comm");
                    if (!File.Exists(statPath)) continue;

                    if (!int.TryParse(Path.GetFileName(tidDir), out int tid)) continue;

                    var fields = File.ReadAllText(statPath).Split(' ');
                    if (fields.Length < 15) continue;
                    long utime = long.Parse(fields[13]);
                    long stime = long.Parse(fields[14]);
                    long j = utime + stime;
                    newJiffies[tid] = j;

                    if (wallSec > 0 && _lastThreadJiffies.TryGetValue(tid, out long prevJ))
                    {
                        double cpu = (j - prevJ) / (wallSec * TicksPerSecond) * 100.0;
                        if (cpu >= 1.0)
                        {
                            string name = File.Exists(commPath)
                                ? File.ReadAllText(commPath).Trim()
                                : tid.ToString();
                            results.Add($"tid={tid}({name}):{cpu:F1}%");
                        }
                    }
                }

                _lastThreadJiffies = newJiffies;
                _lastThreadSample  = now;
                return string.Join(" ", results);
            }
            catch { return ""; }
        }

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

                string cpuOsStr     = ReadProcStatCpu();
                string threadCpuStr = ReadPerThreadCpu(now);

                Directory.CreateDirectory(Path.GetDirectoryName(LogPath)!);
                using var writer = new StreamWriter(LogPath, append: true);
                var line =
                    $"{now:HH:mm:ss.fff};PERF;{label}: " +
                    $"cpu={cpuStr} cpu_os={cpuOsStr} " +
                    $"heap={heapBytes / 1024}KB " +
                    $"gc0={gen0} gc1={gen1} gc2={gen2} " +
                    $"threads={threads}";
                if (!string.IsNullOrEmpty(threadCpuStr))
                    line += $" | {threadCpuStr}";
                writer.WriteLine(line);
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

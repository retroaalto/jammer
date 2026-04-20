# Fix: Gradual CPU Leak Over Time

**Branch:** `fix/gradual-cpu-leak`  
**Date:** 2026-04-20  
**Profile source:** `jammer-final.speedscope.json` (120s trace, PID 189892)

## Background

CPU usage does not spike immediately on launch but climbs steadily over ~1 hour of runtime.
The profiler shows `Thread.Sleep`, `WaitHandle.WaitOneNoCheck`, and `LowLevelLifoSemaphore.WaitNative`
dominating at ~76% combined, which on its own would be a static cost — the time-delayed growth
points to several accumulation and GC-pressure issues that compound over long sessions.

---

## Issues & Fixes

### Issue 1 — `CheckKeyboardAsync` fire-and-forget task storm
**File:** `Jammer.Core/src/Start.cs:179, 224`

```csharp
_ = CheckKeyboardAsync();  // called ~1000×/sec, never awaited
```

`Loop()` runs with a `Thread.Sleep(1)` floor, spinning ~1000 iterations/second. Each iteration
discards the returned `Task` object without awaiting it. Even though the method returns quickly
when no key is available, ~1000 `Task` state-machine allocations/second saturate the thread pool
over time and grow GC pressure.

**Fix:** Add a `static bool _keyboardBusy` guard so only one call is in-flight at a time:

```csharp
if (!_keyboardBusy)
{
    _keyboardBusy = true;
    _ = CheckKeyboardAsync().ContinueWith(_ => _keyboardBusy = false);
}
```

---

### Issue 2 — `InitializeSharpHook` leaks OS hook objects
**File:** `Jammer.Core/src/Keyboard.cs:1142`

```csharp
public static async void InitializeSharpHook()
{
    var hook = new TaskPoolGlobalHook();  // never disposed
    hook.KeyReleased += OnKeyReleased;    // never unsubscribed
    await hook.RunAsync();
}
```

`async void` means the task is untracked and exceptions are swallowed. If `StartUp()` is called
more than once, a new OS-level keyboard hook and listener thread are created without disposing
the previous one.

**Fix:**
- Store the hook in `static TaskPoolGlobalHook? _hook`.
- Guard against re-initialization: if `_hook` is already running, return early.
- Dispose the old hook before creating a new one.
- Change signature to `async Task` and store/track the task.

---

### Issue 3 — `Log.log` unbounded O(n) array growth
**File:** `Jammer.Core/src/Log.cs:5, 18, 21`

```csharp
public static string[] log = Array.Empty<string>();
// on every log call:
log = log.Append(...).ToArray();  // O(n) full copy each time
```

Every log event allocates a new array one element larger than the previous, copying all previous
entries. After thousands of entries the copy cost per-call grows measurably and all old arrays
are queued for GC.

**Fix:** Replace `string[]` with `List<string>` capped at 1000 entries. When the cap is hit,
remove from the front (rolling window). Eliminates both the O(n) copy and unbounded growth.

---

### Issue 4 — `Bass.ChannelSetSync` potentially double-registers end callback
**File:** `Jammer.Core/src/Play.cs:928`

```csharp
Bass.ChannelSetSync(Utils.CurrentMusic, SyncFlags.End, 0, (a, b, c, d) =>
{
    MaybeNextSong();
    ...
}, IntPtr.Zero);
```

`StartPlaying()` is called on every song start, seek-restart, and effects reload. Each call
registers a new `SyncFlags.End` callback without removing any previously registered sync on the
same channel, potentially causing `MaybeNextSong()` to fire multiple times at song end.

**Fix:** Store the returned sync handle in `static int _endSyncHandle`. Before each
`Bass.ChannelSetSync` call, remove the previous handle:

```csharp
if (_endSyncHandle != 0)
    Bass.ChannelRemoveSync(Utils.CurrentMusic, _endSyncHandle);
_endSyncHandle = Bass.ChannelSetSync(...);
```

---

### Issue 5 — Per-frame 164 KB `float[]` allocation in visualizer
**File:** `Jammer.Core/src/Visual.cs:62`

```csharp
var fftData = new float[_bufferSize];  // 41000 floats = 164 KB, every 10–35 ms
```

`GetSongVisual()` is called by the dedicated `EqualizerLoop` thread at up to ~100 fps. A fresh
164 KB array is allocated every frame and never reused. Over an hour with the visualizer on,
this is hundreds of thousands of large-object-heap allocations, causing sustained and growing GC
pressure.

**Fix:** Hoist `fftData` to `static float[] _fftBuffer` on `Visual`, initialized once to
`bufferSize`. Reuse it on every call. `EqualizerLoop` is single-threaded so no concurrency
concern.

---

### Issue 6 — `AnsiConsole.Create()` called on every render tick but result unused
**File:** `Jammer.Core/src/TUI.cs:29–31`

```csharp
var ansiConsoleSettings = new AnsiConsoleSettings();
AnsiConsole.Profile.Encoding = System.Text.Encoding.UTF8;
var ansiConsole = AnsiConsole.Create(ansiConsoleSettings);  // never used
```

`DrawPlayer()` is triggered at minimum every second and on every keypress. A new `IAnsiConsole`
object is created and immediately abandoned — the rest of the method uses the static
`AnsiConsole.*` API. Over one hour this produces thousands of unnecessary allocations.

**Fix:** Delete lines 29 and 31. Move line 30 (`AnsiConsole.Profile.Encoding`) to app startup
(once, in `Start.StartUp()`).

---

### Issue 7 — `JRead.JRead.History` grows unboundedly
**File:** `Jammer.Core/src/Message.cs:303–304`

```csharp
if (!string.IsNullOrEmpty(input))
    JRead.JRead.History.Add(input);
```

Every string typed into any `Message.Input()` prompt is appended to the history list with no
cap. In interactive long sessions (searches, renames, settings) this list grows indefinitely.

**Fix:** After `Add`, trim the front of the list if `History.Count > 500`.

---

### Issue 8 — `EqualizerLoop` immortal background thread
**File:** `Jammer.Core/src/Start.cs:325`

```csharp
private static void EqualizerLoop()
{
    while (true)  // no shutdown check
    {
        ...
        Thread.Sleep(Visual.refreshTime);
    }
}
```

Unlike `Loop()` which checks `while (LoopRunning)`, `EqualizerLoop` runs a literal `while(true)`
with no way to exit. Even after the main loop shuts down, this thread stays alive, continuously
setting `drawVisualizer = true` and blocking garbage collection of anything it references.

**Fix:** Change `while (true)` → `while (LoopRunning)` to align with the existing shutdown
convention.

---

## Priority Order for Implementation

| # | Severity | File | Lines |
|---|----------|------|-------|
| 5 | Critical | `Visual.cs` | 62 |
| 1 | Critical | `Start.cs` | 179, 224 |
| 3 | High | `Log.cs` | 5, 18, 21 |
| 2 | High | `Keyboard.cs` | 1142 |
| 4 | Medium | `Play.cs` | 928 |
| 6 | Medium | `TUI.cs` | 29–31 |
| 8 | Medium | `Start.cs` | 325 |
| 7 | Low | `Message.cs` | 303–304 |

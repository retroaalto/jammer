# Jammer CPU Profiling Report

**Date:** 2026-04-20  
**Process:** Jammer (PID 189892)  
**Runtime:** .NET 8.0.25  
**Tool:** dotnet-trace 9.0.661903  
**Trace duration:** 120 seconds  
**Output file:** `jammer-final.speedscope.json`

---

## Profile Summary

| Metric | Value |
|--------|-------|
| Threads profiled | 16 |
| Total events | 69,974 |
| Unique frames (functions) | 417 |
| Most active thread | Thread (189927) — 67,584 events (~96%) |

---

## Top CPU Consumers

| Rank | % CPU (Exclusive) | Function |
|------|-------------------|----------|
| 1 | 29.72% | `LowLevelLifoSemaphore.WaitNative` |
| 2 | 23.54% | `WaitHandle.WaitOneNoCheck` |
| 3 | 22.94% | `Thread.Sleep` |
| 4 | 11.77% | Missing symbol (native code) |
| 5 | 11.77% | `UioHookProvider.Run()` |
| 6 | 0.26% | `Monitor.Wait` |
| 7 | 0.16% | `Int32[].Resize` |
| 8 | 0.06% | `__Canon[].set_Capacity` |
| 9 | 0.05% | `SyncTextReader.get_KeyAvailable` |
| 10 | 0.03% | `Console!Interop+Sys.Write` |

---

## Analysis

### Primary CPU Driver: `UioHookProvider.Run()` / `Start.Loop()`

`UioHookProvider.Run()` (inclusive: 11.77%) calls `Start.Loop()`, which appears to run a tight event loop for input hook processing (keyboard/mouse hooks via libuiohook). This loop is responsible for the sustained high CPU.

### Blocking Primitives (~76% combined)

The majority of CPU time is spent in blocking/synchronization primitives:

- **`LowLevelLifoSemaphore.WaitNative` (29.72%)** — low-level semaphore wait used by the .NET ThreadPool. Indicates threads frequently waiting for work, waking, and going back to sleep.
- **`WaitHandle.WaitOneNoCheck` (23.54%)** — higher-level blocking wait, likely coordinating between the hook loop and other components.
- **`Thread.Sleep` (22.94%)** — explicit sleep calls, possibly in a polling loop.

These together suggest the application is **spin-polling** or using a tight sleep-wake cycle rather than a true event-driven model, causing the OS to constantly context-switch threads — which registers as high CPU even though little real work is being done.

### Missing Symbol (11.77%)

A significant portion of stack frames is unresolved native code. This is likely the native `libuiohook` library underpinning `UioHookProvider`, which uses OS-level input event APIs.

---

## Conclusion

The ~61% CPU usage is primarily caused by `UioHookProvider.Run()` running a polling loop (via `Start.Loop()`) that repeatedly sleeps and wakes threads using `Thread.Sleep`, `WaitHandle`, and semaphore waits. This is a **busy-wait / polling pattern** rather than an efficient event-driven design. The underlying native input hook library (libuiohook) may also contribute via its own internal loop.

---

## Speedscope Visualization

Load `jammer-final.speedscope.json` at **https://www.speedscope.app** for interactive flame graph exploration.

The most interesting thread to inspect is **Thread (189927)**, which accounts for ~96% of all recorded events.

# Developer Notes

## Profiling with pprof

jammer-go includes an opt-in HTTP profiling endpoint powered by Go's standard
`net/http/pprof` package. It is **off by default** and only activated when you
pass `--pprof` at startup.

### Starting the pprof server

```bash
# Default address: 127.0.0.1:6060
./jammer-go --pprof

# Custom address
./jammer-go --pprof 127.0.0.1:9090
```

The server binds to localhost only and is never exposed to the network.
A log line confirms it started:

```
pprof server listening on http://127.0.0.1:6060/debug/pprof/
```

---

### Available endpoints

| Endpoint | What it shows |
|---|---|
| `/debug/pprof/` | Index of all profiles |
| `/debug/pprof/profile?seconds=N` | CPU profile for N seconds |
| `/debug/pprof/heap` | Heap (memory) allocations |
| `/debug/pprof/goroutine` | All goroutine stacks |
| `/debug/pprof/trace?seconds=N` | Execution trace for N seconds |
| `/debug/pprof/allocs` | Past memory allocations |
| `/debug/pprof/block` | Goroutine blocking events |
| `/debug/pprof/mutex` | Mutex contention |

---

### Diagnosing high CPU usage

**Step 1 — capture a CPU profile (30 seconds)**

```bash
go tool pprof http://127.0.0.1:6060/debug/pprof/profile?seconds=30
```

**Step 2 — explore interactively**

```
(pprof) top10          # top 10 functions by CPU time
(pprof) list <func>    # annotated source for a specific function
(pprof) web            # open flamegraph in browser (requires graphviz)
```

**Step 3 — browser UI with flamegraph**

```bash
# Save profile first
curl -o cpu.prof "http://127.0.0.1:6060/debug/pprof/profile?seconds=30"

# Open interactive UI
go tool pprof -http=:8080 cpu.prof
```

Then open `http://localhost:8080` — navigate to **Flame Graph** for the most
useful view of where CPU time is spent.

---

### Diagnosing high memory / GC pressure

```bash
# Heap profile
curl -o heap.prof http://127.0.0.1:6060/debug/pprof/heap
go tool pprof -http=:8080 heap.prof

# Enable GC tracing at startup (no --pprof needed)
GODEBUG=gctrace=1 ./jammer-go
```

---

### Goroutine / scheduler issues

```bash
# Execution trace (5 seconds)
curl -o trace.out "http://127.0.0.1:6060/debug/pprof/trace?seconds=5"
go tool trace trace.out   # opens browser UI

# Goroutine dump (text)
curl http://127.0.0.1:6060/debug/pprof/goroutine?debug=2

# Scheduler trace at startup
GODEBUG=schedtrace=1000 ./jammer-go   # prints stats every 1 second
```

---

### Prerequisites

```bash
# graphviz (needed for 'web' command and some pprof views)
sudo apt install graphviz   # Debian/Ubuntu
sudo dnf install graphviz   # Fedora

# Go tools (already present if Go is installed)
go tool pprof --help
go tool trace --help
```

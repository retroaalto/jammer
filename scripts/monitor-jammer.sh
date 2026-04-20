#!/usr/bin/env bash
# monitor-jammer.sh — OS-level CPU spike monitor for the Jammer process
#
# Uses /proc/<pid>/task/<tid>/stat to read per-thread CPU jiffies directly,
# requires no root and no external tools (pidstat not available on this system).
#
# What it does:
#   - Polls all threads of the Jammer process every INTERVAL seconds
#   - Computes per-thread CPU% since last poll (utime + stime delta / wall time)
#   - On every sample, logs a one-line summary of total CPU% and the top threads
#   - When total CPU% exceeds SPIKE_THRESHOLD, prints a detailed thread breakdown
#     and optionally invokes `perf record` (requires sudo / perf_event_paranoid<=1)
#   - Output is timestamped and tee'd to a log file for post-mortem review
#
# Usage:
#   ./scripts/monitor-jammer.sh [options]
#
# Options:
#   -i SECONDS     Poll interval (default: 1)
#   -t PERCENT     Spike threshold, total process CPU% (default: 40)
#   -d SECONDS     Duration to run, 0 = forever (default: 0)
#   -p             Enable perf record on spike (needs sudo, perf_event_paranoid<=1)
#   -P SECONDS     How long to perf record when spike detected (default: 15)
#   -o DIR         Output directory for logs and perf data (default: /tmp/jammer-monitor)
#   -w PROCESS     Process name to watch (default: Jammer)
#   -h             Show this help

set -uo pipefail

# ── defaults ──────────────────────────────────────────────────────────────────
INTERVAL=1
SPIKE_THRESHOLD=40
DURATION=0
USE_PERF=0
PERF_DURATION=15
OUTDIR="/tmp/jammer-monitor"
PROCESS_NAME="Jammer"

CLK_TCK=$(getconf CLK_TCK)   # jiffies per second (usually 100)
NCPU=$(nproc)                 # number of logical CPUs (for scaling %)

# ── argument parsing ───────────────────────────────────────────────────────────
usage() {
    grep '^#' "$0" | grep -v '#!/' | sed 's/^# \{0,1\}//'
    exit 0
}

while getopts "i:t:d:pP:o:w:h" opt; do
    case $opt in
        i) INTERVAL=$OPTARG ;;
        t) SPIKE_THRESHOLD=$OPTARG ;;
        d) DURATION=$OPTARG ;;
        p) USE_PERF=1 ;;
        P) PERF_DURATION=$OPTARG ;;
        o) OUTDIR=$OPTARG ;;
        w) PROCESS_NAME=$OPTARG ;;
        h) usage ;;
        *) echo "Unknown option -$OPTARG" >&2; exit 1 ;;
    esac
done

mkdir -p "$OUTDIR"
LOGFILE="$OUTDIR/monitor-$(date +%Y%m%d-%H%M%S).log"

# ── helpers ────────────────────────────────────────────────────────────────────
ts() { date '+%H:%M:%S'; }

log() { echo "$(ts)  $*" | tee -a "$LOGFILE"; }

# Find the PID of the watched process. Exits if not found.
find_pid() {
    pgrep -x "$PROCESS_NAME" 2>/dev/null | head -1 || true
}

# Read /proc/<pid>/task/<tid>/stat and extract utime (field 14) + stime (field 15).
# Returns the sum in jiffies.
read_thread_jiffies() {
    local stat_file="$1"
    # The comm field (field 2) may contain spaces and parentheses, so we strip it
    # by removing everything up to and including the closing ')' before splitting.
    local raw
    raw=$(cat "$stat_file" 2>/dev/null) || return 1
    local stripped="${raw#*) }"   # everything after ") "
    local fields=($stripped)
    # After stripping, field indices shift by 2 (fields 14,15 become 11,12 in 0-based).
    local utime="${fields[11]}"
    local stime="${fields[12]}"
    echo $(( utime + stime ))
}

# Read thread name from /proc/<pid>/task/<tid>/comm (kernel name, max 15 chars).
read_thread_name() {
    local comm_file="$1"
    cat "$comm_file" 2>/dev/null | tr -d '\n' || echo "?"
}

# ── perf record trigger ────────────────────────────────────────────────────────
# Called once when a spike is detected. Runs perf in the background so monitoring
# continues uninterrupted. Only fires if -p flag was given and perf is available.
perf_triggered=0

trigger_perf() {
    local pid="$1"
    if [[ $USE_PERF -eq 0 ]]; then return; fi
    if [[ $perf_triggered -eq 1 ]]; then return; fi
    if ! command -v perf &>/dev/null; then
        log "PERF  perf not found, skipping"
        return
    fi

    local perf_out="$OUTDIR/perf-$(date +%Y%m%d-%H%M%S).data"
    log "PERF  Starting perf record for ${PERF_DURATION}s → $perf_out"
    log "PERF  (requires perf_event_paranoid<=1 or sudo)"
    perf_triggered=1

    # Run in background; suppress its own output to not clutter the log
    (
        perf record -g --call-graph dwarf -p "$pid" \
            -o "$perf_out" -- sleep "$PERF_DURATION" \
            >>"$LOGFILE" 2>&1
        echo "$(ts)  PERF  Done. Analyse with: perf report -i $perf_out" \
            | tee -a "$LOGFILE"
        echo "$(ts)  PERF  Or flamegraph: perf script -i $perf_out | stackcollapse-perf.pl | flamegraph.pl > ${perf_out%.data}.svg" \
            | tee -a "$LOGFILE"
    ) &
}

# ── main monitoring loop ───────────────────────────────────────────────────────
log "Starting Jammer CPU monitor"
log "  Process:   $PROCESS_NAME"
log "  Interval:  ${INTERVAL}s"
log "  Threshold: ${SPIKE_THRESHOLD}%  (spike alert)"
log "  CPUs:      $NCPU  (CLK_TCK=$CLK_TCK)"
log "  Log:       $LOGFILE"
[[ $DURATION -gt 0 ]] && log "  Duration:  ${DURATION}s" || log "  Duration:  until Ctrl-C"
[[ $USE_PERF -eq 1 ]] && log "  Perf:      enabled (${PERF_DURATION}s capture on first spike)"
log "────────────────────────────────────────────────────────────────"

# Associative maps: tid -> previous jiffies, tid -> thread name
declare -A prev_jiffies=()
declare -A thread_names=()

start_time=$(date +%s)
iteration=0
prev_wall=$(date +%s%N)  # nanoseconds for sub-second precision

wait_for_process() {
    # NOTE: log() uses tee which writes to stdout; since this function's stdout
    # is captured by callers via $(...), all log output must be redirected to
    # stderr so that only the bare PID is captured.
    log "Waiting for process '$PROCESS_NAME'..." >&2
    while true; do
        local pid
        pid=$(find_pid)
        if [[ -n "$pid" ]]; then
            log "Found $PROCESS_NAME PID=$pid" >&2
            echo "$pid"
            return
        fi
        sleep 2
    done
}

PID=$(wait_for_process)

while true; do
    # ── check process still alive ──────────────────────────────────────────────
    if [[ ! -d "/proc/$PID" ]]; then
        log "Process $PID exited. Waiting for restart..."
        prev_jiffies=()
        thread_names=()
        perf_triggered=0
        PID=$(wait_for_process)
        prev_wall=$(date +%s%N)
        continue
    fi

    # ── duration check ─────────────────────────────────────────────────────────
    if [[ $DURATION -gt 0 ]]; then
        elapsed=$(( $(date +%s) - start_time ))
        if [[ $elapsed -ge $DURATION ]]; then
            log "Duration ${DURATION}s reached, exiting."
            break
        fi
    fi

    sleep "$INTERVAL"

    now_wall=$(date +%s%N)
    wall_ns=$(( now_wall - prev_wall ))
    prev_wall=$now_wall
    # wall time in jiffies
    wall_jiffies=$(( wall_ns * CLK_TCK / 1000000000 ))
    [[ $wall_jiffies -eq 0 ]] && wall_jiffies=1  # avoid div by zero

    # ── read all threads ───────────────────────────────────────────────────────
    declare -A cur_jiffies=()
    declare -A cur_names=()

    task_dir="/proc/$PID/task"
    for tid_path in "$task_dir"/*/; do
        tid="${tid_path%/}"
        tid="${tid##*/}"
        stat_file="$tid_path/stat"
        comm_file="$tid_path/comm"

        jiffies=$(read_thread_jiffies "$stat_file") || continue
        name=$(read_thread_name "$comm_file")
        cur_jiffies[$tid]=$jiffies
        cur_names[$tid]=$name
    done

    # ── compute deltas ─────────────────────────────────────────────────────────
    total_delta=0
    declare -A deltas=()

    for tid in "${!cur_jiffies[@]}"; do
        prev=${prev_jiffies[$tid]:-${cur_jiffies[$tid]}}
        delta=$(( cur_jiffies[$tid] - prev ))
        [[ $delta -lt 0 ]] && delta=0   # counter wrap guard
        deltas[$tid]=$delta
        total_delta=$(( total_delta + delta ))
    done

    # CPU% = delta_jiffies / wall_jiffies * 100
    # (reflects % of one CPU core; values >100 mean multi-core usage)
    total_pct=$(( total_delta * 100 / wall_jiffies ))

    # ── one-line summary ───────────────────────────────────────────────────────
    # Build top-3 threads by delta for the summary line
    top3=""
    count=0
    while IFS= read -r line; do
        tid="${line##* }"
        delta="${deltas[$tid]:-0}"
        pct=$(( delta * 100 / wall_jiffies ))
        name="${cur_names[$tid]:-?}"
        top3+=" | ${name}(${tid})=${pct}%"
        count=$(( count + 1 ))
        [[ $count -ge 3 ]] && break
    done < <(
        for tid in "${!deltas[@]}"; do
            echo "${deltas[$tid]} $tid"
        done | sort -rn
    )

    marker="     "
    [[ $total_pct -ge $SPIKE_THRESHOLD ]] && marker="SPIKE"

    log "$marker  CPU=${total_pct}%  threads=${#cur_jiffies[@]}${top3}"

    # ── spike detail ───────────────────────────────────────────────────────────
    if [[ $total_pct -ge $SPIKE_THRESHOLD ]]; then
        log "────── SPIKE DETAIL ──────────────────────────────────────────"
        log "  TID        NAME                  CPU%   TOTAL_JIFFIES"

        # Sort threads by delta descending
        while IFS= read -r line; do
            delta="${line%% *}"
            tid="${line##* }"
            pct=$(( delta * 100 / wall_jiffies ))
            name="${cur_names[$tid]:-?}"
            total_j="${cur_jiffies[$tid]:-0}"
            printf "  %-10s %-22s %4d%%  %d\n" "$tid" "$name" "$pct" "$total_j" \
                | tee -a "$LOGFILE"
        done < <(
            for tid in "${!deltas[@]}"; do
                echo "${deltas[$tid]} $tid"
            done | sort -rn | head -20
        )

        log "  /proc/$PID/task/ thread count: $(ls "$task_dir" | wc -l)"
        log "──────────────────────────────────────────────────────────────"

        trigger_perf "$PID"
    fi

    # ── update previous state ─────────────────────────────────────────────────
    prev_jiffies=()
    thread_names=()
    for tid in "${!cur_jiffies[@]}"; do
        prev_jiffies[$tid]=${cur_jiffies[$tid]}
        thread_names[$tid]=${cur_names[$tid]}
    done

    iteration=$(( iteration + 1 ))
done

log "Monitor exited after $iteration samples."

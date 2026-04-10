#!/usr/bin/env bash
# monitor.sh — external monitor for the jammer process and its children/threads.
# Usage:  ./scripts/monitor.sh [--interval N] [--output FILE] [--sudo]
# Defaults: interval=2s, output=~/.jammer/monitor.log

set -uo pipefail

# ── defaults ──────────────────────────────────────────────────────────────────
INTERVAL=2
OUTPUT="${HOME}/.jammer/monitor.log"
USE_SUDO=0
CLK_TCK=$(getconf CLK_TCK 2>/dev/null || echo 100)

# ── argument parsing ───────────────────────────────────────────────────────────
while [[ $# -gt 0 ]]; do
    case "$1" in
        --interval|-i) INTERVAL="$2"; shift 2 ;;
        --output|-o)   OUTPUT="$2";   shift 2 ;;
        --sudo)        USE_SUDO=1;    shift   ;;
        --help|-h)
            echo "Usage: $0 [--interval N] [--output FILE] [--sudo]"
            echo "  --interval N   Poll every N seconds (default: 2)"
            echo "  --output FILE  Log file path (default: ~/.jammer/monitor.log)"
            echo "  --sudo         Enable privileged features (perf stat)"
            exit 0 ;;
        *) echo "Unknown option: $1"; exit 1 ;;
    esac
done

# ── helpers ────────────────────────────────────────────────────────────────────
log() { echo "$1" | tee -a "$OUTPUT"; }

find_pid() {
    # Prefer the inner Jammer binary (child of the AppImage wrapper).
    local pid
    pid=$(pgrep -n -f '/usr/bin/Jammer' 2>/dev/null || true)
    if [[ -z "$pid" ]]; then
        pid=$(pgrep -n -f '[Jj]ammer' 2>/dev/null || true)
    fi
    echo "$pid"
}

# Read a single /proc/<pid>/stat field (1-indexed).
proc_stat_field() {
    local pid=$1 field=$2
    awk "{print \$$field}" /proc/"$pid"/stat 2>/dev/null || echo 0
}

# Return total jiffies (utime+stime) for a pid or tid.
total_jiffies() {
    local statfile=$1
    awk '{print $14+$15}' "$statfile" 2>/dev/null || echo 0
}

# Associative arrays to hold previous sample values.
declare -A PREV_JIFFIES   # [pid_or_tid] = jiffies
declare -A PREV_WALL      # [pid_or_tid] = epoch_ns (via date +%s%N)

now_ns() { date +%s%N; }

# CPU% from two jiffies samples and wall-clock nanoseconds.
# Outputs single-core % (same as top default — can exceed 100% on multi-core).
cpu_percent() {
    local cur_j=$1 prev_j=$2 cur_ns=$3 prev_ns=$4
    local dj=$(( cur_j - prev_j ))
    local dns=$(( cur_ns - prev_ns ))
    if [[ $dns -le 0 || $dj -lt 0 ]]; then echo "?"; return; fi
    # dj jiffies / (dns nanoseconds / 1e9 seconds * CLK_TCK ticks/sec) * 100%
    awk "BEGIN { wall=$dns/1e9; ticks=$dj; hz=$CLK_TCK;
                 if(wall>0) printf \"%.1f\", ticks/hz/wall*100; else print \"?\"; }"
}

# ── startup ────────────────────────────────────────────────────────────────────
mkdir -p "$(dirname "$OUTPUT")"
: > "$OUTPUT"   # truncate / create

log "# jammer monitor started $(date '+%Y-%m-%d %H:%M:%S')"
log "# interval=${INTERVAL}s  output=${OUTPUT}  clk_tck=${CLK_TCK}  sudo=${USE_SUDO}"
log "# columns: timestamp PID cpu% mem_rss_KB threads fds"
log "# thread lines:  T <tid> <name> cpu%"
log "# child lines:   C <pid> <name> cpu%"
log "#"

echo "Watching for jammer... (Ctrl-C to stop)"
echo "Logging to: $OUTPUT"

# ── main loop ──────────────────────────────────────────────────────────────────
while true; do
    PID=$(find_pid)

    if [[ -z "$PID" ]]; then
        log "[$(date '+%H:%M:%S')] jammer not found — waiting..."
        sleep "$INTERVAL"
        continue
    fi

    # Verify /proc entry still exists (process may have just exited).
    if [[ ! -d /proc/$PID ]]; then
        log "[$(date '+%H:%M:%S')] PID=$PID vanished"
        unset PREV_JIFFIES PREV_WALL
        declare -A PREV_JIFFIES
        declare -A PREV_WALL
        sleep "$INTERVAL"
        continue
    fi

    TS=$(date '+%H:%M:%S')
    NOW=$(now_ns)

    # ── main process stat ──────────────────────────────────────────────────────
    STATLINE=$(cat /proc/$PID/stat 2>/dev/null || true)
    if [[ -z "$STATLINE" ]]; then sleep "$INTERVAL"; continue; fi

    CUR_J=$(echo "$STATLINE" | awk '{print $14+$15}')
    RSS_KB=$(awk '/VmRSS/{print $2}' /proc/$PID/status 2>/dev/null || echo 0)
    NUM_THREADS=$(awk '/Threads/{print $2}' /proc/$PID/status 2>/dev/null || echo 0)
    FDS=$(ls /proc/$PID/fd 2>/dev/null | wc -l || echo 0)
    STATE=$(echo "$STATLINE" | awk '{print $3}')

    if [[ -n "${PREV_JIFFIES[$PID]:-}" ]]; then
        PCPU=$(cpu_percent "$CUR_J" "${PREV_JIFFIES[$PID]}" "$NOW" "${PREV_WALL[$PID]}")
    else
        PCPU="?"
    fi
    PREV_JIFFIES[$PID]=$CUR_J
    PREV_WALL[$PID]=$NOW

    # Children (live subprocesses).
    mapfile -t CHILD_LINES < <(
        while read -r CPID CCOMM; do
            [[ -z "$CPID" || "$CPID" == "PID" ]] && continue
            CJ=$(total_jiffies /proc/$CPID/stat)
            if [[ -n "${PREV_JIFFIES[c$CPID]:-}" ]]; then
                CC=$(cpu_percent "$CJ" "${PREV_JIFFIES[c$CPID]}" "$NOW" "${PREV_WALL[c$CPID]}")
            else
                CC="?"
            fi
            PREV_JIFFIES[c$CPID]=$CJ
            PREV_WALL[c$CPID]=$NOW
            echo "  C $CPID $CCOMM cpu=${CC}%"
        done < <(ps --ppid "$PID" -o pid=,comm= 2>/dev/null || true)
    )

    # ── per-thread breakdown ───────────────────────────────────────────────────
    mapfile -t THREAD_LINES < <(
        for TSTAT in /proc/$PID/task/*/stat; do
            [[ -f "$TSTAT" ]] || continue
            TID=$(echo "$TSTAT" | grep -o 'task/[0-9]*' | cut -d/ -f2)
            TCOMM=$(cat /proc/$PID/task/$TID/comm 2>/dev/null | tr -d '\n' || echo "?")
            TJ=$(awk '{print $14+$15}' "$TSTAT" 2>/dev/null || echo 0)
            KEY="t${TID}"
            if [[ -n "${PREV_JIFFIES[$KEY]:-}" ]]; then
                TC=$(cpu_percent "$TJ" "${PREV_JIFFIES[$KEY]}" "$NOW" "${PREV_WALL[$KEY]}")
            else
                TC="?"
            fi
            PREV_JIFFIES[$KEY]=$TJ
            PREV_WALL[$KEY]=$NOW
            # Only log threads that have ever consumed CPU.
            if [[ "$TC" != "0.0" && "$TC" != "?" ]] || [[ "$TJ" != "0" ]]; then
                printf "  T %-7s %-20s cpu=%s%%\n" "$TID" "$TCOMM" "$TC"
            fi
        done
    )

    # ── perf stat (sudo only, 1-second sample) ─────────────────────────────────
    PERF_LINE=""
    if [[ $USE_SUDO -eq 1 ]] && command -v perf &>/dev/null; then
        PERF_OUT=$(sudo perf stat -p "$PID" -e cycles,instructions,cache-misses \
                   --interval-print 1000 -- sleep 1 2>&1 | \
                   awk '/cycles|instructions|cache/{printf "%s=%s ", $3, $1}' || true)
        PERF_LINE="  perf: $PERF_OUT"
    fi

    # ── write log line ─────────────────────────────────────────────────────────
    {
        echo "[$TS] PID=$PID state=$STATE cpu=${PCPU}% rss=${RSS_KB}KB threads=$NUM_THREADS fds=$FDS"
        if [[ ${#CHILD_LINES[@]} -eq 0 ]]; then
            echo "  C [none]"
        else
            printf '%s\n' "${CHILD_LINES[@]}"
        fi
        printf '%s\n' "${THREAD_LINES[@]}"
        [[ -n "$PERF_LINE" ]] && echo "$PERF_LINE"
    } | tee -a "$OUTPUT"

    sleep "$INTERVAL"
done

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
find_pid() {
    local pid
    pid=$(pgrep -n -f '/usr/bin/Jammer' 2>/dev/null || true)
    if [[ -z "$pid" ]]; then
        pid=$(pgrep -n -f '[Jj]ammer' 2>/dev/null || true)
    fi
    echo "$pid"
}

# CPU% from two jiffies samples and wall-clock nanoseconds.
# Single-core % — same as top default (can exceed 100% on multi-core).
cpu_percent() {
    local cur_j=$1 prev_j=$2 cur_ns=$3 prev_ns=$4
    local dj=$(( cur_j - prev_j ))
    local dns=$(( cur_ns - prev_ns ))
    if [[ $dns -le 0 || $dj -lt 0 ]]; then echo "0.0"; return; fi
    awk "BEGIN { wall=$dns/1e9; ticks=$dj; hz=$CLK_TCK;
                 if(wall>0) printf \"%.1f\", ticks/hz/wall*100; else print \"0.0\"; }"
}

now_ns() { date +%s%N; }

# ── persistent state (survives across loop iterations in the same shell) ───────
declare -A PREV_JIFFIES=()
declare -A PREV_WALL=()

# ── startup ────────────────────────────────────────────────────────────────────
mkdir -p "$(dirname "$OUTPUT")"
: > "$OUTPUT"

{
    echo "# jammer monitor started $(date '+%Y-%m-%d %H:%M:%S')"
    echo "# interval=${INTERVAL}s  output=${OUTPUT}  clk_tck=${CLK_TCK}  sudo=${USE_SUDO}"
    echo "# P line: timestamp PID state cpu% rss threads fds"
    echo "# T line: tid name cpu%   (threads with >0 cpu or ever had cpu)"
    echo "# C line: child pid name cpu%"
    echo "#"
} | tee -a "$OUTPUT"

echo "Watching for jammer... (Ctrl-C to stop)"
echo "Logging to: $OUTPUT"

# ── main loop ──────────────────────────────────────────────────────────────────
while true; do
    PID=$(find_pid)

    if [[ -z "$PID" ]]; then
        echo "[$(date '+%H:%M:%S')] jammer not found — waiting..." | tee -a "$OUTPUT"
        sleep "$INTERVAL"
        continue
    fi

    if [[ ! -d /proc/$PID ]]; then
        echo "[$(date '+%H:%M:%S')] PID=$PID vanished" | tee -a "$OUTPUT"
        PREV_JIFFIES=()
        PREV_WALL=()
        sleep "$INTERVAL"
        continue
    fi

    TS=$(date '+%H:%M:%S')
    NOW=$(now_ns)

    # ── main process ──────────────────────────────────────────────────────────
    STATLINE=$(cat /proc/$PID/stat 2>/dev/null || true)
    [[ -z "$STATLINE" ]] && { sleep "$INTERVAL"; continue; }

    CUR_J=$(echo "$STATLINE" | awk '{print $14+$15}')
    STATE=$(echo "$STATLINE"  | awk '{print $3}')
    RSS_KB=$(awk '/VmRSS/{print $2}'  /proc/$PID/status 2>/dev/null || echo 0)
    NUM_THREADS=$(awk '/Threads/{print $2}' /proc/$PID/status 2>/dev/null || echo 0)
    FDS=$(ls /proc/$PID/fd 2>/dev/null | wc -l || echo 0)

    if [[ -n "${PREV_JIFFIES[$PID]:-}" ]]; then
        PCPU=$(cpu_percent "$CUR_J" "${PREV_JIFFIES[$PID]}" "$NOW" "${PREV_WALL[$PID]}")
    else
        PCPU="?"
    fi
    PREV_JIFFIES[$PID]=$CUR_J
    PREV_WALL[$PID]=$NOW

    # Build output in a string — no subshells, so PREV_* arrays stay writable.
    OUT="[$TS] PID=$PID state=$STATE cpu=${PCPU}% rss=${RSS_KB}KB threads=$NUM_THREADS fds=$FDS"$'\n'

    # ── per-thread breakdown ───────────────────────────────────────────────────
    HAS_THREAD=0
    for TSTAT in /proc/$PID/task/*/stat; do
        [[ -f "$TSTAT" ]] || continue
        TID="${TSTAT#/proc/$PID/task/}"; TID="${TID%/stat}"
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
        # Print thread if it has ever had jiffies (filters idle .NET pool threads)
        if [[ "$TJ" != "0" ]]; then
            OUT+=$(printf "  T %-7s %-20s cpu=%s%%\n" "$TID" "$TCOMM" "$TC")
            OUT+=$'\n'
            HAS_THREAD=1
        fi
    done
    [[ $HAS_THREAD -eq 0 ]] && OUT+="  T [none with cpu]"$'\n'

    # ── children ──────────────────────────────────────────────────────────────
    HAS_CHILD=0
    while read -r CPID CCOMM; do
        [[ -z "$CPID" || "$CPID" == "PID" ]] && continue
        CJ=$(awk '{print $14+$15}' /proc/$CPID/stat 2>/dev/null || echo 0)
        KEY="c${CPID}"
        if [[ -n "${PREV_JIFFIES[$KEY]:-}" ]]; then
            CC=$(cpu_percent "$CJ" "${PREV_JIFFIES[$KEY]}" "$NOW" "${PREV_WALL[$KEY]}")
        else
            CC="?"
        fi
        PREV_JIFFIES[$KEY]=$CJ
        PREV_WALL[$KEY]=$NOW
        OUT+=$(printf "  C %-7s %-20s cpu=%s%%\n" "$CPID" "$CCOMM" "$CC")
        OUT+=$'\n'
        HAS_CHILD=1
    done < <(ps --ppid "$PID" -o pid=,comm= 2>/dev/null || true)
    [[ $HAS_CHILD -eq 0 ]] && OUT+="  C [none]"$'\n'

    # ── perf stat (sudo only) ──────────────────────────────────────────────────
    if [[ $USE_SUDO -eq 1 ]] && command -v perf &>/dev/null; then
        PERF_OUT=$(sudo perf stat -p "$PID" -e cycles,instructions,cache-misses \
                   --interval-print 1000 -- sleep 1 2>&1 | \
                   awk '/cycles|instructions|cache/{printf "%s=%s ", $3, $1}' || true)
        OUT+="  perf: $PERF_OUT"$'\n'
    fi

    printf '%s' "$OUT" | tee -a "$OUTPUT"

    sleep "$INTERVAL"
done

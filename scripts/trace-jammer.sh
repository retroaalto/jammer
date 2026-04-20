#!/usr/bin/env bash
# trace-jammer.sh
# Attaches dotnet-trace to a running Jammer.CLI process for 60 seconds.
# If Jammer is already running, attaches immediately.
# Otherwise waits for it to appear, then attaches.
# Usage: ./trace-jammer.sh

export PATH="$PATH:$HOME/.dotnet/tools"

OUTFILE="jammer-trace-$(date +%Y%m%d-%H%M%S).nettrace"

find_pid() {
  dotnet-trace ps 2>/dev/null | grep -i "Jammer" | awk '{print $1}' | head -1
}

pid=$(find_pid)
if [ -n "$pid" ]; then
  echo "Jammer already running (PID $pid) — attaching immediately."
else
  echo "Waiting for Jammer.CLI process..."
  while true; do
    pid=$(find_pid)
    [ -n "$pid" ] && break
    sleep 0.1
  done
  echo "Found PID $pid."
fi

echo "Collecting trace for 60 seconds → $OUTFILE"
dotnet-trace collect -p "$pid" --duration 00:01:00 -o "$OUTFILE"

echo ""
echo "Trace saved: $OUTFILE"
echo "Convert to Speedscope with:"
echo "  dotnet-trace convert --format Speedscope $OUTFILE"

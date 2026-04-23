#!/bin/bash
# profile.sh — Build Jammer AppImage, launch it with a playlist, and collect a dotnet-trace.
#
# Usage:
#   ./scripts/profile.sh <playlist>
#
#   playlist   Playlist name or path passed to Jammer's --play flag.
#
# Output files are written to profiles/ in the repo root.
# After collection the trace is automatically converted to Speedscope format.
# Use Jammer normally, then press 'q' to quit. Conversion runs automatically.
#
# Example:
#   ./scripts/profile.sh MyPlaylist

set -e
DOTNET_CLI_TELEMETRY_OPTOUT=1

# Ensure dotnet and its global tools are on PATH
export DOTNET_ROOT="${DOTNET_ROOT:-$HOME/.dotnet}"
export PATH="$PATH:$DOTNET_ROOT:$DOTNET_ROOT/tools"
REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PROFILES_DIR="$REPO_ROOT/profiles"
TIMESTAMP="$(date +%Y%m%d-%H%M%S)"
TRACE_FILE="$PROFILES_DIR/jammer-$TIMESTAMP.nettrace"

# --- Arguments ---
PLAYLIST="${1:-}"

if [ -z "$PLAYLIST" ]; then
    echo "Usage: $(basename "$0") <playlist>"
    echo ""
    echo "  playlist   Playlist name or path (passed to --play)"
    echo ""
    echo "  Use Jammer normally, then press 'q' to quit."
    echo "  The trace will be converted to Speedscope format automatically."
    exit 1
fi

# --- Ensure profiles output directory exists ---
mkdir -p "$PROFILES_DIR"

# --- Ensure dotnet-trace is available ---
if ! command -v dotnet-trace &>/dev/null; then
    echo "[profile] dotnet-trace not found, installing..."
    dotnet tool install --global dotnet-trace
    export PATH="$PATH:$HOME/.dotnet/tools"
fi

if ! command -v dotnet-trace &>/dev/null; then
    echo "[profile] ERROR: dotnet-trace still not found after install."
    echo "          Make sure ~/.dotnet/tools is in your PATH."
    exit 1
fi

# --- Build AppImage ---
echo "[profile] Building Jammer AppImage (Release)..."
cd "$REPO_ROOT"
bash "$REPO_ROOT/scripts/build.sh"

# Locate the freshly built AppImage
APPIMAGE="$(ls -t "$REPO_ROOT"/jammer-*-x86_64.AppImage 2>/dev/null | head -1)"
if [ -z "$APPIMAGE" ]; then
    echo "[profile] ERROR: No AppImage found after build."
    exit 1
fi
echo "[profile] Using AppImage: $APPIMAGE"

# --- Start Jammer AppImage in the background ---
echo "[profile] Starting Jammer with playlist: $PLAYLIST"
"$APPIMAGE" --play "$PLAYLIST" </dev/tty &
APPIMAGE_PID=$!

echo "[profile] AppImage launcher PID: $APPIMAGE_PID"

# Trap: convert and kill Jammer whenever this script exits
convert_and_cleanup() {
    if [ -f "$TRACE_FILE" ]; then
        echo ""
        echo "[profile] Converting to Speedscope format..."
        # dotnet-trace derives the output name from the input: foo.nettrace -> foo.speedscope.json
        dotnet-trace convert "$TRACE_FILE" --format Speedscope
        ACTUAL_SPEEDSCOPE="${TRACE_FILE%.nettrace}.speedscope.json"
        echo ""
        echo "[profile] Done."
        echo "  nettrace:   $TRACE_FILE"
        echo "  speedscope: $ACTUAL_SPEEDSCOPE"
        echo ""
        echo "  Open the .speedscope.json at https://www.speedscope.app"
    else
        echo "[profile] WARNING: Trace file not found, conversion skipped."
    fi

    # Kill the AppImage launcher (and its children) if still running
    if kill -0 "$APPIMAGE_PID" 2>/dev/null; then
        echo "[profile] Stopping Jammer (PID $APPIMAGE_PID)..."
        kill "$APPIMAGE_PID" 2>/dev/null || true
    fi
}
trap convert_and_cleanup EXIT

# --- Wait for Jammer to be ready ---
echo "[profile] Waiting for Jammer to initialize (5s)..."
sleep 5

if ! kill -0 "$APPIMAGE_PID" 2>/dev/null; then
    echo "[profile] ERROR: Jammer AppImage exited before trace could start."
    exit 1
fi

# --- Find the actual .NET runtime PID ---
# The AppImage launcher wraps the real binary; dotnet-trace ps lists live .NET processes.
echo "[profile] Looking for Jammer .NET process..."
JAMMER_PID=""
for attempt in 1 2 3; do
    JAMMER_PID="$(dotnet-trace ps 2>/dev/null | grep -i 'Jammer' | awk '{print $1}' | head -1)"
    if [ -n "$JAMMER_PID" ]; then
        break
    fi
    echo "[profile]   Not found yet, retrying in 2s (attempt $attempt/3)..."
    sleep 2
done

if [ -z "$JAMMER_PID" ]; then
    echo "[profile] ERROR: Could not find Jammer .NET process via dotnet-trace ps."
    echo "          Running .NET processes:"
    dotnet-trace ps 2>/dev/null || true
    exit 1
fi

echo "[profile] Jammer .NET PID: $JAMMER_PID"

# --- Collect trace (runs until Jammer exits) ---
echo "[profile] Collecting trace -> $TRACE_FILE"
echo "[profile] Use Jammer normally, then press 'q' to quit. Conversion will run automatically."
dotnet-trace collect \
    --process-id "$JAMMER_PID" \
    --output "$TRACE_FILE" </dev/null

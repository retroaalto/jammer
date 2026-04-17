#!/usr/bin/env bash
set -e
cd "$(dirname "$0")"
echo "Building jammer-go..."
go build -o jammer-go .
echo "Starting..."
REPO_ROOT="$(cd .. && pwd)"
export LD_LIBRARY_PATH="$REPO_ROOT/libs/linux/x86_64${LD_LIBRARY_PATH:+:$LD_LIBRARY_PATH}"
exec ./jammer-go "$@"

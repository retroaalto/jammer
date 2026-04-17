#!/usr/bin/env bash
set -e
cd "$(dirname "$0")"
echo "Building jammer-go..."
go build -o jammer-go .
echo "Done: $(pwd)/jammer-go"

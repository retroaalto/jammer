#!/bin/bash

# Set BASS library path for Linux
export LD_LIBRARY_PATH=./libs/linux/x86_64:$LD_LIBRARY_PATH

# Run dotnet test
dotnet test

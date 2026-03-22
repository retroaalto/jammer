# Scripts

This directory contains helper scripts for building, running, packaging, and local development tasks.

## Local .NET install on unsupported Linux distros

If your distro does not provide a usable `dotnet` package, you can use `dotnet-install.sh` to install a local .NET SDK without changing system packages.

`dotnet-install.sh` is the official Microsoft install script copied into this repository for convenience.

Example:

```bash
./scripts/dotnet-install.sh --channel 8.0 --install-dir "$HOME/.dotnet"
export DOTNET_ROOT="$HOME/.dotnet"
export PATH="$DOTNET_ROOT:$PATH"
```

After that, you can build Jammer normally:

```bash
dotnet build jammer.sln
```

# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Jammer is a lightweight CLI music player written in C# (.NET 8) that supports playing songs from local files, YouTube, and SoundCloud. It features a terminal-based user interface built with Spectre.Console and uses BASS audio library for playback.

## Architecture

The project is organized as a .NET solution with two main projects:

- **Jammer.CLI**: Entry point console application that initializes the program
- **Jammer.Core**: Core library containing all business logic and functionality

### Key Components

- **Start.cs**: Main application state machine and initialization (`Start.Run()`)
- **Play.cs**: Audio playback engine using BASS library, handles different audio formats
- **Download.cs**: Downloads songs from YouTube/SoundCloud using YoutubeExplode/SoundCloudExplode
- **TUI.cs**: Terminal user interface rendering with Spectre.Console
- **Playlists.cs**: Playlist management (.jammer format)
- **Songs.cs**: Song file management and operations
- **Preferences.cs**: Settings and configuration management
- **Keybindings.cs**: Keyboard input handling with SharpHook
- **Themes.cs**: UI theming system
- **Effects.cs**: Audio effects (reverb, echo, etc.)

## Build Commands

### Development
```bash
# Run for development
dotnet run --project Jammer.CLI -- [args]

# Set library path on Linux
export LD_LIBRARY_PATH=/path/to/libs/linux/x86_64:$LD_LIBRARY_PATH
```

### Production Builds

#### Windows
```bash
dotnet publish -r win10-x64 -c Release /p:PublishSingleFile=true -p:DefineConstants="CLI_UI" --self-contained
```

#### Linux
```bash
dotnet publish -r linux-x64 -c Release /p:PublishSingleFile=true -p:UseForms=false -p:DefineConstants="CLI_UI" --self-contained
```

#### Linux AppImage
```bash
./build.sh
```

## Dependencies

Key NuGet packages:
- **ManagedBass**: BASS audio library wrapper for .NET
- **Spectre.Console**: Rich terminal UI framework
- **YoutubeExplode**: YouTube video/playlist downloading
- **SoundCloudExplode**: SoundCloud track downloading
- **SharpHook**: Low-level keyboard hooking
- **Newtonsoft.Json**: JSON serialization
- **TagLibSharp**: Audio file metadata
- **PuppeteerSharp**: SoundCloud client ID extraction

## Important Notes

- The application requires BASS audio libraries (libbass.so, bass.dll) in the appropriate libs/ subdirectory
- On Linux, LD_LIBRARY_PATH must include the BASS library path
- User data is stored in `~/jammer` (Linux) or `%USERPROFILE%\jammer` (Windows)
- Settings, playlists, themes, and downloaded songs are stored in the jammer directory
- The main state machine uses `MainStates` enum (idle, play, playing, pause, stop, next, previous)
- Environment variables: `JAMMER_CONFIG_PATH`, `JAMMER_SONGS_PATH`, `JAMMER_PLAYLISTS_PATH`

## File Structure

- `/libs/`: Platform-specific BASS audio libraries
- `/locales/`: Translation files (.ini format)
- `/themes/`: Theme JSON files
- `/example/`: Sample playlist files
- `/nsis/`: Windows installer scripts

## Testing

No formal test framework is configured. Testing is done manually using `testAppImage.sh` for AppImage builds.
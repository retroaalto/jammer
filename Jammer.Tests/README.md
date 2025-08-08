# Jammer.Tests

Comprehensive test suite for the Jammer CLI music player.

## Structure

- **Unit/**: Fast, isolated tests with no external dependencies
  - **Core/**: Tests for Jammer.Core library components
  - **CLI/**: Tests for Jammer.CLI application components
- **Integration/**: Component interaction tests with mocked external services
- **E2E/**: End-to-end tests spawning the CLI as a separate process
- **TestUtilities/**: Shared test infrastructure
  - **Builders/**: Object creation patterns (SongBuilder, etc.)
  - **Doubles/**: Test doubles (NullAudioPlayer, mocks, etc.)
  - **Fixtures/**: Test data and constants

## Running Tests

```bash
# Run all unit tests (default)
dotnet test

# Run with coverage
dotnet test --collect:"XPlat Code Coverage"

# Run specific category
dotnet test --filter "Category=Unit"

# Exclude integration/E2E tests
dotnet test --filter "Category!=Integration&Category!=E2E"

# Use custom settings
dotnet test --settings test.runsettings
```

## Test Categories

Tests are organized by traits:
- `Unit`: Fast, isolated tests
- `Integration`: Component interaction tests  
- `E2E`: End-to-end scenarios
- `Platform`: Platform-specific tests (Windows/Linux)

## Key Testing Principles

1. **Unit tests avoid BASS library dependencies** using NullAudioPlayer
2. **External APIs are mocked** using RichardSzalay.MockHttp
3. **File system operations** use System.IO.Abstractions for testability
4. **Spectre.Console output** is captured using Spectre.Console.Testing
5. **Cross-platform compatibility** is verified with platform-specific tests

## Next Steps

- Implement tests for Utils.cs helper functions
- Add tests for Song.cs data structures
- Create tests for Args.cs command-line parsing
- Add integration tests for Download.cs with mocked APIs
- Implement audio playback tests with NullAudioPlayer
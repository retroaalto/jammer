# Testing Guidelines

This document provides patterns for consistent test implementation, mock setup best practices, and null handling standards for the Jammer test suite.

## Overview

This guide establishes best practices for maintaining test quality and preventing future interface divergence issues.

## Testing Patterns

### Mock Setup Best Practices

#### MockAudioPlayer Usage

The `MockAudioPlayer` class provides a complete mock implementation of the `IAudioPlayer` interface. Always follow these patterns:

```csharp
// Standard setup
var mockPlayer = new MockAudioPlayer();
mockPlayer.Reset(); // Clear any previous state

// Configure behavior
mockPlayer.ShouldFailStreamCreation = false;
mockPlayer.ShouldFailPlayback = false;

// Use in tests
var handle = mockPlayer.CreateStream("test.mp3", 0, 0, BassFlags.Default);
mockPlayer.ChannelPlay(handle);

// Verify calls
Assert.True(mockPlayer.WasMethodCalled("CreateStream"));
Assert.True(mockPlayer.WasMethodCalled("ChannelPlay"));
```

#### Method Call Verification

Use the `WasMethodCalled(string methodName)` method to verify mock interactions:

```csharp
// Verify specific method calls
Assert.True(mockPlayer.WasMethodCalled("ChannelPlay"));

// Check call parameters via MethodCallLog
var playCall = mockPlayer.MethodCallLog.FirstOrDefault(call => call.Contains("ChannelPlay"));
Assert.NotNull(playCall);
Assert.Contains("1000", playCall); // Check handle value
```

#### Test Data Management

Create test data in a consistent, isolated manner:

```csharp
[Fact]
public void TestName_Condition_ExpectedBehavior()
{
    // Arrange
    var mockPlayer = new MockAudioPlayer();
    var testSong = new Song { URI = "test.mp3", Title = "Test Song" };
    
    // Act
    var result = SomeOperation(mockPlayer, testSong);
    
    // Assert
    Assert.NotNull(result);
    mockPlayer.Reset(); // Clean up if needed
}
```

### Null Handling Standards

#### Null-Conditional Operators

Use null-conditional operators to safely access potentially null properties:

```csharp
// Good: Safe property access
themes.selectedTheme?.Border

// Good: Safe method calls
result?.SomeMethod()

// Avoid: Direct null dereference
themes.selectedTheme.Border // Can throw NullReferenceException
```

#### Explicit Null Assertions

When you're confident a value shouldn't be null, use explicit assertions:

```csharp
// Test scenario where null is unexpected
Assert.NotNull(result);
result!.Property // Use null-forgiving operator after assertion

// Or combine check and access
if (result != null)
{
    result.Property // Compiler knows result is not null here
}
```

#### Defensive Programming

Handle null inputs gracefully in test scenarios:

```csharp
public void TestMethod_WithNullInput_HandlesGracefully()
{
    // Arrange
    string? nullInput = null;
    
    // Act & Assert
    Assert.DoesNotThrow(() => SomeMethod(nullInput));
}
```

## Test Organization

### Test Structure

Follow consistent test naming and organization:

```csharp
namespace Jammer.Tests.Unit.Core
{
    public class ComponentNameTests
    {
        [Fact]
        public void MethodName_Condition_ExpectedResult()
        {
            // Arrange
            
            // Act
            
            // Assert
        }
    }
}
```

### Integration vs Unit Tests

**Unit Tests:**
- Test individual methods in isolation
- Use mocks for dependencies
- Fast execution
- Located in `/Unit/` directory

**Integration Tests:**
- Test component interactions
- May use real dependencies where safe
- Test workflows end-to-end
- Located in `/Integration/` directory

## Mock Interface Compliance

### Required Methods

All mock implementations must provide these essential methods:

```csharp
public class MockAudioPlayer : IAudioPlayer
{
    // Method call tracking
    public bool WasMethodCalled(string methodName) { }
    
    // Extended method overloads  
    public bool ChannelPlay(int handle, bool restart) { }
    public void ChannelSetLength(int handle, long length) { }
    
    // All IAudioPlayer interface methods...
}
```

### Mock State Management

Track mock state consistently:

```csharp
public void Reset()
{
    _streams.Clear();
    _effects.Clear();
    MethodCallLog.Clear();
    ShouldFailStreamCreation = false;
    ShouldFailPlayback = false;
    LastErrorValue = Errors.OK;
    _nextHandle = 1000;
}
```

## Error Handling Patterns

### Expected Errors

Test both success and failure scenarios:

```csharp
[Fact]
public void Operation_WhenServiceFails_HandlesGracefully()
{
    // Arrange
    var mockPlayer = new MockAudioPlayer();
    mockPlayer.ShouldFailStreamCreation = true;
    
    // Act
    var result = CreateAudioStream(mockPlayer, "test.mp3");
    
    // Assert
    Assert.Equal(0, result); // 0 indicates failure
}
```

### Directory and File Operations

Handle filesystem operations safely:

```csharp
// Safe directory deletion
if (Directory.Exists(testDir))
{
    Directory.Delete(testDir, recursive: true);
}

// Safe file checks
if (File.Exists(testFile))
{
    File.Delete(testFile);
}
```

## Common Anti-Patterns to Avoid

### ❌ Don't: Direct null dereferencing
```csharp
themes.selectedTheme.Border // Can throw NullReferenceException
```

### ✅ Do: Safe null handling
```csharp
themes.selectedTheme?.Border ?? DefaultBorder
```

### ❌ Don't: Shared mock state across tests
```csharp
// Static mock shared across tests - causes flaky tests
private static MockAudioPlayer SharedMock = new();
```

### ✅ Do: Fresh mock per test
```csharp
[Fact]
public void TestMethod()
{
    var mockPlayer = new MockAudioPlayer(); // Fresh instance
    // Test logic
}
```

### ❌ Don't: Ignore test cleanup
```csharp
// Leaving test files/directories after test completion
```

### ✅ Do: Proper cleanup
```csharp
[Fact]
public void TestWithFileOperations()
{
    var testDir = Path.Combine(Path.GetTempPath(), "test-" + Guid.NewGuid());
    
    try
    {
        // Test logic
    }
    finally
    {
        if (Directory.Exists(testDir))
            Directory.Delete(testDir, true);
    }
}
```

## Future Maintenance Guidelines

### Interface Changes

When updating `IAudioPlayer`:

1. **Update interface first**
2. **Update MockAudioPlayer implementation** 
3. **Update real implementation**
4. **Run all tests to identify needed updates**
5. **Update test expectations as needed**

### Test Validation Requirements

Before merging changes:

1. **All tests must compile** - No compilation errors
2. **No null reference warnings** - Address CS8602 warnings  
3. **Mock interfaces complete** - All required methods implemented
4. **Clean package restore** - No compatibility warnings

### Preventing Interface Divergence

Regular validation checklist:

- [ ] MockAudioPlayer implements all IAudioPlayer methods
- [ ] Method signatures match exactly (parameters, return types)
- [ ] All overloads are implemented in mock
- [ ] Test helper methods (WasMethodCalled, etc.) are available
- [ ] Mock tracking behavior is consistent

## Quick Reference

### Essential Mock Operations

```csharp
var mock = new MockAudioPlayer();
mock.Reset();                                    // Clear state
mock.ShouldFailStreamCreation = true;           // Simulate failures
mock.WasMethodCalled("ChannelPlay");            // Verify calls
mock.MethodCallLog;                             // Access call history
mock.GetStream(handle);                         // Get mock state
```

### Common Null Patterns

```csharp
obj?.Property                    // Null-conditional access
obj?.Method()                   // Null-conditional method call
Assert.NotNull(obj); obj!.Prop  // Explicit null check + forgiving operator
obj ?? DefaultValue             // Null coalescing
```

### Test Structure Template

```csharp
[Fact]
public void Method_Condition_ExpectedResult()
{
    // Arrange
    var mock = new MockAudioPlayer();
    var input = CreateTestInput();
    
    // Act
    var result = SystemUnderTest.Method(mock, input);
    
    // Assert
    Assert.NotNull(result);
    Assert.True(mock.WasMethodCalled("ExpectedMethod"));
}
```

---

*These guidelines ensure consistent, maintainable tests that properly validate Jammer's audio functionality while preventing future interface divergence issues.*

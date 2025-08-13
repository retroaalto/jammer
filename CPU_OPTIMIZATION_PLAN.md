# CPU Usage Optimization Plan for Jammer Music Player

## Problem Analysis

The Jammer music player consumes 100% of a single CPU core after extended use due to aggressive loop polling frequencies. The issue manifests after prolonged usage and is more noticeable on slower CPUs, while the UI remains responsive.

## Root Cause Investigation

### Primary Issues Identified:

1. **Main Loop Over-Polling** (`Start.cs:308`)
   - Uses `Thread.Sleep(1)` in default/all/rss views
   - Results in ~1000 iterations per second
   - Each iteration performs expensive operations: audio position polling, UI updates, keyboard checks

2. **Visualizer Thread High Frequency** (`Start.cs:336`)
   - Uses `Visual.refreshTime = 10ms` (100Hz frequency)
   - Continuously sets drawing flags when visualizer is enabled
   - Adds significant overhead on top of main loop

3. **Continuous Audio Polling** (Every loop iteration)
   - `Bass.ChannelGetPosition()` called 3 times per iteration
   - `Bass.ChannelBytes2Seconds()` called 3 times per iteration  
   - `Bass.ChannelGetLength()` called once per iteration
   - At 1000Hz, this creates substantial BASS library call overhead

4. **Keyboard Input Over-Checking**
   - `CheckKeyboardAsync()` called every 1ms iteration
   - Complex processing even when no keys pressed

### Why It Worsens Over Time:
- Threading overhead accumulates
- Audio driver/BASS library internal state builds up
- OS scheduler becomes less efficient with sustained high-frequency operations

## Proposed Solutions

### Immediate Fixes (Priority 1):

1. **Increase Main Loop Sleep Interval**
   - **File**: `Jammer.Core/src/Start.cs:308`
   - **Change**: `Thread.Sleep(1)` → `Thread.Sleep(16)`
   - **Impact**: Reduces frequency from 1000Hz to ~60Hz (standard for smooth UI)
   - **Result**: Maintains responsiveness while dramatically reducing CPU usage

2. **Optimize Visualizer Refresh Rate**  
   - **File**: `Jammer.Core/src/Visual.cs:39`
   - **Change**: `refreshTime = 10` → `refreshTime = 33`
   - **Impact**: Changes from 100Hz to 30Hz refresh rate
   - **Result**: Still provides smooth visualization but reduces CPU load significantly

### Secondary Optimizations (Priority 2):

3. **Throttle Audio Position Polling**
   - Only poll audio position when actually needed (time updates, seeking)
   - Cache audio position values briefly to avoid redundant BASS calls
   - Implement dirty flag system for position updates
   - **Files affected**: `Start.cs` (Loop method), audio polling sections

4. **Optimize Keyboard Checking**
   - Add small delay between keyboard checks when no input detected
   - Consider using event-driven input handling instead of polling
   - **Files affected**: `Jammer.Core/src/Keyboard.cs`

### Long-term Improvements (Priority 3):

5. **Implement Adaptive Sleep**
   - Dynamic sleep intervals based on activity level
   - Longer sleep when idle, shorter when actively processing
   - Smart throttling based on user interaction

6. **Event-Driven Architecture**
   - Replace polling with event-based updates where possible
   - Use audio callbacks for position updates instead of continuous polling
   - Implement proper observer pattern for UI updates

## Implementation Details

### Code Locations:
- **Main loop**: `Jammer.Core/src/Start.cs` (Loop method, lines 308-311)
- **Visualizer**: `Jammer.Core/src/Visual.cs` (refreshTime variable, line 39)
- **Audio polling**: `Jammer.Core/src/Start.cs` (lines 283-291)
- **Keyboard checking**: `Jammer.Core/src/Keyboard.cs` (CheckKeyboardAsync method)

### Testing Strategy:
1. Monitor CPU usage before and after changes
2. Verify UI responsiveness remains acceptable
3. Test audio playback timing accuracy
4. Validate visualizer smoothness
5. Check keyboard input responsiveness

## Expected Results

- **CPU Usage Reduction**: 80-90% decrease in CPU consumption
- **Maintained Functionality**: All current features continue working
- **Preserved Responsiveness**: UI remains as responsive as before
- **Better Performance**: Especially noticeable on slower CPUs

## Risk Assessment

- **Low Risk**: Changes are minimal and localized
- **Reversible**: Easy to revert if issues arise
- **Non-breaking**: No API or functionality changes
- **Gradual Implementation**: Can be applied incrementally

## Success Metrics

- CPU usage drops from 100% to <20% during normal operation
- UI response time remains under acceptable thresholds
- Audio playback maintains accuracy and smoothness
- Visualizer continues to provide satisfactory visual feedback

# Changes made
  Changes Made:

  1. Fixed Main Loop Sleep Interval (Start.cs)
    - Changed Thread.Sleep(1) to Thread.Sleep(16)
    - Reduces main loop frequency from ~1000Hz to ~60Hz
  2. Fixed Visualizer Refresh Rate (Visual.cs)
    - Changed refreshTime = 10 to refreshTime = 33
    - Reduces visualizer frequency from 100Hz to ~30Hz


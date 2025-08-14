# CPU Usage Debugging Plan - Progressive Performance Degradation

## Root Cause Identified: **Memory Leak in Logging System**

### **CRITICAL ISSUE FOUND:**
**File: `Jammer.Core/src/Log.cs` Lines 15 & 19**
```csharp
log = log.Append(...).ToArray(); // Creates new array every time!
```

**Why this causes progressive CPU increase:**
1. Every log entry creates a completely new array with all previous entries
2. Array copying becomes more expensive as log grows (O(n) complexity per append)
3. With frequent logging in main loop, CPU usage increases exponentially over time
4. After hours of runtime, thousands of log entries cause massive CPU overhead

### **Secondary Contributing Factors:**
1. **Frequent logging calls** in the main loop (Start.cs:53, 88, 99, etc.)
2. **No log rotation** or size limits
3. **String concatenation overhead** in log formatting
4. **Memory pressure** causing GC thrashing

### **Debugging & Fix Strategy:**

#### **Phase 1: Immediate Logging Fix**
- Replace `string[] log` with `List<string>` for O(1) append operations
- Add log rotation/size limits to prevent unbounded growth
- Implement circular buffer for recent log entries only

#### **Phase 2: Performance Monitoring**
- Add performance counters to measure:
  - Log array size over time
  - Memory usage growth
  - GC collection frequency
  - Loop iteration timing

#### **Phase 3: Enhanced Logging Strategy**
- Implement log levels to reduce verbose logging
- Add conditional logging based on debug mode
- Consider asynchronous logging to reduce main thread impact
- Add memory usage monitoring and alerts

### **Specific Implementation Plan:**
1. **Replace growing array with efficient data structure**
2. **Add log file output with rotation** 
3. **Implement performance timing logs** to track CPU usage patterns
4. **Add memory monitoring** to detect other potential leaks
5. **Create debug logging levels** to reduce overhead in production

### **Recommended Logging to Add for Troubleshooting:**
1. **Performance metrics every 10 seconds:**
   - Current log array size
   - Memory usage (Process.WorkingSet64)
   - Loop iteration timing
   - GC collection counts

2. **Debug logging with timestamps:**
   - When CPU usage patterns change
   - Memory allocation spikes
   - Log array growth milestones

3. **Log to file with rotation:**
   - Prevents memory buildup in application
   - Enables external analysis of performance patterns
   - Automatic cleanup of old logs

This explains why CPU starts normal but increases over time - the logging system creates an exponentially expensive memory leak that degrades performance as the application runs longer.
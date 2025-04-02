# ChronoGo: Time-Travel Debugger for Go

ChronoGo is a prototype time-travel debugger for Go applications, integrating the Delve debugger with event recording and replay capabilities.

## Building ChronoGo

```bash
# Always build with debug information enabled (disables optimizations and inlining)
go build -gcflags="all=-N -l" -o chrono.exe cmd/chrono/main.go
```

## Running ChronoGo

```bash
# Run ChronoGo with a target binary
./chrono.exe <target-executable>

# Testing on itself (self-debugging)
./chrono.exe ./chrono.exe

# Specify a custom events file
./chrono.exe -events my-trace.log <target-executable>

# Replay previously recorded events without executing a program
./chrono.exe -replay -events my-trace.log
```

## Command-Line Options

- `-events <file>` - Specify the path to the events file (default: chronogo.events)
- `-replay` - Run in replay mode only, loading events from the specified file

## Important Notes

### Build Process
- **Always use debug flags**: The debugger relies on debug information being present in the binary
- **Avoid `go run`**: This creates temporary binaries that Delve cannot properly debug
- **Build manually**: Use explicit build steps for consistent debugging results

### Setting Breakpoints
- When setting breakpoints, you must use executable statement lines
- Use either path format: 
  - Windows backslash: `D:\SRC\ChronoGo\cmd\chrono\main.go:26`
  - Forward slash: `D:/SRC/ChronoGo/cmd/chrono/main.go:26`
- Function breakpoints: `bp func:myFunction` sets a breakpoint at function entry
- Conditional breakpoints: `bp <file:line> -c "x > 5"` breaks only when condition is true
- The debugger now offers smart alternatives when a breakpoint can't be set at an exact line

### Debugging Commands
- `bp <file:line>` - Set a breakpoint
- `bp func:<funcname>` - Set a function breakpoint
- `bp <file:line> -c <condition>` - Set a conditional breakpoint
- `c` - Continue execution until a breakpoint
- `s` - Step forward one event
- `b` - Step backward one event
- `l` - List active breakpoints
- `p <var>` - Print value of a variable (with complex type inspection)
- `watch <expr>` - Set a watchpoint to monitor memory changes
  - `watch -r <expr>` - Break on reads of memory
  - `watch -w <expr>` - Break on writes to memory
  - `watch -rw <expr>` - Break on reads or writes (default)
- `q` - Quit the debugger

### Using Watchpoints

Watchpoints monitor changes to variables or memory locations. In ChronoGo, they work in two modes:

1. **Live Debugging Mode**: When the Delve process is active, watchpoints use Delve's native watchpoint functionality to monitor memory access during live execution.

2. **Replay Mode**: When viewing recorded events, ChronoGo simulates watchpoints by analyzing statement execution events that contain assignments to variables.

To use watchpoints effectively:

1. Set a breakpoint at a line where the variable is in scope: `bp file:line`
2. Continue execution to that breakpoint: `c`
3. Set a watchpoint on the variable: `watch x` or with options `watch -r x` (read), `watch -w x` (write)
4. Continue execution to see changes: `c`

**Note**: If the Delve process has already completed execution, watchpoints will work in replay-only mode and will highlight potential variable changes in the recorded events.

#### Example workflow:

```
(chrono) bp D:/SRC/ChronoGo/cmd/chrono/main.go:26
(chrono) c
... breakpoint hit ...
(chrono) watch x
(chrono) c
... program will stop when x changes ...
```

## Current Limitations

- Limited debugging capabilities for concurrent programs
- Delve debugger terminates after the program finishes execution
- Windows path handling requires full paths for breakpoints
- Not all Delve debugging features are fully integrated

## Development Roadmap

### Phase 1: Improved Event Recording (Completed)
- [x] Basic function entry/exit recording
- [x] Statement-level instrumentation
- [x] Breakpoint management
- [x] Integration with Delve
- [x] Watchpoint support

### Phase 2: Enhanced Debugging Experience (Current)
- [] Better synchronization between replayer and Delve
- [] Improved variable inspection, especially for complex types
- [] Function breakpoints support
- [] Conditional breakpoints support
- [ ] Memory snapshots for more accurate state replay

### Phase 3: Production-ready Features
- [ ] Automatic instrumentation of entire programs
- [ ] Optimized event recording with lower overhead
- [ ] Cross-platform path handling
- [ ] Advanced UI integration

## Manual Testing Process

1. Make code changes
2. Build with debug flags: `go build -gcflags="all=-N -l" -o chrono.exe cmd/chrono/main.go`
3. Run ChronoGo targeting a test program (or itself): `./chrono.exe ./chrono.exe`
4. Test debugging commands:
   - Set breakpoints at recorded event lines
   - Step forward and backward
   - Verify breakpoints are hit correctly

## Troubleshooting

### Common Issues

1. **"Could not find statement" errors**
   - Make sure you're setting breakpoints on executable statement lines
   - Check the recorded events output for valid line numbers

2. **Paths not working with breakpoints**
   - Use full paths to the source files
   - Try both forward slash and backslash variations

3. **Delve process exiting**
   - This is expected when the program completes
   - The replayer will still work for time-travel operations

4. **Watchpoints not working**
   - Ensure Delve process is still active
   - Variables must be in scope at current execution point
   - Use replay-mode watchpoints when Delve has exited

## Project Structure

- `cmd/chrono/` - Main application entry point
- `pkg/debugger/` - Debugging interface and Delve integration
- `pkg/instrumentation/` - Event recording and code instrumentation
- `pkg/recorder/` - Event storage and management
- `pkg/replay/` - Time-travel replay functionality
- `tests/` - Integration tests

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
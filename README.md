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
```

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
- Look at the recorded events list to see which lines contain executable statements

### Debugging Commands
- `bp <file:line>` - Set a breakpoint
- `c` - Continue execution until a breakpoint
- `s` - Step forward one event
- `b` - Step backward one event
- `l` - List active breakpoints
- `q` - Quit the debugger

## Current Limitations

- Breakpoints only work on lines with recorded events
- Delve debugger terminates after the program finishes execution
- Limited variable inspection capabilities
- Windows path handling requires full paths for breakpoints

## Development Roadmap

### Phase 1: Improved Event Recording (Current)
- [x] Basic function entry/exit recording
- [x] Statement-level instrumentation
- [x] Breakpoint management
- [x] Integration with Delve

### Phase 2: Enhanced Debugging Experience
- [ ] Better synchronization between replayer and Delve
- [ ] Memory snapshots for more accurate state replay
- [ ] Improved variable inspection
- [ ] Function breakpoints support

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

## Project Structure

- `cmd/chrono/` - Main application entry point
- `pkg/debugger/` - Debugging interface and Delve integration
- `pkg/instrumentation/` - Event recording and code instrumentation
- `pkg/recorder/` - Event storage and management
- `pkg/replay/` - Time-travel replay functionality
- `tests/` - Integration tests

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
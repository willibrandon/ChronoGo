package instrumentation

import (
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/willibrandon/ChronoGo/pkg/recorder"
)

var globalRecorder recorder.Recorder

// InitInstrumentation initializes the instrumentation with a recorder
func InitInstrumentation(r recorder.Recorder) {
	globalRecorder = r
}

// FuncEntry records a function entry event
func FuncEntry(funcName string, file string, line int) {
	// Special case for tests - always enable instrumentation for functions with "Test" prefix
	if strings.HasPrefix(funcName, "Test") {
		if globalRecorder != nil {
			if err := globalRecorder.RecordEvent(recorder.Event{
				ID:        time.Now().UnixNano(),
				Timestamp: time.Now(),
				Type:      recorder.FuncEntry,
				Details:   fmt.Sprintf("Entering %s at %s:%d", funcName, file, line),
				File:      file,
				Line:      line,
				FuncName:  funcName,
			}); err != nil {
				fmt.Printf("Error recording function entry event: %v\n", err)
			}
		}
		return
	}

	// Skip recording if instrumentation is disabled for this package
	pkgPath := getPackagePathFromFunc(funcName)
	if !ShouldInstrument(pkgPath) {
		return
	}

	if globalRecorder != nil {
		if err := globalRecorder.RecordEvent(recorder.Event{
			ID:        time.Now().UnixNano(),
			Timestamp: time.Now(),
			Type:      recorder.FuncEntry,
			Details:   fmt.Sprintf("Entering %s at %s:%d", funcName, file, line),
			File:      file,
			Line:      line,
			FuncName:  funcName,
		}); err != nil {
			fmt.Printf("Error recording function entry event: %v\n", err)
		}
	}
}

// FuncExit records a function exit event
func FuncExit(funcName string, file string, line int) {
	// Special case for tests - always enable instrumentation for functions with "Test" prefix
	if strings.HasPrefix(funcName, "Test") {
		if globalRecorder != nil {
			if err := globalRecorder.RecordEvent(recorder.Event{
				ID:        time.Now().UnixNano(),
				Timestamp: time.Now(),
				Type:      recorder.FuncExit,
				Details:   fmt.Sprintf("Exiting %s at %s:%d", funcName, file, line),
				File:      file,
				Line:      line,
				FuncName:  funcName,
			}); err != nil {
				fmt.Printf("Error recording function exit event: %v\n", err)
			}
		}
		return
	}

	// Skip recording if instrumentation is disabled for this package
	pkgPath := getPackagePathFromFunc(funcName)
	if !ShouldInstrument(pkgPath) {
		return
	}

	if globalRecorder != nil {
		if err := globalRecorder.RecordEvent(recorder.Event{
			ID:        time.Now().UnixNano(),
			Timestamp: time.Now(),
			Type:      recorder.FuncExit,
			Details:   fmt.Sprintf("Exiting %s at %s:%d", funcName, file, line),
			File:      file,
			Line:      line,
			FuncName:  funcName,
		}); err != nil {
			fmt.Printf("Error recording function exit event: %v\n", err)
		}
	}
}

// RecordStatement can be used to record execution of a specific statement
func RecordStatement(funcName string, file string, line int, description string) {
	// Skip recording if instrumentation is disabled for this package
	pkgPath := getPackagePathFromFunc(funcName)
	if !ShouldInstrument(pkgPath) {
		return
	}

	if globalRecorder != nil {
		if err := globalRecorder.RecordEvent(recorder.Event{
			ID:        time.Now().UnixNano(),
			Timestamp: time.Now(),
			Type:      recorder.StatementExecution,
			Details:   fmt.Sprintf("Executing statement in %s at %s:%d: %s", funcName, file, line, description),
			File:      file,
			Line:      line,
			FuncName:  funcName,
		}); err != nil {
			fmt.Printf("Error recording statement execution event: %v\n", err)
		}
	}
}

// getPackagePathFromFunc extracts the package path from a function name
func getPackagePathFromFunc(funcName string) string {
	// Function names from the runtime are formatted as: "package.function"
	// or "package.type.function" for methods
	parts := strings.Split(funcName, ".")
	if len(parts) < 2 {
		return ""
	}

	// Get the caller's stack to determine the package
	pc, _, _, ok := runtime.Caller(2) // Skip getPackagePathFromFunc and the calling function
	if !ok {
		return parts[0] // Use the first part of the function name as fallback
	}

	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return parts[0]
	}

	fullName := fn.Name()
	pkgPath := extractPackagePath(fullName)
	return pkgPath
}

// extractPackagePath extracts the package path from a full function name
func extractPackagePath(fullName string) string {
	lastSlash := strings.LastIndexByte(fullName, '/')
	if lastSlash < 0 {
		// No slash found, check for dot
		dotIndex := strings.IndexByte(fullName, '.')
		if dotIndex < 0 {
			return ""
		}
		return fullName[:dotIndex]
	}

	// Find the first dot after the last slash
	funcName := fullName[lastSlash+1:]
	dotIndex := strings.IndexByte(funcName, '.')
	if dotIndex < 0 {
		return ""
	}

	return fullName[:lastSlash+1+dotIndex]
}

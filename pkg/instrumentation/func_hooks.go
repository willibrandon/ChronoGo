package instrumentation

import (
	"fmt"
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
	if globalRecorder != nil {
		globalRecorder.RecordEvent(recorder.Event{
			ID:        time.Now().UnixNano(),
			Timestamp: time.Now(),
			Type:      recorder.FuncEntry,
			Details:   fmt.Sprintf("Entering %s at %s:%d", funcName, file, line),
			File:      file,
			Line:      line,
			FuncName:  funcName,
		})
	}
}

// FuncExit records a function exit event
func FuncExit(funcName string, file string, line int) {
	if globalRecorder != nil {
		globalRecorder.RecordEvent(recorder.Event{
			ID:        time.Now().UnixNano(),
			Timestamp: time.Now(),
			Type:      recorder.FuncExit,
			Details:   fmt.Sprintf("Exiting %s at %s:%d", funcName, file, line),
			File:      file,
			Line:      line,
			FuncName:  funcName,
		})
	}
}

// RecordStatement can be used to record execution of a specific statement
func RecordStatement(funcName string, file string, line int, description string) {
	if globalRecorder != nil {
		globalRecorder.RecordEvent(recorder.Event{
			ID:        time.Now().UnixNano(),
			Timestamp: time.Now(),
			Type:      recorder.StatementExecution,
			Details:   fmt.Sprintf("Executing statement in %s at %s:%d: %s", funcName, file, line, description),
			File:      file,
			Line:      line,
			FuncName:  funcName,
		})
	}
}

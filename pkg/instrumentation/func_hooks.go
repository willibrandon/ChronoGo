package instrumentation

import (
	"time"

	"github.com/willibrandon/ChronoGo/pkg/recorder"
)

var globalRecorder recorder.Recorder

func InitInstrumentation(r recorder.Recorder) {
	globalRecorder = r
}

func FuncEntry(funcName string) {
	if globalRecorder != nil {
		globalRecorder.RecordEvent(recorder.Event{
			ID:        time.Now().UnixNano(),
			Timestamp: time.Now(),
			Type:      recorder.FuncEntry,
			Details:   "Entering " + funcName,
		})
	}
}

func FuncExit(funcName string) {
	if globalRecorder != nil {
		globalRecorder.RecordEvent(recorder.Event{
			ID:        time.Now().UnixNano(),
			Timestamp: time.Now(),
			Type:      recorder.FuncExit,
			Details:   "Exiting " + funcName,
		})
	}
}

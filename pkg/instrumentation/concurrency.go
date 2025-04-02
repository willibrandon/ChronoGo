package instrumentation

import (
	"fmt"
	"runtime"
	"time"

	"github.com/willibrandon/ChronoGo/pkg/recorder"
)

// GoroutineCreate records a goroutine creation event
func GoroutineCreate(gID int) {
	// Skip recording if selective instrumentation is disabled for caller
	if !shouldInstrumentCaller() {
		return
	}

	if globalRecorder != nil {
		err := globalRecorder.RecordEvent(recorder.Event{
			ID:        time.Now().UnixNano(),
			Timestamp: time.Now(),
			Type:      recorder.GoroutineSwitch,
			Details:   fmt.Sprintf("Goroutine %d created", gID),
		})
		if err != nil {
			fmt.Printf("Error recording goroutine creation: %v\n", err)
		}
	}
}

// GoroutineSwitch records a scheduler switch between goroutines
func GoroutineSwitch(fromID, toID int) {
	// Skip recording if selective instrumentation is disabled for caller
	if !shouldInstrumentCaller() {
		return
	}

	if globalRecorder != nil {
		err := globalRecorder.RecordEvent(recorder.Event{
			ID:        time.Now().UnixNano(),
			Timestamp: time.Now(),
			Type:      recorder.GoroutineSwitch,
			Details:   fmt.Sprintf("Goroutine switch from %d to %d", fromID, toID),
		})
		if err != nil {
			fmt.Printf("Error recording goroutine switch: %v\n", err)
		}
	}
}

// ChannelSend records a channel send operation
func ChannelSend(chID, senderID int, value interface{}) {
	// Skip recording if selective instrumentation is disabled for caller
	if !shouldInstrumentCaller() {
		return
	}

	if globalRecorder != nil {
		err := globalRecorder.RecordEvent(recorder.Event{
			ID:        time.Now().UnixNano(),
			Timestamp: time.Now(),
			Type:      recorder.ChannelOperation,
			Details:   fmt.Sprintf("Channel %d: send by goroutine %d, value: %v", chID, senderID, value),
		})
		if err != nil {
			fmt.Printf("Error recording channel send: %v\n", err)
		}
	}
}

// ChannelRecv records a channel receive operation
func ChannelRecv(chID, receiverID int, value interface{}) {
	// Skip recording if selective instrumentation is disabled for caller
	if !shouldInstrumentCaller() {
		return
	}

	if globalRecorder != nil {
		err := globalRecorder.RecordEvent(recorder.Event{
			ID:        time.Now().UnixNano(),
			Timestamp: time.Now(),
			Type:      recorder.ChannelOperation,
			Details:   fmt.Sprintf("Channel %d: receive by goroutine %d, value: %v", chID, receiverID, value),
		})
		if err != nil {
			fmt.Printf("Error recording channel receive: %v\n", err)
		}
	}
}

// ChannelClose records a channel close operation
func ChannelClose(chID, goroutineID int) {
	// Skip recording if selective instrumentation is disabled for caller
	if !shouldInstrumentCaller() {
		return
	}

	if globalRecorder != nil {
		err := globalRecorder.RecordEvent(recorder.Event{
			ID:        time.Now().UnixNano(),
			Timestamp: time.Now(),
			Type:      recorder.ChannelOperation,
			Details:   fmt.Sprintf("Channel %d: closed by goroutine %d", chID, goroutineID),
		})
		if err != nil {
			fmt.Printf("Error recording channel close: %v\n", err)
		}
	}
}

// MutexLock records a mutex lock acquisition
func MutexLock(mutexID, goroutineID int) {
	// Skip recording if selective instrumentation is disabled for caller
	if !shouldInstrumentCaller() {
		return
	}

	if globalRecorder != nil {
		err := globalRecorder.RecordEvent(recorder.Event{
			ID:        time.Now().UnixNano(),
			Timestamp: time.Now(),
			Type:      recorder.SyncOperation,
			Details:   fmt.Sprintf("Mutex %d: locked by goroutine %d", mutexID, goroutineID),
		})
		if err != nil {
			fmt.Printf("Error recording mutex lock: %v\n", err)
		}
	}
}

// MutexUnlock records a mutex unlock operation
func MutexUnlock(mutexID, goroutineID int) {
	// Skip recording if selective instrumentation is disabled for caller
	if !shouldInstrumentCaller() {
		return
	}

	if globalRecorder != nil {
		err := globalRecorder.RecordEvent(recorder.Event{
			ID:        time.Now().UnixNano(),
			Timestamp: time.Now(),
			Type:      recorder.SyncOperation,
			Details:   fmt.Sprintf("Mutex %d: unlocked by goroutine %d", mutexID, goroutineID),
		})
		if err != nil {
			fmt.Printf("Error recording mutex unlock: %v\n", err)
		}
	}
}

// shouldInstrumentCaller checks if the caller's package should be instrumented
func shouldInstrumentCaller() bool {
	// Skip 2 frames to get the actual caller (not this function or the instrumentation function)
	pc, _, _, ok := runtime.Caller(2)
	if !ok {
		// If we can't determine caller, default to instrumenting
		return true
	}

	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return true
	}

	fullName := fn.Name()
	pkgPath := extractPackagePath(fullName)
	return ShouldInstrument(pkgPath)
}

package instrumentation

import (
	"fmt"
	"time"

	"github.com/willibrandon/ChronoGo/pkg/recorder"
)

// GoroutineCreate records a goroutine creation event
func GoroutineCreate(gID int) {
	if globalRecorder != nil {
		globalRecorder.RecordEvent(recorder.Event{
			ID:        time.Now().UnixNano(),
			Timestamp: time.Now(),
			Type:      recorder.GoroutineSwitch,
			Details:   fmt.Sprintf("Goroutine %d created", gID),
		})
	}
}

// GoroutineSwitch records a scheduler switch between goroutines
func GoroutineSwitch(fromID, toID int) {
	if globalRecorder != nil {
		globalRecorder.RecordEvent(recorder.Event{
			ID:        time.Now().UnixNano(),
			Timestamp: time.Now(),
			Type:      recorder.GoroutineSwitch,
			Details:   fmt.Sprintf("Goroutine switch from %d to %d", fromID, toID),
		})
	}
}

// ChannelSend records a channel send operation
func ChannelSend(chID, senderID int, value interface{}) {
	if globalRecorder != nil {
		globalRecorder.RecordEvent(recorder.Event{
			ID:        time.Now().UnixNano(),
			Timestamp: time.Now(),
			Type:      recorder.ChannelOperation,
			Details:   fmt.Sprintf("Channel %d: send by goroutine %d, value: %v", chID, senderID, value),
		})
	}
}

// ChannelRecv records a channel receive operation
func ChannelRecv(chID, receiverID int, value interface{}) {
	if globalRecorder != nil {
		globalRecorder.RecordEvent(recorder.Event{
			ID:        time.Now().UnixNano(),
			Timestamp: time.Now(),
			Type:      recorder.ChannelOperation,
			Details:   fmt.Sprintf("Channel %d: receive by goroutine %d, value: %v", chID, receiverID, value),
		})
	}
}

// ChannelClose records a channel close operation
func ChannelClose(chID, goroutineID int) {
	if globalRecorder != nil {
		globalRecorder.RecordEvent(recorder.Event{
			ID:        time.Now().UnixNano(),
			Timestamp: time.Now(),
			Type:      recorder.ChannelOperation,
			Details:   fmt.Sprintf("Channel %d: closed by goroutine %d", chID, goroutineID),
		})
	}
}

// MutexLock records a mutex lock acquisition
func MutexLock(mutexID, goroutineID int) {
	if globalRecorder != nil {
		globalRecorder.RecordEvent(recorder.Event{
			ID:        time.Now().UnixNano(),
			Timestamp: time.Now(),
			Type:      recorder.SyncOperation,
			Details:   fmt.Sprintf("Mutex %d: locked by goroutine %d", mutexID, goroutineID),
		})
	}
}

// MutexUnlock records a mutex unlock operation
func MutexUnlock(mutexID, goroutineID int) {
	if globalRecorder != nil {
		globalRecorder.RecordEvent(recorder.Event{
			ID:        time.Now().UnixNano(),
			Timestamp: time.Now(),
			Type:      recorder.SyncOperation,
			Details:   fmt.Sprintf("Mutex %d: unlocked by goroutine %d", mutexID, goroutineID),
		})
	}
}

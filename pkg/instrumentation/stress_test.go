package instrumentation

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/willibrandon/ChronoGo/pkg/recorder"
)

// TestStressConcurrentRecording tests the runtime/trace integration under heavy concurrency
func TestStressConcurrentRecording(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	// Create a recorder
	rec := recorder.NewInMemoryRecorder()

	// Initialize runtime tracing
	err := InitRuntimeTracing(rec)
	if err != nil {
		t.Fatalf("Failed to initialize runtime tracing: %v", err)
	}
	defer StopRuntimeTracing()

	// Number of goroutines to create
	const numGoroutines = 10

	// Number of operations per goroutine
	const opsPerGoroutine = 5

	// Create channels
	channels := make([]chan int, numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		channels[i] = make(chan int, opsPerGoroutine/2) // Some buffering to avoid deadlocks
	}

	// Create mutexes
	mutexes := make([]sync.Mutex, numGoroutines)

	// Create a WaitGroup to synchronize goroutines
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Launch worker goroutines
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			ch := channels[id]
			mu := &mutexes[id]

			// Perform multiple mutex operations
			for j := 0; j < opsPerGoroutine; j++ {
				mu.Lock()
				time.Sleep(time.Millisecond) // Small sleep to simulate work
				mu.Unlock()
			}

			// Perform sending operations
			for j := 0; j < opsPerGoroutine; j++ {
				// Explicitly trace for reliability
				TraceChannelOperation(ch, "send", j)
				ch <- j
			}

			// Close our channel when done
			TraceChannelOperation(ch, "close", nil)
			close(ch)
		}(i)
	}

	// Launch reader goroutines
	var readerWg sync.WaitGroup
	readerWg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer readerWg.Add(-1)

			ch := channels[id]

			// Read from the channel until it's closed
			for val := range ch {
				// Explicitly trace for reliability
				TraceChannelOperation(ch, "recv", val)
				time.Sleep(time.Millisecond) // Small sleep to simulate work
			}
		}(i)
	}

	// Wait for all workers to complete
	wg.Wait()

	// Wait for all readers to complete
	readerWg.Wait()

	// Give the runtime tracer time to process events
	time.Sleep(200 * time.Millisecond)

	// Get the recorded events
	events := rec.GetEvents()

	// Verify that a substantial number of events were recorded
	if len(events) < numGoroutines*2 {
		t.Errorf("Expected at least %d events, got %d", numGoroutines*2, len(events))
	}

	// Count event types to verify we captured different kinds of concurrency events
	counts := make(map[recorder.EventType]int)
	for _, event := range events {
		counts[event.Type]++
	}

	t.Logf("Recorded events: %v", counts)

	// Check that we recorded at least some channel operations
	if counts[recorder.ChannelOperation] < numGoroutines {
		t.Errorf("Expected at least %d channel operations, got %d",
			numGoroutines, counts[recorder.ChannelOperation])
	}

	// Check that we recorded at least some goroutine switches
	if counts[recorder.GoroutineSwitch] < numGoroutines {
		t.Errorf("Expected at least %d goroutine switches, got %d",
			numGoroutines, counts[recorder.GoroutineSwitch])
	}
}

// TestMixedPatternConcurrency tests recording with a mix of different concurrency patterns
func TestMixedPatternConcurrency(t *testing.T) {
	// Create a recorder
	rec := recorder.NewInMemoryRecorder()

	// Initialize runtime tracing
	err := InitRuntimeTracing(rec)
	if err != nil {
		t.Fatalf("Failed to initialize runtime tracing: %v", err)
	}
	defer StopRuntimeTracing()

	// Create different concurrency primitives
	unbufferedCh := make(chan int)
	bufferedCh := make(chan string, 5)
	syncChan := make(chan struct{}) // For synchronization

	var mutex sync.Mutex
	var wg sync.WaitGroup

	// Test select statements with multiple channels
	wg.Add(1)
	go func() {
		defer wg.Done()

		select {
		case val := <-unbufferedCh:
			TraceChannelOperation(unbufferedCh, "recv", val)

			// Use mutex with explicit tracing
			mutex.Lock()
			TraceMutexOperation(&mutex, "lock")

			time.Sleep(5 * time.Millisecond)

			mutex.Unlock()
			TraceMutexOperation(&mutex, "unlock")

		case <-time.After(100 * time.Millisecond):
			// Timeout case
		}

		// Signal completion
		syncChan <- struct{}{}
	}()

	// Give goroutine time to start
	time.Sleep(10 * time.Millisecond)

	// Send on unbuffered channel
	TraceChannelOperation(unbufferedCh, "send", 123)
	unbufferedCh <- 123

	// Wait for first goroutine to finish
	<-syncChan

	// Test buffered channel
	for i := 0; i < 3; i++ {
		// Fix: Properly convert int to string
		value := fmt.Sprintf("test%d", i)
		TraceChannelOperation(bufferedCh, "send", value)
		bufferedCh <- value
	}

	// Test consumer for buffered channel
	wg.Add(1)
	go func() {
		defer wg.Done()

		// Read all values from the buffered channel
		for i := 0; i < 3; i++ {
			val := <-bufferedCh
			TraceChannelOperation(bufferedCh, "recv", val)
		}
	}()

	// Wait for all goroutines to complete
	wg.Wait()

	// Close channels
	TraceChannelOperation(unbufferedCh, "close", nil)
	close(unbufferedCh)
	TraceChannelOperation(bufferedCh, "close", nil)
	close(bufferedCh)
	close(syncChan)

	// Give the runtime tracer time to process events
	time.Sleep(100 * time.Millisecond)

	// Get the recorded events
	events := rec.GetEvents()

	// Verify different event types were captured
	channelOpCount := 0
	syncOpCount := 0
	goroutineSwitchCount := 0

	for _, e := range events {
		switch e.Type {
		case recorder.ChannelOperation:
			channelOpCount++
		case recorder.SyncOperation:
			syncOpCount++
		case recorder.GoroutineSwitch:
			goroutineSwitchCount++
		}
	}

	t.Logf("Recorded: %d channel ops, %d sync ops, %d goroutine switches",
		channelOpCount, syncOpCount, goroutineSwitchCount)

	// We should have at least 6 channel operations (3 sends + 3 receives)
	if channelOpCount < 6 {
		t.Errorf("Expected at least 6 channel operations, got %d", channelOpCount)
	}

	// We should have at least 2 goroutine switches
	if goroutineSwitchCount < 2 {
		t.Errorf("Expected at least 2 goroutine switches, got %d", goroutineSwitchCount)
	}

	// We should have at least 2 sync operations (lock + unlock)
	if syncOpCount < 2 {
		t.Errorf("Expected at least 2 sync operations, got %d", syncOpCount)
	}
}

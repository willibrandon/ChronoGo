package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/willibrandon/ChronoGo/pkg/instrumentation"
	"github.com/willibrandon/ChronoGo/pkg/recorder"
	"github.com/willibrandon/ChronoGo/pkg/replay"
)

// This program demonstrates integration with Go's runtime/trace package
// to automatically capture concurrency events without manual instrumentation.
func main() {
	fmt.Println("Starting runtime/trace integration demo...")

	// Create a recorder
	rec := recorder.NewInMemoryRecorder()

	// Initialize runtime trace integration
	err := instrumentation.InitRuntimeTracing(rec)
	if err != nil {
		fmt.Printf("Error initializing runtime tracing: %v\n", err)
		return
	}
	defer instrumentation.StopRuntimeTracing()

	// Run a concurrent program with channels and goroutines
	runConcurrentProgram()

	// Display recorded events
	events := rec.GetEvents()
	fmt.Printf("\nRecorded %d events using runtime/trace integration\n", len(events))

	// Replay events
	fmt.Println("\n--- Replaying events ---")
	replayer := replay.NewBasicReplayer()
	replayer.LoadEvents(events)
	replayer.ReplayForward()

	fmt.Println("\nDemo complete. The trace output has been saved to chrono_trace.out")
	fmt.Println("You can view it with: go tool trace chrono_trace.out")
}

// runConcurrentProgram runs a simple concurrent program with multiple goroutines and channels
func runConcurrentProgram() {
	// Create channels
	ch1 := make(chan int)
	ch2 := make(chan string)

	// Create a WaitGroup to synchronize goroutines
	var wg sync.WaitGroup
	wg.Add(2)

	// First worker: receives ints, sends strings
	go func() {
		defer wg.Done()

		// Use the wrapped channel operations so they're properly traced
		for i := 0; i < 3; i++ {
			// Receive an int
			val := <-ch1
			fmt.Printf("Worker 1: Received %d\n", val)

			// Manually trace the operation (normally done automatically)
			instrumentation.TraceChannelOperation(ch1, "recv", val)

			// Process and send a string
			result := fmt.Sprintf("Processed-%d", val)

			// Trace the send operation
			instrumentation.TraceChannelOperation(ch2, "send", result)
			ch2 <- result
		}
	}()

	// Second worker: uses mutexes
	go func() {
		defer wg.Done()

		// Create a mutex
		var mu sync.Mutex

		for i := 0; i < 3; i++ {
			// Trace mutex lock
			instrumentation.TraceMutexOperation(&mu, "lock")
			mu.Lock()

			fmt.Printf("Worker 2: Critical section %d\n", i)
			time.Sleep(10 * time.Millisecond)

			// Trace mutex unlock
			instrumentation.TraceMutexOperation(&mu, "unlock")
			mu.Unlock()

			time.Sleep(20 * time.Millisecond)
		}
	}()

	// Send values in the main goroutine
	for i := 1; i <= 3; i++ {
		fmt.Printf("Main: Sending %d\n", i)

		// Trace the send operation
		instrumentation.TraceChannelOperation(ch1, "send", i)
		ch1 <- i

		// Receive the processed result
		result := <-ch2

		// Trace the receive operation
		instrumentation.TraceChannelOperation(ch2, "recv", result)
		fmt.Printf("Main: Received %s\n", result)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Trace channel closes
	instrumentation.TraceChannelOperation(ch1, "close", nil)
	close(ch1)
	instrumentation.TraceChannelOperation(ch2, "close", nil)
	close(ch2)
}

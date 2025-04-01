package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/willibrandon/ChronoGo/pkg/instrumentation"
	"github.com/willibrandon/ChronoGo/pkg/recorder"
	"github.com/willibrandon/ChronoGo/pkg/replay"
)

// This program demonstrates concurrency instrumentation with ChronoGo.
// In a real implementation, hooks would be automatically injected into the Go runtime,
// but for this demo, we manually call the instrumentation functions.
func main() {
	// Set up recording
	rec := recorder.NewInMemoryRecorder()
	instrumentation.InitInstrumentation(rec)

	fmt.Println("Starting concurrency demo with time-travel debugging...")

	// Create a channel for communication
	ch := make(chan int)

	// Create a WaitGroup to synchronize goroutines
	var wg sync.WaitGroup
	wg.Add(1)

	// Record that we're creating a goroutine (ID 2)
	instrumentation.GoroutineCreate(2)

	// Launch a worker goroutine
	go func() {
		defer wg.Done()

		// Record goroutine switch to worker (ID 2)
		instrumentation.GoroutineSwitch(1, 2)

		// Worker waits for a value
		fmt.Println("Worker: waiting to receive value...")
		val := <-ch

		// Record the channel receive
		instrumentation.ChannelRecv(1, 2, val)
		fmt.Printf("Worker: received value %d\n", val)

		// Record a mutex operation
		instrumentation.MutexLock(1, 2)
		fmt.Println("Worker: doing some work...")
		time.Sleep(100 * time.Millisecond)
		instrumentation.MutexUnlock(1, 2)

		// Record goroutine switch back to main
		instrumentation.GoroutineSwitch(2, 1)
	}()

	// Give worker time to start
	time.Sleep(50 * time.Millisecond)

	// Record sending to a channel
	fmt.Println("Main: sending value...")
	instrumentation.ChannelSend(1, 1, 42)
	ch <- 42

	// Wait for worker to complete
	wg.Wait()

	// Record closing the channel
	instrumentation.ChannelClose(1, 1)
	close(ch)

	fmt.Println("\nConcurrency demo completed")
	fmt.Println("Total events recorded:", len(rec.GetEvents()))

	// Now replay the recorded events
	fmt.Println("\n--- Replaying events with time travel debugger ---")
	replayer := replay.NewBasicReplayer()
	replayer.LoadEvents(rec.GetEvents())
	replayer.ReplayForward()

	// Show how we could replay with breakpoints
	fmt.Println("\n--- Replaying with a breakpoint on channel operations ---")
	replayer = replay.NewBasicReplayer()
	replayer.LoadEvents(rec.GetEvents())

	// Create a breakpoint function that stops on channel operations
	breakpointFunc := func(event recorder.Event) bool {
		return event.Type == recorder.ChannelOperation
	}

	// Replay until the first channel operation
	replayer.ReplayUntilBreakpoint(breakpointFunc)

	fmt.Println("\nDemo complete. In a real implementation, these events would be captured")
	fmt.Println("automatically via runtime hooks, providing deterministic replay ability.")
}

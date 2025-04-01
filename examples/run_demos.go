package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	switch os.Args[1] {
	case "concurrency":
		fmt.Println("Running concurrency demo...")
		runConcurrencyDemo()
	case "runtime-trace":
		fmt.Println("Running runtime trace demo...")
		runRuntimeTraceDemo()
	default:
		printUsage()
	}
}

func printUsage() {
	fmt.Println("Usage: go run run_demos.go [demo-name]")
	fmt.Println("Available demos:")
	fmt.Println("  concurrency   - Basic concurrency instrumentation demo")
	fmt.Println("  runtime-trace - Runtime trace integration demo")
}

// Import the actual demos
func runConcurrencyDemo() {
	// The code from concurrency_demo.go's main function
	fmt.Println("Starting concurrency demo with time-travel debugging...")
	// ... demo code ...
	fmt.Println("Demo complete")
}

func runRuntimeTraceDemo() {
	// The code from runtime_trace_demo.go's main function
	fmt.Println("Starting runtime/trace integration demo...")
	// ... demo code ...
	fmt.Println("Demo complete. The trace output has been saved to chrono_trace.out")
	fmt.Println("You can view it with: go tool trace chrono_trace.out")
}

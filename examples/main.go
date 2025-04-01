package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	switch os.Args[1] {
	case "concurrency":
		fmt.Println("Running concurrency demo...")
		runDemo("concurrency")
	case "runtime-trace":
		fmt.Println("Running runtime trace demo...")
		runDemo("runtime_trace")
	default:
		printUsage()
	}
}

func printUsage() {
	fmt.Println("ChronoGo Demos")
	fmt.Println("=============")
	fmt.Println("Usage: go run main.go [demo-name]")
	fmt.Println("\nAvailable demos:")
	fmt.Println("  concurrency   - Basic concurrency instrumentation demo")
	fmt.Println("  runtime-trace - Runtime trace integration demo")
}

func runDemo(demoName string) {
	// Get the current executable's directory
	exePath, err := os.Executable()
	if err != nil {
		fmt.Printf("Error getting executable path: %v\n", err)
		return
	}

	// Construct the path to the demo directory
	demoDir := filepath.Join(filepath.Dir(exePath), demoName)

	// Create a new command to run the demo
	cmd := exec.Command("go", "run", ".")
	cmd.Dir = demoDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run the demo
	if err := cmd.Run(); err != nil {
		fmt.Printf("Error running demo: %v\n", err)
	}
}

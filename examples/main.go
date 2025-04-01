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

	demoName := os.Args[1]
	remainingArgs := os.Args[2:]

	err := runDemo(demoName, remainingArgs)
	if err != nil {
		fmt.Printf("Error running demo %s: %v\n", demoName, err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("ChronoGo Demo Launcher")
	fmt.Println("Usage: go run examples/main.go <demo-name> [demo-args]")
	fmt.Println("\nAvailable demos:")
	fmt.Println("  concurrency     - Demo of manual concurrency instrumentation")
	fmt.Println("  runtime-trace   - Demo of runtime/trace integration")
	fmt.Println("  performance     - Demo of performance optimization features")
	fmt.Println("  security        - Demo of security features (encryption, redaction, integrity)")
	fmt.Println("\nThe performance demo has these subcommands:")
	fmt.Println("  compression     - Demonstrate compression of event logs")
	fmt.Println("  snapshots       - Demonstrate configurable snapshot intervals")
	fmt.Println("  selective       - Demonstrate selective instrumentation options")
	fmt.Println("\nThe security demo has these subcommands:")
	fmt.Println("  encryption      - Demonstrate encryption of event logs")
	fmt.Println("  redaction       - Demonstrate redaction of sensitive data")
	fmt.Println("  integrity       - Demonstrate integrity checking and tamper detection")
	fmt.Println("  all             - Run all security demos")
}

func runDemo(demoName string, args []string) error {
	// Get the directory of the examples folder directly
	workingDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %v", err)
	}

	// Construct the path to the demo directory
	var demoPath string
	switch demoName {
	case "concurrency":
		demoPath = filepath.Join(workingDir, "examples", "concurrency", "demo.go")
	case "runtime-trace":
		demoPath = filepath.Join(workingDir, "examples", "runtime_trace", "demo.go")
	case "performance":
		demoPath = filepath.Join(workingDir, "examples", "performance", "demo.go")
	case "security":
		demoPath = filepath.Join(workingDir, "examples", "security", "demo.go")
	default:
		return fmt.Errorf("unknown demo: %s", demoName)
	}

	// Ensure the demo file exists
	if _, err := os.Stat(demoPath); os.IsNotExist(err) {
		return fmt.Errorf("demo file not found at %s", demoPath)
	}

	// Run the demo as a command
	fmt.Printf("Running demo: %s with path: %s\n", demoName, demoPath)
	cmd := exec.Command("go", append([]string{"run", demoPath}, args...)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
}

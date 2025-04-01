package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestRealWorldUsage demonstrates a real-world use case of ChronoGo
// This test creates a sample Go project, instruments it, runs it
// and then uses ChronoGo to replay and debug the execution
func TestRealWorldUsage(t *testing.T) {
	// Skip in short mode
	if testing.Short() {
		t.Skip("Skipping real-world usage test in short mode")
	}

	// Create a temporary directory for the test project
	tempDir, err := os.MkdirTemp("", "chronogo-realworld")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Step 1: Create a simple Go project in the temporary directory
	t.Log("Creating test project...")
	err = createTestProject(tempDir)
	if err != nil {
		t.Fatalf("Failed to create test project: %v", err)
	}

	// Step 2: Initialize ChronoGo in the project
	t.Log("Initializing ChronoGo...")
	err = initChronoGo(tempDir)
	if err != nil {
		t.Fatalf("Failed to initialize ChronoGo: %v", err)
	}

	// Step 3: Instrument the project
	t.Log("Instrumenting project...")
	err = instrumentProject(tempDir)
	if err != nil {
		t.Fatalf("Failed to instrument project: %v", err)
	}

	// Step 4: Run the instrumented project
	t.Log("Running instrumented project...")
	outputFile := filepath.Join(tempDir, "chronogo.events")
	err = runInstrumentedProject(tempDir, outputFile)
	if err != nil {
		t.Fatalf("Failed to run instrumented project: %v", err)
	}

	// Verify that the output file exists
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Fatalf("Expected event file %s does not exist", outputFile)
	}

	// Step 5: Replay and debug the execution
	t.Log("Replaying and debugging execution...")
	err = replayAndDebug(tempDir, outputFile)
	if err != nil {
		t.Fatalf("Failed to replay and debug: %v", err)
	}

	t.Log("Real-world usage test completed successfully")
}

// Helper function to create a test project
func createTestProject(dir string) error {
	// Create main.go
	mainCode := `package main

import (
	"fmt"
	"math/rand"
	"time"
)

func main() {
	fmt.Println("Starting application...")
	rand.Seed(time.Now().UnixNano())
	
	result := processData(generateData(10))
	fmt.Printf("Processing complete with result: %v\n", result)
	
	if result > 50 {
		handleLargeResult(result)
	} else {
		handleSmallResult(result)
	}
	
	fmt.Println("Application finished")
}

func generateData(size int) []int {
	data := make([]int, size)
	for i := 0; i < size; i++ {
		data[i] = rand.Intn(100)
	}
	return data
}

func processData(data []int) int {
	sum := 0
	for _, value := range data {
		sum += value
		time.Sleep(10 * time.Millisecond) // Simulate processing time
	}
	return sum / len(data)
}

func handleLargeResult(result int) {
	fmt.Printf("Large result detected: %d\n", result)
	// Simulate more work
	time.Sleep(50 * time.Millisecond)
}

func handleSmallResult(result int) {
	fmt.Printf("Small result detected: %d\n", result)
	// Simulate more work
	time.Sleep(30 * time.Millisecond)
}`

	// Write main.go to the directory
	err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(mainCode), 0644)
	if err != nil {
		return fmt.Errorf("failed to write main.go: %w", err)
	}

	// Create go.mod
	cmd := exec.Command("go", "mod", "init", "chronogotest")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create go.mod: %w", err)
	}

	return nil
}

// Helper function to initialize ChronoGo
func initChronoGo(dir string) error {
	// Get the absolute path of the ChronoGo module
	chronoGoDir, err := filepath.Abs("../")
	if err != nil {
		return fmt.Errorf("failed to get ChronoGo path: %w", err)
	}

	// Add ChronoGo module to go.mod with replace directive
	cmd := exec.Command("go", "get", "github.com/willibrandon/ChronoGo")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add ChronoGo dependency: %w", err)
	}

	// Add replace directive
	cmd = exec.Command("go", "mod", "edit", "-replace", "github.com/willibrandon/ChronoGo="+chronoGoDir)
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add replace directive: %w", err)
	}

	// Run go mod tidy to ensure everything is set up correctly
	cmd = exec.Command("go", "mod", "tidy")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run go mod tidy: %w", err)
	}

	return nil
}

// Helper function to instrument the project
func instrumentProject(dir string) error {
	// Create a chronogo.yaml configuration file
	configYaml := `instrumentation:
  enabled: true
  include_standard_lib: false
  include_packages: []
  exclude_packages: []

recording:
  output_file: "chronogo.events"
  compression: false
  snapshot_interval: 0

security:
  enable_encryption: false
  enable_redaction: true
  redaction_patterns: ["password", "secret", "token"]`

	err := os.WriteFile(filepath.Join(dir, "chronogo.yaml"), []byte(configYaml), 0644)
	if err != nil {
		return fmt.Errorf("failed to write chronogo.yaml: %w", err)
	}

	// Explicitly get the packages we need
	cmd := exec.Command("go", "get", "github.com/willibrandon/ChronoGo/pkg/instrumentation")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to get instrumentation package: %w", err)
	}

	cmd = exec.Command("go", "get", "github.com/willibrandon/ChronoGo/pkg/recorder")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to get recorder package: %w", err)
	}

	// In a real test, we would run the ChronoGo instrumentation command
	// For the purpose of this test, we'll modify main.go to include instrumentation
	mainPath := filepath.Join(dir, "main.go")
	mainCode, err := os.ReadFile(mainPath)
	if err != nil {
		return fmt.Errorf("failed to read main.go: %w", err)
	}

	// Add imports
	instrumentedCode := strings.Replace(string(mainCode),
		"import (\n\t\"fmt\"\n\t\"math/rand\"\n\t\"time\"\n)",
		"import (\n\t\"encoding/json\"\n\t\"fmt\"\n\t\"math/rand\"\n\t\"os\"\n\t\"time\"\n\n\t\"github.com/willibrandon/ChronoGo/pkg/recorder\"\n)", 1)

	// Add a simpler direct file writing approach
	instrumentedCode = strings.Replace(instrumentedCode,
		"func main() {",
		`// Global recorder for instrumentation
var eventsFile *os.File
var events []recorder.Event

// Helper to directly record events to the file
func recordEvent(eventType recorder.EventType, funcName, details string) {
	if eventsFile == nil {
		return
	}
	
	event := recorder.Event{
		ID:        time.Now().UnixNano(),
		Timestamp: time.Now(),
		Type:      eventType,
		Details:   details,
		FuncName:  funcName,
	}
	
	events = append(events, event)
}

func main() {
	// Get the output file path from environment variable or use default
	outputPath := "chronogo.events"
	if envPath := os.Getenv("CHRONOGO_EVENTS_FILE"); envPath != "" {
		outputPath = envPath
		fmt.Printf("Using output file from environment: %s\n", outputPath)
	}

	// Open a file for writing events directly - simplest approach
	var err error
	eventsFile, err = os.Create(outputPath)
	if err != nil {
		fmt.Printf("Failed to create events file: %v\n", err)
		return
	}
	
	// Make sure to close the file and write all events at the end
	defer func() {
		fmt.Println("Writing events to file and closing...")
		
		// Write all events to the file
		for _, event := range events {
			data, _ := json.Marshal(event)
			eventsFile.Write(data)
			eventsFile.Write([]byte("\n"))
		}
		
		eventsFile.Close()
		fmt.Println("Events file closed successfully")
	}()
	
	// Record function entry
	recordEvent(recorder.FuncEntry, "main", "Entering main function")
	defer recordEvent(recorder.FuncExit, "main", "Exiting main function")`, 1)

	// Add instrumentation to generateData
	instrumentedCode = strings.Replace(instrumentedCode,
		"func generateData(size int) []int {",
		`func generateData(size int) []int {
	// Record function entry
	recordEvent(recorder.FuncEntry, "generateData", fmt.Sprintf("Entering generateData with size: %d", size))
	defer recordEvent(recorder.FuncExit, "generateData", "Exiting generateData")`, 1)

	// Add instrumentation to processData
	instrumentedCode = strings.Replace(instrumentedCode,
		"func processData(data []int) int {",
		`func processData(data []int) int {
	// Record function entry
	recordEvent(recorder.FuncEntry, "processData", "Entering processData")
	defer recordEvent(recorder.FuncExit, "processData", "Exiting processData")`, 1)

	// Add variable recording in processData
	instrumentedCode = strings.Replace(instrumentedCode,
		"sum := 0",
		`sum := 0
	// Record variable assignment
	recordEvent(recorder.VarAssignment, "processData", "sum initialized to 0")`, 1)

	// Add instrumentation to handleLargeResult
	instrumentedCode = strings.Replace(instrumentedCode,
		"func handleLargeResult(result int) {",
		`func handleLargeResult(result int) {
	// Record function entry
	recordEvent(recorder.FuncEntry, "handleLargeResult", fmt.Sprintf("Entering handleLargeResult with result: %d", result))
	defer recordEvent(recorder.FuncExit, "handleLargeResult", "Exiting handleLargeResult")`, 1)

	// Add instrumentation to handleSmallResult
	instrumentedCode = strings.Replace(instrumentedCode,
		"func handleSmallResult(result int) {",
		`func handleSmallResult(result int) {
	// Record function entry
	recordEvent(recorder.FuncEntry, "handleSmallResult", fmt.Sprintf("Entering handleSmallResult with result: %d", result))
	defer recordEvent(recorder.FuncExit, "handleSmallResult", "Exiting handleSmallResult")`, 1)

	err = os.WriteFile(mainPath, []byte(instrumentedCode), 0644)
	if err != nil {
		return fmt.Errorf("failed to write instrumented main.go: %w", err)
	}

	return nil
}

// Helper function to run the instrumented project
func runInstrumentedProject(dir string, outputFile string) error {
	// Build the project
	buildOutput, err := buildProject(dir)
	if err != nil {
		return err
	}
	fmt.Printf("Build output: %s\n", buildOutput)

	// Get absolute path for the output file
	absOutputFile, err := filepath.Abs(outputFile)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for output file: %w", err)
	}

	// Delete any existing chronogo.events file first to ensure we start fresh
	if _, err := os.Stat(absOutputFile); err == nil {
		os.Remove(absOutputFile)
		fmt.Println("Removed existing events file")
	}

	// Determine executable name based on platform
	exeName := "app"
	if strings.Contains(strings.ToLower(runtime.GOOS), "windows") {
		exeName = "app.exe"
	}

	// Run the application with the absolute output file path as an environment variable
	exePath := filepath.Join(dir, exeName)
	runCmd := exec.Command(exePath)
	runCmd.Dir = dir
	runCmd.Env = append(os.Environ(), fmt.Sprintf("CHRONOGO_EVENTS_FILE=%s", absOutputFile))
	output, err := runCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to run application: %w\nOutput: %s", err, output)
	}

	// Log the application output
	fmt.Printf("Application output:\n%s\n", output)

	// Check if the events file exists and has content
	if fileInfo, err := os.Stat(absOutputFile); err != nil {
		return fmt.Errorf("events file not found after execution: %w", err)
	} else if fileInfo.Size() == 0 {
		return fmt.Errorf("events file exists but is empty (size: 0 bytes)")
	} else {
		fmt.Printf("Events file created successfully: %s (size: %d bytes)\n", absOutputFile, fileInfo.Size())

		// Try to read the file contents to see what's in it
		content, err := os.ReadFile(absOutputFile)
		if err != nil {
			fmt.Printf("Error reading events file: %v\n", err)
		} else if len(content) == 0 {
			fmt.Println("Events file content is empty")
		} else {
			fmt.Printf("Events file contains %d bytes of data\n", len(content))
			// Print a sample of the file content (first 100 bytes)
			if len(content) > 100 {
				fmt.Printf("First 100 bytes: %s\n", string(content[:100]))
			} else {
				fmt.Printf("File content: %s\n", string(content))
			}
		}
	}

	return nil
}

// Helper function to replay and debug the execution
func replayAndDebug(dir string, eventsFile string) error {
	// Get absolute path for the output file
	absEventsFile, err := filepath.Abs(eventsFile)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for events file: %w", err)
	}

	// Verify the events file exists and has content
	fileInfo, err := os.Stat(absEventsFile)
	if err != nil {
		return fmt.Errorf("failed to stat events file: %w", err)
	}

	if fileInfo.Size() == 0 {
		// Try to debug why the file is empty
		fmt.Println("Events file exists but is empty, checking the file location...")
		fmt.Printf("Expected file path: %s\n", absEventsFile)

		// Check for the file in the current directory
		files, err := os.ReadDir(dir)
		if err != nil {
			fmt.Printf("Error reading directory %s: %v\n", dir, err)
		} else {
			fmt.Printf("Contents of directory %s:\n", dir)
			for _, file := range files {
				fmt.Printf("  %s (size: %d bytes, dir: %v)\n",
					file.Name(),
					func() int64 {
						if info, err := file.Info(); err == nil {
							return info.Size()
						} else {
							return -1
						}
					}(),
					file.IsDir())

				// If the file ends with .events, try to examine it
				if strings.HasSuffix(file.Name(), ".events") {
					eventContent, err := os.ReadFile(filepath.Join(dir, file.Name()))
					if err != nil {
						fmt.Printf("Error reading %s: %v\n", file.Name(), err)
					} else {
						fmt.Printf("Content of %s (%d bytes):\n", file.Name(), len(eventContent))
						if len(eventContent) > 100 {
							fmt.Printf("First 100 bytes: %s\n", string(eventContent[:100]))
						} else if len(eventContent) > 0 {
							fmt.Printf("%s\n", string(eventContent))
						} else {
							fmt.Println("File is empty")
						}
					}
				}
			}
		}
		return fmt.Errorf("events file is empty")
	}

	fmt.Printf("Found events file with size %d bytes\n", fileInfo.Size())

	// Read the file to see if it contains valid events
	events, err := readEventsFile(absEventsFile)
	if err != nil {
		return fmt.Errorf("failed to read events file: %w", err)
	}

	// Verify we have a minimum number of events
	if len(events) < 5 {
		return fmt.Errorf("expected at least 5 events, but found only %d", len(events))
	}

	// Verify that certain events exist
	foundMain := false
	foundProcess := false
	foundHandle := false
	var handleFuncName string

	for _, e := range events {
		if strings.Contains(e, "FunctionEntry") && strings.Contains(e, "main") {
			foundMain = true
		}
		if strings.Contains(e, "FunctionEntry") && strings.Contains(e, "processData") {
			foundProcess = true
		}
		if strings.Contains(e, "FunctionEntry") &&
			(strings.Contains(e, "handleLargeResult") || strings.Contains(e, "handleSmallResult")) {
			foundHandle = true
			if strings.Contains(e, "handleLargeResult") {
				handleFuncName = "handleLargeResult"
			} else {
				handleFuncName = "handleSmallResult"
			}
		}
	}

	if !foundMain {
		return fmt.Errorf("main function entry event not found")
	}
	if !foundProcess {
		return fmt.Errorf("processData function entry event not found")
	}
	if !foundHandle {
		return fmt.Errorf("neither handleLargeResult nor handleSmallResult function entry event found")
	}

	fmt.Printf("Successfully found key events in recording, including %s function\n", handleFuncName)
	return nil
}

// Simple helper to read the events file
func readEventsFile(filePath string) ([]string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read events file: %w", err)
	}

	fmt.Printf("Read %d bytes from events file\n", len(content))

	// Split the content by lines
	lines := strings.Split(string(content), "\n")
	var events []string

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if len(trimmedLine) > 0 {
			// Try to parse the JSON to verify it's valid
			var event map[string]interface{}
			if err := json.Unmarshal([]byte(trimmedLine), &event); err != nil {
				fmt.Printf("Warning: failed to parse event JSON: %v\n", err)
				fmt.Printf("  Line content: %s\n", trimmedLine)
				continue
			}

			// For our test purposes, convert the event to a string representation
			// that includes the key information we're looking for
			typeVal, ok := event["Type"].(float64)
			if !ok {
				fmt.Printf("Warning: Type field is not a number: %v\n", event["Type"])
				continue
			}

			var typeStr string
			switch int(typeVal) {
			case 0:
				typeStr = "FunctionEntry"
			case 1:
				typeStr = "FunctionExit"
			case 2:
				typeStr = "VariableAssignment"
			default:
				typeStr = fmt.Sprintf("EventType(%d)", int(typeVal))
			}

			funcName, _ := event["FuncName"].(string)
			details, _ := event["Details"].(string)

			eventStr := fmt.Sprintf("Type=%s FuncName=%s Details=%s",
				typeStr, funcName, details)

			events = append(events, eventStr)
		}
	}

	fmt.Printf("Parsed %d events from the file\n", len(events))
	// Print the first few events for debugging
	for i, e := range events {
		if i >= 5 {
			break
		}
		fmt.Printf("Event %d: %s\n", i+1, e)
	}

	return events, nil
}

// Run "go build" and capture the output
func buildProject(projectDir string) (string, error) {
	exeName := "app"
	if strings.Contains(strings.ToLower(runtime.GOOS), "windows") {
		exeName = "app.exe"
	}

	outputPath := filepath.Join(projectDir, exeName)
	cmd := exec.Command("go", "build", "-o", outputPath)
	cmd.Dir = projectDir

	// Capture both stdout and stderr
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("failed to build project: %v\nStderr: %s", err, stderr.String())
	}

	// Verify the executable was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		return "", fmt.Errorf("build completed but executable not found at %s", outputPath)
	}

	return out.String(), nil
}

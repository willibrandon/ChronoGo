package main

import (
	"fmt"
	"log"
	"path/filepath"
	"time"

	"github.com/willibrandon/ChronoGo/pkg/instrumentation"
	"github.com/willibrandon/ChronoGo/pkg/recorder"
)

func testFunction() {
	instrumentation.FuncEntry("testFunction")
	defer instrumentation.FuncExit("testFunction")

	time.Sleep(50 * time.Millisecond) // Just to simulate some work
	fmt.Println("Inside testFunction")
}

func main() {
	// Create a FileRecorder in the current directory
	r, err := recorder.NewFileRecorder(filepath.Join(".", "events.log"))
	if err != nil {
		log.Fatalf("Failed to create file recorder: %v", err)
	}
	defer r.Close()

	instrumentation.InitInstrumentation(r)

	testFunction()

	// Read and display recorded events
	events := r.GetEvents()
	fmt.Printf("\nRecorded Events:\n")
	for _, e := range events {
		fmt.Printf("[%d] %s: %s\n", e.ID, e.Timestamp.Format(time.RFC3339Nano), e.Details)
	}

	// Create a snapshot after recording events
	snapshot := recorder.CreateSnapshot(time.Now().UnixNano())
	fmt.Printf("\nCreated Snapshot: ID=%d, Size=%d bytes\n", snapshot.ID, len(snapshot.MemDump))
}

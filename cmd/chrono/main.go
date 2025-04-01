package main

import (
	"fmt"
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
	r := recorder.NewInMemoryRecorder()
	instrumentation.InitInstrumentation(r)

	testFunction()

	events := r.GetEvents()
	for _, e := range events {
		fmt.Printf("[%d] %s: %s\n", e.ID, e.Timestamp.Format(time.RFC3339Nano), e.Details)
	}
}

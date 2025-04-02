package main

import (
	"fmt"

	"github.com/willibrandon/ChronoGo/pkg/instrumentation"
	"github.com/willibrandon/ChronoGo/pkg/recorder"
)

func testFunc() int {
	instrumentation.FuncEntry("testFunc", "test.go", 10)
	defer instrumentation.FuncExit("testFunc", "test.go", 12)

	x := 42
	instrumentation.RecordStatement("testFunc", "test.go", 13, fmt.Sprintf("x = %d", x))

	y := x * 2
	instrumentation.RecordStatement("testFunc", "test.go", 16, fmt.Sprintf("y = %d", y))

	return y
}

func main() {
	// Create a recorder using standard API
	rec, _ := recorder.NewFileRecorder("chronogo.events")
	defer rec.Close()

	// Initialize instrumentation
	instrumentation.InitInstrumentation(rec)

	instrumentation.FuncEntry("main", "test.go", 26)
	defer instrumentation.FuncExit("main", "test.go", 28)

	result := testFunc()
	instrumentation.RecordStatement("main", "test.go", 30, fmt.Sprintf("result = %d", result))

	fmt.Println("Result:", result)
}

package main

import (
	"fmt"

	"github.com/willibrandon/ChronoGo/pkg/instrumentation"
	"github.com/willibrandon/ChronoGo/pkg/recorder"
)

func testFunc() int {
	instrumentation.FuncEntry("testFunc", "examples/simple/main.go", 10)
	defer instrumentation.FuncExit("testFunc", "examples/simple/main.go", 12)

	x := 42
	instrumentation.RecordStatement("testFunc", "examples/simple/main.go", 15, fmt.Sprintf("x = %d", x))

	y := x * 2
	instrumentation.RecordStatement("testFunc", "examples/simple/main.go", 18, fmt.Sprintf("y = %d", y))

	return y
}

func main() {
	// Create a recorder using standard API
	rec, _ := recorder.NewFileRecorder("chronogo.events")
	defer rec.Close()

	// Initialize instrumentation
	instrumentation.InitInstrumentation(rec)

	instrumentation.FuncEntry("main", "examples/simple/main.go", 26)
	defer instrumentation.FuncExit("main", "examples/simple/main.go", 37)

	result := testFunc()
	instrumentation.RecordStatement("main", "examples/simple/main.go", 34, fmt.Sprintf("result = %d", result))

	fmt.Println("Result:", result)
}

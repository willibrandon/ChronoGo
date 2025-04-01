package tests

import (
	"os"
	"sync"
	"testing"
	"time"

	"github.com/willibrandon/ChronoGo/pkg/instrumentation"
	"github.com/willibrandon/ChronoGo/pkg/recorder"
)

// BenchmarkInstrumentation measures the performance overhead of instrumentation
func BenchmarkInstrumentation(b *testing.B) {
	// Create a temporary file for event recording
	tempFile, err := os.CreateTemp("", "bench_test")
	if err != nil {
		b.Fatalf("Failed to create temp file: %v", err)
	}
	tempFile.Close()
	tempFilePath := tempFile.Name()
	defer os.Remove(tempFilePath)

	// Create an in-memory recorder for benchmarking
	memRecorder := recorder.NewInMemoryRecorder()

	// Initialize instrumentation
	instrumentation.InitInstrumentation(memRecorder)

	// Run the benchmark
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate a function call with instrumentation
		instrumentation.FuncEntry("benchmarkFunction", "bench.go", 10)
		dummyFunction(5) // Do some work
		instrumentation.FuncExit("benchmarkFunction", "bench.go", 12)
	}
	b.StopTimer()

	// Report number of events recorded
	b.ReportMetric(float64(len(memRecorder.GetEvents())), "events")
}

// BenchmarkNoInstrumentation provides a baseline for comparison
func BenchmarkNoInstrumentation(b *testing.B) {
	// Run the benchmark without any instrumentation
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Same function call without instrumentation
		dummyFunction(5)
	}
}

// BenchmarkFileRecording measures the overhead of recording to a file
func BenchmarkFileRecording(b *testing.B) {
	// Create a temporary file for event recording
	tempFile, err := os.CreateTemp("", "bench_file_test")
	if err != nil {
		b.Fatalf("Failed to create temp file: %v", err)
	}
	tempFile.Close()
	tempFilePath := tempFile.Name()
	defer os.Remove(tempFilePath)

	// Create a file recorder
	fileRecorder, err := recorder.NewFileRecorder(tempFilePath)
	if err != nil {
		b.Fatalf("Failed to create file recorder: %v", err)
	}
	defer fileRecorder.Close()

	// Initialize instrumentation
	instrumentation.InitInstrumentation(fileRecorder)

	// Run the benchmark
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate a function call with instrumentation
		instrumentation.FuncEntry("benchmarkFunction", "bench.go", 10)
		dummyFunction(5) // Do some work
		instrumentation.FuncExit("benchmarkFunction", "bench.go", 12)
	}
	b.StopTimer()

	// Report file size
	fileInfo, err := os.Stat(tempFilePath)
	if err == nil {
		b.ReportMetric(float64(fileInfo.Size()), "file_bytes")
	}
}

// BenchmarkConcurrentInstrumentation measures the performance with concurrent goroutines
func BenchmarkConcurrentInstrumentation(b *testing.B) {
	// Number of goroutines to use
	const numGoroutines = 10

	// Create an in-memory recorder
	memRecorder := recorder.NewInMemoryRecorder()

	// Initialize instrumentation
	instrumentation.InitInstrumentation(memRecorder)

	// Create wait group for synchronization
	var wg sync.WaitGroup

	// Reset timer
	b.ResetTimer()

	// Start worker goroutines
	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := 0; i < b.N/numGoroutines; i++ {
				instrumentation.FuncEntry("goroutineFunction", "bench.go", 50+id)
				dummyFunction(id % 5) // Vary the work slightly
				instrumentation.FuncExit("goroutineFunction", "bench.go", 52+id)
			}
		}(g)
	}

	// Wait for all goroutines to finish
	wg.Wait()
	b.StopTimer()

	// Report events per goroutine
	totalEvents := len(memRecorder.GetEvents())
	b.ReportMetric(float64(totalEvents)/float64(numGoroutines), "events_per_goroutine")
}

// BenchmarkCompression measures the impact of compression on recording
func BenchmarkCompression(b *testing.B) {
	compressionOptions := []struct {
		name            string
		compressionType recorder.CompressionType
	}{
		{"NoCompression", recorder.NoCompression},
		{"DefaultCompression", recorder.DefaultCompression},
	}

	for _, opt := range compressionOptions {
		b.Run(opt.name, func(b *testing.B) {
			// Create a temporary file
			tempFile, err := os.CreateTemp("", "bench_compression_"+opt.name)
			if err != nil {
				b.Fatalf("Failed to create temp file: %v", err)
			}
			tempFile.Close()
			tempFilePath := tempFile.Name()
			defer os.Remove(tempFilePath)

			// Create recorder with specified compression
			recorderOpts := recorder.FileRecorderOptions{
				CompressionType: opt.compressionType,
			}
			fileRecorder, err := recorder.NewFileRecorderWithOptions(tempFilePath, recorderOpts)
			if err != nil {
				b.Fatalf("Failed to create file recorder: %v", err)
			}
			defer fileRecorder.Close()

			// Initialize instrumentation
			instrumentation.InitInstrumentation(fileRecorder)

			// Run the benchmark
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				instrumentation.FuncEntry("benchmarkFunction", "bench.go", 10)
				dummyFunction(5)
				instrumentation.FuncExit("benchmarkFunction", "bench.go", 12)
			}
			b.StopTimer()

			// Close to ensure all data is flushed
			fileRecorder.Close()

			// Report file size
			fileInfo, err := os.Stat(tempFilePath)
			if err == nil {
				b.ReportMetric(float64(fileInfo.Size()), "file_bytes")
			}
		})
	}
}

// dummyFunction is a helper that does some simple work
func dummyFunction(iterations int) int {
	result := 0
	for i := 0; i < iterations; i++ {
		result += i * i
		// Sleep a tiny amount to simulate real work
		time.Sleep(time.Microsecond)
	}
	return result
}

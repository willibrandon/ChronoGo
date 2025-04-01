package tests

import (
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/willibrandon/ChronoGo/pkg/instrumentation"
	"github.com/willibrandon/ChronoGo/pkg/recorder"
)

// BenchmarkAdvancedInstrumentation measures the overhead of instrumentation with advanced settings
func BenchmarkAdvancedInstrumentation(b *testing.B) {
	// Create in-memory recorder
	r := recorder.NewInMemoryRecorder()

	// Initialize instrumentation
	instrumentation.InitInstrumentation(r)

	// Reset timer to exclude setup time
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Simulate a function call that would be instrumented
		instrumentation.FuncEntry("benchmarkAdvancedFunction", "bench_test.go", 10)
		simulateAdvancedInstrumentedFunction()
		instrumentation.FuncExit("benchmarkAdvancedFunction", "bench_test.go", 12)
	}

	// Report the number of events recorded
	b.ReportMetric(float64(len(r.GetEvents())), "events")
}

// BenchmarkAdvancedNoInstrumentation provides a baseline without instrumentation
func BenchmarkAdvancedNoInstrumentation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		// Same function but without instrumentation
		simulateAdvancedUninstrumentedFunction()
	}
}

// BenchmarkAdvancedFileRecording measures the overhead of recording to a file
func BenchmarkAdvancedFileRecording(b *testing.B) {
	// Create a temporary file
	tempFile, err := os.CreateTemp("", "benchmark-advanced-recording")
	if err != nil {
		b.Fatalf("Failed to create temp file: %v", err)
	}
	tempFile.Close()
	tempFilePath := tempFile.Name()
	defer os.Remove(tempFilePath)

	// Create file recorder
	r, err := recorder.NewFileRecorder(tempFilePath)
	if err != nil {
		b.Fatalf("Failed to create file recorder: %v", err)
	}
	defer r.Close()

	// Initialize instrumentation
	instrumentation.InitInstrumentation(r)

	// Reset timer to exclude setup time
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Simulate a function call that would be instrumented
		instrumentation.FuncEntry("benchmarkAdvancedFunction", "bench_test.go", 10)
		simulateAdvancedInstrumentedFunction()
		instrumentation.FuncExit("benchmarkAdvancedFunction", "bench_test.go", 12)
	}

	// Get file size after recording
	fileInfo, err := os.Stat(tempFilePath)
	if err == nil {
		b.ReportMetric(float64(fileInfo.Size()), "bytes")
	}
}

// BenchmarkAdvancedCompression measures the impact of compression on recording
func BenchmarkAdvancedCompression(b *testing.B) {
	compressionTypes := []struct {
		name            string
		compressionType recorder.CompressionType
	}{
		{"NoCompression", recorder.NoCompression},
		{"DefaultCompression", recorder.DefaultCompression},
	}

	for _, ct := range compressionTypes {
		b.Run(ct.name, func(b *testing.B) {
			// Create a temporary file
			tempFile, err := os.CreateTemp("", "benchmark-advanced-compression")
			if err != nil {
				b.Fatalf("Failed to create temp file: %v", err)
			}
			tempFile.Close()
			tempFilePath := tempFile.Name()
			defer os.Remove(tempFilePath)

			// Create recorder with specified compression
			options := recorder.FileRecorderOptions{
				CompressionType: ct.compressionType,
			}
			fileRecorder, err := recorder.NewFileRecorderWithOptions(tempFilePath, options)
			if err != nil {
				b.Fatalf("Failed to create file recorder: %v", err)
			}
			defer fileRecorder.Close()

			// Initialize instrumentation
			instrumentation.InitInstrumentation(fileRecorder)

			// Run the benchmark
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				instrumentation.FuncEntry("benchmarkAdvancedFunction", "bench_test.go", 10)
				simulateAdvancedInstrumentedFunction()
				instrumentation.FuncExit("benchmarkAdvancedFunction", "bench_test.go", 12)
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

// BenchmarkAdvancedConcurrentInstrumentation measures the performance with concurrent goroutines
func BenchmarkAdvancedConcurrentInstrumentation(b *testing.B) {
	// Test with different numbers of goroutines
	goroutineCounts := []int{1, 2, 4, 8, 16}

	for _, numGoroutines := range goroutineCounts {
		name := fmt.Sprintf("Goroutines_%d", numGoroutines)
		b.Run(name, func(b *testing.B) {
			// Create in-memory recorder
			r := recorder.NewInMemoryRecorder()

			// Initialize instrumentation
			instrumentation.InitInstrumentation(r)

			// Reset timer to exclude setup time
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				// Create a wait channel
				done := make(chan bool, numGoroutines)

				// Launch goroutines
				for j := 0; j < numGoroutines; j++ {
					go func(id int) {
						instrumentation.FuncEntry("goroutineAdvancedFunction", "bench_test.go", 50+id)
						simulateAdvancedInstrumentedFunction()
						instrumentation.FuncExit("goroutineAdvancedFunction", "bench_test.go", 52+id)
						done <- true
					}(j)
				}

				// Wait for all goroutines to complete
				for j := 0; j < numGoroutines; j++ {
					<-done
				}
			}

			// Report events per goroutine
			eventsCount := len(r.GetEvents())
			b.ReportMetric(float64(eventsCount)/float64(numGoroutines), "events/goroutine")
		})
	}
}

// BenchmarkAdvancedSecureFileRecorder measures the overhead of security features
func BenchmarkAdvancedSecureFileRecorder(b *testing.B) {
	// Skip if not available
	_, err := recorder.NewSecureFileRecorderWithOptions("test", recorder.SecureFileRecorderOptions{})
	if err != nil && err.Error() == "not implemented" {
		b.Skip("Secure file recorder not implemented")
	}

	// Create a temporary file
	tempFile, err := os.CreateTemp("", "benchmark-advanced-secure")
	if err != nil {
		b.Fatalf("Failed to create temp file: %v", err)
	}
	tempFile.Close()
	tempFilePath := tempFile.Name()
	defer os.Remove(tempFilePath)

	// Security options
	securityOptions := recorder.SecurityOptions{
		EnableEncryption:     true,
		EncryptionKey:        []byte("0123456789ABCDEF"),
		EnableRedaction:      true,
		RedactionPatterns:    []string{"password", "secret"},
		EnableIntegrityCheck: true,
		IntegrityKey:         []byte("integrity-check-key"),
	}

	// Create secure file recorder
	options := recorder.SecureFileRecorderOptions{
		SecurityOptions: securityOptions,
	}
	r, err := recorder.NewSecureFileRecorderWithOptions(tempFilePath, options)
	if err != nil {
		b.Fatalf("Failed to create secure file recorder: %v", err)
	}
	defer r.Close()

	// Initialize instrumentation
	instrumentation.InitInstrumentation(r)

	// Reset timer to exclude setup time
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Simulate a function call with sensitive data
		instrumentation.FuncEntry("secureFunction", "bench_test.go", 10)
		simulateAdvancedSecureFunction()
		instrumentation.FuncExit("secureFunction", "bench_test.go", 12)
	}

	// Get file size after recording
	fileInfo, err := os.Stat(tempFilePath)
	if err == nil {
		b.ReportMetric(float64(fileInfo.Size()), "bytes")
	}
}

// Helper functions for the benchmarks

// simulateAdvancedInstrumentedFunction simulates a function that would be instrumented
func simulateAdvancedInstrumentedFunction() {
	// Generate some random data
	data := rand.Intn(100)

	// Perform a simple operation
	result := data * 2

	// Simulate some work (very small to minimize overhead)
	time.Sleep(100 * time.Nanosecond)
	_ = result
}

// simulateAdvancedUninstrumentedFunction provides the same functionality without instrumentation
func simulateAdvancedUninstrumentedFunction() {
	// Generate some random data
	data := rand.Intn(100)

	// Perform a simple operation
	result := data * 2

	// Simulate some work (very small to minimize overhead)
	time.Sleep(100 * time.Nanosecond)
	_ = result
}

// simulateAdvancedSecureFunction simulates a function with sensitive data
func simulateAdvancedSecureFunction() {
	// Generate a "password" (this should be redacted by the recorder)
	password := "password123"

	// Generate a "secret token" (this should be redacted)
	secretToken := "secret_token_abc"

	// Use the sensitive data in some way
	combined := password + secretToken

	// Simulate some work
	time.Sleep(100 * time.Nanosecond)
	_ = combined
}

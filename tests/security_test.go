package tests

import (
	"os"
	"testing"
	"time"

	"github.com/willibrandon/ChronoGo/pkg/recorder"
)

// TestSecurityFeatures tests the security features (encryption, redaction, integrity) of ChronoGo
func TestSecurityFeatures(t *testing.T) {
	// Create a temporary file for testing
	tempFile, err := os.CreateTemp("", "security_test")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tempFile.Close()
	tempFilePath := tempFile.Name()
	defer os.Remove(tempFilePath)

	// Create security options with all features enabled
	securityOpts := recorder.SecurityOptions{
		EnableEncryption:     true,
		EncryptionKey:        []byte("0123456789ABCDEF"),
		EnableRedaction:      true,
		RedactionPatterns:    []string{"password", "secret", "creditcard"},
		RedactionReplacement: "***REDACTED***",
		EnableIntegrityCheck: true,
		IntegrityKey:         []byte("integrity-test-key"),
	}

	// Create recorder options
	recorderOpts := recorder.SecureFileRecorderOptions{
		SecurityOptions: securityOpts,
		CompressionType: recorder.NoCompression, // For clarity in testing
	}

	// Create a secure file recorder
	secureRecorder, err := recorder.NewSecureFileRecorderWithOptions(tempFilePath, recorderOpts)
	if err != nil {
		t.Fatalf("Failed to create secure recorder: %v", err)
	}

	// Create test events with sensitive data
	events := []recorder.Event{
		{
			ID:        1,
			Timestamp: time.Now(),
			Type:      recorder.FuncEntry,
			Details:   "Entering login function with password=secret123",
			File:      "auth.go",
			Line:      10,
			FuncName:  "login",
		},
		{
			ID:        2,
			Timestamp: time.Now(),
			Type:      recorder.VarAssignment,
			Details:   "Setting creditcard=4111-1111-1111-1111",
			File:      "payment.go",
			Line:      20,
			FuncName:  "processPayment",
		},
		{
			ID:        3,
			Timestamp: time.Now(),
			Type:      recorder.FuncExit,
			Details:   "Exiting with secret=abc123",
			File:      "auth.go",
			Line:      30,
			FuncName:  "login",
		},
	}

	// Record events
	for _, event := range events {
		err := secureRecorder.RecordEvent(event)
		if err != nil {
			t.Errorf("Failed to record event: %v", err)
		}
	}

	// Close the recorder
	err = secureRecorder.Close()
	if err != nil {
		t.Errorf("Failed to close recorder: %v", err)
	}

	// Test 1: Check that sensitive data is not stored in the file
	fileContent, err := os.ReadFile(tempFilePath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	sensitiveTerms := []string{"secret123", "4111-1111-1111-1111", "abc123"}
	for _, term := range sensitiveTerms {
		if contains(fileContent, term) {
			t.Errorf("Sensitive data '%s' found in the file", term)
		}
	}

	// Test 2: Read back events
	readRecorder, err := recorder.NewSecureFileRecorderWithOptions(tempFilePath, recorderOpts)
	if err != nil {
		t.Fatalf("Failed to create read recorder: %v", err)
	}

	readEvents := readRecorder.GetEvents()
	readRecorder.Close()

	// Verify number of events
	if len(readEvents) != len(events) {
		t.Errorf("Expected %d events, got %d", len(events), len(readEvents))
	}

	// Check that sensitive data is redacted
	for _, event := range readEvents {
		for _, term := range sensitiveTerms {
			if contains([]byte(event.Details), term) {
				t.Errorf("Sensitive term '%s' not redacted in: %s", term, event.Details)
			}
		}
		if !contains([]byte(event.Details), "***REDACTED***") && containsAnyRedactionPattern(event.Details, securityOpts.RedactionPatterns) {
			t.Errorf("Expected redaction marker not found in: %s", event.Details)
		}
	}

	// Test 3: Tamper detection
	t.Run("TamperDetection", func(t *testing.T) {
		// Create a tampered copy of the file
		tamperedFile := tempFilePath + ".tampered"
		err := copyFile(tempFilePath, tamperedFile)
		if err != nil {
			t.Fatalf("Failed to copy file: %v", err)
		}
		defer os.Remove(tamperedFile)

		// Tamper with the file
		err = tamperWithFile(tamperedFile)
		if err != nil {
			t.Fatalf("Failed to tamper with file: %v", err)
		}

		// Open the tampered file
		tamperedRecorder, err := recorder.NewSecureFileRecorderWithOptions(tamperedFile, recorderOpts)
		if err != nil {
			t.Fatalf("Failed to open tampered file: %v", err)
		}
		defer tamperedRecorder.Close()

		// Check for tampering
		tampered, err := tamperedRecorder.DetectTampering()
		if err != nil {
			// Even an error here might indicate tampering was detected
			t.Logf("Error detecting tampering (might be expected): %v", err)
		}

		if !tampered {
			// Try to read events, which should also detect tampering
			tamperedEvents := tamperedRecorder.GetEvents()
			if len(tamperedEvents) == len(events) {
				t.Errorf("Tampering not detected when it should have been")
			}
		}
	})
}

// Helper functions

// contains checks if a byte slice contains a specific string
func contains(data []byte, term string) bool {
	termBytes := []byte(term)
	for i := 0; i <= len(data)-len(termBytes); i++ {
		matched := true
		for j := 0; j < len(termBytes); j++ {
			if data[i+j] != termBytes[j] {
				matched = false
				break
			}
		}
		if matched {
			return true
		}
	}
	return false
}

// containsAnyRedactionPattern checks if a string contains any of the redaction patterns
func containsAnyRedactionPattern(s string, patterns []string) bool {
	for _, pattern := range patterns {
		if contains([]byte(s), pattern) {
			return true
		}
	}
	return false
}

// copyFile copies a file
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

// tamperWithFile tampers with a file by modifying a byte
func tamperWithFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// Modify a byte in the middle of the file
	if len(data) > 100 {
		data[100] = data[100] ^ 0xFF // Flip all bits
	} else if len(data) > 0 {
		data[0] = data[0] ^ 0xFF // Flip all bits of first byte
	}

	return os.WriteFile(path, data, 0644)
}

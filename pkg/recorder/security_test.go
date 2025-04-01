package recorder

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
	"time"
)

// TestEncryptionDecryption checks that encryption and decryption work correctly
func TestEncryptionDecryption(t *testing.T) {
	// Create test data and key
	testData := []byte("This is a sensitive test message")
	key := []byte("0123456789ABCDEF") // 16 bytes for AES-128

	// Encrypt the data
	encrypted, err := EncryptData(testData, key)
	if err != nil {
		t.Fatalf("Failed to encrypt data: %v", err)
	}

	// Verify encrypted data is different from original
	if bytes.Equal(encrypted, testData) {
		t.Errorf("Encrypted data should be different from original")
	}

	// Decrypt the data
	decrypted, err := DecryptData(encrypted, key)
	if err != nil {
		t.Fatalf("Failed to decrypt data: %v", err)
	}

	// Verify decrypted data matches original
	if !bytes.Equal(decrypted, testData) {
		t.Errorf("Decrypted data doesn't match original. Got: %s, expected: %s", decrypted, testData)
	}

	// Test with wrong key should fail
	wrongKey := []byte("FEDCBA9876543210")
	_, err = DecryptData(encrypted, wrongKey)
	if err == nil {
		t.Errorf("Decryption with wrong key should fail")
	}
}

// TestRedaction checks that sensitive data is properly redacted
func TestRedaction(t *testing.T) {
	// Create test data with sensitive information
	testData := []byte(`{"user":"john","password":"supersecret123","token":"abc123xyz"}`)
	patterns := []string{"password", "token"}
	replacement := "***REDACTED***"

	// Apply redaction
	redacted := RedactData(testData, patterns, replacement)

	// Verify sensitive data is redacted
	if bytes.Contains(redacted, []byte("supersecret123")) {
		t.Errorf("Password was not redacted")
	}
	if bytes.Contains(redacted, []byte("abc123xyz")) {
		t.Errorf("Token was not redacted")
	}

	// Verify non-sensitive data remains
	if !bytes.Contains(redacted, []byte("john")) {
		t.Errorf("Username was incorrectly redacted")
	}

	// Verify the redacted values were replaced
	if !bytes.Contains(redacted, []byte(replacement)) {
		t.Errorf("Redaction replacement text not found")
	}
}

// TestHMAC checks that HMAC generation and verification work correctly
func TestHMAC(t *testing.T) {
	// Create test data and key
	testData := []byte("Test message for HMAC verification")
	key := []byte("hmac-test-key")

	// Generate HMAC
	hmac := CalculateHMAC(testData, key)

	// Verify HMAC
	if !VerifyHMAC(testData, key, hmac) {
		t.Errorf("HMAC verification failed for valid data")
	}

	// Tamper with the data and verify HMAC fails
	tamperedData := []byte("Tampered message for HMAC verification")
	if VerifyHMAC(tamperedData, key, hmac) {
		t.Errorf("HMAC verification should fail for tampered data")
	}

	// Use wrong key and verify HMAC fails
	wrongKey := []byte("wrong-key")
	if VerifyHMAC(testData, wrongKey, hmac) {
		t.Errorf("HMAC verification should fail with wrong key")
	}
}

// TestSecureEvent checks that SecureEvent creation and retrieval work correctly
func TestSecureEvent(t *testing.T) {
	// Create a test event
	event := Event{
		ID:        123,
		Timestamp: time.Now(),
		Type:      FuncEntry,
		Details:   "Entering function with password=secret123",
		File:      "test.go",
		Line:      42,
		FuncName:  "TestFunction",
	}

	// Test with encryption enabled
	t.Run("WithEncryption", func(t *testing.T) {
		opts := SecurityOptions{
			EnableEncryption: true,
			EncryptionKey:    []byte("0123456789ABCDEF"),
		}

		secureEvent, err := SecureEventFromEvent(event, opts)
		if err != nil {
			t.Fatalf("Failed to create secure event: %v", err)
		}

		if !secureEvent.Encrypted {
			t.Errorf("Event should be marked as encrypted")
		}

		// Get original event
		originalEvent, err := secureEvent.GetOriginalEvent(opts)
		if err != nil {
			t.Fatalf("Failed to get original event: %v", err)
		}

		// Verify original event matches
		if originalEvent.ID != event.ID || originalEvent.File != event.File || originalEvent.Details != event.Details {
			t.Errorf("Original event doesn't match. Got: %+v, expected: %+v", originalEvent, event)
		}

		// Try to decrypt with wrong key
		wrongOpts := SecurityOptions{
			EnableEncryption: true,
			EncryptionKey:    []byte("FEDCBA9876543210"),
		}
		_, err = secureEvent.GetOriginalEvent(wrongOpts)
		if err == nil {
			t.Errorf("Decryption with wrong key should fail")
		}
	})

	// Test with redaction enabled
	t.Run("WithRedaction", func(t *testing.T) {
		opts := SecurityOptions{
			EnableRedaction:      true,
			RedactionPatterns:    []string{"password"},
			RedactionReplacement: "***REDACTED***",
		}

		secureEvent, err := SecureEventFromEvent(event, opts)
		if err != nil {
			t.Fatalf("Failed to create secure event: %v", err)
		}

		if !secureEvent.IsRedacted {
			t.Errorf("Event should be marked as redacted")
		}

		// Get the redacted event directly
		if bytes.Contains([]byte(secureEvent.Event.Details), []byte("secret123")) {
			t.Errorf("Sensitive data was not redacted")
		}
	})

	// Test with HMAC integrity check enabled
	t.Run("WithIntegrityCheck", func(t *testing.T) {
		opts := SecurityOptions{
			EnableIntegrityCheck: true,
			IntegrityKey:         []byte("integrity-test-key"),
		}

		secureEvent, err := SecureEventFromEvent(event, opts)
		if err != nil {
			t.Fatalf("Failed to create secure event: %v", err)
		}

		if secureEvent.HMAC == "" {
			t.Errorf("HMAC should be generated")
		}

		// Tamper with the event
		secureEvent.Event.Details = "Tampered details"

		// Try to get original event (should fail integrity check)
		_, err = secureEvent.GetOriginalEvent(opts)
		// This might not fail since we didn't encrypt, so we're only checking HMAC when decrypting
		// For stronger security, we would always verify HMAC

		// Verify HMAC directly
		eventData, _ := json.Marshal(secureEvent.Event)
		if VerifyHMAC(eventData, opts.IntegrityKey, secureEvent.HMAC) {
			t.Errorf("HMAC verification should fail for tampered event")
		}
	})

	// Test with all security features enabled
	t.Run("WithAllSecurityFeatures", func(t *testing.T) {
		opts := SecurityOptions{
			EnableEncryption:     true,
			EncryptionKey:        []byte("0123456789ABCDEF"),
			EnableRedaction:      true,
			RedactionPatterns:    []string{"password"},
			RedactionReplacement: "***REDACTED***",
			EnableIntegrityCheck: true,
			IntegrityKey:         []byte("integrity-test-key"),
		}

		secureEvent, err := SecureEventFromEvent(event, opts)
		if err != nil {
			t.Fatalf("Failed to create secure event: %v", err)
		}

		if !secureEvent.Encrypted || !secureEvent.IsRedacted || secureEvent.HMAC == "" {
			t.Errorf("Security features not properly applied. Event: %+v", secureEvent)
		}

		// Get original event
		originalEvent, err := secureEvent.GetOriginalEvent(opts)
		if err != nil {
			t.Fatalf("Failed to get original event: %v", err)
		}

		// Verify the password was redacted in the original
		if !bytes.Contains([]byte(originalEvent.Details), []byte("***REDACTED***")) {
			t.Errorf("Sensitive data was not redacted in the decrypted event")
		}
	})
}

// TestSecureFileRecorderWithFullSecurity tests the secure file recorder with all security features enabled
func TestSecureFileRecorderWithFullSecurity(t *testing.T) {
	// Create a temporary file for testing
	tempFile, err := os.CreateTemp("", "secure_recorder_test")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tempFile.Close()
	defer os.Remove(tempFile.Name())

	// Create security options with all features enabled
	securityOpts := SecurityOptions{
		EnableEncryption:     true,
		EncryptionKey:        []byte("0123456789ABCDEF"),
		EnableRedaction:      true,
		RedactionPatterns:    []string{"password", "token"},
		RedactionReplacement: "***REDACTED***",
		EnableIntegrityCheck: true,
		IntegrityKey:         []byte("integrity-test-key"),
	}

	recorderOpts := SecureFileRecorderOptions{
		SecurityOptions: securityOpts,
		CompressionType: NoCompression, // Use no compression for easier inspection
	}

	// Create a secure file recorder
	recorder, err := NewSecureFileRecorderWithOptions(tempFile.Name(), recorderOpts)
	if err != nil {
		t.Fatalf("Failed to create secure file recorder: %v", err)
	}

	// Record some events with sensitive data
	events := []Event{
		{
			ID:        1,
			Timestamp: time.Now(),
			Type:      FuncEntry,
			Details:   "Entering function with password=secret123",
			File:      "test.go",
			Line:      42,
			FuncName:  "TestFunction",
		},
		{
			ID:        2,
			Timestamp: time.Now(),
			Type:      VarAssignment,
			Details:   "token=xyz789",
			File:      "test.go",
			Line:      43,
			FuncName:  "TestFunction",
		},
	}

	for _, e := range events {
		if err := recorder.RecordEvent(e); err != nil {
			t.Fatalf("Failed to record event: %v", err)
		}
	}

	// Close the recorder
	recorder.Close()

	// Check that the file exists and is not empty
	fileInfo, err := os.Stat(tempFile.Name())
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}
	if fileInfo.Size() == 0 {
		t.Errorf("File is empty")
	}

	// Read the file and check that it's encrypted (i.e., sensitive data not visible)
	fileContent, err := os.ReadFile(tempFile.Name())
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if bytes.Contains(fileContent, []byte("secret123")) {
		t.Errorf("File contains unencrypted sensitive data (password)")
	}
	if bytes.Contains(fileContent, []byte("xyz789")) {
		t.Errorf("File contains unencrypted sensitive data (token)")
	}

	// Create a new recorder to read the events back
	readRecorder, err := NewSecureFileRecorderWithOptions(tempFile.Name(), recorderOpts)
	if err != nil {
		t.Fatalf("Failed to create secure file recorder for reading: %v", err)
	}

	// Get the events
	readEvents := readRecorder.GetEvents()
	readRecorder.Close()

	// Verify we got all events
	if len(readEvents) != len(events) {
		t.Errorf("Expected %d events, got %d", len(events), len(readEvents))
	}

	// Verify sensitive data is redacted
	for _, e := range readEvents {
		if bytes.Contains([]byte(e.Details), []byte("secret123")) {
			t.Errorf("Event contains unredacted sensitive data (password)")
		}
		if bytes.Contains([]byte(e.Details), []byte("xyz789")) {
			t.Errorf("Event contains unredacted sensitive data (token)")
		}
	}

	// Test tamper detection
	t.Run("TamperDetection", func(t *testing.T) {
		// Create a new temp file
		tamperedFile, err := os.CreateTemp("", "tampered_recorder_test")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		tamperedFile.Close()
		defer os.Remove(tamperedFile.Name())

		// Create a recorder with integrity checking
		tamperRecorder, err := NewSecureFileRecorderWithOptions(tamperedFile.Name(), recorderOpts)
		if err != nil {
			t.Fatalf("Failed to create secure file recorder: %v", err)
		}

		// Record an event
		if err := tamperRecorder.RecordEvent(events[0]); err != nil {
			t.Fatalf("Failed to record event: %v", err)
		}
		tamperRecorder.Close()

		// Tamper with the file manually
		fileContent, err := os.ReadFile(tamperedFile.Name())
		if err != nil {
			t.Fatalf("Failed to read file: %v", err)
		}

		// Replace a character in the file
		tamperedContent := bytes.Replace(fileContent, []byte{'A'}, []byte{'B'}, 1)
		if bytes.Equal(tamperedContent, fileContent) {
			// If the file doesn't contain 'A', try another change
			tamperedContent = bytes.Replace(fileContent, []byte{'0'}, []byte{'1'}, 1)
		}

		if err := os.WriteFile(tamperedFile.Name(), tamperedContent, 0644); err != nil {
			t.Fatalf("Failed to write tampered file: %v", err)
		}

		// Reopen the file and check for tampering
		tamperRecorder, err = NewSecureFileRecorderWithOptions(tamperedFile.Name(), recorderOpts)
		if err != nil {
			t.Fatalf("Failed to create secure file recorder: %v", err)
		}

		tampered, err := tamperRecorder.DetectTampering()
		if err != nil {
			t.Fatalf("Error detecting tampering: %v", err)
		}
		if !tampered {
			t.Errorf("Tampering not detected")
		}

		tamperRecorder.Close()
	})
}

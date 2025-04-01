package main

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/willibrandon/ChronoGo/pkg/recorder"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	demoType := os.Args[1]

	switch strings.ToLower(demoType) {
	case "encryption":
		demoEncryption()
	case "redaction":
		demoRedaction()
	case "integrity":
		demoIntegrity()
	case "all":
		demoAll()
	default:
		fmt.Printf("Unknown demo type: %s\n", demoType)
		printUsage()
	}
}

func printUsage() {
	fmt.Println("ChronoGo Security Demo")
	fmt.Println("Usage: go run demo.go <demo-type>")
	fmt.Println("\nAvailable demo types:")
	fmt.Println("  encryption - Demonstrate encryption of event logs")
	fmt.Println("  redaction  - Demonstrate redaction of sensitive data")
	fmt.Println("  integrity  - Demonstrate integrity checking and tamper detection")
	fmt.Println("  all        - Run all security demos")
}

// demoEncryption demonstrates encryption of event logs
func demoEncryption() {
	fmt.Println("===== Encryption Demo =====")
	fmt.Println("This demo shows how to encrypt event logs to protect sensitive data.")

	// Create a temporary file for the encrypted log
	encryptedFile := "encrypted_log.json"
	defer os.Remove(encryptedFile)

	// Create a temporary file for comparison with standard logging
	plainFile := "plain_log.json"
	defer os.Remove(plainFile)

	// Create a standard file recorder (no encryption)
	plainRecorder, err := recorder.NewFileRecorder(plainFile)
	if err != nil {
		fmt.Printf("Error creating plain recorder: %v\n", err)
		return
	}
	defer plainRecorder.Close()

	// Create a secure file recorder with encryption
	encryptionKey := []byte("0123456789ABCDEF") // 16 bytes for AES-128
	securityOpts := recorder.SecurityOptions{
		EnableEncryption: true,
		EncryptionKey:    encryptionKey,
	}
	recorderOpts := recorder.SecureFileRecorderOptions{
		SecurityOptions: securityOpts,
		CompressionType: recorder.NoCompression, // No compression for demo clarity
	}

	encryptedRecorder, err := recorder.NewSecureFileRecorderWithOptions(encryptedFile, recorderOpts)
	if err != nil {
		fmt.Printf("Error creating encrypted recorder: %v\n", err)
		return
	}
	defer encryptedRecorder.Close()

	// Generate some events with sensitive data
	fmt.Println("Recording events with sensitive data...")
	for i := 0; i < 5; i++ {
		event := recorder.Event{
			ID:        int64(i + 1),
			Timestamp: time.Now(),
			Type:      recorder.FuncEntry,
			Details:   fmt.Sprintf("Function with password=%s and credit_card=%s", generatePassword(), generateCreditCard()),
			File:      "sensitive_file.go",
			Line:      42 + i,
			FuncName:  "SensitiveFunction",
		}

		// Record to both recorders
		plainRecorder.RecordEvent(event)
		encryptedRecorder.RecordEvent(event)
	}

	// Close the recorders to ensure all data is written
	plainRecorder.Close()
	encryptedRecorder.Close()

	// Compare file contents
	fmt.Println("\nComparing plain and encrypted files:")

	plainContent, _ := os.ReadFile(plainFile)
	encryptedContent, _ := os.ReadFile(encryptedFile)

	fmt.Printf("Plain file size: %d bytes\n", len(plainContent))
	fmt.Printf("Encrypted file size: %d bytes\n", len(encryptedContent))

	fmt.Println("\nPlain file contains sensitive data:")
	if containsSensitiveData(plainContent) {
		fmt.Println("✓ Sensitive data found in plain file (expected)")
	} else {
		fmt.Println("✗ No sensitive data found in plain file (unexpected)")
	}

	fmt.Println("\nEncrypted file should not contain sensitive data:")
	if containsSensitiveData(encryptedContent) {
		fmt.Println("✗ Sensitive data found in encrypted file (unexpected)")
	} else {
		fmt.Println("✓ No sensitive data found in encrypted file (good)")
	}

	// Read back the encrypted events
	fmt.Println("\nReading back encrypted events:")
	readRecorder, _ := recorder.NewSecureFileRecorderWithOptions(encryptedFile, recorderOpts)
	events := readRecorder.GetEvents()
	readRecorder.Close()

	fmt.Printf("Retrieved %d events from encrypted file\n", len(events))
	for i, e := range events {
		fmt.Printf("Event %d: %s\n", i+1, e.Details)
	}
}

// demoRedaction demonstrates redaction of sensitive data
func demoRedaction() {
	fmt.Println("===== Redaction Demo =====")
	fmt.Println("This demo shows how to redact sensitive data from event logs.")

	// Create a temporary file for the redacted log
	redactedFile := "redacted_log.json"
	defer os.Remove(redactedFile)

	// Create a temporary file for comparison with standard logging
	plainFile := "plain_log.json"
	defer os.Remove(plainFile)

	// Create a standard file recorder (no redaction)
	plainRecorder, err := recorder.NewFileRecorder(plainFile)
	if err != nil {
		fmt.Printf("Error creating plain recorder: %v\n", err)
		return
	}
	defer plainRecorder.Close()

	// Create a secure file recorder with redaction
	securityOpts := recorder.SecurityOptions{
		EnableRedaction:      true,
		RedactionPatterns:    []string{"password", "credit_card", "ssn", "token", "apikey"},
		RedactionReplacement: "***REDACTED***",
	}
	recorderOpts := recorder.SecureFileRecorderOptions{
		SecurityOptions: securityOpts,
		CompressionType: recorder.NoCompression, // No compression for demo clarity
	}

	redactedRecorder, err := recorder.NewSecureFileRecorderWithOptions(redactedFile, recorderOpts)
	if err != nil {
		fmt.Printf("Error creating redacted recorder: %v\n", err)
		return
	}
	defer redactedRecorder.Close()

	// Generate some events with sensitive data
	fmt.Println("Recording events with sensitive data...")
	sensitiveData := []string{
		fmt.Sprintf("password=%s", generatePassword()),
		fmt.Sprintf("credit_card=%s", generateCreditCard()),
		fmt.Sprintf("ssn=123-45-6789"),
		fmt.Sprintf("token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"),
		fmt.Sprintf("apikey=sk_test_123456789"),
	}

	for i, data := range sensitiveData {
		event := recorder.Event{
			ID:        int64(i + 1),
			Timestamp: time.Now(),
			Type:      recorder.FuncEntry,
			Details:   fmt.Sprintf("Function with %s", data),
			File:      "sensitive_file.go",
			Line:      42 + i,
			FuncName:  "SensitiveFunction",
		}

		// Record to both recorders
		plainRecorder.RecordEvent(event)
		redactedRecorder.RecordEvent(event)
	}

	// Close the recorders to ensure all data is written
	plainRecorder.Close()
	redactedRecorder.Close()

	// Compare file contents
	fmt.Println("\nComparing plain and redacted files:")

	plainContent, _ := os.ReadFile(plainFile)
	redactedContent, _ := os.ReadFile(redactedFile)

	fmt.Printf("Plain file size: %d bytes\n", len(plainContent))
	fmt.Printf("Redacted file size: %d bytes\n", len(redactedContent))

	fmt.Println("\nPlain file contains sensitive data:")
	if containsSensitiveData(plainContent) {
		fmt.Println("✓ Sensitive data found in plain file (expected)")
	} else {
		fmt.Println("✗ No sensitive data found in plain file (unexpected)")
	}

	// Check for redaction marker - using bytes.Contains
	fmt.Println("\nRedacted file should contain redaction markers:")
	if bytes.Contains(redactedContent, []byte("***REDACTED***")) {
		fmt.Println("✓ Redaction markers found in redacted file (good)")
	} else {
		fmt.Println("✗ No redaction markers found in redacted file (unexpected)")
	}

	// Read back the redacted events
	fmt.Println("\nReading back redacted events:")
	readRecorder, _ := recorder.NewSecureFileRecorderWithOptions(redactedFile, recorderOpts)
	events := readRecorder.GetEvents()
	readRecorder.Close()

	fmt.Printf("Retrieved %d events from redacted file\n", len(events))
	for i, e := range events {
		fmt.Printf("Event %d: %s\n", i+1, e.Details)
	}
}

// demoIntegrity demonstrates integrity checking and tamper detection
func demoIntegrity() {
	fmt.Println("===== Integrity Check Demo =====")
	fmt.Println("This demo shows how to verify the integrity of event logs and detect tampering.")

	// Create a temporary file for the integrity-protected log
	integrityFile := "integrity_log.json"
	defer os.Remove(integrityFile)

	// Create a secure file recorder with integrity checking
	integrityKey := []byte("integrity-verification-key")
	securityOpts := recorder.SecurityOptions{
		EnableIntegrityCheck: true,
		IntegrityKey:         integrityKey,
	}
	recorderOpts := recorder.SecureFileRecorderOptions{
		SecurityOptions: securityOpts,
		CompressionType: recorder.NoCompression, // No compression for demo clarity
	}

	integrityRecorder, err := recorder.NewSecureFileRecorderWithOptions(integrityFile, recorderOpts)
	if err != nil {
		fmt.Printf("Error creating integrity recorder: %v\n", err)
		return
	}

	// Generate some events
	fmt.Println("Recording events with integrity protection...")
	for i := 0; i < 5; i++ {
		event := recorder.Event{
			ID:        int64(i + 1),
			Timestamp: time.Now(),
			Type:      recorder.FuncEntry,
			Details:   fmt.Sprintf("Function call %d", i+1),
			File:      "integrity_test.go",
			Line:      42 + i,
			FuncName:  "TestFunction",
		}

		integrityRecorder.RecordEvent(event)
	}

	// Close the recorder to ensure all data is written
	integrityRecorder.Close()

	// Create a copy of the file for tampering
	tamperedFile := "tampered_log.json"
	copyFile(integrityFile, tamperedFile)
	defer os.Remove(tamperedFile)

	// Tamper with the copy
	fmt.Println("\nTampering with the log file...")
	tamperWithFile(tamperedFile)

	// Check the original file integrity
	fmt.Println("\nChecking integrity of the original file:")
	originalRecorder, _ := recorder.NewSecureFileRecorderWithOptions(integrityFile, recorderOpts)
	tampered, err := originalRecorder.DetectTampering()
	if err != nil {
		fmt.Printf("Error checking integrity: %v\n", err)
	} else if tampered {
		fmt.Println("✗ Tampering detected in original file (unexpected)")
	} else {
		fmt.Println("✓ No tampering detected in original file (good)")
	}
	originalRecorder.Close()

	// Check the tampered file integrity
	fmt.Println("\nChecking integrity of the tampered file:")
	tamperedRecorder, _ := recorder.NewSecureFileRecorderWithOptions(tamperedFile, recorderOpts)
	tampered, err = tamperedRecorder.DetectTampering()
	if err != nil {
		fmt.Printf("Error checking integrity: %v\n", err)
	} else if tampered {
		fmt.Println("✓ Tampering detected in tampered file (good)")
	} else {
		fmt.Println("✗ No tampering detected in tampered file (unexpected)")
	}
	tamperedRecorder.Close()
}

// demoAll runs all security demos
func demoAll() {
	fmt.Println("===== ChronoGo Security Features Demo =====")
	fmt.Println("This demo shows all security features working together.")

	// Create a temporary file for the secure log
	secureFile := "secure_log.json"
	defer os.Remove(secureFile)

	// Create a temporary file for comparison with standard logging
	plainFile := "plain_log.json"
	defer os.Remove(plainFile)

	// Create a standard file recorder (no security)
	plainRecorder, err := recorder.NewFileRecorder(plainFile)
	if err != nil {
		fmt.Printf("Error creating plain recorder: %v\n", err)
		return
	}
	defer plainRecorder.Close()

	// Create a secure file recorder with all security features
	securityOpts := recorder.SecurityOptions{
		EnableEncryption:     true,
		EncryptionKey:        []byte("0123456789ABCDEF"),
		EnableRedaction:      true,
		RedactionPatterns:    []string{"password", "credit_card", "ssn", "token", "apikey"},
		RedactionReplacement: "***REDACTED***",
		EnableIntegrityCheck: true,
		IntegrityKey:         []byte("integrity-verification-key"),
	}
	recorderOpts := recorder.SecureFileRecorderOptions{
		SecurityOptions: securityOpts,
		CompressionType: recorder.NoCompression, // No compression for demo clarity
	}

	secureRecorder, err := recorder.NewSecureFileRecorderWithOptions(secureFile, recorderOpts)
	if err != nil {
		fmt.Printf("Error creating secure recorder: %v\n", err)
		return
	}
	defer secureRecorder.Close()

	// Generate some events with sensitive data
	fmt.Println("Recording events with sensitive data...")
	for i := 0; i < 5; i++ {
		event := recorder.Event{
			ID:        int64(i + 1),
			Timestamp: time.Now(),
			Type:      recorder.FuncEntry,
			Details:   fmt.Sprintf("Function with password=%s and credit_card=%s", generatePassword(), generateCreditCard()),
			File:      "sensitive_file.go",
			Line:      42 + i,
			FuncName:  "SensitiveFunction",
		}

		// Record to both recorders
		plainRecorder.RecordEvent(event)
		secureRecorder.RecordEvent(event)
	}

	// Close the recorders to ensure all data is written
	plainRecorder.Close()
	secureRecorder.Close()

	// Compare file contents
	fmt.Println("\nComparing plain and secure files:")

	plainContent, _ := os.ReadFile(plainFile)
	secureContent, _ := os.ReadFile(secureFile)

	fmt.Printf("Plain file size: %d bytes\n", len(plainContent))
	fmt.Printf("Secure file size: %d bytes\n", len(secureContent))

	fmt.Println("\nPlain file contains sensitive data:")
	if containsSensitiveData(plainContent) {
		fmt.Println("✓ Sensitive data found in plain file (expected)")
	} else {
		fmt.Println("✗ No sensitive data found in plain file (unexpected)")
	}

	fmt.Println("\nSecure file should not contain sensitive data:")
	if containsSensitiveData(secureContent) {
		fmt.Println("✗ Sensitive data found in secure file (unexpected)")
	} else {
		fmt.Println("✓ No sensitive data found in secure file (good)")
	}

	// Create a copy of the file for tampering
	tamperedFile := "tampered_secure_log.json"
	copyFile(secureFile, tamperedFile)
	defer os.Remove(tamperedFile)

	// Tamper with the copy
	fmt.Println("\nTampering with the secure log file...")
	tamperWithFile(tamperedFile)

	// Check the original file integrity
	fmt.Println("\nChecking integrity of the original secure file:")
	originalRecorder, _ := recorder.NewSecureFileRecorderWithOptions(secureFile, recorderOpts)
	tampered, err := originalRecorder.DetectTampering()
	if err != nil {
		fmt.Printf("Error checking integrity: %v\n", err)
	} else if tampered {
		fmt.Println("✗ Tampering detected in original file (unexpected)")
	} else {
		fmt.Println("✓ No tampering detected in original file (good)")
	}
	originalRecorder.Close()

	// Check the tampered file integrity
	fmt.Println("\nChecking integrity of the tampered secure file:")
	tamperedRecorder, _ := recorder.NewSecureFileRecorderWithOptions(tamperedFile, recorderOpts)
	tampered, err = tamperedRecorder.DetectTampering()
	if err != nil {
		fmt.Printf("Error checking integrity: %v\n", err)
	} else if tampered {
		fmt.Println("✓ Tampering detected in tampered file (good)")
	} else {
		fmt.Println("✗ No tampering detected in tampered file (unexpected)")
	}
	tamperedRecorder.Close()

	// Read back the secure events
	fmt.Println("\nReading back secure events:")
	readRecorder, _ := recorder.NewSecureFileRecorderWithOptions(secureFile, recorderOpts)
	events := readRecorder.GetEvents()
	readRecorder.Close()

	fmt.Printf("Retrieved %d events from secure file\n", len(events))
	for i, e := range events {
		fmt.Printf("Event %d: %s\n", i+1, e.Details)
	}

	fmt.Println("\nAll security features (encryption, redaction, integrity) demonstrated successfully!")
}

// Helper functions

// generatePassword generates a random password for demo purposes
func generatePassword() string {
	return "P@ssw0rd123!"
}

// generateCreditCard generates a random credit card number for demo purposes
func generateCreditCard() string {
	return "4111-1111-1111-1111"
}

// containsSensitiveData checks if the given data contains sensitive information
func containsSensitiveData(data []byte) bool {
	sensitivePatterns := []string{
		"P@ssw0rd", "4111", "123-45-6789", "eyJhbGci", "sk_test",
	}

	strData := string(data)
	for _, pattern := range sensitivePatterns {
		if strings.Contains(strData, pattern) {
			return true
		}
	}

	return false
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	return os.WriteFile(dst, data, 0644)
}

// tamperWithFile modifies a file to simulate tampering
func tamperWithFile(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	// Replace a character at multiple positions
	modifiedData := data
	for i := 100; i < len(data) && i < 1000; i += 100 {
		if data[i] == 'a' {
			modifiedData[i] = 'b'
		} else if data[i] == '0' {
			modifiedData[i] = '1'
		} else {
			modifiedData[i] = 'X'
		}
	}

	return os.WriteFile(filePath, modifiedData, 0644)
}

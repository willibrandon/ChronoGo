package recorder

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// SecureFileRecorder records events to a file with security features
type SecureFileRecorder struct {
	file            *os.File
	writer          io.Writer
	bufWriter       *bufio.Writer
	path            string
	securityOpts    SecurityOptions
	compressionType CompressionType
	eventCount      int
}

// SecureFileRecorderOptions contains options for creating a secure file recorder
type SecureFileRecorderOptions struct {
	SecurityOptions SecurityOptions
	CompressionType CompressionType
}

// DefaultSecureFileRecorderOptions returns default options for secure file recorder
func DefaultSecureFileRecorderOptions() SecureFileRecorderOptions {
	return SecureFileRecorderOptions{
		SecurityOptions: DefaultSecurityOptions(),
		CompressionType: DefaultCompression,
	}
}

// NewSecureFileRecorder creates a new secure file recorder with default options
func NewSecureFileRecorder(path string) (*SecureFileRecorder, error) {
	return NewSecureFileRecorderWithOptions(path, DefaultSecureFileRecorderOptions())
}

// NewSecureFileRecorderWithOptions creates a new secure file recorder with the given options
func NewSecureFileRecorderWithOptions(path string, options SecureFileRecorderOptions) (*SecureFileRecorder, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	bufWriter := bufio.NewWriter(f)
	compressedWriter := NewCompressedWriter(bufWriter, options.CompressionType)

	return &SecureFileRecorder{
		file:            f,
		writer:          compressedWriter,
		bufWriter:       bufWriter,
		path:            path,
		securityOpts:    options.SecurityOptions,
		compressionType: options.CompressionType,
		eventCount:      0,
	}, nil
}

// RecordEvent applies security features and writes an event to the file
func (sfr *SecureFileRecorder) RecordEvent(e Event) error {
	// Apply security features to the event
	secureEvent, err := SecureEventFromEvent(e, sfr.securityOpts)
	if err != nil {
		return err
	}

	// Serialize the secure event
	data, err := json.Marshal(secureEvent)
	if err != nil {
		return err
	}

	// Write the JSON data
	if _, err := sfr.writer.Write(data); err != nil {
		return err
	}

	// Write a newline
	if _, err := sfr.writer.Write([]byte{'\n'}); err != nil {
		return err
	}

	// Flush bufWriter to ensure data is written to the file
	if err := sfr.bufWriter.Flush(); err != nil {
		return err
	}

	// Increment event count
	sfr.eventCount++

	// Check if we need to create a snapshot based on the global interval
	if SnapshotInterval > 0 && sfr.eventCount%SnapshotInterval == 0 {
		snapshot := CreateSnapshot(e.ID)
		// Store snapshot metadata with the event
		if err := sfr.recordSnapshotEvent(snapshot, sfr.eventCount); err != nil {
			return err
		}
	}

	return nil
}

// recordSnapshotEvent records a snapshot event to the file
func (sfr *SecureFileRecorder) recordSnapshotEvent(snapshot Snapshot, eventIdx int) error {
	// Create a special event to mark the snapshot
	snapshotEvent := Event{
		ID:        snapshot.ID,
		Timestamp: CurrentTime(),
		Type:      SnapshotEvent,
		Details:   "Snapshot created",
	}

	// Apply security features to the snapshot event
	secureEvent, err := SecureEventFromEvent(snapshotEvent, sfr.securityOpts)
	if err != nil {
		return err
	}

	data, err := json.Marshal(secureEvent)
	if err != nil {
		return err
	}

	// Write the snapshot event
	if _, err := sfr.writer.Write(data); err != nil {
		return err
	}
	if _, err := sfr.writer.Write([]byte{'\n'}); err != nil {
		return err
	}

	return sfr.bufWriter.Flush()
}

// GetEvents reads all events from the file, applying security features in reverse
func (sfr *SecureFileRecorder) GetEvents() []Event {
	// Ensure data is flushed to disk
	if err := CloseCompressedWriter(sfr.writer, sfr.compressionType); err != nil {
		// Log the error but continue - we still want to try reading events
		fmt.Printf("Warning: Error closing compressed writer: %v\n", err)
	}
	sfr.bufWriter.Flush()

	// Open the file for reading
	f, err := os.Open(sfr.path)
	if err != nil {
		return nil
	}
	defer f.Close()

	// Create a reader with decompression if needed
	reader, err := NewCompressedReader(f, sfr.compressionType)
	if err != nil {
		return nil
	}

	var events []Event
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		// Parse the secure event
		var secureEvent SecureEvent
		if err := json.Unmarshal(scanner.Bytes(), &secureEvent); err != nil {
			continue
		}

		// Extract the original event
		event, err := secureEvent.GetOriginalEvent(sfr.securityOpts)
		if err != nil {
			// Skip events that can't be decrypted or verified
			continue
		}

		events = append(events, event)
	}

	// Reopen the writer since we closed it
	sfr.writer = NewCompressedWriter(sfr.bufWriter, sfr.compressionType)

	return events
}

// Clear clears the file and resets the recorder
func (sfr *SecureFileRecorder) Clear() {
	// Ignore errors in Clear() as per interface
	if err := CloseCompressedWriter(sfr.writer, sfr.compressionType); err != nil {
		fmt.Printf("Warning: Error closing compressed writer: %v\n", err)
	}
	sfr.bufWriter.Flush()
	sfr.file.Close()
	if err := os.Truncate(sfr.path, 0); err != nil {
		fmt.Printf("Warning: Error truncating file: %v\n", err)
	}

	// Reopen the file
	f, err := os.OpenFile(sfr.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err == nil {
		sfr.file = f
		sfr.bufWriter = bufio.NewWriter(f)
		sfr.writer = NewCompressedWriter(sfr.bufWriter, sfr.compressionType)
		sfr.eventCount = 0
	}
}

// Close flushes and closes the file
func (sfr *SecureFileRecorder) Close() error {
	// Close the compressed writer if needed
	if err := CloseCompressedWriter(sfr.writer, sfr.compressionType); err != nil {
		return err
	}

	if err := sfr.bufWriter.Flush(); err != nil {
		return err
	}

	return sfr.file.Close()
}

// DetectTampering checks the file for any signs of tampering
func (sfr *SecureFileRecorder) DetectTampering() (bool, error) {
	// Open the file for reading
	f, err := os.Open(sfr.path)
	if err != nil {
		return false, err
	}
	defer f.Close()

	// Create a reader with decompression if needed
	reader, err := NewCompressedReader(f, sfr.compressionType)
	if err != nil {
		return false, err
	}

	// If integrity check is disabled, we can't detect tampering
	if !sfr.securityOpts.EnableIntegrityCheck {
		return false, nil
	}

	// Check each event
	scanner := bufio.NewScanner(reader)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		// Parse the secure event
		var secureEvent SecureEvent
		if err := json.Unmarshal(scanner.Bytes(), &secureEvent); err != nil {
			return true, err // Corrupted JSON is considered tampering
		}

		// Skip events without HMAC
		if secureEvent.HMAC == "" {
			continue
		}

		// If encrypted, verify HMAC of the encrypted data
		if secureEvent.Encrypted {
			encryptedData, err := json.Marshal(secureEvent.Event)
			if err != nil {
				return true, err
			}

			if !VerifyHMAC(encryptedData, sfr.securityOpts.IntegrityKey, secureEvent.HMAC) {
				return true, nil // Tampering detected
			}
		} else {
			// Verify HMAC of the event data
			eventData, err := json.Marshal(secureEvent.Event)
			if err != nil {
				return true, err
			}

			if !VerifyHMAC(eventData, sfr.securityOpts.IntegrityKey, secureEvent.HMAC) {
				return true, nil // Tampering detected
			}
		}
	}

	if scanner.Err() != nil {
		return true, scanner.Err() // Error during scanning is considered tampering
	}

	return false, nil // No tampering detected
}

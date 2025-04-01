package recorder

import (
	"os"
	"testing"
	"time"
)

func TestInMemoryRecorder(t *testing.T) {
	// Create a recorder
	recorder := NewInMemoryRecorder()

	// Check initial state
	events := recorder.GetEvents()
	if len(events) != 0 {
		t.Errorf("Expected 0 events initially, got %d", len(events))
	}

	// Record some events
	testEvents := []Event{
		{
			ID:        1,
			Timestamp: time.Now(),
			Type:      FuncEntry,
			Details:   "Entering function1",
			File:      "test.go",
			Line:      10,
			FuncName:  "function1",
		},
		{
			ID:        2,
			Timestamp: time.Now(),
			Type:      FuncExit,
			Details:   "Exiting function1",
			File:      "test.go",
			Line:      20,
			FuncName:  "function1",
		},
	}

	for _, event := range testEvents {
		err := recorder.RecordEvent(event)
		if err != nil {
			t.Errorf("Unexpected error recording event: %v", err)
		}
	}

	// Check recorded events
	events = recorder.GetEvents()
	if len(events) != len(testEvents) {
		t.Errorf("Expected %d events, got %d", len(testEvents), len(events))
	}

	// Check event details
	for i, event := range events {
		if event.ID != testEvents[i].ID {
			t.Errorf("Event %d: expected ID %d, got %d", i, testEvents[i].ID, event.ID)
		}
		if event.Type != testEvents[i].Type {
			t.Errorf("Event %d: expected Type %v, got %v", i, testEvents[i].Type, event.Type)
		}
		if event.Details != testEvents[i].Details {
			t.Errorf("Event %d: expected Details %s, got %s", i, testEvents[i].Details, event.Details)
		}
	}

	// Clear the recorder
	recorder.Clear()

	// Check events after clearing
	events = recorder.GetEvents()
	if len(events) != 0 {
		t.Errorf("Expected 0 events after clearing, got %d", len(events))
	}
}

func TestFileRecorder(t *testing.T) {
	// Create a temporary file for testing
	tempFile, err := os.CreateTemp("", "file_recorder_test")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tempFile.Close()
	tempFilePath := tempFile.Name()
	defer os.Remove(tempFilePath)

	// Create a file recorder
	recorder, err := NewFileRecorder(tempFilePath)
	if err != nil {
		t.Fatalf("Failed to create file recorder: %v", err)
	}

	// Record some events
	testEvents := []Event{
		{
			ID:        1,
			Timestamp: time.Now(),
			Type:      FuncEntry,
			Details:   "Entering function1",
			File:      "test.go",
			Line:      10,
			FuncName:  "function1",
		},
		{
			ID:        2,
			Timestamp: time.Now(),
			Type:      FuncExit,
			Details:   "Exiting function1",
			File:      "test.go",
			Line:      20,
			FuncName:  "function1",
		},
	}

	for _, event := range testEvents {
		err := recorder.RecordEvent(event)
		if err != nil {
			t.Errorf("Unexpected error recording event: %v", err)
		}
	}

	// Close the recorder
	err = recorder.Close()
	if err != nil {
		t.Errorf("Unexpected error closing recorder: %v", err)
	}

	// Create a new recorder to read back events
	readRecorder, err := NewFileRecorder(tempFilePath)
	if err != nil {
		t.Fatalf("Failed to create read recorder: %v", err)
	}
	defer readRecorder.Close()

	// Get the events
	events := readRecorder.GetEvents()

	// Check number of events (should be at least the number we wrote)
	// Note: There might be additional events due to automatic snapshots
	if len(events) < len(testEvents) {
		t.Errorf("Expected at least %d events, got %d", len(testEvents), len(events))
	}

	// Check event details for the first events (which should match our test events)
	for i := 0; i < len(testEvents); i++ {
		if events[i].ID != testEvents[i].ID {
			t.Errorf("Event %d: expected ID %d, got %d", i, testEvents[i].ID, events[i].ID)
		}
		if events[i].Type != testEvents[i].Type {
			t.Errorf("Event %d: expected Type %v, got %v", i, testEvents[i].Type, events[i].Type)
		}
		if events[i].Details != testEvents[i].Details {
			t.Errorf("Event %d: expected Details %s, got %s", i, testEvents[i].Details, events[i].Details)
		}
	}

	// Test Clear method
	readRecorder.Clear()

	// Check file after clearing
	fileInfo, err := os.Stat(tempFilePath)
	if err != nil {
		t.Errorf("Unexpected error checking file: %v", err)
	}
	if fileInfo.Size() != 0 {
		t.Errorf("Expected empty file after clearing, got size %d", fileInfo.Size())
	}
}

func TestFileRecorderWithOptions(t *testing.T) {
	// Create a temporary file for testing
	tempFile, err := os.CreateTemp("", "file_recorder_opts_test")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tempFile.Close()
	tempFilePath := tempFile.Name()
	defer os.Remove(tempFilePath)

	// Test with different compression options
	compressionTypes := []CompressionType{
		NoCompression,
		DefaultCompression,
	}

	for _, compressionType := range compressionTypes {
		t.Run(compressionTypeToString(compressionType), func(t *testing.T) {
			// Create options with this compression type
			options := FileRecorderOptions{
				CompressionType: compressionType,
			}

			// Create recorder
			recorder, err := NewFileRecorderWithOptions(tempFilePath, options)
			if err != nil {
				t.Fatalf("Failed to create recorder with %s compression: %v",
					compressionTypeToString(compressionType), err)
			}

			// Record an event
			event := Event{
				ID:        1,
				Timestamp: time.Now(),
				Type:      FuncEntry,
				Details:   "Test with " + compressionTypeToString(compressionType),
			}

			err = recorder.RecordEvent(event)
			if err != nil {
				t.Errorf("Failed to record event: %v", err)
			}

			// Close the recorder
			err = recorder.Close()
			if err != nil {
				t.Errorf("Failed to close recorder: %v", err)
			}

			// Create a new recorder with the same options to read back
			readRecorder, err := NewFileRecorderWithOptions(tempFilePath, options)
			if err != nil {
				t.Fatalf("Failed to create read recorder: %v", err)
			}
			defer readRecorder.Close()

			// Get events
			events := readRecorder.GetEvents()

			// Check we got at least one event back
			if len(events) < 1 {
				t.Errorf("Expected at least 1 event, got %d", len(events))
			} else {
				// Check event details
				if events[0].Details != event.Details {
					t.Errorf("Expected details %q, got %q", event.Details, events[0].Details)
				}
			}

			// Clear file for next test
			readRecorder.Clear()
		})
	}
}

func TestDefaulFileRecorderOptions(t *testing.T) {
	options := DefaultFileRecorderOptions()

	if options.CompressionType != DefaultCompression {
		t.Errorf("Expected default compression type %v, got %v",
			DefaultCompression, options.CompressionType)
	}
}

// Helper to convert compression type to string for test names
func compressionTypeToString(ct CompressionType) string {
	switch ct {
	case NoCompression:
		return "NoCompression"
	case DefaultCompression:
		return "DefaultCompression"
	default:
		return "Unknown"
	}
}

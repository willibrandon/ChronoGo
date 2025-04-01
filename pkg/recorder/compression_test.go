package recorder

import (
	"bytes"
	"testing"
)

func TestCompression(t *testing.T) {
	// Test data
	testData := []byte("This is test data for compression. It should be smaller when compressed.")

	// Test compression and decompression
	compressed, err := CompressData(testData, ZstdCompression)
	if err != nil {
		t.Fatalf("Failed to compress data: %v", err)
	}

	// Verify compression actually reduced size
	if len(compressed) >= len(testData) {
		t.Logf("Warning: Compressed data (%d bytes) is not smaller than original (%d bytes)",
			len(compressed), len(testData))
		// Note: For very small data, compression might not reduce size
	}

	// Test decompression
	decompressed, err := DecompressData(compressed, ZstdCompression)
	if err != nil {
		t.Fatalf("Failed to decompress data: %v", err)
	}

	// Verify decompressed data matches original
	if !bytes.Equal(decompressed, testData) {
		t.Fatalf("Decompressed data does not match original")
	}
}

func TestCompressedWriter(t *testing.T) {
	// Setup buffer to write to
	var buf bytes.Buffer

	// Create compressed writer
	writer := NewCompressedWriter(&buf, ZstdCompression)

	// Test data
	testData := []byte("This is test data for the compressed writer.")

	// Write data
	n, err := writer.Write(testData)
	if err != nil {
		t.Fatalf("Failed to write to compressed writer: %v", err)
	}
	if n != len(testData) {
		t.Fatalf("Expected to write %d bytes, wrote %d", len(testData), n)
	}

	// Close writer
	if err := CloseCompressedWriter(writer, ZstdCompression); err != nil {
		t.Fatalf("Failed to close compressed writer: %v", err)
	}

	// Verify compressed data was written
	if buf.Len() == 0 {
		t.Fatal("No data was written to buffer")
	}

	// Verify we can decompress it
	reader, err := NewCompressedReader(bytes.NewReader(buf.Bytes()), ZstdCompression)
	if err != nil {
		t.Fatalf("Failed to create compressed reader: %v", err)
	}

	// Read decompressed data
	decompressed := make([]byte, len(testData)*2) // Ensure buffer is large enough
	n, err = reader.Read(decompressed)
	if err != nil && err.Error() != "EOF" {
		t.Fatalf("Failed to read from compressed reader: %v", err)
	}
	decompressed = decompressed[:n]

	// Verify decompressed data matches original
	if !bytes.Equal(decompressed, testData) {
		t.Fatalf("Decompressed data does not match original")
	}
}

func TestFileRecorderWithCompression(t *testing.T) {
	// Create a temporary file
	tempFile := t.TempDir() + "/test_compressed_events.json.zst"

	// Create a file recorder with compression
	options := FileRecorderOptions{
		CompressionType: ZstdCompression,
	}
	recorder, err := NewFileRecorderWithOptions(tempFile, options)
	if err != nil {
		t.Fatalf("Failed to create file recorder: %v", err)
	}

	// Record some test events
	for i := 0; i < 10; i++ {
		event := Event{
			ID:        int64(i),
			Timestamp: CurrentTime(),
			Type:      ChannelOperation,
			Details:   "Test event",
		}
		if err := recorder.RecordEvent(event); err != nil {
			t.Fatalf("Failed to record event: %v", err)
		}
	}

	// Close recorder
	if err := recorder.Close(); err != nil {
		t.Fatalf("Failed to close recorder: %v", err)
	}

	// Reopen recorder
	reopened, err := NewFileRecorderWithOptions(tempFile, options)
	if err != nil {
		t.Fatalf("Failed to reopen file recorder: %v", err)
	}
	defer reopened.Close()

	// Get events and verify
	events := reopened.GetEvents()
	if len(events) != 10 {
		t.Fatalf("Expected 10 events, got %d", len(events))
	}

	// Verify event data
	for i, event := range events {
		if event.ID != int64(i) {
			t.Errorf("Event %d has wrong ID: expected %d, got %d", i, i, event.ID)
		}
		if event.Type != ChannelOperation {
			t.Errorf("Event %d has wrong type: expected %v, got %v", i, ChannelOperation, event.Type)
		}
	}
}

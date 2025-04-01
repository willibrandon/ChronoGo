package recorder

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
)

// FileRecorder records events to a file with optional compression
type FileRecorder struct {
	file            *os.File
	writer          io.Writer
	bufWriter       *bufio.Writer
	path            string
	compressionType CompressionType
	eventCount      int
}

// FileRecorderOptions contains options for creating a file recorder
type FileRecorderOptions struct {
	CompressionType CompressionType
}

// DefaultFileRecorderOptions returns default options for file recorder
func DefaultFileRecorderOptions() FileRecorderOptions {
	return FileRecorderOptions{
		CompressionType: DefaultCompression,
	}
}

// NewFileRecorder creates a new file recorder with default options
func NewFileRecorder(path string) (*FileRecorder, error) {
	return NewFileRecorderWithOptions(path, DefaultFileRecorderOptions())
}

// NewFileRecorderWithOptions creates a new file recorder with the given options
func NewFileRecorderWithOptions(path string, options FileRecorderOptions) (*FileRecorder, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	bufWriter := bufio.NewWriter(f)
	compressedWriter := NewCompressedWriter(bufWriter, options.CompressionType)

	return &FileRecorder{
		file:            f,
		writer:          compressedWriter,
		bufWriter:       bufWriter,
		path:            path,
		compressionType: options.CompressionType,
		eventCount:      0,
	}, nil
}

// RecordEvent writes an event to the file with compression
func (fr *FileRecorder) RecordEvent(e Event) error {
	data, err := json.Marshal(e)
	if err != nil {
		return err
	}

	// Write the JSON data
	if _, err := fr.writer.Write(data); err != nil {
		return err
	}

	// Write a newline
	if _, err := fr.writer.Write([]byte{'\n'}); err != nil {
		return err
	}

	// Flush bufWriter to ensure data is written to the file
	if err := fr.bufWriter.Flush(); err != nil {
		return err
	}

	// Increment event count
	fr.eventCount++

	// Check if we need to create a snapshot based on the global interval
	if SnapshotInterval > 0 && fr.eventCount%SnapshotInterval == 0 {
		snapshot := CreateSnapshot(e.ID)
		// Store snapshot metadata with the event
		// In a real implementation, we would store the actual memory state
		fr.recordSnapshotEvent(snapshot, fr.eventCount)
	}

	return nil
}

// recordSnapshotEvent records a snapshot event to the file
func (fr *FileRecorder) recordSnapshotEvent(snapshot Snapshot, eventIdx int) error {
	// Create a special event to mark the snapshot
	snapshotEvent := Event{
		ID:        snapshot.ID,
		Timestamp: CurrentTime(),
		Type:      SnapshotEvent,
		Details:   "Snapshot created",
	}

	data, err := json.Marshal(snapshotEvent)
	if err != nil {
		return err
	}

	// Write the snapshot event
	if _, err := fr.writer.Write(data); err != nil {
		return err
	}
	if _, err := fr.writer.Write([]byte{'\n'}); err != nil {
		return err
	}

	return fr.bufWriter.Flush()
}

// GetEvents reads all events from the file, decompressing if necessary
func (fr *FileRecorder) GetEvents() []Event {
	// Ensure data is flushed to disk
	CloseCompressedWriter(fr.writer, fr.compressionType)
	fr.bufWriter.Flush()

	// Open the file for reading
	f, err := os.Open(fr.path)
	if err != nil {
		return nil
	}
	defer f.Close()

	// Create a reader with decompression if needed
	reader, err := NewCompressedReader(f, fr.compressionType)
	if err != nil {
		return nil
	}

	var events []Event
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		var event Event
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			continue
		}
		events = append(events, event)
	}

	// Reopen the writer since we closed it
	fr.writer = NewCompressedWriter(fr.bufWriter, fr.compressionType)

	return events
}

// Clear clears the file and resets the recorder
func (fr *FileRecorder) Clear() {
	// Ignore errors in Clear() as per interface
	CloseCompressedWriter(fr.writer, fr.compressionType)
	fr.bufWriter.Flush()
	fr.file.Close()
	os.Truncate(fr.path, 0)

	// Reopen the file
	f, err := os.OpenFile(fr.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err == nil {
		fr.file = f
		fr.bufWriter = bufio.NewWriter(f)
		fr.writer = NewCompressedWriter(fr.bufWriter, fr.compressionType)
		fr.eventCount = 0
	}
}

// Close flushes and closes the file
func (fr *FileRecorder) Close() error {
	// Close the compressed writer if needed
	if err := CloseCompressedWriter(fr.writer, fr.compressionType); err != nil {
		return err
	}

	if err := fr.bufWriter.Flush(); err != nil {
		return err
	}

	return fr.file.Close()
}

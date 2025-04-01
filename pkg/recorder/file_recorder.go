package recorder

import (
	"bufio"
	"encoding/json"
	"os"
)

type FileRecorder struct {
	file   *os.File
	writer *bufio.Writer
	path   string
}

func NewFileRecorder(path string) (*FileRecorder, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	return &FileRecorder{
		file:   f,
		writer: bufio.NewWriter(f),
		path:   path,
	}, nil
}

func (fr *FileRecorder) RecordEvent(e Event) error {
	data, err := json.Marshal(e)
	if err != nil {
		return err
	}

	if _, err := fr.writer.Write(data); err != nil {
		return err
	}
	if err := fr.writer.WriteByte('\n'); err != nil {
		return err
	}
	return fr.writer.Flush()
}

func (fr *FileRecorder) GetEvents() []Event {
	// Open the file for reading
	f, err := os.Open(fr.path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var events []Event
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var event Event
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			continue
		}
		events = append(events, event)
	}
	return events
}

func (fr *FileRecorder) Clear() {
	// Ignore errors in Clear() as per interface
	fr.writer.Flush()
	fr.file.Close()
	os.Truncate(fr.path, 0)
	f, err := os.OpenFile(fr.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err == nil {
		fr.file = f
		fr.writer = bufio.NewWriter(f)
	}
}

func (fr *FileRecorder) Close() error {
	if err := fr.writer.Flush(); err != nil {
		return err
	}
	return fr.file.Close()
}

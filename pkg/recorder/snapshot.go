package recorder

type Snapshot struct {
	ID      int64
	MemDump []byte // Could be a serialized representation of memory
	// Additional metadata (heap size, stack traces, etc.)
}

func CreateSnapshot(id int64) Snapshot {
	// TODO: integrate Delve or runtime hooks for real memory capture
	return Snapshot{
		ID:      id,
		MemDump: []byte("mock state"),
	}
}

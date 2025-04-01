package recorder

type Recorder interface {
	RecordEvent(e Event) error
	GetEvents() []Event
	Clear()
}

type InMemoryRecorder struct {
	events []Event
}

func NewInMemoryRecorder() *InMemoryRecorder {
	return &InMemoryRecorder{events: []Event{}}
}

func (r *InMemoryRecorder) RecordEvent(e Event) error {
	r.events = append(r.events, e)
	return nil
}

func (r *InMemoryRecorder) GetEvents() []Event {
	return r.events
}

func (r *InMemoryRecorder) Clear() {
	r.events = []Event{}
}

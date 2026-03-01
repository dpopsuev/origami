package backend

import (
	"sync"
	"time"
)

// EventStore is an in-memory event store for circuit runs.
// Community edition uses this; Enterprise edition can swap for Postgres.
type EventStore struct {
	mu     sync.RWMutex
	events []StudioEvent
	runs   map[string]*RunInfo
	nextID int
	subs   map[int]chan StudioEvent
	subID  int
}

// NewEventStore creates an empty event store.
func NewEventStore() *EventStore {
	return &EventStore{
		runs: make(map[string]*RunInfo),
		subs: make(map[int]chan StudioEvent),
	}
}

// Append stores an event and notifies subscribers.
func (s *EventStore) Append(evt StudioEvent) {
	s.mu.Lock()
	s.nextID++
	evt.ID = s.nextID
	s.events = append(s.events, evt)

	for _, ch := range s.subs {
		select {
		case ch <- evt:
		default:
		}
	}
	s.mu.Unlock()
}

// Events returns all events for a run.
func (s *EventStore) Events(runID string) []StudioEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []StudioEvent
	for _, evt := range s.events {
		if evt.RunID == runID {
			result = append(result, evt)
		}
	}
	return result
}

// EventsSince returns events for a run after the given ID.
func (s *EventStore) EventsSince(runID string, afterID int) []StudioEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []StudioEvent
	for _, evt := range s.events {
		if evt.RunID == runID && evt.ID > afterID {
			result = append(result, evt)
		}
	}
	return result
}

// Subscribe returns a channel that receives new events.
func (s *EventStore) Subscribe() (int, <-chan StudioEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ch := make(chan StudioEvent, 64)
	s.subID++
	s.subs[s.subID] = ch
	return s.subID, ch
}

// Unsubscribe removes a subscriber.
func (s *EventStore) Unsubscribe(id int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ch, ok := s.subs[id]; ok {
		close(ch)
		delete(s.subs, id)
	}
}

// RegisterRun records a new run.
func (s *EventStore) RegisterRun(info RunInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.runs[info.ID] = &info
}

// CompleteRun marks a run as completed.
func (s *EventStore) CompleteRun(runID, status string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if r, ok := s.runs[runID]; ok {
		r.Status = status
		r.EndedAt = time.Now().UTC()
	}
}

// Runs returns all registered runs.
func (s *EventStore) Runs() []RunInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]RunInfo, 0, len(s.runs))
	for _, r := range s.runs {
		result = append(result, *r)
	}
	return result
}

// Run returns a specific run by ID.
func (s *EventStore) Run(id string) *RunInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.runs[id]
	if !ok {
		return nil
	}
	copy := *r
	return &copy
}

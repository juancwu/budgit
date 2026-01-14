package event

import (
	"fmt"
	"log/slog"
	"sync"
)

type Event struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

type Broker struct {
	mu          sync.RWMutex
	subscribers map[string][]chan Event // map[spaceID][]chan Event
}

func NewBroker() *Broker {
	return &Broker{
		subscribers: make(map[string][]chan Event),
	}
}

func (b *Broker) Subscribe(spaceID string) chan Event {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan Event, 10) // buffer slightly to prevent blocking
	b.subscribers[spaceID] = append(b.subscribers[spaceID], ch)

	slog.Info("new subscriber", "space_id", spaceID)
	return ch
}

func (b *Broker) Unsubscribe(spaceID string, ch chan Event) {
	b.mu.Lock()
	defer b.mu.Unlock()

	subs := b.subscribers[spaceID]
	for i, sub := range subs {
		if sub == ch {
			// Remove from slice
			b.subscribers[spaceID] = append(subs[:i], subs[i+1:]...)
			close(ch)
			slog.Info("subscriber removed", "space_id", spaceID)
			break
		}
	}

	if len(b.subscribers[spaceID]) == 0 {
		delete(b.subscribers, spaceID)
	}
}

func (b *Broker) Publish(spaceID string, eventType string, data interface{}) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	subs, ok := b.subscribers[spaceID]
	if !ok {
		return
	}

	event := Event{
		Type: eventType,
		Data: data,
	}

	slog.Info("publishing event", "space_id", spaceID, "type", eventType)

	for _, ch := range subs {
		select {
		case ch <- event:
		default:
			slog.Warn("subscriber channel full, dropping event", "space_id", spaceID)
		}
	}
}

// String format for SSE data
func (e Event) String() string {
	return fmt.Sprintf("event: %s\ndata: %v\n\n", e.Type, e.Data)
}

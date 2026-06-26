package sse

import (
	"reflect"
	"sync"
	"time"
)

type EventBroker struct {
	mu      sync.RWMutex
	clients map[string]map[chan SSEEvent]struct{}
}

type SSEEvent struct {
	InstanceID string `json:"instance_id"`
	Type       string `json:"type"`   // "status", "error", "credentials"
	Status     string `json:"status"` // "provisioning", "running", "failed"
	Step       string `json:"step"`   // "launching_instance", "waiting_for_boot", "installing_apps", "ready"
	Message    string `json:"message"`
	Data       any    `json:"data"`
	Timestamp  int64  `json:"timestamp"` // unix milliseconds
}

func NewEventBroker() *EventBroker {
	return &EventBroker{
		clients: make(map[string]map[chan SSEEvent]struct{}),
	}
}

func (b *EventBroker) Subscribe(instanceID string) <-chan SSEEvent {
	ch := make(chan SSEEvent, 64)

	b.mu.Lock()
	defer b.mu.Unlock()

	if b.clients[instanceID] == nil {
		b.clients[instanceID] = make(map[chan SSEEvent]struct{})
	}
	b.clients[instanceID][ch] = struct{}{}

	return ch
}

func (b *EventBroker) Unsubscribe(instanceID string, ch <-chan SSEEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if channels, ok := b.clients[instanceID]; ok {
		for c := range channels {
			if reflect.ValueOf(c).Pointer() == reflect.ValueOf(ch).Pointer() {
				delete(channels, c)
				close(c)
				break
			}
		}
		if len(channels) == 0 {
			delete(b.clients, instanceID)
		}
	}
}

func (b *EventBroker) Publish(instanceID string, event SSEEvent) {
	if event.Timestamp == 0 {
		event.Timestamp = time.Now().UnixMilli()
	}
	if event.InstanceID == "" {
		event.InstanceID = instanceID
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	for ch := range b.clients[instanceID] {
		select {
		case ch <- event:
		default:
			// drop for slow consumer
		}
	}
}

func (b *EventBroker) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()

	for instanceID, channels := range b.clients {
		for ch := range channels {
			close(ch)
		}
		delete(b.clients, instanceID)
	}
}

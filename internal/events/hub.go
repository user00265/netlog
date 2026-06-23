// Package events provides a small in-memory publish/subscribe hub used to push
// Server-Sent Events to connected clients (e.g. "the backend has changes you
// should sync"). It is process-local, which is sufficient for the single-binary
// deployment NetLog targets.
package events

import (
	"fmt"
	"sync"
)

// Hub fans out messages to all subscribed SSE connections.
type Hub struct {
	mu   sync.Mutex
	subs map[chan string]struct{}
}

// NewHub returns an empty Hub.
func NewHub() *Hub {
	return &Hub{subs: make(map[chan string]struct{})}
}

// Subscribe registers a new subscriber and returns its buffered channel. The
// caller must Unsubscribe when done.
func (h *Hub) Subscribe() chan string {
	ch := make(chan string, 8)
	h.mu.Lock()
	h.subs[ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

// Unsubscribe removes and closes a subscriber channel.
func (h *Hub) Unsubscribe(ch chan string) {
	h.mu.Lock()
	if _, ok := h.subs[ch]; ok {
		delete(h.subs, ch)
		close(ch)
	}
	h.mu.Unlock()
}

// Broadcast sends a named SSE event with the given data to every subscriber. A
// slow subscriber whose buffer is full is skipped rather than blocking the
// broadcast (it will catch up on its next reconnect/sync).
func (h *Hub) Broadcast(event, data string) {
	msg := fmt.Sprintf("event: %s\ndata: %s\n\n", event, data)
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subs {
		select {
		case ch <- msg:
		default:
		}
	}
}

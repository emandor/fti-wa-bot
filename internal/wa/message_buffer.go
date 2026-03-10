package wa

import "sync"

// MessageBuffer is a thread-safe, bounded FIFO buffer for recent messages.
type MessageBuffer struct {
	mu       sync.RWMutex
	messages []MessageSummary
	maxSize  int
}

func NewMessageBuffer(maxSize int) *MessageBuffer {
	if maxSize <= 0 {
		maxSize = 500
	}
	return &MessageBuffer{maxSize: maxSize}
}

// Add prepends a message to the buffer, evicting the oldest if at capacity.
func (b *MessageBuffer) Add(msg MessageSummary) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.messages = append([]MessageSummary{msg}, b.messages...)
	if len(b.messages) > b.maxSize {
		b.messages = b.messages[:b.maxSize]
	}
}

// List returns up to limit messages (most recent first).
func (b *MessageBuffer) List(limit int) []MessageSummary {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if len(b.messages) == 0 {
		return nil
	}

	n := limit
	if limit <= 0 || limit > b.maxSize {
		n = 100
	}
	if n > len(b.messages) {
		n = len(b.messages)
	}

	out := make([]MessageSummary, n)
	copy(out, b.messages[:n])
	return out
}

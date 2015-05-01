package cypress

import "sync"

// A simple type that stores any Received message into a buffer.
// Mostly for testing.
type BufferReceiver struct {
	lock     sync.Mutex
	Messages []*Message
}

// Store the message into the internal buffer
func (b *BufferReceiver) Receive(m *Message) error {
	b.lock.Lock()

	b.Messages = append(b.Messages, m)

	b.lock.Unlock()

	return nil
}

func (b *BufferReceiver) Close() error {
	return nil
}

// Used for testing to syncronize goroutines using the value
func (b *BufferReceiver) SyncTo() {
	b.lock.Lock()
	b.lock.Unlock()
}

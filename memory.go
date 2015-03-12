package cypress

import "sync"

type BufferReceiver struct {
	lock     sync.Mutex
	Messages []*Message
}

func (b *BufferReceiver) Receive(m *Message) error {
	b.lock.Lock()

	b.Messages = append(b.Messages, m)

	b.lock.Unlock()

	return nil
}

func (b *BufferReceiver) SyncTo() {
	b.lock.Lock()
	b.lock.Unlock()
}

package cypress

import "sync"

type SerialReceiver struct {
	Receiver Receiver
	lock     sync.Mutex
}

func NewSerialReceiver(r Receiver) *SerialReceiver {
	return &SerialReceiver{Receiver: r}
}

func (s *SerialReceiver) Receive(m *Message) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	return s.Receiver.Receive(m)
}

package cypress

import (
	"net"
	"sync"
	"time"
)

type TCPSend struct {
	host string
	c    net.Conn
	s    *Send

	lock        sync.Mutex
	outstanding int

	newMessages chan *Message
	closed      chan bool

	shutdown bool

	nacked []*Message
}

const DefaultTCPBuffer = 128

func NewTCPSend(host string, window, buffer int) (*TCPSend, error) {
	c, err := net.Dial("tcp", host)
	if err != nil {
		return nil, err
	}

	s := NewSend(c, window)
	err = s.SendHandshake()
	if err != nil {
		return nil, err
	}

	tcp := &TCPSend{
		host:        host,
		c:           c,
		s:           s,
		newMessages: make(chan *Message, buffer),
		closed:      make(chan bool),
	}

	s.OnClosed = tcp.onClosed

	go tcp.drain()

	return tcp, nil
}

func (t *TCPSend) Close() error {
	t.shutdown = true
	return t.c.Close()
}

func (t *TCPSend) Ack(m *Message) {
	t.lock.Lock()
	defer t.lock.Unlock()

	t.outstanding--
}

func (t *TCPSend) Nack(m *Message) {
	t.lock.Lock()
	defer t.lock.Unlock()

	t.outstanding--
	t.nacked = append(t.nacked, m)
}

func (t *TCPSend) onClosed() {
	t.closed <- true
}

func (t *TCPSend) reconnect() {
	t.lock.Lock()

tryagain:

	c, err := net.Dial("tcp", t.host)
	if err != nil {
		if t.shutdown {
			t.lock.Unlock()
			return
		}

		time.Sleep(1 * time.Second)
		goto tryagain
	}

	s := NewSend(c, 0)
	err = s.SendHandshake()
	if err != nil {
		c.Close()
		goto tryagain
	}

	t.c = c
	t.s = s

	for idx, msg := range t.nacked {
		t.outstanding++
		err = t.s.Send(msg, t)
		if err != nil {
			t.nacked = t.nacked[idx:]
			goto tryagain
		}
	}

	t.nacked = nil
}

func (t *TCPSend) Receive(m *Message) error {
	t.newMessages <- m
	return nil
}

func (t *TCPSend) drain() {
	for {
		select {
		case <-t.closed:
			t.reconnect()
		case m := <-t.newMessages:
			t.outstanding++
			t.s.Send(m, t)
		}
	}
}

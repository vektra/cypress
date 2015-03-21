package cypress

import (
	"net"
	"sort"
	"sync"
	"time"
)

type TCPSend struct {
	host   string
	window int
	c      net.Conn
	s      *Send

	lock        sync.Mutex
	outstanding int

	newMessages chan *Message
	closed      chan bool

	shutdown bool

	nacked Messages
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
		window:      window,
		c:           c,
		s:           s,
		newMessages: make(chan *Message, buffer),
		closed:      make(chan bool, 1),
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

	var (
		c   net.Conn
		s   *Send
		err error
	)

	t.c.Close()

	for {
		c, err = net.Dial("tcp", t.host)
		if err != nil {
			if t.shutdown {
				t.lock.Unlock()
				return
			}

			time.Sleep(1 * time.Second)
			continue
		}

		s = NewSend(c, t.window)
		s.OnClosed = t.onClosed
		err = s.SendHandshake()
		if err != nil {
			c.Close()
			continue
		}

		break
	}

	t.c = c
	t.s = s

	nacked := t.nacked
	t.nacked = nil

	t.lock.Unlock()

	for idx, msg := range nacked {
		t.outstanding++
		err = t.s.Send(msg, t)
		if err != nil {
			t.lock.Lock()
			t.nacked = append(nacked[idx+1:], t.nacked...)
			sort.Sort(t.nacked)

			// don't retry here because the OnClose handler will
			// prime the closed channel, so we return from here, pick
			// up the value from the channel, then this is called again.
			t.lock.Unlock()
			return
		}
	}
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
			t.lock.Lock()
			t.outstanding++
			t.lock.Unlock()
			t.s.Send(m, t)
		}
	}
}

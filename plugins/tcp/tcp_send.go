package tcp

import (
	"errors"
	"math/rand"
	"net"
	"sort"
	"sync"
	"time"

	"gopkg.in/tomb.v2"

	"github.com/vektra/cypress"
)

type TCPSend struct {
	hosts  []string
	window int
	c      net.Conn
	s      *cypress.Send

	lock        sync.Mutex
	outstanding int

	newMessages chan *cypress.Message
	closed      chan bool

	flush    chan struct{}
	shutdown bool

	nacked cypress.Messages

	t tomb.Tomb
}

const DefaultTCPBuffer = 128

func NewTCPSend(hosts []string, window, buffer int) (*TCPSend, error) {
	tcp := &TCPSend{
		hosts:       hosts,
		window:      window,
		newMessages: make(chan *cypress.Message, buffer),
		closed:      make(chan bool, 1),
		flush:       make(chan struct{}),
	}

	for {
		err := tcp.Connect()
		if err != nil {
			if err == ErrNoAvailableHosts {
				time.Sleep(1 * time.Second)
				continue
			}

			return nil, err
		}

		break
	}

	tcp.t.Go(tcp.drain)

	return tcp, nil
}

var ErrNoAvailableHosts = errors.New("no available hosts")

func shuffle(a []string) {
	for i := range a {
		j := rand.Intn(i + 1)
		a[i], a[j] = a[j], a[i]
	}
}

func (t *TCPSend) Connect() error {
	shuffle(t.hosts)

	for _, host := range t.hosts {
		c, err := net.Dial("tcp", host)
		if err != nil {
			continue
		}

		s := cypress.NewSend(c, t.window)
		err = s.SendHandshake()
		if err != nil {
			c.Close()
			continue
		}

		t.c = c
		t.s = s

		s.OnClosed = t.onClosed

		return nil
	}

	return ErrNoAvailableHosts
}

func (t *TCPSend) Close() error {
	t.shutdown = true

	t.t.Kill(nil)

	return t.t.Wait()
}

func (t *TCPSend) Flush() error {
	t.flush <- struct{}{}
	return nil
}

func (t *TCPSend) Ack(m *cypress.Message) {
	t.lock.Lock()
	defer t.lock.Unlock()

	t.outstanding--
}

func (t *TCPSend) Nack(m *cypress.Message) {
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

	var err error

	t.s.Close()

	for {
		err = t.Connect()
		if err != nil {
			if t.shutdown {
				t.lock.Unlock()
				return
			}

			time.Sleep(1 * time.Second)
			continue
		}

		break
	}

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

func (t *TCPSend) Receive(m *cypress.Message) error {
	t.newMessages <- m
	return nil
}

func (t *TCPSend) drain() error {
	for {
		select {
		case <-t.closed:
			t.reconnect()
		case <-t.flush:
			t.s.Flush()
		case m := <-t.newMessages:
			t.lock.Lock()
			t.outstanding++
			t.lock.Unlock()
			t.s.Send(m, t)
		case <-t.t.Dying():
			for {
				select {
				case m := <-t.newMessages:
					t.lock.Lock()
					t.outstanding++
					t.lock.Unlock()
					t.s.Send(m, t)
				default:
					t.s.Close()
					return nil
				}
			}
		}
	}
}

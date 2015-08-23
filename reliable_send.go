package cypress

import (
	"sort"
	"sync"
	"time"

	"gopkg.in/tomb.v2"
)

type Connector interface {
	Connect() (*Send, error)
}

type ReliableSend struct {
	connector Connector

	s *Send

	lock        sync.Mutex
	outstanding int

	newMessages chan *Message
	closed      chan bool

	flush    chan struct{}
	shutdown bool

	nacked Messages

	t tomb.Tomb
}

func NewReliableSend(c Connector, buffer int) *ReliableSend {
	return &ReliableSend{
		connector:   c,
		newMessages: make(chan *Message, buffer),
		closed:      make(chan bool, 1),
		flush:       make(chan struct{}),
	}
}

func (r *ReliableSend) Start() error {
	r.reconnect()

	r.t.Go(r.drain)

	return nil
}

func (r *ReliableSend) Close() error {
	r.shutdown = true

	r.t.Kill(nil)

	return r.t.Wait()
}

func (r *ReliableSend) Flush() error {
	r.flush <- struct{}{}
	return nil
}

func (r *ReliableSend) Outstanding() int {
	return r.outstanding
}

func (r *ReliableSend) Ack(m *Message) {
	r.lock.Lock()
	defer r.lock.Unlock()

	r.outstanding--
}

func (r *ReliableSend) Nack(m *Message) {
	r.lock.Lock()
	defer r.lock.Unlock()

	r.outstanding--
	r.nacked = append(r.nacked, m)
}

func (r *ReliableSend) onClosed() {
	r.closed <- true
}

func (r *ReliableSend) reconnect() {
	r.lock.Lock()

	var err error

	if r.s != nil {
		r.s.Close()
	}

	for {
		s, err := r.connector.Connect()
		if err != nil {
			if r.shutdown {
				r.lock.Unlock()
				return
			}

			time.Sleep(1 * time.Second)
			continue
		}

		s.OnClosed = r.onClosed
		r.s = s

		break
	}

	nacked := r.nacked
	r.nacked = nil

	r.lock.Unlock()

	for idx, msg := range nacked {
		r.outstanding++
		err = r.s.Send(msg, r)
		if err != nil {
			r.lock.Lock()
			r.nacked = append(nacked[idx+1:], r.nacked...)
			sort.Sort(r.nacked)

			// don't retry here because the OnClose handler will
			// prime the closed channel, so we return from here, pick
			// up the value from the channel, then this is called again.
			r.lock.Unlock()
			return
		}
	}
}

func (r *ReliableSend) Receive(m *Message) error {
	r.newMessages <- m
	return nil
}

func (r *ReliableSend) drain() error {
	for {
		select {
		case <-r.closed:
			r.reconnect()
		case <-r.flush:
			r.s.Flush()
		case m := <-r.newMessages:
			r.lock.Lock()
			r.outstanding++
			r.lock.Unlock()
			r.s.Send(m, r)
		case <-r.t.Dying():
			for {
				select {
				case m := <-r.newMessages:
					r.lock.Lock()
					r.outstanding++
					r.lock.Unlock()
					r.s.Send(m, r)
				default:
					r.s.Close()
					return nil
				}
			}
		}
	}
}

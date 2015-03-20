package cypress

import (
	"container/list"
	"errors"
	"io"
	"sync"
	"time"
)

type Send struct {
	OnClosed func()

	rw  io.ReadWriter
	enc *StreamEncoder
	buf []byte

	closed    bool
	window    int
	available int32
	reqs      *list.List

	ackLock sync.Mutex
	ackCond *sync.Cond
}

/*
* Note on window size: to maximize throughput, attempt to make this
* equation work: t * w = d * 2 or w =  d * 2 / t
*
* t = time between generated messages. Ie, if you're generating 1000
*     messages per second, t = 1ms
* w = the window size
* d = the transmission delay of the network
*
* So, t = 0.1ms and d = 0.05ms, then w = 2. This is the minimum window
* size to maximize throughput.
 */

// Given the transmission delay of the network (t) and the
// expected messages per second (mps), calculate the minimum
// window to use to maximize throughput.
func MinimumSendWindow(d time.Duration, mps int) int {
	t := time.Duration(mps/1000) * time.Millisecond

	return int((d * 20) / t)
}

var (
	// Disable windowing, acknowledge each message immediately
	NoWindow int = -1

	// An average messages/sec rate to calculate against
	DefaultMPSRate int = 1000

	// A decent minimum window that assures some improved throughput
	MinimumWindow = MinimumSendWindow(1*time.Millisecond, DefaultMPSRate)

	// A window for use on a fast lan where transmission delay is very small
	FastLanWindow = MinimumWindow

	// A window for use on a slower lan (cloud infrastructer, across AZ)
	SlowLanWindow = MinimumSendWindow(3*time.Millisecond, DefaultMPSRate)

	// A window for use over faster internet paths
	FastInternetWindow = MinimumSendWindow(10*time.Millisecond, DefaultMPSRate)

	// A window for use over slowe internet paths
	SlowInternetWindow = MinimumSendWindow(50*time.Millisecond, DefaultMPSRate)
)

func NewSend(rw io.ReadWriter, window int) *Send {
	switch window {
	case -1:
		window = 1
	case 0:
		window = MinimumWindow
	}

	s := &Send{
		rw:        rw,
		enc:       NewStreamEncoder(rw),
		buf:       make([]byte, window),
		window:    window,
		available: int32(window),
		reqs:      list.New(),
	}

	s.ackCond = sync.NewCond(&s.ackLock)

	go s.backgroundAck()

	return s
}

func (s *Send) SendHandshake() error {
	hdr := &StreamHeader{
		Compression: SNAPPY.Enum(),
		Mode:        StreamHeader_RELIABLE.Enum(),
	}

	return s.enc.WriteCustomHeader(hdr)
}

func (s *Send) transmit(m *Message) error {
	err := s.enc.Receive(m)
	if err != nil {
		s.sendNacks()
		return ErrClosed
	}

	return nil
}

var ErrStreamUnsynced = errors.New("stream unsynced")

type sendInFlight struct {
	req SendRequest
	m   *Message
}

func (s *Send) readAck() error {
	n, err := s.rw.Read(s.buf)
	if err != nil {
		return err
	}

	for i := 0; i < n; i++ {
		if s.buf[0] != 'k' {
			return ErrStreamUnsynced
		}

		f := s.reqs.Back()

		if f == nil {
			continue
		}

		if inf, ok := f.Value.(sendInFlight); ok {
			if inf.req != nil {
				inf.req.Ack(inf.m)
			}
		}

		s.reqs.Remove(f)
	}

	s.ackLock.Lock()
	s.available += int32(n)
	s.ackCond.Signal()
	s.ackLock.Unlock()

	return nil
}

func (s *Send) sendNacks() {
	if s.closed {
		return
	}

	for e := s.reqs.Front(); e != nil; e = e.Next() {
		if inf, ok := e.Value.(sendInFlight); ok {
			if inf.req != nil {
				inf.req.Nack(inf.m)
			}
		}
	}

	s.closed = true

	if s.OnClosed != nil {
		s.OnClosed()
	}

	s.ackCond.Signal()
}

func (s *Send) backgroundAck() {
	for {
		err := s.readAck()
		if err != nil {
			s.ackLock.Lock()
			defer s.ackLock.Unlock()

			s.sendNacks()
			return
		}
	}
}

func (s *Send) Receive(m *Message) error {
	return s.Send(m, nil)
}

var ErrClosed = errors.New("send closed")

func (s *Send) Send(m *Message, req SendRequest) error {
	s.ackLock.Lock()
	defer s.ackLock.Unlock()

	if s.closed {
		if req != nil {
			req.Nack(m)
		}

		return ErrClosed
	}

	s.reqs.PushFront(sendInFlight{req, m})

	s.available--

	err := s.transmit(m)
	if err != nil {
		return err
	}

	for s.available == 0 {
		if s.closed {
			return ErrClosed
		}

		s.ackCond.Wait()
	}

	return nil

}

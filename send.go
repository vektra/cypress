package cypress

import (
	"container/list"
	"errors"
	"io"
	"sync"
	"time"
)

// A type use to send a stream of Messages reliably. This type works in
// coordination with Recv to make transport the stream reliably by
// buffering and acking messages.
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

// Create a new Send, reading and writing from rw. Window controls
// the size of the ack window to use. See MinimumSendWindow and the Window
// variables for information window sizes. If the window is set to 0, the
// default window size is used.
// NOTE: The window size has a big effect on the throughput of Send, so
// be sure to consider it's value. The larger the window, the higher
// the memory usage and throughput. Fast lans only require a small window
// because there is a very small transmission delay.
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

// Send the start of a stream to the remote side. This will initialize
// the stream to use Snappy for compression and reliable transmission.
func (s *Send) SendHandshake() error {
	hdr := &StreamHeader{
		Compression: SNAPPY.Enum(),
		Mode:        StreamHeader_RELIABLE.Enum(),
	}

	return s.enc.WriteCustomHeader(hdr)
}

// Send the Message. If there is an error, nack the message so it can
// be sent again later.
func (s *Send) transmit(m *Message) error {
	err := s.enc.Receive(m)
	if err != nil {
		s.sendNacks()
		return ErrClosed
	}

	return nil
}

// Indicates that both sides of the stream have gotten confused and are
// no longer is sync.
var ErrStreamUnsynced = errors.New("stream unsynced")

// Used to track all messages that are currently not ack'd by the remote
// side.
type sendInFlight struct {
	req SendRequest
	m   *Message
}

// Read any acks from the stream and remove them from the requests list.
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

// Tell the sender about all the messages that it was not able to get
// acks about and thus should be resent.
func (s *Send) sendNacks() {
	if s.closed {
		return
	}

	for e := s.reqs.Back(); e != nil; e = e.Prev() {
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

// Read acks forever and if there is an error reading acks, nack all
// inflight requests.
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

// Send a Message to the remote side
func (s *Send) Receive(m *Message) error {
	return s.Send(m, nil)
}

// Indicate that this Send is closed and can not be used
var ErrClosed = errors.New("send closed")

// Send a Message to the remote side. if req is not nil, then
// it will be updated as to the status of m, calling either
// Ack or Nack depending on if things go ok or not.
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

package cypress

import (
	"errors"
	"io"
)

type Send struct {
	rw  io.ReadWriter
	enc *StreamEncoder
	buf []byte

	window    int
	available int
}

func NewSend(rw io.ReadWriter, window int) *Send {
	return &Send{
		rw:        rw,
		enc:       NewStreamEncoder(rw),
		buf:       make([]byte, window+1),
		window:    window,
		available: window,
	}
}

func (s *Send) SendHandshake() error {
	hdr := &StreamHeader{
		Compression: SNAPPY.Enum(),
		Mode:        StreamHeader_RELIABLE.Enum(),
	}

	return s.enc.WriteCustomHeader(hdr)
}

func (s *Send) transmit(m *Message) error {
	return s.enc.Receive(m)
}

var ErrStreamUnsynced = errors.New("stream unsynced")

func (s *Send) readAck() error {
	n, err := s.rw.Read(s.buf)
	if err != nil {
		return err
	}

	for i := 0; i < n; i++ {
		if s.buf[0] != 'k' {
			return ErrStreamUnsynced
		}
	}

	if s.window > 0 {
		s.available += n
	}

	return nil
}

func (s *Send) Receive(m *Message) error {
	err := s.transmit(m)
	if err != nil {
		return err
	}

	if s.available == 0 {
		return s.readAck()
	}

	s.available--

	return nil
}

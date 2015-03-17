package cypress

import (
	"errors"
	"io"
)

type Send struct {
	rw  io.ReadWriter
	enc *StreamEncoder
	buf []byte
}

func NewSend(rw io.ReadWriter) *Send {
	return &Send{rw, NewStreamEncoder(rw), make([]byte, 1)}
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
	_, err := s.rw.Read(s.buf)
	if err != nil {
		return err
	}

	if s.buf[0] != 'k' {
		return ErrStreamUnsynced
	}

	return nil
}

func (s *Send) Receive(m *Message) error {
	err := s.transmit(m)
	if err != nil {
		return err
	}

	return s.readAck()
}

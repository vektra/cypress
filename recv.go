package cypress

import (
	"io"
)

type Recv struct {
	rw  io.ReadWriter
	dec *StreamDecoder
}

func NewRecv(rw io.ReadWriter) (*Recv, error) {
	dec, err := NewStreamDecoder(rw)
	if err != nil {
		return nil, err
	}

	return &Recv{rw, dec}, nil
}

func (r *Recv) recvMessage() (*Message, error) {
	return r.dec.Generate()
}

var ReliableAckBytes = []byte{'k'}

func (r *Recv) sendAck() error {
	_, err := r.rw.Write(ReliableAckBytes)
	return err
}

func (r *Recv) Generate() (*Message, error) {
	m, err := r.recvMessage()
	if err != nil {
		return nil, err
	}

	if r.dec.Header.GetMode() == StreamHeader_RELIABLE {
		r.sendAck()
	}

	return m, nil
}

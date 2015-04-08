package cypress

import "io"

// A type which can recieve a stream of Messages reliabliy.
// Recv works in coordination with Send to reliablity send Messages
// using ack'ing.
type Recv struct {
	rw  io.ReadWriter
	dec *StreamDecoder
}

// Create a new Recv, reading and writing from rw.
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

// Generate a new Message reading from the stream. If the stream
// is in reliable mode (the default) then an ack is sent back.
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

// To satisify the Generator interface
func (r *Recv) Close() error {
	return nil
}

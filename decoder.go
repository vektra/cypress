package cypress

import (
	"encoding/binary"
	"io"
)

type Decoder struct {
	buf []byte
}

func NewDecoder() *Decoder {
	return &Decoder{
		buf: make([]byte, 1024),
	}
}

func (d *Decoder) DecodeFrom(r io.Reader) (*Message, error) {
	_, err := io.ReadFull(r, d.buf[:5])
	if err != nil {
		return nil, err
	}

	if d.buf[0] != '+' {
		return nil, ErrInvalidMessage
	}

	dataLen := binary.BigEndian.Uint32(d.buf[1:5])

	if uint32(len(d.buf)) < dataLen {
		d.buf = make([]byte, dataLen)
	}

	buf := d.buf[:dataLen]

	_, err = io.ReadFull(r, buf)
	if err != nil {
		return nil, err
	}

	m := &Message{}

	err = m.Unmarshal(buf)
	if err != nil {
		return nil, err
	}

	return m, nil
}

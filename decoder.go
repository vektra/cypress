package cypress

import (
	"bufio"
	"encoding/binary"
	"io"
)

type Decoder struct {
	r   *bufio.Reader
	buf []byte
}

func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{
		r:   bufio.NewReader(r),
		buf: make([]byte, 1024),
	}
}

func (d *Decoder) Decode() (*Message, error) {
	b, err := d.r.ReadByte()
	if err != nil {
		return nil, err
	}

	if b != '+' {
		return nil, ErrInvalidMessage
	}

	dataLen, err := binary.ReadUvarint(d.r)
	if err != nil {
		return nil, err
	}

	if uint64(len(d.buf)) < dataLen {
		d.buf = make([]byte, dataLen)
	}

	buf := d.buf[:dataLen]

	_, err = io.ReadFull(d.r, buf)
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

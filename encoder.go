package cypress

import (
	"encoding/binary"
	"io"
)

type Encoder struct {
	buf []byte
}

func NewEncoder() *Encoder {
	return &Encoder{
		make([]byte, 1024),
	}
}

func (e *Encoder) EncodeTo(m *Message, w io.Writer) (uint64, error) {
	sz := m.Size()

	e.buf[0] = '+'

	binary.BigEndian.PutUint32(e.buf[1:5], uint32(sz))

	_, err := w.Write(e.buf[:5])
	if err != nil {
		return 0, err
	}

	if len(e.buf) < sz {
		e.buf = make([]byte, sz)
	}

	cnt, err := m.MarshalTo(e.buf)
	if err != nil {
		return 0, err
	}

	_, err = w.Write(e.buf[:cnt])
	if err != nil {
		return 0, err
	}

	return uint64(sz) + 5, nil
}

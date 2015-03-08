package cypress

import (
	"encoding/binary"
	"io"
)

type Encoder struct {
	w   io.Writer
	buf []byte
}

func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{
		w:   w,
		buf: make([]byte, 1024),
	}
}

func (e *Encoder) Encode(m *Message) (uint64, error) {
	sz := m.Size()

	e.buf[0] = '+'

	cnt := binary.PutUvarint(e.buf[1:], uint64(sz))

	_, err := e.w.Write(e.buf[:cnt+1])
	if err != nil {
		return 0, err
	}

	if len(e.buf) < sz {
		e.buf = make([]byte, sz)
	}

	cnt, err = m.MarshalTo(e.buf)
	if err != nil {
		return 0, err
	}

	_, err = e.w.Write(e.buf[:cnt])
	if err != nil {
		return 0, err
	}

	return uint64(sz) + 5, nil
}

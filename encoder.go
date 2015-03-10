package cypress

import (
	"encoding/binary"
	"io"
)

type Encoder struct {
	w io.Writer
}

func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{
		w: w,
	}
}

func (e *Encoder) Encode(m *Message) (uint64, error) {
	sz := m.Size()

	buf := pbBufPool.Get().([]byte)

	buf[0] = '+'

	cnt := binary.PutUvarint(buf[1:], uint64(sz))

	_, err := e.w.Write(buf[:cnt+1])
	if err != nil {
		pbBufPool.Put(buf)
		return 0, err
	}

	if len(buf) < sz {
		buf = make([]byte, sz)
	}

	cnt, err = m.MarshalTo(buf)
	if err != nil {
		pbBufPool.Put(buf)
		return 0, err
	}

	_, err = e.w.Write(buf[:cnt])

	pbBufPool.Put(buf)

	if err != nil {
		return 0, err
	}

	return uint64(sz) + 5, nil
}

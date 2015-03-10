package cypress

import (
	"io"
	"sync"
)

var headerBufPool sync.Pool

func (h *StreamHeader) UnmarshalFrom(r io.Reader) error {
	var buf []byte

	val := headerBufPool.Get()
	if val == nil {
		buf = make([]byte, 128)
	} else {
		buf = val.([]byte)
	}

	size, err := ReadUvarint(r, buf)
	if err != nil {
		headerBufPool.Put(buf)
		return err
	}

	if len(buf) < int(size) {
		buf = make([]byte, size)
	}

	_, err = io.ReadFull(r, buf[:size])
	if err != nil {
		headerBufPool.Put(buf)
		return err
	}

	err = h.Unmarshal(buf[:size])
	headerBufPool.Put(buf)

	return err
}

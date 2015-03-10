package cypress

import "io"

func (h *StreamHeader) UnmarshalFrom(r io.Reader) error {
	buf := pbBufPool.Get().([]byte)

	size, err := ReadUvarint(r, buf)
	if err != nil {
		pbBufPool.Put(buf)
		return err
	}

	if len(buf) < int(size) {
		buf = make([]byte, size)
	}

	_, err = io.ReadFull(r, buf[:size])
	if err != nil {
		pbBufPool.Put(buf)
		return err
	}

	err = h.Unmarshal(buf[:size])
	pbBufPool.Put(buf)

	return err
}

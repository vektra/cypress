package cypress

import (
	"encoding/binary"
	"errors"
	"io"
)

var ErrOverflow = errors.New("overflow parsing varint")

func ReadUvarint(r io.Reader, buf []byte) (uint64, error) {
	var x uint64
	var s uint

	if buf == nil {
		buf = make([]byte, 1)
	}

	for i := 0; ; i++ {
		_, err := r.Read(buf[:1])
		if err != nil {
			return 0, err
		}

		b := buf[0]

		if b < 0x80 {
			if i > 9 || i == 9 && b > 1 {
				return 0, ErrOverflow
			}

			return x | uint64(b)<<s, nil
		}

		x |= uint64(b&0x7f) << s
		s += 7
	}
}

func WriteUvarint(w io.Writer, x uint64) (int, error) {
	var buf [10]byte

	cnt := binary.PutUvarint(buf[:], x)

	return w.Write(buf[:cnt])
}

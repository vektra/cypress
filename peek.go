package cypress

import "io"

type bytePeekReader struct {
	b    byte
	used bool

	r io.Reader
}

func (b *bytePeekReader) Read(buf []byte) (int, error) {
	if !b.used {
		b.used = true
		buf[0] = b.b
		buf = buf[1:]

		cnt, err := b.r.Read(buf)
		return cnt + 1, err
	}

	return b.r.Read(buf)
}

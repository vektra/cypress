package cypress

import "bytes"

type ByteBuffer struct {
	bytes.Buffer
}

func (bb *ByteBuffer) Close() error {
	return nil
}

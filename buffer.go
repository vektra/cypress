package cypress

import "sync"

var pbBufPool sync.Pool

func init() {
	pbBufPool.New = newBuf
}

func newBuf() interface{} {
	return make([]byte, 128)
}

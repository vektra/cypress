package cypress

import "sync"

var pbBufPool sync.Pool

func init() {
	pbBufPool.New = func() interface{} {
		return make([]byte, 128)
	}
}

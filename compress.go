package cypress

import (
	"compress/zlib"
	"io"

	"code.google.com/p/snappy-go/snappy"
)

var NONE = StreamHeader_NONE
var SNAPPY = StreamHeader_SNAPPY
var ZLIB = StreamHeader_ZLIB

func WriteCompressed(w io.Writer, comp StreamHeader_Compression) io.Writer {
	switch comp {
	case StreamHeader_NONE:
		return w
	case StreamHeader_SNAPPY:
		return snappy.NewWriter(w)
	case StreamHeader_ZLIB:
		return zlib.NewWriter(w)
	default:
		panic("unknown compression requested")
	}
}

func ReadCompressed(r io.Reader, comp StreamHeader_Compression) io.Reader {
	switch comp {
	case StreamHeader_NONE:
		return r
	case StreamHeader_SNAPPY:
		return snappy.NewReader(r)
	case StreamHeader_ZLIB:
		zw, err := zlib.NewReader(r)
		if err != nil {
			panic(err)
		}

		return zw
	default:
		panic("unknown compression requested")
	}
}

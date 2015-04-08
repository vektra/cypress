package cypress

import (
	"compress/zlib"
	"io"

	"code.google.com/p/snappy-go/snappy"
)

// No compression is applied
var NONE = StreamHeader_NONE

// Snappy compression is used
var SNAPPY = StreamHeader_SNAPPY

// ZLib compression is used
var ZLIB = StreamHeader_ZLIB

// Given a compression level, return a wrapped Writer
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

// Given a compression level, return a wrapped Reader
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

package cypress

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"io"

	"github.com/golang/snappy/snappy"
)

// No compression is applied
var NONE = StreamHeader_NONE

// Snappy compression is used
var SNAPPY = StreamHeader_SNAPPY

// ZLib compression is used
var ZLIB = StreamHeader_ZLIB

const defaultSnappyChunkBuffer = 1024 * 10

type snappyWriter struct {
	*snappy.Writer

	chunkBuffer int
	buf         bytes.Buffer
}

func (s *snappyWriter) Write(data []byte) (int, error) {
	n, err := s.buf.Write(data)
	if err != nil {
		return 0, err
	}

	if s.buf.Len() < s.chunkBuffer {
		return n, nil
	}

	_, err = s.Writer.Write(s.buf.Bytes())
	if err != nil {
		return 0, err
	}

	s.buf.Reset()

	return n, nil
}

func (s *snappyWriter) Flush() error {
	if s.buf.Len() > 0 {
		_, err := s.Writer.Write(s.buf.Bytes())
		s.buf.Reset()
		return err
	}

	return nil
}

func (s *snappyWriter) Close() error {
	return s.Flush()
}

type bufWriter struct {
	*bufio.Writer
}

func (b *bufWriter) Close() error {
	return b.Flush()
}

// Given a compression level, return a wrapped Writer
func WriteCompressed(w io.WriteCloser, comp StreamHeader_Compression) io.WriteCloser {
	switch comp {
	case StreamHeader_NONE:
		return &bufWriter{bufio.NewWriter(w)}
	case StreamHeader_SNAPPY:
		return &snappyWriter{
			Writer:      snappy.NewWriter(w),
			chunkBuffer: defaultSnappyChunkBuffer,
		}
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

package cypress

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"testing"

	"code.google.com/p/snappy-go/snappy"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektra/neko"
)

func TestStreamNegotiator(t *testing.T) {
	n := neko.Start(t)

	var (
		buf  *bytes.Buffer
		sink *logSink
		neg  *StreamNegotiator
	)

	n.Setup(func() {
		buf = new(bytes.Buffer)
		sink = &logSink{}
		neg = NewStreamNegotiator(buf, sink)
	})

	n.It("reads a stream and sorts out the message encoding", func() {
		buf.Write([]byte(`> greeting="hello"` + "\n"))
		buf.Write([]byte(`> greeting="hello"`))

		parser, err := neg.Negotiate()
		require.NoError(t, err)

		err = parser.Parse()
		require.NoError(t, err)

		greeting, ok := sink.Messages[0].GetString("greeting")
		require.True(t, ok)

		assert.Equal(t, greeting, "hello")

		greeting, ok = sink.Messages[1].GetString("greeting")
		require.True(t, ok)

		assert.Equal(t, greeting, "hello")
	})

	n.It("detects the presence of a header and at least strips it", func() {
		hdr := &StreamHeader{
			Compression: StreamHeader_NONE.Enum(),
		}

		buf.Write([]byte("-"))

		szbuf := make([]byte, 128)

		cnt := binary.PutUvarint(szbuf, uint64(hdr.Size()))

		buf.Write(szbuf[:cnt])

		data, err := hdr.Marshal()
		require.NoError(t, err)

		buf.Write(data)

		buf.Write([]byte(`> greeting="hello"` + "\n"))
		buf.Write([]byte(`> greeting="hello"`))

		parser, err := neg.Negotiate()
		require.NoError(t, err)

		err = parser.Parse()
		require.NoError(t, err)

		greeting, ok := sink.Messages[0].GetString("greeting")
		require.True(t, ok)

		assert.Equal(t, greeting, "hello")

		greeting, ok = sink.Messages[1].GetString("greeting")
		require.True(t, ok)

		assert.Equal(t, greeting, "hello")
	})

	n.It("detects the presence of a header and sets up snappy decompression", func() {
		hdr := &StreamHeader{
			Compression: StreamHeader_SNAPPY.Enum(),
		}

		buf.Write([]byte("-"))

		szbuf := make([]byte, 128)

		cnt := binary.PutUvarint(szbuf, uint64(hdr.Size()))

		buf.Write(szbuf[:cnt])

		data, err := hdr.Marshal()
		require.NoError(t, err)

		buf.Write(data)

		msg := []byte(`> greeting="hello"` + "\n")

		_, err = snappy.NewWriter(buf).Write(msg)
		require.NoError(t, err)

		parser, err := neg.Negotiate()
		require.NoError(t, err)

		err = parser.Parse()
		require.NoError(t, err)

		greeting, ok := sink.Messages[0].GetString("greeting")
		require.True(t, ok)

		assert.Equal(t, greeting, "hello")
	})

	n.It("detects the presence of a header and sets up zlib decompression", func() {
		hdr := &StreamHeader{
			Compression: StreamHeader_ZLIB.Enum(),
		}

		buf.Write([]byte("-"))

		szbuf := make([]byte, 128)

		cnt := binary.PutUvarint(szbuf, uint64(hdr.Size()))

		buf.Write(szbuf[:cnt])

		data, err := hdr.Marshal()
		require.NoError(t, err)

		buf.Write(data)

		msg := []byte(`> greeting="hello"` + "\n")

		zw := zlib.NewWriter(buf)

		_, err = zw.Write(msg)
		require.NoError(t, err)

		zw.Close()

		parser, err := neg.Negotiate()
		require.NoError(t, err)

		err = parser.Parse()
		require.NoError(t, err)

		greeting, ok := sink.Messages[0].GetString("greeting")
		require.True(t, ok)

		assert.Equal(t, greeting, "hello")
	})

	n.Meow()
}

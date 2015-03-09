package cypress

import (
	"bytes"
	"encoding/binary"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektra/neko"
)

func TestStreamEncoder(t *testing.T) {
	n := neko.Start(t)

	var (
		buf *bytes.Buffer
		se  *StreamEncoder
	)

	n.Setup(func() {
		buf = new(bytes.Buffer)
		se = NewStreamEncoder(buf)
	})

	n.It("writes a header", func() {
		err := se.WriteHeader(SNAPPY)
		require.NoError(t, err)

		hdr := &StreamHeader{}

		bt := buf.Bytes()

		assert.Equal(t, byte('-'), bt[0])

		val, cnt := binary.Uvarint(bt[1:])

		err = hdr.Unmarshal(bt[1+cnt : 1+cnt+int(val)])
		require.NoError(t, err)

		assert.Equal(t, StreamHeader_SNAPPY, hdr.GetCompression())
	})

	n.It("encodes a message to the output", func() {
		m := Log()
		m.Add("hello", "world")

		err := se.Receive(m)
		require.NoError(t, err)

		input := bytes.NewReader(buf.Bytes())

		dec := NewDecoder(input)

		m2, err := dec.Decode()
		require.NoError(t, err)

		assert.Equal(t, m, m2)
	})

	n.It("encodes a message to the output matching the stream's params", func() {
		err := se.Init(SNAPPY)
		require.NoError(t, err)

		m := Log()
		m.Add("hello", "world")

		err = se.Receive(m)
		require.NoError(t, err)

		input := bytes.NewReader(buf.Bytes())

		probe := NewProbe(input)

		err = probe.Probe()
		require.NoError(t, err)

		dec := NewDecoder(probe.Reader())

		m2, err := dec.Decode()
		require.NoError(t, err)

		assert.Equal(t, m, m2)
	})

	n.It("figures out the right settings for a file", func() {
		f, err := ioutil.TempFile("", "cypress")
		require.NoError(t, err)

		defer os.Remove(f.Name())

		err = se.Init(SNAPPY)
		require.NoError(t, err)

		_, err = f.Write(buf.Bytes())
		require.NoError(t, err)

		_, err = f.Seek(0, os.SEEK_SET)
		require.NoError(t, err)

		se = NewStreamEncoder(f)

		err = se.OpenFile(f)
		require.NoError(t, err)

		m := Log()
		m.Add("hello", "world")

		err = se.Receive(m)
		require.NoError(t, err)

		_, err = f.Seek(0, os.SEEK_SET)

		probe := NewProbe(f)

		err = probe.Probe()
		require.NoError(t, err)

		dec := NewDecoder(probe.Reader())

		m2, err := dec.Decode()
		require.NoError(t, err)

		assert.Equal(t, m, m2)
	})

	n.Meow()
}

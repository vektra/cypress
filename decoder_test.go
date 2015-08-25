package cypress

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektra/neko"
)

func TestDecoder(t *testing.T) {
	n := neko.Start(t)

	var (
		buf bytes.Buffer
		dec *Decoder
	)

	n.Setup(func() {
		dec = NewDecoder(&buf)
	})

	n.It("can decode a native format message", func() {
		m := Log()
		m.Add("hello", "world")

		buf.WriteString("+")

		data, err := m.Marshal()
		require.NoError(t, err)

		_, err = WriteUvarint(&buf, uint64(len(data)))
		require.NoError(t, err)

		buf.Write(data)

		m2, err := dec.Decode()
		require.NoError(t, err)

		assert.Equal(t, m, m2)
	})

	n.It("can decode a kv format message", func() {
		m := Log()
		m.Add("hello", "world")

		buf.WriteString(m.KVString())

		m2, err := dec.Decode()
		require.NoError(t, err)

		assert.Equal(t, m, m2)
	})

	n.It("can decode a json format message", func() {
		m := Log()
		m.Add("hello", "world")

		data, err := json.Marshal(m)
		require.NoError(t, err)

		buf.Write(data)

		m2, err := dec.Decode()
		require.NoError(t, err)

		assert.Equal(t, m, m2)
	})

	n.It("can decode unformatted text", func() {
		str := "this is some text that isn't formatted"
		buf.WriteString(str)
		buf.WriteByte('\n')

		m2, err := dec.Decode()
		require.NoError(t, err)

		t.Log(m2.KVString())

		out, ok := m2.GetString("message")
		require.True(t, ok)

		assert.Equal(t, str, out)
	})

	n.Meow()
}

package cypress

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektra/neko"
)

func TestStreamDecoder(t *testing.T) {
	n := neko.Start(t)

	var (
		buf ByteBuffer
	)

	n.It("can decode a stream created by stream encoder", func() {
		se := NewStreamEncoder(&buf)

		err := se.Init(SNAPPY)
		require.NoError(t, err)

		m := Log()
		m.Add("hello", "world")

		err = se.Receive(m)
		require.NoError(t, err)

		err = se.Close()
		require.NoError(t, err)

		sd, err := NewStreamDecoder(&buf)
		require.NoError(t, err)

		m2, err := sd.Generate()
		require.NoError(t, err)

		assert.Equal(t, m, m2)
	})

	n.Meow()
}

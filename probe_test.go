package cypress

import (
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektra/neko"
)

func TestProbe(t *testing.T) {
	n := neko.Start(t)

	n.It("can detect if there is a header and read it", func() {
		var buf bytes.Buffer

		enc := NewStreamEncoder(&buf)

		enc.WriteHeader(SNAPPY)

		probe := NewProbe(&buf)

		err := probe.Probe()
		require.NoError(t, err)

		assert.Equal(t, SNAPPY, probe.Compression())
	})

	n.It("sets up a stream to use after the probe with a header", func() {
		var buf bytes.Buffer

		enc := NewStreamEncoder(&buf)

		enc.WriteHeader(SNAPPY)

		probe := NewProbe(&buf)

		err := probe.Probe()
		require.NoError(t, err)

		assert.Equal(t, SNAPPY, probe.Compression())
	})

	n.It("sets up a stream to use after the probe with no header", func() {
		var buf bytes.Buffer

		buf.WriteString("{}\n")

		probe := NewProbe(&buf)

		err := probe.Probe()
		require.NoError(t, err)

		all, err := ioutil.ReadAll(probe.Stream)
		require.NoError(t, err)

		assert.Equal(t, "{}\n", string(all))
	})

	n.Meow()
}

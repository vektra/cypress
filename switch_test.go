package cypress

import (
	"bytes"
	"encoding/json"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektra/neko"
)

type logSink struct {
	mu       sync.Mutex
	Messages []*Message
}

func (v *logSink) Receive(m *Message) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.Messages = append(v.Messages, m)

	return nil
}

func TestSwitchStream(t *testing.T) {
	n := neko.Start(t)

	var sink *logSink
	var ss *SwitchStream
	var buf *bytes.Buffer

	n.Setup(func() {
		buf = new(bytes.Buffer)
		sink = &logSink{}
		ss = &SwitchStream{Input: buf, Out: sink}
	})

	n.It("detects and parses a kv message", func() {
		buf.Write([]byte(`> greeting="hello"` + "\n"))
		buf.Write([]byte(`> greeting="hello"`))
		err := ss.Parse()
		require.NoError(t, err)

		greeting, ok := sink.Messages[0].GetString("greeting")
		require.True(t, ok)

		assert.Equal(t, greeting, "hello")

		greeting, ok = sink.Messages[1].GetString("greeting")
		require.True(t, ok)

		assert.Equal(t, greeting, "hello")
	})

	n.It("detects and parses a protobuf message", func() {
		m := Log()
		m.Add("greeting", "hello")

		enc := NewEncoder()

		_, err := enc.EncodeTo(m, buf)
		require.NoError(t, err)

		_, err = enc.EncodeTo(m, buf)
		require.NoError(t, err)

		err = ss.Parse()
		require.NoError(t, err)

		greeting, ok := sink.Messages[0].GetString("greeting")
		require.True(t, ok)

		assert.Equal(t, greeting, "hello")

		greeting, ok = sink.Messages[1].GetString("greeting")
		require.True(t, ok)

		assert.Equal(t, greeting, "hello")
	})

	n.It("detects and parses a json message", func() {
		m := Log()
		m.Add("greeting", "hello")

		err := json.NewEncoder(buf).Encode(m)
		require.NoError(t, err)

		err = json.NewEncoder(buf).Encode(m)
		require.NoError(t, err)

		err = ss.Parse()
		require.NoError(t, err)

		greeting, ok := sink.Messages[0].GetString("greeting")
		require.True(t, ok)

		assert.Equal(t, greeting, "hello")

		greeting, ok = sink.Messages[1].GetString("greeting")
		require.True(t, ok)

		assert.Equal(t, greeting, "hello")
	})

	n.Meow()
}

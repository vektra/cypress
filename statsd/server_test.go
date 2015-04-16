package statsd

import (
	"bytes"
	"net"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektra/neko"
)

func TestServer(t *testing.T) {
	n := neko.Start(t)

	n.It("can parse a packet with one metric", func() {
		var buf bytes.Buffer

		buf.WriteString("fun:100|c")

		metrics, err := parsePacket(buf.Bytes())
		require.NoError(t, err)

		assert.Equal(t, COUNTER, metrics[0].Type)
		assert.Equal(t, "fun", metrics[0].Bucket)
		assert.Equal(t, float64(100), metrics[0].Value)
	})

	n.It("can parse a packet with multiple metrics", func() {
		var buf bytes.Buffer

		buf.WriteString("fun:100|c\nbar:88|ms")

		metrics, err := parsePacket(buf.Bytes())
		require.NoError(t, err)

		require.True(t, len(metrics) == 2)

		assert.Equal(t, COUNTER, metrics[0].Type)
		assert.Equal(t, "fun", metrics[0].Bucket)
		assert.Equal(t, float64(100), metrics[0].Value)

		assert.Equal(t, TIMER, metrics[1].Type)
		assert.Equal(t, "bar", metrics[1].Bucket)
		assert.Equal(t, float64(88), metrics[1].Value)
	})

	n.It("can deal with a metric terminating in a newline", func() {
		var buf bytes.Buffer

		buf.WriteString("fun:100|c\nbar:88|ms\n")

		metrics, err := parsePacket(buf.Bytes())
		require.NoError(t, err)

		require.True(t, len(metrics) == 2)

		assert.Equal(t, COUNTER, metrics[0].Type)
		assert.Equal(t, "fun", metrics[0].Bucket)
		assert.Equal(t, float64(100), metrics[0].Value)

		assert.Equal(t, TIMER, metrics[1].Type)
		assert.Equal(t, "bar", metrics[1].Bucket)
		assert.Equal(t, float64(88), metrics[1].Value)
	})

	n.It("listens on a udp socket and accepts packets", func() {
		var (
			s      Server
			metric *Metric
		)

		s.Addr = ":0"
		s.Handler = HandlerFunc(func(m *Metric) {
			metric = m
			s.Close()
		})

		err := s.Listen()
		require.NoError(t, err)

		c, err := net.Dial("udp", s.Addr)
		require.NoError(t, err)

		_, err = c.Write([]byte(`tests.run:2|c`))
		require.NoError(t, err)

		var wg sync.WaitGroup

		wg.Add(1)
		go func() {
			defer wg.Done()

			err := s.ListenAndReceive()
			require.NoError(t, err)
		}()

		wg.Wait()
	})

	n.Meow()
}

package statsd

import (
	"net"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektra/cypress"
	"github.com/vektra/neko"
)

func TestStatsd(t *testing.T) {
	n := neko.Start(t)

	n.It("converts statsd metrics into message", func() {
		var buf cypress.BufferReceiver

		s, err := NewStatsdEndpoint(&buf, ":0")
		require.NoError(t, err)

		var wg sync.WaitGroup

		wg.Add(1)
		go func() {
			defer wg.Done()
			s.Run()
		}()

		c, err := net.Dial("udp", s.Server.Addr)
		require.NoError(t, err)

		_, err = c.Write([]byte(`tests.run:12|c`))
		c.Close()

		// The Close() here is to get Run() to return, but there is a race
		// that cause the packet sent above to not get there before the
		// close so we sched to be sure it does.
		time.Sleep(10 * time.Millisecond)
		s.Server.Close()

		require.NoError(t, err)

		wg.Wait()

		m := buf.Messages[0]

		name, ok := m.GetString("name")
		require.True(t, ok)

		assert.Equal(t, "tests.run", name)

		typ, ok := m.GetString("type")
		require.True(t, ok)

		assert.Equal(t, "counter", typ)

		val, ok := m.GetFloat("value")
		require.True(t, ok)

		assert.Equal(t, 12, val)
	})

	n.It("converts a timer to an Internal", func() {
		var buf cypress.BufferReceiver

		s, err := NewStatsdEndpoint(&buf, ":0")
		require.NoError(t, err)

		var wg sync.WaitGroup

		wg.Add(1)
		go func() {
			defer wg.Done()
			s.Run()
		}()

		c, err := net.Dial("udp", s.Server.Addr)
		require.NoError(t, err)

		_, err = c.Write([]byte(`tests.run:83.1|ms`))
		c.Close()

		// The Close() here is to get Run() to return, but there is a race
		// that cause the packet sent above to not get there before the
		// close so we sched to be sure it does.
		runtime.Gosched()
		s.Server.Close()

		require.NoError(t, err)

		wg.Wait()

		m := buf.Messages[0]

		name, ok := m.GetString("name")
		require.True(t, ok)

		assert.Equal(t, "tests.run", name)

		typ, ok := m.GetString("type")
		require.True(t, ok)

		assert.Equal(t, "timer", typ)

		val, ok := m.GetInterval("value")
		require.True(t, ok)

		assert.Equal(t, 83100000, val.Duration())
	})

	n.Meow()
}
